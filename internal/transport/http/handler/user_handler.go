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

type UserService interface {
	CreateUser(ctx context.Context, customerID uuid.UUID, actorRole string, in service.CreateUserInput) (*service.UserItem, error)
	UpdateUser(ctx context.Context, customerID uuid.UUID, actorRole string, in service.UpdateUserInput) (*service.UserItem, error)
	ListUsers(ctx context.Context, customerID uuid.UUID, in service.ListUsersInput) (*service.ListUsersOutput, error)
	DeleteUser(ctx context.Context, customerID uuid.UUID, actorRole string, id uuid.UUID) (*service.DeleteUserOutput, error)
}

type UserHandler struct {
	service   UserService
	validator *validator.Validator
	logger    *slog.Logger
}

func NewUserHandler(svc UserService, v *validator.Validator, logger *slog.Logger) *UserHandler {
	return &UserHandler{service: svc, validator: v, logger: logger}
}

func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalOr500(w, r, h.logger)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)

	var req dto.CreateUserHTTPRequest
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

	out, err := h.service.CreateUser(r.Context(), principal.CustomerID, principal.Role, req.ToServiceInput())
	if err != nil {
		h.writeUserError(w, r, "create user", err)
		return
	}
	response.WriteJSON(w, http.StatusCreated, dto.FromUserItem(out))
}

func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalOr500(w, r, h.logger)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)

	var req dto.UpdateUserHTTPRequest
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

	out, err := h.service.UpdateUser(r.Context(), principal.CustomerID, principal.Role, req.ToServiceInput())
	if err != nil {
		h.writeUserError(w, r, "update user", err)
		return
	}
	response.WriteJSON(w, http.StatusOK, dto.FromUserItem(out))
}

func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalOr500(w, r, h.logger)
	if !ok {
		return
	}
	q := r.URL.Query()
	in := service.ListUsersInput{
		Page:     atoiDefault(q.Get("page"), 0),
		PageSize: atoiDefault(q.Get("page_size"), 0),
		Role:     q.Get("role"),
		IsActive: parseBoolPtr(q.Get("is_active")),
		Search:   q.Get("search"),
		SortBy:   q.Get("sort_by"),
		Order:    q.Get("order"),
	}
	out, err := h.service.ListUsers(r.Context(), principal.CustomerID, in)
	if err != nil {
		h.writeUserError(w, r, "list users", err)
		return
	}
	response.WriteJSON(w, http.StatusOK, dto.FromListUsersOutput(out))
}

func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
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

	out, err := h.service.DeleteUser(r.Context(), principal.CustomerID, principal.Role, req.ID)
	if err != nil {
		h.writeUserError(w, r, "delete user", err)
		return
	}
	response.WriteJSON(w, http.StatusOK, dto.FromDeleteUserOutput(out))
}

func (h *UserHandler) writeUserError(w http.ResponseWriter, r *http.Request, op string, err error) {
	var verr domain.ValidationErrors
	switch {
	case errors.As(err, &verr):
		response.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "domain validation failed", domainValidationDetails(verr)...)
	case errors.Is(err, domain.ErrForbidden):
		response.WriteError(w, http.StatusForbidden, "FORBIDDEN", "insufficient permissions")
	case errors.Is(err, domain.ErrNotFound):
		response.WriteError(w, http.StatusNotFound, "USER_NOT_FOUND", "user not found")
	case errors.Is(err, domain.ErrAlreadyExists):
		response.WriteError(w, http.StatusConflict, "USER_EXISTS", "user with this email already exists")
	default:
		h.logger.Error(op, "err", err, "request_id", mw.RequestIDFromContext(r.Context()))
		response.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	}
}
