package dto

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/YuriyDubinin/bms-api/internal/service"
)

// ───────────────────────── Process ─────────────────────────

type CreateProcessHTTPRequest struct {
	Name        string          `json:"name"        validate:"required,min=2,max=255"`
	Description string          `json:"description" validate:"omitempty,max=5000"`
	Status      string          `json:"status"      validate:"omitempty,oneof=ACTIVE COMPLETED FAILED"`
	Attributes  json.RawMessage `json:"attributes"  validate:"omitempty"`
}

func (r CreateProcessHTTPRequest) ToServiceInput() service.CreateProcessInput {
	return service.CreateProcessInput{
		Name:        r.Name,
		Description: r.Description,
		Status:      r.Status,
		Attributes:  r.Attributes,
	}
}

type UpdateProcessHTTPRequest struct {
	ID          uuid.UUID       `json:"id"          validate:"required"`
	Name        string          `json:"name"        validate:"required,min=2,max=255"`
	Description string          `json:"description" validate:"omitempty,max=5000"`
	Status      string          `json:"status"      validate:"omitempty,oneof=ACTIVE COMPLETED FAILED"`
	Attributes  json.RawMessage `json:"attributes"  validate:"omitempty"`
}

func (r UpdateProcessHTTPRequest) ToServiceInput() service.UpdateProcessInput {
	return service.UpdateProcessInput{
		ID:          r.ID,
		Name:        r.Name,
		Description: r.Description,
		Status:      r.Status,
		Attributes:  r.Attributes,
	}
}

type ProcessHTTPResponse struct {
	ID          uuid.UUID       `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Status      string          `json:"status"`
	ClosedAt    *time.Time      `json:"closed_at"`
	Attributes  json.RawMessage `json:"attributes"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

func fromProcessItem(p service.ProcessItem) ProcessHTTPResponse {
	attrs := p.Attributes
	if len(attrs) == 0 {
		attrs = json.RawMessage(`{}`)
	}
	return ProcessHTTPResponse{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		Status:      p.Status,
		ClosedAt:    p.ClosedAt,
		Attributes:  attrs,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

func FromProcessItem(p *service.ProcessItem) ProcessHTTPResponse {
	return fromProcessItem(*p)
}

type ListProcessesHTTPResponse struct {
	Items      []ProcessHTTPResponse `json:"items"`
	Pagination PaginationResponse    `json:"pagination"`
}

func FromListProcessesOutput(o *service.ListProcessesOutput) ListProcessesHTTPResponse {
	items := make([]ProcessHTTPResponse, 0, len(o.Items))
	for _, p := range o.Items {
		items = append(items, fromProcessItem(p))
	}
	return ListProcessesHTTPResponse{Items: items, Pagination: fromPagination(o.Pagination)}
}

type DeleteProcessHTTPResponse struct {
	ID        uuid.UUID `json:"id"`
	DeletedAt time.Time `json:"deleted_at"`
}

func FromDeleteProcessOutput(o *service.DeleteProcessOutput) DeleteProcessHTTPResponse {
	return DeleteProcessHTTPResponse{ID: o.ID, DeletedAt: o.DeletedAt}
}

// ───────────────────────── Process stages ─────────────────────────

type CreateProcessStageHTTPRequest struct {
	ProcessID   uuid.UUID       `json:"process_id"  validate:"required"`
	Name        string          `json:"name"        validate:"required,min=2,max=255"`
	Description string          `json:"description" validate:"omitempty,max=5000"`
	Position    int             `json:"position"    validate:"omitempty"`
	Attributes  json.RawMessage `json:"attributes"  validate:"omitempty"`
}

func (r CreateProcessStageHTTPRequest) ToServiceInput() service.CreateProcessStageInput {
	return service.CreateProcessStageInput{
		ProcessID:   r.ProcessID,
		Name:        r.Name,
		Description: r.Description,
		Position:    r.Position,
		Attributes:  r.Attributes,
	}
}

type UpdateProcessStageHTTPRequest struct {
	ID          uuid.UUID       `json:"id"          validate:"required"`
	Name        string          `json:"name"        validate:"required,min=2,max=255"`
	Description string          `json:"description" validate:"omitempty,max=5000"`
	Position    int             `json:"position"    validate:"omitempty"`
	Attributes  json.RawMessage `json:"attributes"  validate:"omitempty"`
}

func (r UpdateProcessStageHTTPRequest) ToServiceInput() service.UpdateProcessStageInput {
	return service.UpdateProcessStageInput{
		ID:          r.ID,
		Name:        r.Name,
		Description: r.Description,
		Position:    r.Position,
		Attributes:  r.Attributes,
	}
}

type ProcessStageHTTPResponse struct {
	ID          uuid.UUID       `json:"id"`
	ProcessID   uuid.UUID       `json:"process_id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Position    int             `json:"position"`
	Attributes  json.RawMessage `json:"attributes"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

func fromProcessStageItem(s service.ProcessStageItem) ProcessStageHTTPResponse {
	attrs := s.Attributes
	if len(attrs) == 0 {
		attrs = json.RawMessage(`{}`)
	}
	return ProcessStageHTTPResponse{
		ID:          s.ID,
		ProcessID:   s.ProcessID,
		Name:        s.Name,
		Description: s.Description,
		Position:    s.Position,
		Attributes:  attrs,
		CreatedAt:   s.CreatedAt,
		UpdatedAt:   s.UpdatedAt,
	}
}

func FromProcessStageItem(s *service.ProcessStageItem) ProcessStageHTTPResponse {
	return fromProcessStageItem(*s)
}

type ListProcessStagesHTTPResponse struct {
	Items []ProcessStageHTTPResponse `json:"items"`
}

func FromProcessStageItems(items []service.ProcessStageItem) ListProcessStagesHTTPResponse {
	out := make([]ProcessStageHTTPResponse, 0, len(items))
	for _, s := range items {
		out = append(out, fromProcessStageItem(s))
	}
	return ListProcessStagesHTTPResponse{Items: out}
}

type DeleteProcessStageHTTPResponse struct {
	ID        uuid.UUID `json:"id"`
	DeletedAt time.Time `json:"deleted_at"`
}

func FromDeleteProcessStageOutput(o *service.DeleteProcessStageOutput) DeleteProcessStageHTTPResponse {
	return DeleteProcessStageHTTPResponse{ID: o.ID, DeletedAt: o.DeletedAt}
}
