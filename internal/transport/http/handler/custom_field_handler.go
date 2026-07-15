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

type CustomFieldService interface {
	CreateDefinition(ctx context.Context, customerID uuid.UUID, in service.CreateDefinitionInput) (*service.DefinitionItem, error)
	UpdateDefinition(ctx context.Context, customerID uuid.UUID, in service.UpdateDefinitionInput) (*service.DefinitionItem, error)
	ListDefinitions(ctx context.Context, customerID uuid.UUID, in service.ListDefinitionsInput) (*service.ListDefinitionsOutput, error)
	DeleteDefinition(ctx context.Context, customerID, id uuid.UUID) error
	SetValue(ctx context.Context, customerID uuid.UUID, in service.SetValueInput) (*service.ValueItem, error)
	ListValues(ctx context.Context, customerID, entityID uuid.UUID) ([]service.ValueItem, error)
	UnsetValue(ctx context.Context, customerID, entityID, fieldDefinitionID uuid.UUID) error
}

type CustomFieldHandler struct {
	service   CustomFieldService
	validator *validator.Validator
	logger    *slog.Logger
}

func NewCustomFieldHandler(svc CustomFieldService, v *validator.Validator, logger *slog.Logger) *CustomFieldHandler {
	return &CustomFieldHandler{service: svc, validator: v, logger: logger}
}

func (h *CustomFieldHandler) CreateDefinition(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalOr500(w, r, h.logger)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)

	var req dto.CreateDefinitionHTTPRequest
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

	out, err := h.service.CreateDefinition(r.Context(), principal.CustomerID, req.ToServiceInput())
	if err != nil {
		h.writeDefError(w, r, "create custom field definition", err)
		return
	}
	response.WriteJSON(w, http.StatusCreated, dto.FromDefinitionItem(out))
}

func (h *CustomFieldHandler) UpdateDefinition(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalOr500(w, r, h.logger)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)

	var req dto.UpdateDefinitionHTTPRequest
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

	out, err := h.service.UpdateDefinition(r.Context(), principal.CustomerID, req.ToServiceInput())
	if err != nil {
		h.writeDefError(w, r, "update custom field definition", err)
		return
	}
	response.WriteJSON(w, http.StatusOK, dto.FromDefinitionItem(out))
}

func (h *CustomFieldHandler) ListDefinitions(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalOr500(w, r, h.logger)
	if !ok {
		return
	}
	q := r.URL.Query()
	in := service.ListDefinitionsInput{
		Page:       atoiDefault(q.Get("page"), 0),
		PageSize:   atoiDefault(q.Get("page_size"), 0),
		EntityType: q.Get("entity_type"),
		Order:      q.Get("order"),
	}
	out, err := h.service.ListDefinitions(r.Context(), principal.CustomerID, in)
	if err != nil {
		h.writeDefError(w, r, "list custom field definitions", err)
		return
	}
	response.WriteJSON(w, http.StatusOK, dto.FromListDefinitionsOutput(out))
}

func (h *CustomFieldHandler) DeleteDefinition(w http.ResponseWriter, r *http.Request) {
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

	if err := h.service.DeleteDefinition(r.Context(), principal.CustomerID, req.ID); err != nil {
		h.writeDefError(w, r, "delete custom field definition", err)
		return
	}
	response.WriteJSON(w, http.StatusOK, map[string]any{"id": req.ID, "status": "DELETED"})
}

func (h *CustomFieldHandler) SetValue(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalOr500(w, r, h.logger)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)

	var req dto.SetValueHTTPRequest
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

	out, err := h.service.SetValue(r.Context(), principal.CustomerID, req.ToServiceInput())
	if err != nil {
		var verr domain.ValidationErrors
		switch {
		case errors.As(err, &verr):
			response.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "domain validation failed", domainValidationDetails(verr)...)
		case errors.Is(err, domain.ErrNotFound):
			response.WriteError(w, http.StatusNotFound, "CUSTOM_FIELD_NOT_FOUND", "custom field definition not found")
		default:
			h.logger.Error("set custom field value", "err", err, "request_id", mw.RequestIDFromContext(r.Context()))
			response.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		}
		return
	}
	response.WriteJSON(w, http.StatusOK, dto.FromValueItem(out))
}

func (h *CustomFieldHandler) ListValues(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalOr500(w, r, h.logger)
	if !ok {
		return
	}
	entityID := parseUUIDPtr(r.URL.Query().Get("entity_id"))
	if entityID == nil {
		response.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "entity_id is required")
		return
	}

	out, err := h.service.ListValues(r.Context(), principal.CustomerID, *entityID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			response.WriteError(w, http.StatusNotFound, "ENTITY_NOT_FOUND", "entity not found")
			return
		}
		h.logger.Error("list custom field values", "err", err, "request_id", mw.RequestIDFromContext(r.Context()))
		response.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}
	response.WriteJSON(w, http.StatusOK, dto.FromValueItems(out))
}

func (h *CustomFieldHandler) UnsetValue(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalOr500(w, r, h.logger)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)

	var req dto.UnsetValueHTTPRequest
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

	if err := h.service.UnsetValue(r.Context(), principal.CustomerID, req.EntityID, req.FieldDefinitionID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			response.WriteError(w, http.StatusNotFound, "CUSTOM_FIELD_VALUE_NOT_FOUND", "custom field value not found")
			return
		}
		h.logger.Error("unset custom field value", "err", err, "request_id", mw.RequestIDFromContext(r.Context()))
		response.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}
	response.WriteJSON(w, http.StatusOK, map[string]string{"status": "REMOVED", "message": "custom field value removed"})
}

func (h *CustomFieldHandler) writeDefError(w http.ResponseWriter, r *http.Request, op string, err error) {
	var verr domain.ValidationErrors
	switch {
	case errors.As(err, &verr):
		response.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "domain validation failed", domainValidationDetails(verr)...)
	case errors.Is(err, domain.ErrNotFound):
		response.WriteError(w, http.StatusNotFound, "CUSTOM_FIELD_NOT_FOUND", "custom field definition not found")
	case errors.Is(err, domain.ErrAlreadyExists):
		response.WriteError(w, http.StatusConflict, "CUSTOM_FIELD_EXISTS", "custom field with this key already exists")
	default:
		h.logger.Error(op, "err", err, "request_id", mw.RequestIDFromContext(r.Context()))
		response.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	}
}
