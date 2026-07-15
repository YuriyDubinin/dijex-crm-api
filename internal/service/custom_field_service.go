package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/YuriyDubinin/bms-api/internal/domain"
)

const dateLayout = "2006-01-02"

// ───────────────────────── Service DTO ─────────────────────────

type CreateDefinitionInput struct {
	EntityType string
	FieldKey   string
	Label      string
	FieldType  string
	Options    json.RawMessage
	IsRequired bool
	SortOrder  int
}

type UpdateDefinitionInput struct {
	ID         uuid.UUID
	Label      string
	FieldType  string
	Options    json.RawMessage
	IsRequired bool
	SortOrder  int
}

type ListDefinitionsInput struct {
	Page       int
	PageSize   int
	EntityType string
	Order      string
}

type DefinitionItem struct {
	ID         uuid.UUID
	EntityType string
	FieldKey   string
	Label      string
	FieldType  string
	Options    json.RawMessage
	IsRequired bool
	SortOrder  int
	CreatedAt  time.Time
}

type ListDefinitionsOutput struct {
	Items      []DefinitionItem
	Pagination Pagination
}

type SetValueInput struct {
	EntityID          uuid.UUID
	FieldDefinitionID uuid.UUID
	Value             json.RawMessage
}

type ValueItem struct {
	FieldDefinitionID uuid.UUID
	Value             json.RawMessage
}

// ───────────────────────── Service ─────────────────────────

type CustomFieldService struct {
	repo     domain.CustomFieldRepository
	entities EntityResolver
	logger   *slog.Logger
	clock    func() time.Time
}

func NewCustomFieldService(repo domain.CustomFieldRepository, entities EntityResolver, logger *slog.Logger) *CustomFieldService {
	return &CustomFieldService{repo: repo, entities: entities, logger: logger, clock: time.Now}
}

func (s *CustomFieldService) CreateDefinition(ctx context.Context, customerID uuid.UUID, in CreateDefinitionInput) (*DefinitionItem, error) {
	in.EntityType = strings.ToUpper(strings.TrimSpace(in.EntityType))
	in.FieldKey = strings.TrimSpace(in.FieldKey)
	in.Label = strings.TrimSpace(in.Label)
	in.FieldType = strings.ToUpper(strings.TrimSpace(in.FieldType))

	if errs := validateDefinition(in.EntityType, in.FieldKey, in.Label, in.FieldType, in.Options); len(errs) > 0 {
		return nil, errs
	}

	d := &domain.CustomFieldDefinition{
		ID:         uuid.New(),
		CustomerID: customerID,
		EntityType: in.EntityType,
		FieldKey:   in.FieldKey,
		Label:      in.Label,
		FieldType:  in.FieldType,
		Options:    in.Options,
		IsRequired: in.IsRequired,
		SortOrder:  in.SortOrder,
	}
	if err := s.repo.CreateDefinition(ctx, d); err != nil {
		return nil, err
	}
	s.logger.Info("custom field definition created", "definition_id", d.ID, "customer_id", customerID)
	return defToItem(d), nil
}

func (s *CustomFieldService) UpdateDefinition(ctx context.Context, customerID uuid.UUID, in UpdateDefinitionInput) (*DefinitionItem, error) {
	in.Label = strings.TrimSpace(in.Label)
	in.FieldType = strings.ToUpper(strings.TrimSpace(in.FieldType))

	var errs domain.ValidationErrors
	if in.Label == "" || utf8.RuneCountInString(in.Label) > 255 {
		errs = append(errs, &domain.ValidationError{Field: "label", Message: "must be between 1 and 255 characters"})
	}
	if !domain.IsValidCustomFieldType(in.FieldType) {
		errs = append(errs, &domain.ValidationError{Field: "field_type", Message: "invalid custom field type"})
	}
	if len(in.Options) > 0 && !json.Valid(in.Options) {
		errs = append(errs, &domain.ValidationError{Field: "options", Message: "must be valid JSON"})
	}
	if len(errs) > 0 {
		return nil, errs
	}

	d := &domain.CustomFieldDefinition{
		ID:         in.ID,
		CustomerID: customerID,
		Label:      in.Label,
		FieldType:  in.FieldType,
		Options:    in.Options,
		IsRequired: in.IsRequired,
		SortOrder:  in.SortOrder,
	}
	if err := s.repo.UpdateDefinition(ctx, customerID, d); err != nil {
		return nil, err
	}
	s.logger.Info("custom field definition updated", "definition_id", d.ID, "customer_id", customerID)
	return defToItem(d), nil
}

