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

type ServiceService interface {
	CreateService(ctx context.Context, customerID uuid.UUID, in service.CreateServiceInput) (*service.ServiceItem, error)
	UpdateService(ctx context.Context, customerID uuid.UUID, in service.UpdateServiceInput) (*service.ServiceItem, error)
	ListServices(ctx context.Context, customerID uuid.UUID, in service.ListServicesInput) (*service.ListServicesOutput, error)
	DeleteService(ctx context.Context, customerID, id uuid.UUID) (*service.DeleteServiceOutput, error)
}

type ServiceHandler struct {
	service   ServiceService
	validator *validator.Validator
	logger    *slog.Logger
}

func NewServiceHandler(svc ServiceService, v *validator.Validator, logger *slog.Logger) *ServiceHandler {
	return &ServiceHandler{service: svc, validator: v, logger: logger}
}

func (h *ServiceHandler) Create(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalOr500(w, r, h.logger)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)

	var req dto.CreateServiceHTTPRequest
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

	out, err := h.service.CreateService(r.Context(), principal.CustomerID, req.ToServiceInput())
	if err != nil {
		h.writeServiceError(w, r, "create service", err)
		return
	}
	response.WriteJSON(w, http.StatusCreated, dto.FromServiceItem(out))
}

func (h *ServiceHandler) Update(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalOr500(w, r, h.logger)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)

	var req dto.UpdateServiceHTTPRequest
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

	out, err := h.service.UpdateService(r.Context(), principal.CustomerID, req.ToServiceInput())
	if err != nil {
		h.writeServiceError(w, r, "update service", err)
		return
	}
	response.WriteJSON(w, http.StatusOK, dto.FromServiceItem(out))
}

func (h *ServiceHandler) List(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalOr500(w, r, h.logger)
	if !ok {
		return
	}
	q := r.URL.Query()
	in := service.ListServicesInput{
		Page:      atoiDefault(q.Get("page"), 0),
		PageSize:  atoiDefault(q.Get("page_size"), 0),
		ProjectID: parseUUIDPtr(q.Get("project_id")),
		Status:    q.Get("status"),
		Category:  q.Get("category"),
		Search:    q.Get("search"),
		SortBy:    q.Get("sort_by"),
		Order:     q.Get("order"),
	}
	out, err := h.service.ListServices(r.Context(), principal.CustomerID, in)
	if err != nil {
		h.writeServiceError(w, r, "list services", err)
		return
	}
	response.WriteJSON(w, http.StatusOK, dto.FromListServicesOutput(out))
}

func (h *ServiceHandler) Delete(w http.ResponseWriter, r *http.Request) {
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

	out, err := h.service.DeleteService(r.Context(), principal.CustomerID, req.ID)
	if err != nil {
		h.writeServiceError(w, r, "delete service", err)
		return
	}
	response.WriteJSON(w, http.StatusOK, dto.FromDeleteServiceOutput(out))
}

func (h *ServiceHandler) writeServiceError(w http.ResponseWriter, r *http.Request, op string, err error) {
	var verr domain.ValidationErrors
	switch {
	case errors.As(err, &verr):
		response.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "domain validation failed", domainValidationDetails(verr)...)
	case errors.Is(err, domain.ErrNotFound):
		response.WriteError(w, http.StatusNotFound, "SERVICE_NOT_FOUND", "service not found")
	case errors.Is(err, domain.ErrAlreadyExists):
		response.WriteError(w, http.StatusConflict, "SERVICE_EXISTS", "service already exists")
	default:
		h.logger.Error(op, "err", err, "request_id", mw.RequestIDFromContext(r.Context()))
		response.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	}
}
