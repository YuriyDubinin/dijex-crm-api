package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/YuriyDubinin/bms-api/internal/domain"
)

// ───────────────────────── Service DTO ─────────────────────────

type CreateEventInput struct {
	ProjectID      *uuid.UUID
	Title          string
	Description    string
	Location       string
	StartsAt       time.Time
	EndsAt         time.Time
	AllDay         bool
	Status         string
	RecurrenceRule string
	Timezone       string
	CreatedBy      uuid.UUID
	Attributes     json.RawMessage
}

type UpdateEventInput struct {
	ID             uuid.UUID
	ProjectID      *uuid.UUID
	Title          string
	Description    string
	Location       string
	StartsAt       time.Time
	EndsAt         time.Time
	AllDay         bool
	Status         string
	RecurrenceRule string
	Timezone       string
	Attributes     json.RawMessage
}

type ListEventsInput struct {
	Page      int
	PageSize  int
	ProjectID *uuid.UUID
	Status    string
	Search    string
	From      *time.Time
	To        *time.Time
	SortBy    string
	Order     string
}

type EventItem struct {
	ID             uuid.UUID
	ProjectID      *uuid.UUID
	Title          string
	Description    string
	Location       string
	StartsAt       time.Time
	EndsAt         time.Time
	AllDay         bool
	Status         string
	RecurrenceRule string
	Timezone       string
	CreatedBy      *uuid.UUID
	Attributes     json.RawMessage
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type ListEventsOutput struct {
	Items      []EventItem
	Pagination Pagination
}

type DeleteEventOutput struct {
	ID        uuid.UUID
	DeletedAt time.Time
}

type AddAttendeeInput struct {
	EventID          uuid.UUID
	AttendeeEntityID uuid.UUID
	ResponseStatus   string
}

type AttendeeItem struct {
	ID               uuid.UUID
	EventID          uuid.UUID
	AttendeeEntityID uuid.UUID
	ResponseStatus   string
	CreatedAt        time.Time
}

// ───────────────────────── Service ─────────────────────────

type CalendarEventService struct {
	repo     domain.CalendarEventRepository
	projects ProjectChecker
	entities EntityResolver
	logger   *slog.Logger
	clock    func() time.Time
}

func NewCalendarEventService(repo domain.CalendarEventRepository, projects ProjectChecker, entities EntityResolver, logger *slog.Logger) *CalendarEventService {
	return &CalendarEventService{repo: repo, projects: projects, entities: entities, logger: logger, clock: time.Now}
}

func (s *CalendarEventService) CreateEvent(ctx context.Context, customerID uuid.UUID, in CreateEventInput) (*EventItem, error) {
	in.Title = strings.TrimSpace(in.Title)
	in.Description = strings.TrimSpace(in.Description)
	in.Location = strings.TrimSpace(in.Location)
	in.Status = normalizeStatus(in.Status, domain.CalendarEventStatusScheduled)
	in.Timezone = normalizeTimezone(in.Timezone)

	errs := validateEvent(in.Title, in.Status, in.StartsAt, in.EndsAt, in.Attributes)
	if ve := ensureProjectOwned(ctx, s.projects, customerID, in.ProjectID); ve != nil {
		errs = append(errs, ve)
	}
	if len(errs) > 0 {
		return nil, errs
	}

	now := s.clock()
	createdBy := in.CreatedBy
	e := &domain.CalendarEvent{
		ID:             uuid.New(),
		CustomerID:     customerID,
		ProjectID:      in.ProjectID,
		Title:          in.Title,
		Description:    in.Description,
		Location:       in.Location,
		StartsAt:       in.StartsAt,
		EndsAt:         in.EndsAt,
		AllDay:         in.AllDay,
		Status:         in.Status,
		RecurrenceRule: in.RecurrenceRule,
		Timezone:       in.Timezone,
		CreatedBy:      &createdBy,
		Attributes:     in.Attributes,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.repo.Create(ctx, e); err != nil {
		return nil, err
	}
	s.logger.Info("calendar event created", "event_id", e.ID, "customer_id", customerID)
	return eventToItem(e), nil
}

func (s *CalendarEventService) UpdateEvent(ctx context.Context, customerID uuid.UUID, in UpdateEventInput) (*EventItem, error) {
	in.Title = strings.TrimSpace(in.Title)
	in.Description = strings.TrimSpace(in.Description)
	in.Location = strings.TrimSpace(in.Location)
	in.Status = normalizeStatus(in.Status, domain.CalendarEventStatusScheduled)
	in.Timezone = normalizeTimezone(in.Timezone)

	errs := validateEvent(in.Title, in.Status, in.StartsAt, in.EndsAt, in.Attributes)
	if ve := ensureProjectOwned(ctx, s.projects, customerID, in.ProjectID); ve != nil {
		errs = append(errs, ve)
	}
	if len(errs) > 0 {
		return nil, errs
	}

	e := &domain.CalendarEvent{
		ID:             in.ID,
		CustomerID:     customerID,
		ProjectID:      in.ProjectID,
		Title:          in.Title,
		Description:    in.Description,
		Location:       in.Location,
		StartsAt:       in.StartsAt,
		EndsAt:         in.EndsAt,
		AllDay:         in.AllDay,
		Status:         in.Status,
		RecurrenceRule: in.RecurrenceRule,
		Timezone:       in.Timezone,
		Attributes:     in.Attributes,
	}
	if err := s.repo.Update(ctx, customerID, e); err != nil {
		return nil, err
	}
	s.logger.Info("calendar event updated", "event_id", e.ID, "customer_id", customerID)
	return eventToItem(e), nil
}

func (s *CalendarEventService) ListEvents(ctx context.Context, customerID uuid.UUID, in ListEventsInput) (*ListEventsOutput, error) {
	status := normalizeStatus(in.Status, "")
	if status != "" && !domain.IsValidCalendarEventStatus(status) {
		return nil, domain.ValidationErrors{{Field: "status", Message: "invalid calendar event status"}}
	}

	page, size, offset := clampPagination(in.Page, in.PageSize)
	filter := domain.CalendarEventListFilter{
		ProjectID: in.ProjectID,
		Status:    status,
		Search:    strings.TrimSpace(in.Search),
		From:      in.From,
		To:        in.To,
		Limit:     size,
		Offset:    offset,
		SortBy:    in.SortBy,
		SortDesc:  strings.ToLower(in.Order) == "desc",
	}

	items, total, err := s.repo.List(ctx, customerID, filter)
	if err != nil {
		return nil, err
	}
	out := &ListEventsOutput{
		Items:      make([]EventItem, 0, len(items)),
		Pagination: Pagination{Page: page, PageSize: size, Total: total, TotalPages: totalPages(total, size)},
	}
	for _, e := range items {
		out.Items = append(out.Items, *eventToItem(e))
	}
	return out, nil
}

func (s *CalendarEventService) DeleteEvent(ctx context.Context, customerID, id uuid.UUID) (*DeleteEventOutput, error) {
	deletedAt, err := s.repo.SoftDelete(ctx, customerID, id)
	if err != nil {
		return nil, err
	}
	s.logger.Info("calendar event deleted", "event_id", id, "customer_id", customerID)
	return &DeleteEventOutput{ID: id, DeletedAt: deletedAt}, nil
}

// AddAttendee проверяет принадлежность события и attendee-entity тенанту и
// добавляет участника.
func (s *CalendarEventService) AddAttendee(ctx context.Context, customerID uuid.UUID, in AddAttendeeInput) (*AttendeeItem, error) {
	if _, err := s.repo.GetByID(ctx, customerID, in.EventID); err != nil {
		return nil, err // ErrNotFound, если событие чужое/отсутствует
	}
	if _, err := s.entities.ResolveType(ctx, customerID, in.AttendeeEntityID); err != nil {
		return nil, domain.ValidationErrors{{Field: "attendee_entity_id", Message: "entity not found in this tenant"}}
	}

	status := normalizeStatus(in.ResponseStatus, domain.AttendeeResponsePending)
	if !domain.IsValidAttendeeResponse(status) {
		return nil, domain.ValidationErrors{{Field: "response_status", Message: "invalid response status"}}
	}

	a := &domain.CalendarEventAttendee{
		ID:               uuid.New(),
		EventID:          in.EventID,
		AttendeeEntityID: in.AttendeeEntityID,
		ResponseStatus:   status,
	}
	if err := s.repo.AddAttendee(ctx, customerID, a); err != nil {
		return nil, err
	}
	return attendeeToItem(a), nil
}

func (s *CalendarEventService) RemoveAttendee(ctx context.Context, customerID, eventID, attendeeEntityID uuid.UUID) error {
	return s.repo.RemoveAttendee(ctx, customerID, eventID, attendeeEntityID)
}

func (s *CalendarEventService) ListAttendees(ctx context.Context, customerID, eventID uuid.UUID) ([]AttendeeItem, error) {
	if _, err := s.repo.GetByID(ctx, customerID, eventID); err != nil {
		return nil, err
	}
	items, err := s.repo.ListAttendees(ctx, customerID, eventID)
	if err != nil {
		return nil, err
	}
	out := make([]AttendeeItem, 0, len(items))
	for _, a := range items {
		out = append(out, *attendeeToItem(a))
	}
	return out, nil
}

// ───────────────────────── helpers ─────────────────────────

func validateEvent(title, status string, startsAt, endsAt time.Time, attrs json.RawMessage) domain.ValidationErrors {
	var errs domain.ValidationErrors
	if title == "" {
		errs = append(errs, &domain.ValidationError{Field: "title", Message: "is required"})
	} else if l := utf8.RuneCountInString(title); l < 2 || l > 255 {
		errs = append(errs, &domain.ValidationError{Field: "title", Message: "must be between 2 and 255 characters"})
	}
	if startsAt.IsZero() || endsAt.IsZero() {
		errs = append(errs, &domain.ValidationError{Field: "starts_at", Message: "starts_at and ends_at are required"})
	} else if !endsAt.After(startsAt) {
		errs = append(errs, &domain.ValidationError{Field: "ends_at", Message: "must be after starts_at"})
	}
	if !domain.IsValidCalendarEventStatus(status) {
		errs = append(errs, &domain.ValidationError{Field: "status", Message: "must be one of SCHEDULED, COMPLETED, CANCELLED"})
	}
	if !validAttributes(attrs) {
		errs = append(errs, attrError("attributes"))
	}
	return errs
}

func eventToItem(e *domain.CalendarEvent) *EventItem {
	return &EventItem{
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
		Attributes:     e.Attributes,
		CreatedAt:      e.CreatedAt,
		UpdatedAt:      e.UpdatedAt,
	}
}

func attendeeToItem(a *domain.CalendarEventAttendee) *AttendeeItem {
	return &AttendeeItem{
		ID:               a.ID,
		EventID:          a.EventID,
		AttendeeEntityID: a.AttendeeEntityID,
		ResponseStatus:   a.ResponseStatus,
		CreatedAt:        a.CreatedAt,
	}
}

func normalizeTimezone(tz string) string {
	tz = strings.TrimSpace(tz)
	if tz == "" {
		return "UTC"
	}
	return tz
}