func (s *CustomFieldService) ListDefinitions(ctx context.Context, customerID uuid.UUID, in ListDefinitionsInput) (*ListDefinitionsOutput, error) {
	entityType := strings.ToUpper(strings.TrimSpace(in.EntityType))
	if entityType != "" && !domain.IsValidEntityType(entityType) {
		return nil, domain.ValidationErrors{{Field: "entity_type", Message: "invalid entity type"}}
	}

	page, size, offset := clampPagination(in.Page, in.PageSize)
	filter := domain.CustomFieldDefinitionListFilter{
		EntityType: entityType,
		Limit:      size,
		Offset:     offset,
		SortDesc:   strings.ToLower(in.Order) == "desc",
	}
	items, total, err := s.repo.ListDefinitions(ctx, customerID, filter)
	if err != nil {
		return nil, err
	}
	out := &ListDefinitionsOutput{
		Items:      make([]DefinitionItem, 0, len(items)),
		Pagination: Pagination{Page: page, PageSize: size, Total: total, TotalPages: totalPages(total, size)},
	}
	for _, d := range items {
		out.Items = append(out.Items, *defToItem(d))
	}
	return out, nil
}

func (s *CustomFieldService) DeleteDefinition(ctx context.Context, customerID, id uuid.UUID) error {
	if err := s.repo.DeleteDefinition(ctx, customerID, id); err != nil {
		return err
	}
	s.logger.Info("custom field definition deleted", "definition_id", id, "customer_id", customerID)
	return nil
}

// SetValue проверяет определение и сущность (принадлежность тенанту + совпадение
// типа сущности), приводит значение к типу поля и делает upsert.
func (s *CustomFieldService) SetValue(ctx context.Context, customerID uuid.UUID, in SetValueInput) (*ValueItem, error) {
	def, err := s.repo.GetDefinitionByID(ctx, customerID, in.FieldDefinitionID)
	if err != nil {
		return nil, err // ErrNotFound → 404 (определение не найдено)
	}
	entityType, err := s.entities.ResolveType(ctx, customerID, in.EntityID)
	if err != nil {
		return nil, domain.ValidationErrors{{Field: "entity_id", Message: "entity not found in this tenant"}}
	}
	if string(entityType) != def.EntityType {
		return nil, domain.ValidationErrors{{Field: "entity_id", Message: "entity type does not match field definition"}}
	}

	v := &domain.CustomFieldValue{
		ID:                uuid.New(),
		EntityID:          in.EntityID,
		FieldDefinitionID: in.FieldDefinitionID,
	}
	if verr := coerceValue(def.FieldType, in.Value, v); verr != nil {
		return nil, domain.ValidationErrors{verr}
	}
	if err := s.repo.SetValue(ctx, customerID, v); err != nil {
		return nil, err
	}
	return &ValueItem{FieldDefinitionID: v.FieldDefinitionID, Value: valueToJSON(v)}, nil
}

func (s *CustomFieldService) ListValues(ctx context.Context, customerID, entityID uuid.UUID) ([]ValueItem, error) {
	if _, err := s.entities.ResolveType(ctx, customerID, entityID); err != nil {
		return nil, err // ErrNotFound → 404
	}
	values, err := s.repo.ListValues(ctx, customerID, entityID)
	if err != nil {
		return nil, err
	}
	out := make([]ValueItem, 0, len(values))
	for _, v := range values {
		out = append(out, ValueItem{FieldDefinitionID: v.FieldDefinitionID, Value: valueToJSON(v)})
	}
	return out, nil
}

func (s *CustomFieldService) UnsetValue(ctx context.Context, customerID, entityID, fieldDefinitionID uuid.UUID) error {
	return s.repo.UnsetValue(ctx, customerID, entityID, fieldDefinitionID)
}

// ───────────────────────── helpers ─────────────────────────

