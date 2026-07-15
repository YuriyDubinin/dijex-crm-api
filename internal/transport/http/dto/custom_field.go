package dto

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/YuriyDubinin/bms-api/internal/service"
)

// ───────────────────────── Определения ─────────────────────────

type CreateDefinitionHTTPRequest struct {
	EntityType string          `json:"entity_type" validate:"required,oneof=PROJECT SERVICE CLIENT TASK CALENDAR_EVENT DEAL CUSTOMER_USER"`
	FieldKey   string          `json:"field_key"   validate:"required,max=100"`
	Label      string          `json:"label"       validate:"required,max=255"`
	FieldType  string          `json:"field_type"  validate:"required,oneof=TEXT NUMBER BOOLEAN DATE DATETIME JSON SELECT"`
	Options    json.RawMessage `json:"options"     validate:"omitempty"`
	IsRequired bool            `json:"is_required"`
	SortOrder  int             `json:"sort_order"`
}

func (r CreateDefinitionHTTPRequest) ToServiceInput() service.CreateDefinitionInput {
	return service.CreateDefinitionInput{
		EntityType: r.EntityType,
		FieldKey:   r.FieldKey,
		Label:      r.Label,
		FieldType:  r.FieldType,
		Options:    r.Options,
		IsRequired: r.IsRequired,
		SortOrder:  r.SortOrder,
	}
}

type UpdateDefinitionHTTPRequest struct {
	ID         uuid.UUID       `json:"id"          validate:"required"`
	Label      string          `json:"label"       validate:"required,max=255"`
	FieldType  string          `json:"field_type"  validate:"required,oneof=TEXT NUMBER BOOLEAN DATE DATETIME JSON SELECT"`
	Options    json.RawMessage `json:"options"     validate:"omitempty"`
	IsRequired bool            `json:"is_required"`
	SortOrder  int             `json:"sort_order"`
}

func (r UpdateDefinitionHTTPRequest) ToServiceInput() service.UpdateDefinitionInput {
	return service.UpdateDefinitionInput{
		ID:         r.ID,
		Label:      r.Label,
		FieldType:  r.FieldType,
		Options:    r.Options,
		IsRequired: r.IsRequired,
		SortOrder:  r.SortOrder,
	}
}

type DefinitionHTTPResponse struct {
	ID         uuid.UUID       `json:"id"`
	EntityType string          `json:"entity_type"`
	FieldKey   string          `json:"field_key"`
	Label      string          `json:"label"`
	FieldType  string          `json:"field_type"`
	Options    json.RawMessage `json:"options"`
	IsRequired bool            `json:"is_required"`
	SortOrder  int             `json:"sort_order"`
	CreatedAt  time.Time       `json:"created_at"`
}

func fromDefinitionItem(d service.DefinitionItem) DefinitionHTTPResponse {
	return DefinitionHTTPResponse{
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

func FromDefinitionItem(d *service.DefinitionItem) DefinitionHTTPResponse {
	return fromDefinitionItem(*d)
}

type ListDefinitionsHTTPResponse struct {
	Items      []DefinitionHTTPResponse `json:"items"`
	Pagination PaginationResponse       `json:"pagination"`
}

func FromListDefinitionsOutput(o *service.ListDefinitionsOutput) ListDefinitionsHTTPResponse {
	items := make([]DefinitionHTTPResponse, 0, len(o.Items))
	for _, d := range o.Items {
		items = append(items, fromDefinitionItem(d))
	}
	return ListDefinitionsHTTPResponse{Items: items, Pagination: fromPagination(o.Pagination)}
}

// ───────────────────────── Значения ─────────────────────────

type SetValueHTTPRequest struct {
	EntityID          uuid.UUID       `json:"entity_id"           validate:"required"`
	FieldDefinitionID uuid.UUID       `json:"field_definition_id" validate:"required"`
	Value             json.RawMessage `json:"value"               validate:"required"`
}

func (r SetValueHTTPRequest) ToServiceInput() service.SetValueInput {
	return service.SetValueInput{
		EntityID:          r.EntityID,
		FieldDefinitionID: r.FieldDefinitionID,
		Value:             r.Value,
	}
}

type UnsetValueHTTPRequest struct {
	EntityID          uuid.UUID `json:"entity_id"           validate:"required"`
	FieldDefinitionID uuid.UUID `json:"field_definition_id" validate:"required"`
}

type ValueHTTPResponse struct {
	FieldDefinitionID uuid.UUID       `json:"field_definition_id"`
	Value             json.RawMessage `json:"value"`
}

func FromValueItem(v *service.ValueItem) ValueHTTPResponse {
	return ValueHTTPResponse{FieldDefinitionID: v.FieldDefinitionID, Value: v.Value}
}

type ListValuesHTTPResponse struct {
	Items []ValueHTTPResponse `json:"items"`
}

func FromValueItems(items []service.ValueItem) ListValuesHTTPResponse {
	out := make([]ValueHTTPResponse, 0, len(items))
	for i := range items {
		out = append(out, FromValueItem(&items[i]))
	}
	return ListValuesHTTPResponse{Items: out}
}
