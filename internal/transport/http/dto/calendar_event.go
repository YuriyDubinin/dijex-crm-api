package dto

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/YuriyDubinin/bms-api/internal/service"
)

type CreateEventHTTPRequest struct {
	ProjectID      *uuid.UUID      `json:"project_id"      validate:"omitempty"`
	Title          string          `json:"title"           validate:"required,min=2,max=255"`
	Description    string          `json:"description"     validate:"omitempty,max=10000"`
	Location       string          `json:"location"        validate:"omitempty,max=500"`
	StartsAt       time.Time       `json:"starts_at"       validate:"required"`
	EndsAt         time.Time       `json:"ends_at"         validate:"required"`
	AllDay         bool            `json:"all_day"`
	Status         string          `json:"status"          validate:"omitempty,oneof=SCHEDULED COMPLETED CANCELLED"`
	RecurrenceRule string          `json:"recurrence_rule" validate:"omitempty,max=1000"`
	Timezone       string          `json:"timezone"        validate:"omitempty,max=100"`
	Attributes     json.RawMessage `json:"attributes"      validate:"omitempty"`
}

func (r CreateEventHTTPRequest) ToServiceInput() service.CreateEventInput {
	return service.CreateEventInput{
		ProjectID:      r.ProjectID,
		Title:          r.Title,
		Description:    r.Description,
		Location:       r.Location,
		StartsAt:       r.StartsAt,
		EndsAt:         r.EndsAt,
		AllDay:         r.AllDay,
		Status:         r.Status,
		RecurrenceRule: r.RecurrenceRule,
		Timezone:       r.Timezone,
		Attributes:     r.Attributes,
	}
}

type UpdateEventHTTPRequest struct {
	ID             uuid.UUID       `json:"id"              validate:"required"`
	ProjectID      *uuid.UUID      `json:"project_id"      validate:"omitempty"`
	Title          string          `json:"title"           validate:"required,min=2,max=255"`
	Description    string          `json:"description"     validate:"omitempty,max=10000"`
	Location       string          `json:"location"        validate:"omitempty,max=500"`
	StartsAt       time.Time       `json:"starts_at"       validate:"required"`
	EndsAt         time.Time       `json:"ends_at"         validate:"required"`
	AllDay         bool            `json:"all_day"`
	Status         string          `json:"status"          validate:"omitempty,oneof=SCHEDULED COMPLETED CANCELLED"`
	RecurrenceRule string          `json:"recurrence_rule" validate:"omitempty,max=1000"`
	Timezone       string          `json:"timezone"        validate:"omitempty,max=100"`
	Attributes     json.RawMessage `json:"attributes"      validate:"omitempty"`
}

func (r UpdateEventHTTPRequest) ToServiceInput() service.UpdateEventInput {
	return service.UpdateEventInput{
		ID:             r.ID,
		ProjectID:      r.ProjectID,
		Title:          r.Title,
		Description:    r.Description,
		Location:       r.Location,
		StartsAt:       r.StartsAt,
		EndsAt:         r.EndsAt,
		AllDay:         r.AllDay,
		Status:         r.Status,
		RecurrenceRule: r.RecurrenceRule,
		Timezone:       r.Timezone,
		Attributes:     r.Attributes,
	}
}