func validateDefinition(entityType, fieldKey, label, fieldType string, options json.RawMessage) domain.ValidationErrors {
	var errs domain.ValidationErrors
	if !domain.IsValidEntityType(entityType) {
		errs = append(errs, &domain.ValidationError{Field: "entity_type", Message: "invalid entity type"})
	}
	if fieldKey == "" || utf8.RuneCountInString(fieldKey) > 100 {
		errs = append(errs, &domain.ValidationError{Field: "field_key", Message: "must be between 1 and 100 characters"})
	}
	if label == "" || utf8.RuneCountInString(label) > 255 {
		errs = append(errs, &domain.ValidationError{Field: "label", Message: "must be between 1 and 255 characters"})
	}
	if !domain.IsValidCustomFieldType(fieldType) {
		errs = append(errs, &domain.ValidationError{Field: "field_type", Message: "invalid custom field type"})
	}
	if len(options) > 0 && !json.Valid(options) {
		errs = append(errs, &domain.ValidationError{Field: "options", Message: "must be valid JSON"})
	}
	return errs
}

// coerceValue разбирает JSON-значение согласно типу поля и заполняет ровно одну
// value_*-колонку в v. Ошибка формата → *domain.ValidationError по полю "value".
func coerceValue(fieldType string, raw json.RawMessage, v *domain.CustomFieldValue) *domain.ValidationError {
	if len(raw) == 0 || string(raw) == "null" {
		return &domain.ValidationError{Field: "value", Message: "is required"}
	}
	verr := func(msg string) *domain.ValidationError { return &domain.ValidationError{Field: "value", Message: msg} }

	switch fieldType {
	case domain.CustomFieldTypeText, domain.CustomFieldTypeSelect:
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return verr("must be a string")
		}
		v.ValueText = &s
	case domain.CustomFieldTypeNumber:
		var n float64
		if err := json.Unmarshal(raw, &n); err != nil {
			return verr("must be a number")
		}
		v.ValueNumber = &n
	case domain.CustomFieldTypeBoolean:
		var b bool
		if err := json.Unmarshal(raw, &b); err != nil {
			return verr("must be a boolean")
		}
		v.ValueBoolean = &b
	case domain.CustomFieldTypeDate:
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return verr("must be a date string YYYY-MM-DD")
		}
		t, err := time.Parse(dateLayout, s)
		if err != nil {
			return verr("must be a date string YYYY-MM-DD")
		}
		v.ValueDate = &t
	case domain.CustomFieldTypeDatetime:
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return verr("must be an RFC3339 datetime string")
		}
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return verr("must be an RFC3339 datetime string")
		}
		v.ValueDatetime = &t
	case domain.CustomFieldTypeJSON:
		if !json.Valid(raw) {
			return verr("must be valid JSON")
		}
		v.ValueJSON = raw
	default:
		return verr("unsupported field type")
	}
	return nil
}

// valueToJSON собирает хранимое значение обратно в JSON согласно заполненной колонке.
func valueToJSON(v *domain.CustomFieldValue) json.RawMessage {
	switch {
	case v.ValueText != nil:
		b, _ := json.Marshal(*v.ValueText)
		return b
	case v.ValueNumber != nil:
		b, _ := json.Marshal(*v.ValueNumber)
		return b
	case v.ValueBoolean != nil:
		b, _ := json.Marshal(*v.ValueBoolean)
		return b
	case v.ValueDate != nil:
		b, _ := json.Marshal(v.ValueDate.Format(dateLayout))
		return b
	case v.ValueDatetime != nil:
		b, _ := json.Marshal(v.ValueDatetime.Format(time.RFC3339))
		return b
	case len(v.ValueJSON) > 0:
		return v.ValueJSON
	default:
		return json.RawMessage("null")
	}
}

func defToItem(d *domain.CustomFieldDefinition) *DefinitionItem {
	return &DefinitionItem{
		ID:         d.ID,
		EntityType: d.EntityType,
		FieldKey:   d.FieldKey,
		Label:      d.Label,
		FieldType:  d.FieldType,
		Options:    d.Options,
		IsRequired: d.IsRequired,
		SortOrder:  d.SortOrder,
		CreatedAt:  d.CreatedAt,
	}
}
