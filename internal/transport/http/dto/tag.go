package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/YuriyDubinin/bms-api/internal/service"
)

type CreateTagHTTPRequest struct {
	Name  string `json:"name"  validate:"required,min=1,max=100"`
	Color string `json:"color" validate:"omitempty,max=32"`
}

func (r CreateTagHTTPRequest) ToServiceInput() service.CreateTagInput {
	return service.CreateTagInput{
		Name:  r.Name,
		Color: r.Color,
	}
}

type UpdateTagHTTPRequest struct {
	ID    uuid.UUID `json:"id"    validate:"required"`
	Name  string    `json:"name"  validate:"required,min=1,max=100"`
	Color string    `json:"color" validate:"omitempty,max=32"`
}

func (r UpdateTagHTTPRequest) ToServiceInput() service.UpdateTagInput {
	return service.UpdateTagInput{
		ID:    r.ID,
		Name:  r.Name,
		Color: r.Color,
	}
}

// TagEntityHTTPRequest — тело запроса привязки/отвязки тега к сущности.
// Используется и для attach, и для detach.
type TagEntityHTTPRequest struct {
	TagID    uuid.UUID `json:"tag_id"    validate:"required"`
	EntityID uuid.UUID `json:"entity_id" validate:"required"`
}

type TagHTTPResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Color     string    `json:"color"`
	CreatedAt time.Time `json:"created_at"`
}

func fromTagItem(t service.TagItem) TagHTTPResponse {
	return TagHTTPResponse{
		ID:        t.ID,
		Name:      t.Name,
		Color:     t.Color,
		CreatedAt: t.CreatedAt,
	}
}

func FromTagItem(t *service.TagItem) TagHTTPResponse {
	return fromTagItem(*t)
}

type ListTagsHTTPResponse struct {
	Items      []TagHTTPResponse  `json:"items"`
	Pagination PaginationResponse `json:"pagination"`
}

func FromListTagsOutput(o *service.ListTagsOutput) ListTagsHTTPResponse {
	items := make([]TagHTTPResponse, 0, len(o.Items))
	for _, t := range o.Items {
		items = append(items, fromTagItem(t))
	}
	return ListTagsHTTPResponse{Items: items, Pagination: fromPagination(o.Pagination)}
}

type DeleteTagHTTPResponse struct {
	ID uuid.UUID `json:"id"`
}

func FromDeleteTagOutput(o *service.DeleteTagOutput) DeleteTagHTTPResponse {
	return DeleteTagHTTPResponse{ID: o.ID}
}
