package dto

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/YuriyDubinin/bms-api/internal/service"
)

type CreateProjectHTTPRequest struct {
	Name        string          `json:"name"        validate:"required,min=2,max=255"`
	Slug        string          `json:"slug"        validate:"omitempty,max=100"`
	Direction   string          `json:"direction"   validate:"omitempty,max=255"`
	Description string          `json:"description" validate:"omitempty,max=5000"`
	Status      string          `json:"status"      validate:"omitempty,oneof=DRAFT ACTIVE ON_HOLD ARCHIVED"`
	Attributes  json.RawMessage `json:"attributes"  validate:"omitempty"`
	StartsAt    *string         `json:"starts_at"   validate:"omitempty,datetime=2006-01-02"`
	EndsAt      *string         `json:"ends_at"     validate:"omitempty,datetime=2006-01-02"`
}

func (r CreateProjectHTTPRequest) ToServiceInput() service.CreateProjectInput {
	return service.CreateProjectInput{
		Name:        r.Name,
		Slug:        r.Slug,
		Direction:   r.Direction,
		Description: r.Description,
		Status:      r.Status,
		Attributes:  r.Attributes,
		StartsAt:    parseDatePtr(r.StartsAt),
		EndsAt:      parseDatePtr(r.EndsAt),
	}
}

type UpdateProjectHTTPRequest struct {
	ID          uuid.UUID       `json:"id"          validate:"required"`
	Name        string          `json:"name"        validate:"required,min=2,max=255"`
	Slug        string          `json:"slug"        validate:"omitempty,max=100"`
	Direction   string          `json:"direction"   validate:"omitempty,max=255"`
	Description string          `json:"description" validate:"omitempty,max=5000"`
	Status      string          `json:"status"      validate:"omitempty,oneof=DRAFT ACTIVE ON_HOLD ARCHIVED"`
	Attributes  json.RawMessage `json:"attributes"  validate:"omitempty"`
	StartsAt    *string         `json:"starts_at"   validate:"omitempty,datetime=2006-01-02"`
	EndsAt      *string         `json:"ends_at"     validate:"omitempty,datetime=2006-01-02"`
}

func (r UpdateProjectHTTPRequest) ToServiceInput() service.UpdateProjectInput {
	return service.UpdateProjectInput{
		ID:          r.ID,
		Name:        r.Name,
		Slug:        r.Slug,
		Direction:   r.Direction,
		Description: r.Description,
		Status:      r.Status,
		Attributes:  r.Attributes,
		StartsAt:    parseDatePtr(r.StartsAt),
		EndsAt:      parseDatePtr(r.EndsAt),
	}
}

type IDHTTPRequest struct {
	ID uuid.UUID `json:"id" validate:"required"`
}

type ProjectHTTPResponse struct {
	ID          uuid.UUID       `json:"id"`
	Name        string          `json:"name"`
	Slug        string          `json:"slug"`
	Direction   string          `json:"direction"`
	Description string          `json:"description"`
	Status      string          `json:"status"`
	Attributes  json.RawMessage `json:"attributes"`
	StartsAt    *string         `json:"starts_at"`
	EndsAt      *string         `json:"ends_at"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

func fromProjectItem(p service.ProjectItem) ProjectHTTPResponse {
	attrs := p.Attributes
	if len(attrs) == 0 {
		attrs = json.RawMessage(`{}`)
	}
	return ProjectHTTPResponse{
		ID:          p.ID,
		Name:        p.Name,
		Slug:        p.Slug,
		Direction:   p.Direction,
		Description: p.Description,
		Status:      p.Status,
		Attributes:  attrs,
		StartsAt:    formatDatePtr(p.StartsAt),
		EndsAt:      formatDatePtr(p.EndsAt),
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

func FromProjectItem(p *service.ProjectItem) ProjectHTTPResponse {
	return fromProjectItem(*p)
}

type ListProjectsHTTPResponse struct {
	Items      []ProjectHTTPResponse `json:"items"`
	Pagination PaginationResponse    `json:"pagination"`
}

func FromListProjectsOutput(o *service.ListProjectsOutput) ListProjectsHTTPResponse {
	items := make([]ProjectHTTPResponse, 0, len(o.Items))
	for _, p := range o.Items {
		items = append(items, fromProjectItem(p))
	}
	return ListProjectsHTTPResponse{Items: items, Pagination: fromPagination(o.Pagination)}
}

type DeleteProjectHTTPResponse struct {
	ID        uuid.UUID `json:"id"`
	DeletedAt time.Time `json:"deleted_at"`
}

func FromDeleteProjectOutput(o *service.DeleteProjectOutput) DeleteProjectHTTPResponse {
	return DeleteProjectHTTPResponse{ID: o.ID, DeletedAt: o.DeletedAt}
}
