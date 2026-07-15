package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/YuriyDubinin/bms-api/internal/domain"
)

// ───────────────────────── Service DTO ─────────────────────────

type CreateLinkInput struct {
	SrcEntityID  uuid.UUID
	DstEntityID  uuid.UUID
	RelationType string
	Attributes   json.RawMessage
	CreatedBy    uuid.UUID
}

type ListLinksInput struct {
	Page         int
	PageSize     int
	EntityID     *uuid.UUID
	RelationType string
	LinkedType   string
	Order        string
}

type LinkItem struct {
	LinkID         uuid.UUID
	EntityID       uuid.UUID
	LinkedEntityID uuid.UUID
	LinkedType     string
	LinkedLabel    string
	RelationType   string
	Attributes     json.RawMessage
	CreatedAt      time.Time
}

type ListLinksOutput struct {
	Items      []LinkItem
	Pagination Pagination
}

type DeleteLinkOutput struct {
	ID uuid.UUID
}

// ───────────────────────── Service ─────────────────────────

type EntityLinkService struct {
	repo     domain.EntityLinkRepository
	entities EntityResolver
	logger   *slog.Logger
	clock    func() time.Time
}

func NewEntityLinkService(repo domain.EntityLinkRepository, entities EntityResolver, logger *slog.Logger) *EntityLinkService {
	return &EntityLinkService{repo: repo, entities: entities, logger: logger, clock: time.Now}
}

func (s *EntityLinkService) CreateLink(ctx context.Context, customerID uuid.UUID, in CreateLinkInput) (*LinkItem, error) {
	relation := strings.ToUpper(strings.TrimSpace(in.RelationType))
	if relation == "" {
		relation = domain.RelationTypeDefault
	}

	var errs domain.ValidationErrors
	if in.SrcEntityID == in.DstEntityID {
		errs = append(errs, &domain.ValidationError{Field: "dst_entity_id", Message: "cannot link an entity to itself"})
	}
	if !validAttributes(in.Attributes) {
		errs = append(errs, attrError("attributes"))
	}
	// Обе стороны обязаны существовать и принадлежать тенанту.
	if _, err := s.entities.ResolveType(ctx, customerID, in.SrcEntityID); err != nil {
		errs = append(errs, &domain.ValidationError{Field: "src_entity_id", Message: "entity not found in this tenant"})
	}
	dstType, err := s.entities.ResolveType(ctx, customerID, in.DstEntityID)
	if err != nil {
		errs = append(errs, &domain.ValidationError{Field: "dst_entity_id", Message: "entity not found in this tenant"})
	}
	if len(errs) > 0 {
		return nil, errs
	}

	createdBy := in.CreatedBy
	l := &domain.EntityLink{
		ID:           uuid.New(),
		CustomerID:   customerID,
		SrcEntityID:  in.SrcEntityID,
		DstEntityID:  in.DstEntityID,
		RelationType: relation,
		Attributes:   in.Attributes,
		CreatedBy:    &createdBy,
	}
	if err := s.repo.Create(ctx, l); err != nil {
		return nil, err
	}
	s.logger.Info("entity link created", "link_id", l.ID, "customer_id", customerID, "relation", relation)

	return &LinkItem{
		LinkID:         l.ID,
		EntityID:       l.SrcEntityID,
		LinkedEntityID: l.DstEntityID,
		LinkedType:     string(dstType),
		RelationType:   l.RelationType,
		Attributes:     l.Attributes,
		CreatedAt:      l.CreatedAt,
	}, nil
}

func (s *EntityLinkService) ListLinks(ctx context.Context, customerID uuid.UUID, in ListLinksInput) (*ListLinksOutput, error) {
	linkedType := strings.ToUpper(strings.TrimSpace(in.LinkedType))
	if linkedType != "" && !domain.IsValidEntityType(linkedType) {
		return nil, domain.ValidationErrors{{Field: "linked_type", Message: "invalid entity type"}}
	}

	page, size, offset := clampPagination(in.Page, in.PageSize)
	filter := domain.EntityLinkListFilter{
		EntityID:     in.EntityID,
		RelationType: strings.ToUpper(strings.TrimSpace(in.RelationType)),
		LinkedType:   linkedType,
		Limit:        size,
		Offset:       offset,
		SortDesc:     strings.ToLower(in.Order) != "asc",
	}

	items, total, err := s.repo.List(ctx, customerID, filter)
	if err != nil {
		return nil, err
	}
	out := &ListLinksOutput{
		Items:      make([]LinkItem, 0, len(items)),
		Pagination: Pagination{Page: page, PageSize: size, Total: total, TotalPages: totalPages(total, size)},
	}
	for _, le := range items {
		out.Items = append(out.Items, LinkItem{
			LinkID:         le.LinkID,
			EntityID:       le.EntityID,
			LinkedEntityID: le.LinkedEntityID,
			LinkedType:     le.LinkedType,
			LinkedLabel:    le.LinkedLabel,
			RelationType:   le.RelationType,
			Attributes:     le.Attributes,
			CreatedAt:      le.CreatedAt,
		})
	}
	return out, nil
}

func (s *EntityLinkService) DeleteLink(ctx context.Context, customerID, id uuid.UUID) (*DeleteLinkOutput, error) {
	if err := s.repo.Delete(ctx, customerID, id); err != nil {
		return nil, err
	}
	s.logger.Info("entity link deleted", "link_id", id, "customer_id", customerID)
	return &DeleteLinkOutput{ID: id}, nil
}
