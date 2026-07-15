package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/YuriyDubinin/bms-api/internal/domain"
	"github.com/YuriyDubinin/bms-api/internal/service"
	"github.com/YuriyDubinin/bms-api/internal/transport/http/dto"
	mw "github.com/YuriyDubinin/bms-api/internal/transport/http/middleware"
	"github.com/YuriyDubinin/bms-api/internal/transport/http/response"
	"github.com/YuriyDubinin/bms-api/pkg/validator"
)

type CalendarEventService interface {
	CreateEvent(ctx context.Context, customerID uuid.UUID, in service.CreateEventInput) (*service.EventItem, error)
	UpdateEvent(ctx context.Context, customerID uuid.UUID, in service.UpdateEventInput) (*service.EventItem, error)
	ListEvents(ctx context.Context, customerID uuid.UUID, in service.ListEventsInput) (*service.ListEventsOutput, error)
	DeleteEvent(ctx context.Context, customerID, id uuid.UUID) (*service.DeleteEventOutput, error)
	AddAttendee(ctx context.Context, customerID uuid.UUID, in service.AddAttendeeInput) (*service.AttendeeItem, error)
	RemoveAttendee(ctx context.Context, customerID, eventID, attendeeEntityID uuid.UUID) error
	ListAttendees(ctx context.Context, customerID, eventID uuid.UUID) ([]service.AttendeeItem, error)
}

type CalendarEventHandler struct {
	service   CalendarEventService
	validator *validator.Validator
	logger    *slog.Logger
}

func NewCalendarEventHandler(svc CalendarEventService, v *validator.Validator, logger *slog.Logger) *CalendarEventHandler {
	return &CalendarEventHandler{service: svc, validator: v, logger: logger}
}

func (h *CalendarEventHandler) Create(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalOr500(w, r, h.logger)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)

	var req dto.CreateEventHTTPRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_JSON", "invalid JSON body")
		return
	}
	if err := h.validator.Struct(req); err != nil {
		details := toResponseFieldErrors(validator.TranslateErrors(err))
		response.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "request validation failed", details...)
		return
	}

	in := req.ToServiceInput()
	in.CreatedBy = principal.UserID
	out, err := h.service.CreateEvent(r.Context(), principal.CustomerID, in)
	if err != nil {
		h.writeEventError(w, r, "create calendar event", err)
		return
	}
	response.WriteJSON(w, http.StatusCreated, dto.FromEventItem(out))
}

func (h *CalendarEventHandler) Update(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalOr500(w, r, h.logger)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)

	var req dto.UpdateEventHTTPRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_JSON", "invalid JSON body")
		return
	}
	if err := h.validator.Struct(req); err != nil {
		details := toResponseFieldErrors(validator.TranslateErrors(err))
		response.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "request validation failed", details...)
		return
	}

	out, err := h.service.UpdateEvent(r.Context(), principal.CustomerID, req.ToServiceInput())
	if err != nil {
		h.writeEventError(w, r, "update calendar event", err)
		return
	}
	response.WriteJSON(w, http.StatusOK, dto.FromEventItem(out))
}

func (h *CalendarEventHandler) List(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalOr500(w, r, h.logger)
	if !ok {
		return
	}
	q := r.URL.Query()
	in := service.ListEventsInput{
		Page:      atoiDefault(q.Get("page"), 0),
		PageSize:  atoiDefault(q.Get("page_size"), 0),
		ProjectID: parseUUIDPtr(q.Get("project_id")),
		Status:    q.Get("status"),
		Search:    q.Get("search"),
		From:      parseTimePtr(q.Get("from")),
		To:        parseTimePtr(q.Get("to")),
		SortBy:    q.Get("sort_by"),
		Order:     q.Get("order"),
	}
	out, err := h.service.ListEvents(r.Context(), principal.CustomerID, in)
	if err != nil {
		h.writeEventError(w, r, "list calendar events", err)
		return
	}
	response.WriteJSON(w, http.StatusOK, dto.FromListEventsOutput(out))
}

