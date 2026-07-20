package dto

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/YuriyDubinin/bms-api/internal/service"
)

type CreateDealHTTPRequest struct {
	ProjectID       *uuid.UUID      `json:"project_id"        validate:"omitempty"`
	ClientID        *uuid.UUID      `json:"client_id"         validate:"omitempty"`
	ServiceID       *uuid.UUID      `json:"service_id"        validate:"omitempty"`
	ProcessID       *uuid.UUID      `json:"process_id"        validate:"omitempty"`
	Title           string          `json:"title"             validate:"required,min=2,max=255"`
	Description     string          `json:"description"       validate:"omitempty,max=10000"`
	Status          string          `json:"status"            validate:"omitempty,oneof=NEW PENDING WON LOST"`
	Type            string          `json:"type"              validate:"omitempty,oneof=INCOME EXPENSE"`
	Amount          *float64        `json:"amount"            validate:"omitempty,gte=0"`
	Currency        string          `json:"currency"          validate:"omitempty,len=3,alpha"`
	Probability     *int            `json:"probability"       validate:"omitempty,min=0,max=100"`
	ExpectedCloseAt *string         `json:"expected_close_at" validate:"omitempty,datetime=2006-01-02"`
	AssignedTo      *uuid.UUID      `json:"assigned_to"       validate:"omitempty"`
	Attributes      json.RawMessage `json:"attributes"        validate:"omitempty"`
}

func (r CreateDealHTTPRequest) ToServiceInput() service.CreateDealInput {
	return service.CreateDealInput{
		ProjectID:       r.ProjectID,
		ClientID:        r.ClientID,
		ServiceID:       r.ServiceID,
		ProcessID:       r.ProcessID,
		Title:           r.Title,
		Description:     r.Description,
		Status:          r.Status,
		Type:            r.Type,
		Amount:          r.Amount,
		Currency:        r.Currency,
		Probability:     r.Probability,
		ExpectedCloseAt: parseDatePtr(r.ExpectedCloseAt),
		AssignedTo:      r.AssignedTo,
		Attributes:      r.Attributes,
	}
}

type UpdateDealHTTPRequest struct {
	ID              uuid.UUID       `json:"id"                validate:"required"`
	ProjectID       *uuid.UUID      `json:"project_id"        validate:"omitempty"`
	ClientID        *uuid.UUID      `json:"client_id"         validate:"omitempty"`
	ServiceID       *uuid.UUID      `json:"service_id"        validate:"omitempty"`
	ProcessID       *uuid.UUID      `json:"process_id"        validate:"omitempty"`
	Title           string          `json:"title"             validate:"required,min=2,max=255"`
	Description     string          `json:"description"       validate:"omitempty,max=10000"`
	Status          string          `json:"status"            validate:"omitempty,oneof=NEW PENDING WON LOST"`
	Type            string          `json:"type"              validate:"omitempty,oneof=INCOME EXPENSE"`
	Amount          *float64        `json:"amount"            validate:"omitempty,gte=0"`
	Currency        string          `json:"currency"          validate:"omitempty,len=3,alpha"`
	Probability     *int            `json:"probability"       validate:"omitempty,min=0,max=100"`
	ExpectedCloseAt *string         `json:"expected_close_at" validate:"omitempty,datetime=2006-01-02"`
	AssignedTo      *uuid.UUID      `json:"assigned_to"       validate:"omitempty"`
	Attributes      json.RawMessage `json:"attributes"        validate:"omitempty"`
}

func (r UpdateDealHTTPRequest) ToServiceInput() service.UpdateDealInput {
	return service.UpdateDealInput{
		ID:              r.ID,
		ProjectID:       r.ProjectID,
		ClientID:        r.ClientID,
		ServiceID:       r.ServiceID,
		ProcessID:       r.ProcessID,
		Title:           r.Title,
		Description:     r.Description,
		Status:          r.Status,
		Type:            r.Type,
		Amount:          r.Amount,
		Currency:        r.Currency,
		Probability:     r.Probability,
		ExpectedCloseAt: parseDatePtr(r.ExpectedCloseAt),
		AssignedTo:      r.AssignedTo,
		Attributes:      r.Attributes,
	}
}

type DealHTTPResponse struct {
	ID              uuid.UUID       `json:"id"`
	ProjectID       *uuid.UUID      `json:"project_id"`
	ClientID        *uuid.UUID      `json:"client_id"`
	ServiceID       *uuid.UUID      `json:"service_id"`
	ProcessID       *uuid.UUID      `json:"process_id"`
	Title           string          `json:"title"`
	Description     string          `json:"description"`
	Status          string          `json:"status"`
	Type            string          `json:"type"`
	Amount          *float64        `json:"amount"`
	Currency        string          `json:"currency"`
	Probability     *int            `json:"probability"`
	ExpectedCloseAt *string         `json:"expected_close_at"`
	ClosedAt        *time.Time      `json:"closed_at"`
	AssignedTo      *uuid.UUID      `json:"assigned_to"`
	CreatedBy       *uuid.UUID      `json:"created_by"`
	Attributes      json.RawMessage `json:"attributes"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

func fromDealItem(d service.DealItem) DealHTTPResponse {
	attrs := d.Attributes
	if len(attrs) == 0 {
		attrs = json.RawMessage(`{}`)
	}
	return DealHTTPResponse{
		ID:              d.ID,
		ProjectID:       d.ProjectID,
		ClientID:        d.ClientID,
		ServiceID:       d.ServiceID,
		ProcessID:       d.ProcessID,
		Title:           d.Title,
		Description:     d.Description,
		Status:          d.Status,
		Type:            d.Type,
		Amount:          d.Amount,
		Currency:        d.Currency,
		Probability:     d.Probability,
		ExpectedCloseAt: formatDatePtr(d.ExpectedCloseAt),
		ClosedAt:        d.ClosedAt,
		AssignedTo:      d.AssignedTo,
		CreatedBy:       d.CreatedBy,
		Attributes:      attrs,
		CreatedAt:       d.CreatedAt,
		UpdatedAt:       d.UpdatedAt,
	}
}

func FromDealItem(d *service.DealItem) DealHTTPResponse {
	return fromDealItem(*d)
}

type ListDealsHTTPResponse struct {
	Items      []DealHTTPResponse `json:"items"`
	Pagination PaginationResponse `json:"pagination"`
}

func FromListDealsOutput(o *service.ListDealsOutput) ListDealsHTTPResponse {
	items := make([]DealHTTPResponse, 0, len(o.Items))
	for _, d := range o.Items {
		items = append(items, fromDealItem(d))
	}
	return ListDealsHTTPResponse{Items: items, Pagination: fromPagination(o.Pagination)}
}

type DeleteDealHTTPResponse struct {
	ID        uuid.UUID `json:"id"`
	DeletedAt time.Time `json:"deleted_at"`
}

func FromDeleteDealOutput(o *service.DeleteDealOutput) DeleteDealHTTPResponse {
	return DeleteDealHTTPResponse{ID: o.ID, DeletedAt: o.DeletedAt}
}
