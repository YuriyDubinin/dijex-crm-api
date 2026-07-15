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

type CustomerService interface {
	GetCustomer(ctx context.Context, customerID uuid.UUID) (*service.CustomerItem, error)
	UpdateCustomer(ctx context.Context, customerID uuid.UUID, actorRole string, in service.UpdateCustomerInput) (*service.CustomerItem, error)
}

type CustomerHandler struct {
	service   CustomerService
	validator *validator.Validator
	logger    *slog.Logger
}

func NewCustomerHandler(svc CustomerService, v *validator.Validator, logger *slog.Logger) *CustomerHandler {
	return &CustomerHandler{service: svc, validator: v, logger: logger}
}

// Get отдаёт текущий тенант (из Principal.CustomerID).
func (h *CustomerHandler) Get(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalOr500(w, r, h.logger)
	if !ok {
		return
	}
	out, err := h.service.GetCustomer(r.Context(), principal.CustomerID)
	if err != nil {
		h.writeCustomerError(w, r, "get customer", err)
		return
	}
	response.WriteJSON(w, http.StatusOK, dto.FromCustomerItem(out))
}

func (h *CustomerHandler) Update(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalOr500(w, r, h.logger)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)

	var req dto.UpdateCustomerHTTPRequest
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

	out, err := h.service.UpdateCustomer(r.Context(), principal.CustomerID, principal.Role, req.ToServiceInput())
	if err != nil {
		h.writeCustomerError(w, r, "update customer", err)
		return
	}
	response.WriteJSON(w, http.StatusOK, dto.FromCustomerItem(out))
}

func (h *CustomerHandler) writeCustomerError(w http.ResponseWriter, r *http.Request, op string, err error) {
	var verr domain.ValidationErrors
	switch {
	case errors.As(err, &verr):
		response.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "domain validation failed", domainValidationDetails(verr)...)
	case errors.Is(err, domain.ErrForbidden):
		response.WriteError(w, http.StatusForbidden, "FORBIDDEN", "insufficient permissions")
	case errors.Is(err, domain.ErrNotFound):
		response.WriteError(w, http.StatusNotFound, "CUSTOMER_NOT_FOUND", "customer not found")
	default:
		h.logger.Error(op, "err", err, "request_id", mw.RequestIDFromContext(r.Context()))
		response.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	}
}
