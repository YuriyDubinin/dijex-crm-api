package dto

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/YuriyDubinin/bms-api/internal/service"
)

type CreateLinkHTTPRequest struct {
	SrcEntityID  uuid.UUID       `json:"src_entity_id" validate:"required"`
	DstEntityID  uuid.UUID       `json:"dst_entity_id" validate:"required"`
	RelationType string          `json:"relation_type" validate:"omitempty,max=100"`
	Attributes   json.RawMessage `json:"attributes"    validate:"omitempty"`
}

func (r CreateLinkHTTPRequest) ToServiceInput() service.CreateLinkInput {
	return service.CreateLinkInput{
		SrcEntityID:  r.SrcEntityID,
		DstEntityID:  r.DstEntityID,
		RelationType: r.RelationType,
		Attributes:   r.Attributes,
	}
}

type LinkHTTPResponse struct {
	LinkID         uuid.UUID       `json:"link_id"`
	EntityID       uuid.UUID       `json:"entity_id"`
	LinkedEntityID uuid.UUID       `json:"linked_entity_id"`
	LinkedType     string          `json:"linked_type"`
	LinkedLabel    string          `json:"linked_label"`
	RelationType   string          `json:"relation_type"`
	Attributes     json.RawMessage `json:"attributes"`
	CreatedAt      time.Time       `json:"created_at"`
}

func fromLinkItem(l service.LinkItem) LinkHTTPResponse {
	attrs := l.Attributes
	if len(attrs) == 0 {
		attrs = json.RawMessage(`{}`)
	}
	return LinkHTTPResponse{
		LinkID:         l.LinkID,
		EntityID:       l.EntityID,
		LinkedEntityID: l.LinkedEntityID,
		LinkedType:     l.LinkedType,
		LinkedLabel:    l.LinkedLabel,
		RelationType:   l.RelationType,
		Attributes:     attrs,
		CreatedAt:      l.CreatedAt,
	}
}

func FromLinkItem(l *service.LinkItem) LinkHTTPResponse {
	return fromLinkItem(*l)
}

type ListLinksHTTPResponse struct {
	Items      []LinkHTTPResponse `json:"items"`
	Pagination PaginationResponse `json:"pagination"`
}

func FromListLinksOutput(o *service.ListLinksOutput) ListLinksHTTPResponse {
	items := make([]LinkHTTPResponse, 0, len(o.Items))
	for _, l := range o.Items {
		items = append(items, fromLinkItem(l))
	}
	return ListLinksHTTPResponse{Items: items, Pagination: fromPagination(o.Pagination)}
}

type DeleteLinkHTTPResponse struct {
	ID uuid.UUID `json:"id"`
}

func FromDeleteLinkOutput(o *service.DeleteLinkOutput) DeleteLinkHTTPResponse {
	return DeleteLinkHTTPResponse{ID: o.ID}
}