type EventHTTPResponse struct {
	ID             uuid.UUID       `json:"id"`
	ProjectID      *uuid.UUID      `json:"project_id"`
	Title          string          `json:"title"`
	Description    string          `json:"description"`
	Location       string          `json:"location"`
	StartsAt       time.Time       `json:"starts_at"`
	EndsAt         time.Time       `json:"ends_at"`
	AllDay         bool            `json:"all_day"`
	Status         string          `json:"status"`
	RecurrenceRule string          `json:"recurrence_rule"`
	Timezone       string          `json:"timezone"`
	CreatedBy      *uuid.UUID      `json:"created_by"`
	Attributes     json.RawMessage `json:"attributes"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

func fromEventItem(e service.EventItem) EventHTTPResponse {
	attrs := e.Attributes
	if len(attrs) == 0 {
		attrs = json.RawMessage(`{}`)
	}
	return EventHTTPResponse{
		ID:             e.ID,
		ProjectID:      e.ProjectID,
		Title:          e.Title,
		Description:    e.Description,
		Location:       e.Location,
		StartsAt:       e.StartsAt,
		EndsAt:         e.EndsAt,
		AllDay:         e.AllDay,
		Status:         e.Status,
		RecurrenceRule: e.RecurrenceRule,
		Timezone:       e.Timezone,
		CreatedBy:      e.CreatedBy,
		Attributes:     attrs,
		CreatedAt:      e.CreatedAt,
		UpdatedAt:      e.UpdatedAt,
	}
}

func FromEventItem(e *service.EventItem) EventHTTPResponse {
	return fromEventItem(*e)
}

type ListEventsHTTPResponse struct {
	Items      []EventHTTPResponse `json:"items"`
	Pagination PaginationResponse  `json:"pagination"`
}

func FromListEventsOutput(o *service.ListEventsOutput) ListEventsHTTPResponse {
	items := make([]EventHTTPResponse, 0, len(o.Items))
	for _, e := range o.Items {
		items = append(items, fromEventItem(e))
	}
	return ListEventsHTTPResponse{Items: items, Pagination: fromPagination(o.Pagination)}
}

type DeleteEventHTTPResponse struct {
	ID        uuid.UUID `json:"id"`
	DeletedAt time.Time `json:"deleted_at"`
}

func FromDeleteEventOutput(o *service.DeleteEventOutput) DeleteEventHTTPResponse {
	return DeleteEventHTTPResponse{ID: o.ID, DeletedAt: o.DeletedAt}
}

// ───────────────────────── Участники ─────────────────────────

type AddAttendeeHTTPRequest struct {
	EventID          uuid.UUID `json:"event_id"           validate:"required"`
	AttendeeEntityID uuid.UUID `json:"attendee_entity_id" validate:"required"`
	ResponseStatus   string    `json:"response_status"    validate:"omitempty,oneof=PENDING ACCEPTED DECLINED TENTATIVE"`
}

func (r AddAttendeeHTTPRequest) ToServiceInput() service.AddAttendeeInput {
	return service.AddAttendeeInput{
		EventID:          r.EventID,
		AttendeeEntityID: r.AttendeeEntityID,
		ResponseStatus:   r.ResponseStatus,
	}
}

type RemoveAttendeeHTTPRequest struct {
	EventID          uuid.UUID `json:"event_id"           validate:"required"`
	AttendeeEntityID uuid.UUID `json:"attendee_entity_id" validate:"required"`
}

type AttendeeHTTPResponse struct {
	ID               uuid.UUID `json:"id"`
	EventID          uuid.UUID `json:"event_id"`
	AttendeeEntityID uuid.UUID `json:"attendee_entity_id"`
	ResponseStatus   string    `json:"response_status"`
	CreatedAt        time.Time `json:"created_at"`
}

func FromAttendeeItem(a *service.AttendeeItem) AttendeeHTTPResponse {
	return AttendeeHTTPResponse{
		ID:               a.ID,
		EventID:          a.EventID,
		AttendeeEntityID: a.AttendeeEntityID,
		ResponseStatus:   a.ResponseStatus,
		CreatedAt:        a.CreatedAt,
	}
}

type ListAttendeesHTTPResponse struct {
	Items []AttendeeHTTPResponse `json:"items"`
}

func FromAttendeeItems(items []service.AttendeeItem) ListAttendeesHTTPResponse {
	out := make([]AttendeeHTTPResponse, 0, len(items))
	for i := range items {
		out = append(out, FromAttendeeItem(&items[i]))
	}
	return ListAttendeesHTTPResponse{Items: out}
}
