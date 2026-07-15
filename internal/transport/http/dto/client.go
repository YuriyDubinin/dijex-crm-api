package dto

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/YuriyDubinin/bms-api/internal/service"
)

type CreateClientHTTPRequest struct {
	ProjectID   *uuid.UUID      `json:"project_id"   validate:"omitempty"`
	FirstName   string          `json:"first_name"   validate:"omitempty,max=255"`
	LastName    string          `json:"last_name"    validate:"omitempty,max=255"`
	CompanyName string          `json:"company_name" validate:"omitempty,max=255"`
	Email       string          `json:"email"        validate:"omitempty,email,max=255"`
	Phone       string          `json:"phone"        validate:"omitempty,max=50"`
	Status      string          `json:"status"       validate:"omitempty,oneof=LEAD ACTIVE INACTIVE ARCHIVED"`
	Source      string          `json:"source"       validate:"omitempty,max=255"`
	Address     json.RawMessage `json:"address"      validate:"omitempty"`
	Attributes  json.RawMessage `json:"attributes"   validate:"omitempty"`
}

func (r CreateClientHTTPRequest) ToServiceInput() service.CreateClientInput {
	return service.CreateClientInput{
		ProjectID:   r.ProjectID,
		FirstName:   r.FirstName,
		LastName:    r.LastName,
		CompanyName: r.CompanyName,
		Email:       r.Email,
		Phone:       r.Phone,
		Status:      r.Status,
		Source:      r.Source,
		Address:     r.Address,
		Attributes:  r.Attributes,
	}
}

type UpdateClientHTTPRequest struct {
	ID          uuid.UUID       `json:"id"           validate:"required"`
	ProjectID   *uuid.UUID      `json:"project_id"   validate:"omitempty"`
	FirstName   string          `json:"first_name"   validate:"omitempty,max=255"`
	LastName    string          `json:"last_name"    validate:"omitempty,max=255"`
	CompanyName string          `json:"company_name" validate:"omitempty,max=255"`
	Email       string          `json:"email"        validate:"omitempty,email,max=255"`
	Phone       string          `json:"phone"        validate:"omitempty,max=50"`
	Status      string          `json:"status"       validate:"omitempty,oneof=LEAD ACTIVE INACTIVE ARCHIVED"`
	Source      string          `json:"source"       validate:"omitempty,max=255"`
	Address     json.RawMessage `json:"address"      validate:"omitempty"`
	Attributes  json.RawMessage `json:"attributes"   validate:"omitempty"`
}

func (r UpdateClientHTTPRequest) ToServiceInput() service.UpdateClientInput {
	return service.UpdateClientInput{
		ID:          r.ID,
		ProjectID:   r.ProjectID,
		FirstName:   r.FirstName,
		LastName:    r.LastName,
		CompanyName: r.CompanyName,
		Email:       r.Email,
		Phone:       r.Phone,
		Status:      r.Status,
		Source:      r.Source,
		Address:     r.Address,
		Attributes:  r.Attributes,
	}
}

type ClientHTTPResponse struct {
	ID          uuid.UUID       `json:"id"`
	ProjectID   *uuid.UUID      `json:"project_id"`
	FirstName   string          `json:"first_name"`
	LastName    string          `json:"last_name"`
	CompanyName string          `json:"company_name"`
	Email       string          `json:"email"`
	Phone       string          `json:"phone"`
	Status      string          `json:"status"`
	Source      string          `json:"source"`
	Address     json.RawMessage `json:"address"`
	Attributes  json.RawMessage `json:"attributes"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

func fromClientItem(c service.ClientItem) ClientHTTPResponse {
	attrs := c.Attributes
	if len(attrs) == 0 {
		attrs = json.RawMessage(`{}`)
	}
	return ClientHTTPResponse{
		ID:          c.ID,
		ProjectID:   c.ProjectID,
		FirstName:   c.FirstName,
		LastName:    c.LastName,
		CompanyName: c.CompanyName,
		Email:       c.Email,
		Phone:       c.Phone,
		Status:      c.Status,
		Source:      c.Source,
		Address:     c.Address,
		Attributes:  attrs,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
	}
}

func FromClientItem(c *service.ClientItem) ClientHTTPResponse {
	return fromClientItem(*c)
}

type ListClientsHTTPResponse struct {
	Items      []ClientHTTPResponse `json:"items"`
	Pagination PaginationResponse   `json:"pagination"`
}

func FromListClientsOutput(o *service.ListClientsOutput) ListClientsHTTPResponse {
	items := make([]ClientHTTPResponse, 0, len(o.Items))
	for _, c := range o.Items {
		items = append(items, fromClientItem(c))
	}
	return ListClientsHTTPResponse{Items: items, Pagination: fromPagination(o.Pagination)}
}

type DeleteClientHTTPResponse struct {
	ID        uuid.UUID `json:"id"`
	DeletedAt time.Time `json:"deleted_at"`
}

func FromDeleteClientOutput(o *service.DeleteClientOutput) DeleteClientHTTPResponse {
	return DeleteClientHTTPResponse{ID: o.ID, DeletedAt: o.DeletedAt}
}
