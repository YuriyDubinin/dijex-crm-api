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

// ProjectService — узкий контракт хендлера. Реализуется *service.ProjectService.
type ProjectService interface {
	CreateProject(ctx context.Context, customerID uuid.UUID, in service.CreateProjectInput) (*service.ProjectItem, error)
	UpdateProject(ctx context.Context, customerID uuid.UUID, in service.UpdateProjectInput) (*service.ProjectItem, error)
	ListProjects(ctx context.Context, customerID uuid.UUID, in service.ListProjectsInput) (*service.ListProjectsOutput, error)
	DeleteProject(ctx context.Context, customerID, id uuid.UUID) (*service.DeleteProjectOutput, error)
}

type ProjectHandler struct {
	service   ProjectService
	validator *validator.Validator
	logger    *slog.Logger
}

func NewProjectHandler(svc ProjectService, v *validator.Validator, logger *slog.Logger) *ProjectHandler {
	return &ProjectHandler{service: svc, validator: v, logger: logger}
}

func (h *ProjectHandler) Create(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalOr500(w, r, h.logger)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)

	var req dto.CreateProjectHTTPRequest
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

	out, err := h.service.CreateProject(r.Context(), principal.CustomerID, req.ToServiceInput())
	if err != nil {
		h.writeServiceError(w, r, "create project", err)
		return
	}
	response.WriteJSON(w, http.StatusCreated, dto.FromProjectItem(out))
}

func (h *ProjectHandler) Update(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalOr500(w, r, h.logger)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)

	var req dto.UpdateProjectHTTPRequest
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

	out, err := h.service.UpdateProject(r.Context(), principal.CustomerID, req.ToServiceInput())
	if err != nil {
		h.writeServiceError(w, r, "update project", err)
		return
	}
	response.WriteJSON(w, http.StatusOK, dto.FromProjectItem(out))
}

func (h *ProjectHandler) List(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalOr500(w, r, h.logger)
	if !ok {
		return
	}
	q := r.URL.Query()
	in := service.ListProjectsInput{
		Page:     atoiDefault(q.Get("page"), 0),
		PageSize: atoiDefault(q.Get("page_size"), 0),
		Status:   q.Get("status"),
		Search:   q.Get("search"),
		From:     parseStatsBound(q.Get("from"), false),
		To:       parseStatsBound(q.Get("to"), true),
		SortBy:   q.Get("sort_by"),
		Order:    q.Get("order"),
	}
	out, err := h.service.ListProjects(r.Context(), principal.CustomerID, in)
	if err != nil {
		h.writeServiceError(w, r, "list projects", err)
		return
	}
	response.WriteJSON(w, http.StatusOK, dto.FromListProjectsOutput(out))
}

func (h *ProjectHandler) Delete(w http.ResponseWriter, r *http.Request) {
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

	out, err := h.service.DeleteProject(r.Context(), principal.CustomerID, req.ID)
	if err != nil {
		h.writeServiceError(w, r, "delete project", err)
		return
	}
	response.WriteJSON(w, http.StatusOK, dto.FromDeleteProjectOutput(out))
}

// writeServiceError — единый маппинг доменных ошибок проекта в HTTP.
func (h *ProjectHandler) writeServiceError(w http.ResponseWriter, r *http.Request, op string, err error) {
	var verr domain.ValidationErrors
	switch {
	case errors.As(err, &verr):
		response.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "domain validation failed", domainValidationDetails(verr)...)
	case errors.Is(err, domain.ErrNotFound):
		response.WriteError(w, http.StatusNotFound, "PROJECT_NOT_FOUND", "project not found")
	case errors.Is(err, domain.ErrAlreadyExists):
		response.WriteError(w, http.StatusConflict, "PROJECT_EXISTS", "project with this slug already exists")
	default:
		h.logger.Error(op, "err", err, "request_id", mw.RequestIDFromContext(r.Context()))
		response.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	}
}