func (h *CalendarEventHandler) Delete(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalOr500(w, r, h.logger)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)

	var req dto.IDHTTPRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_JSON", "invalid JSON body")
		return
	}
	if err := h.validator.Struct(req); err != nil {
		details := toResponseFieldErrors(validator.TranslateErrors(err))
		response.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "request validation failed", details...)
		return
	}

	out, err := h.service.DeleteEvent(r.Context(), principal.CustomerID, req.ID)
	if err != nil {
		h.writeEventError(w, r, "delete calendar event", err)
		return
	}
	response.WriteJSON(w, http.StatusOK, dto.FromDeleteEventOutput(out))
}

func (h *CalendarEventHandler) AddAttendee(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalOr500(w, r, h.logger)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)

	var req dto.AddAttendeeHTTPRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_JSON", "invalid JSON body")
		return
	}
	if err := h.validator.Struct(req); err != nil {
		details := toResponseFieldErrors(validator.TranslateErrors(err))
		response.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "request validation failed", details...)
		return
	}

	out, err := h.service.AddAttendee(r.Context(), principal.CustomerID, req.ToServiceInput())
	if err != nil {
		var verr domain.ValidationErrors
		switch {
		case errors.As(err, &verr):
			response.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "domain validation failed", domainValidationDetails(verr)...)
		case errors.Is(err, domain.ErrNotFound):
			response.WriteError(w, http.StatusNotFound, "EVENT_NOT_FOUND", "calendar event not found")
		case errors.Is(err, domain.ErrAlreadyExists):
			response.WriteError(w, http.StatusConflict, "ATTENDEE_EXISTS", "attendee already added")
		default:
			h.logger.Error("add attendee", "err", err, "request_id", mw.RequestIDFromContext(r.Context()))
			response.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		}
		return
	}
	response.WriteJSON(w, http.StatusCreated, dto.FromAttendeeItem(out))
}

func (h *CalendarEventHandler) RemoveAttendee(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalOr500(w, r, h.logger)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)

	var req dto.RemoveAttendeeHTTPRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_JSON", "invalid JSON body")
		return
	}
	if err := h.validator.Struct(req); err != nil {
		details := toResponseFieldErrors(validator.TranslateErrors(err))
		response.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "request validation failed", details...)
		return
	}

	if err := h.service.RemoveAttendee(r.Context(), principal.CustomerID, req.EventID, req.AttendeeEntityID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			response.WriteError(w, http.StatusNotFound, "ATTENDEE_NOT_FOUND", "attendee not found")
			return
		}
		h.logger.Error("remove attendee", "err", err, "request_id", mw.RequestIDFromContext(r.Context()))
		response.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}
	response.WriteJSON(w, http.StatusOK, map[string]string{"status": "REMOVED", "message": "attendee removed"})
}

func (h *CalendarEventHandler) ListAttendees(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalOr500(w, r, h.logger)
	if !ok {
		return
	}
	eventID := parseUUIDPtr(r.URL.Query().Get("event_id"))
	if eventID == nil {
		response.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "event_id is required")
		return
	}

	out, err := h.service.ListAttendees(r.Context(), principal.CustomerID, *eventID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			response.WriteError(w, http.StatusNotFound, "EVENT_NOT_FOUND", "calendar event not found")
			return
		}
		h.logger.Error("list attendees", "err", err, "request_id", mw.RequestIDFromContext(r.Context()))
		response.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}
	response.WriteJSON(w, http.StatusOK, dto.FromAttendeeItems(out))
}

func (h *CalendarEventHandler) writeEventError(w http.ResponseWriter, r *http.Request, op string, err error) {
	var verr domain.ValidationErrors
	switch {
	case errors.As(err, &verr):
		response.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "domain validation failed", domainValidationDetails(verr)...)
	case errors.Is(err, domain.ErrNotFound):
		response.WriteError(w, http.StatusNotFound, "EVENT_NOT_FOUND", "calendar event not found")
	case errors.Is(err, domain.ErrAlreadyExists):
		response.WriteError(w, http.StatusConflict, "EVENT_EXISTS", "calendar event already exists")
	default:
		h.logger.Error(op, "err", err, "request_id", mw.RequestIDFromContext(r.Context()))
		response.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	}
}
