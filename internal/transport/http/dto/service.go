package dto

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/YuriyDubinin/bms-api/internal/service"
)

type CreateServiceHTTPRequest struct {
	ProjectID   *uuid.UUID      `json:"project_id"   validate:"omitempty"`
	Name        string          `json:"name"         validate:"required,min=2,max=255"`
	Category    string          `json:"category"     validate:"omitempty,max=255"`
	Description string          `json:"description"  validate:"omitempty,max=5000"`
	Price       *float64        `json:"price"        validate:"omitempty,gte=0"`
	Currency    string          `json:"currency"     validate:"omitempty,len=3,alpha"`
	DurationMin *int            `json:"duration_min" validate:"omitempty,min=1"`
	Status      string          `json:"status"       validate:"omitempty,oneof=ACTIVE INACTIVE ARCHIVED"`
	Attributes  json.RawMessage `json:"attributes"   validate:"omitempty"`
}

func (r CreateServiceHTTPRequest) ToServiceInput() service.CreateServiceInput {
	return service.CreateServiceInput{
		ProjectID:   r.ProjectID,
		Name:        r.Name,
		Category:    r.Category,
		Description: r.Description,
		Price:       r.Price,
		Currency:    r.Currency,
		DurationMin: r.DurationMin,
		Status:      r.Status,
		Attributes:  r.Attributes,
	}
}

type UpdateServiceHTTPRequest struct {
	ID          uuid.UUID       `json:"id"           validate:"required"`
	ProjectID   *uuid.UUID      `json:"project_id"   validate:"omitempty"`
	Name        string          `json:"name"         validate:"required,min=2,max=255"`
	Category    string          `json:"category"     validate:"omitempty,max=255"`
	Description string          `json:"description"  validate:"omitempty,max=5000"`
	Price       *float64        `json:"price"        validate:"omitempty,gte=0"`
	Currency    string          `json:"currency"     validate:"omitempty,len=3,alpha"`
	DurationMin *int            `json:"duration_min" validate:"omitempty,min=1"`
	Status      string          `json:"status"       validate:"omitempty,oneof=ACTIVE INACTIVE ARCHIVED"`
	Attributes  json.RawMessage `json:"attributes"   validate:"omitempty"`
}

func (r UpdateServiceHTTPRequest) ToServiceInput() service.UpdateServiceInput {
	return service.UpdateServiceInput{
		ID:          r.ID,
		ProjectID:   r.ProjectID,
		Name:        r.Name,
		Category:    r.Category,
		Description: r.Description,
		Price:       r.Price,
		Currency:    r.Currency,
		DurationMin: r.DurationMin,
		Status:      r.Status,
		Attributes:  r.Attributes,
	}
}

type ServiceHTTPResponse struct {
	ID          uuid.UUID       `json:"id"`
	ProjectID   *uuid.UUID      `json:"project_id"`
	Name        string          `json:"name"`
	Category    string          `json:"category"`
	Description string          `json:"description"`
	Price       *float64        `json:"price"`
	Currency    string          `json:"currency"`
	DurationMin *int            `json:"duration_min"`
	Status      string          `json:"status"`
	Attributes  json.RawMessage `json:"attributes"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

func fromServiceItem(s service.ServiceItem) ServiceHTTPResponse {
	attrs := s.Attributes
	if len(attrs) == 0 {
		attrs = json.RawMessage(`{}`)
	}
	return ServiceHTTPResponse{
		ID:          s.ID,
		ProjectID:   s.ProjectID,
		Name:        s.Name,
		Category:    s.Category,
		Description: s.Description,
		Price:       s.Price,
		Currency:    s.Currency,
		DurationMin: s.DurationMin,
		Status:      s.Status,
		Attributes:  attrs,
		CreatedAt:   s.CreatedAt,
		UpdatedAt:   s.UpdatedAt,
	}
}

func FromServiceItem(s *service.ServiceItem) ServiceHTTPResponse {
	return fromServiceItem(*s)
}

type ListServicesHTTPResponse struct {
	Items      []ServiceHTTPResponse `json:"items"`
	Pagination PaginationResponse    `json:"pagination"`
}

func FromListServicesOutput(o *service.ListServicesOutput) ListServicesHTTPResponse {
	items := make([]ServiceHTTPResponse, 0, len(o.Items))
	for _, s := range o.Items {
		items = append(items, fromServiceItem(s))
	}
	return ListServicesHTTPResponse{Items: items, Pagination: fromPagination(o.Pagination)}
}

type DeleteServiceHTTPResponse struct {
	ID        uuid.UUID `json:"id"`
	DeletedAt time.Time `json:"deleted_at"`
}

func FromDeleteServiceOutput(o *service.DeleteServiceOutput) DeleteServiceHTTPResponse {
	return DeleteServiceHTTPResponse{ID: o.ID, DeletedAt: o.DeletedAt}
}
