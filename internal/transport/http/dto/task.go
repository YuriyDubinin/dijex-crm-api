package dto

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/YuriyDubinin/bms-api/internal/service"
)

type CreateTaskHTTPRequest struct {
	ProjectID   *uuid.UUID      `json:"project_id"  validate:"omitempty"`
	ClientID    *uuid.UUID      `json:"client_id"   validate:"omitempty"`
	DealID      *uuid.UUID      `json:"deal_id"     validate:"omitempty"`
	Title       string          `json:"title"       validate:"required,min=2,max=500"`
	Description string          `json:"description" validate:"omitempty,max=10000"`
	Status      string          `json:"status"      validate:"omitempty,oneof=TODO IN_PROGRESS BLOCKED DONE CANCELLED"`
	Priority    string          `json:"priority"    validate:"omitempty,oneof=LOW MEDIUM HIGH URGENT"`
	AssignedTo  *uuid.UUID      `json:"assigned_to" validate:"omitempty"`
	DueAt       *time.Time      `json:"due_at"      validate:"omitempty"`
	Attributes  json.RawMessage `json:"attributes"  validate:"omitempty"`
}

// ToServiceInput НЕ проставляет CreatedBy — его задаёт хендлер из
// аутентифицированного principal.
func (r CreateTaskHTTPRequest) ToServiceInput() service.CreateTaskInput {
	return service.CreateTaskInput{
		ProjectID:   r.ProjectID,
		ClientID:    r.ClientID,
		DealID:      r.DealID,
		Title:       r.Title,
		Description: r.Description,
		Status:      r.Status,
		Priority:    r.Priority,
		AssignedTo:  r.AssignedTo,
		DueAt:       r.DueAt,
		Attributes:  r.Attributes,
	}
}

type UpdateTaskHTTPRequest struct {
	ID          uuid.UUID       `json:"id"          validate:"required"`
	ProjectID   *uuid.UUID      `json:"project_id"  validate:"omitempty"`
	ClientID    *uuid.UUID      `json:"client_id"   validate:"omitempty"`
	DealID      *uuid.UUID      `json:"deal_id"     validate:"omitempty"`
	Title       string          `json:"title"       validate:"required,min=2,max=500"`
	Description string          `json:"description" validate:"omitempty,max=10000"`
	Status      string          `json:"status"      validate:"omitempty,oneof=TODO IN_PROGRESS BLOCKED DONE CANCELLED"`
	Priority    string          `json:"priority"    validate:"omitempty,oneof=LOW MEDIUM HIGH URGENT"`
	AssignedTo  *uuid.UUID      `json:"assigned_to" validate:"omitempty"`
	DueAt       *time.Time      `json:"due_at"      validate:"omitempty"`
	Attributes  json.RawMessage `json:"attributes"  validate:"omitempty"`
}

func (r UpdateTaskHTTPRequest) ToServiceInput() service.UpdateTaskInput {
	return service.UpdateTaskInput{
		ID:          r.ID,
		ProjectID:   r.ProjectID,
		ClientID:    r.ClientID,
		DealID:      r.DealID,
		Title:       r.Title,
		Description: r.Description,
		Status:      r.Status,
		Priority:    r.Priority,
		AssignedTo:  r.AssignedTo,
		DueAt:       r.DueAt,
		Attributes:  r.Attributes,
	}
}

type TaskHTTPResponse struct {
	ID          uuid.UUID       `json:"id"`
	ProjectID   *uuid.UUID      `json:"project_id"`
	ClientID    *uuid.UUID      `json:"client_id"`
	DealID      *uuid.UUID      `json:"deal_id"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Status      string          `json:"status"`
	Priority    string          `json:"priority"`
	AssignedTo  *uuid.UUID      `json:"assigned_to"`
	CreatedBy   *uuid.UUID      `json:"created_by"`
	DueAt       *time.Time      `json:"due_at"`
	CompletedAt *time.Time      `json:"completed_at"`
	Attributes  json.RawMessage `json:"attributes"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

func fromTaskItem(t service.TaskItem) TaskHTTPResponse {
	attrs := t.Attributes
	if len(attrs) == 0 {
		attrs = json.RawMessage(`{}`)
	}
	return TaskHTTPResponse{
		ID:          t.ID,
		ProjectID:   t.ProjectID,
		ClientID:    t.ClientID,
		DealID:      t.DealID,
		Title:       t.Title,
		Description: t.Description,
		Status:      t.Status,
		Priority:    t.Priority,
		AssignedTo:  t.AssignedTo,
		CreatedBy:   t.CreatedBy,
		DueAt:       t.DueAt,
		CompletedAt: t.CompletedAt,
		Attributes:  attrs,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}

func FromTaskItem(t *service.TaskItem) TaskHTTPResponse {
	return fromTaskItem(*t)
}

type ListTasksHTTPResponse struct {
	Items      []TaskHTTPResponse `json:"items"`
	Pagination PaginationResponse `json:"pagination"`
}

func FromListTasksOutput(o *service.ListTasksOutput) ListTasksHTTPResponse {
	items := make([]TaskHTTPResponse, 0, len(o.Items))
	for _, t := range o.Items {
		items = append(items, fromTaskItem(t))
	}
	return ListTasksHTTPResponse{Items: items, Pagination: fromPagination(o.Pagination)}
}

type DeleteTaskHTTPResponse struct {
	ID        uuid.UUID `json:"id"`
	DeletedAt time.Time `json:"deleted_at"`
}

func FromDeleteTaskOutput(o *service.DeleteTaskOutput) DeleteTaskHTTPResponse {
	return DeleteTaskHTTPResponse{ID: o.ID, DeletedAt: o.DeletedAt}
}
