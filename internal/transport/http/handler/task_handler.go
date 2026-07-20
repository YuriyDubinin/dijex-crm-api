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

type TaskService interface {
	CreateTask(ctx context.Context, customerID uuid.UUID, in service.CreateTaskInput) (*service.TaskItem, error)
	UpdateTask(ctx context.Context, customerID uuid.UUID, in service.UpdateTaskInput) (*service.TaskItem, error)
	ListTasks(ctx context.Context, customerID uuid.UUID, in service.ListTasksInput) (*service.ListTasksOutput, error)
	DeleteTask(ctx context.Context, customerID, id uuid.UUID) (*service.DeleteTaskOutput, error)
}

type TaskHandler struct {
	service   TaskService
	validator *validator.Validator
	logger    *slog.Logger
}

func NewTaskHandler(svc TaskService, v *validator.Validator, logger *slog.Logger) *TaskHandler {
	return &TaskHandler{service: svc, validator: v, logger: logger}
}

func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalOr500(w, r, h.logger)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)

	var req dto.CreateTaskHTTPRequest
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
	out, err := h.service.CreateTask(r.Context(), principal.CustomerID, in)
	if err != nil {
		h.writeServiceError(w, r, "create task", err)
		return
	}
	response.WriteJSON(w, http.StatusCreated, dto.FromTaskItem(out))
}

func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalOr500(w, r, h.logger)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)

	var req dto.UpdateTaskHTTPRequest
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

	out, err := h.service.UpdateTask(r.Context(), principal.CustomerID, req.ToServiceInput())
	if err != nil {
		h.writeServiceError(w, r, "update task", err)
		return
	}
	response.WriteJSON(w, http.StatusOK, dto.FromTaskItem(out))
}

func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalOr500(w, r, h.logger)
	if !ok {
		return
	}
	q := r.URL.Query()
	in := service.ListTasksInput{
		Page:           atoiDefault(q.Get("page"), 0),
		PageSize:       atoiDefault(q.Get("page_size"), 0),
		ProjectID:      parseUUIDPtr(q.Get("project_id")),
		ClientID:       parseUUIDPtr(q.Get("client_id")),
		DealID:         parseUUIDPtr(q.Get("deal_id")),
		ServiceID:      parseUUIDPtr(q.Get("service_id")),
		ProcessID:      parseUUIDPtr(q.Get("process_id")),
		ProcessStageID: parseUUIDPtr(q.Get("process_stage_id")),
		Status:         q.Get("status"),
		Priority:       q.Get("priority"),
		AssignedTo:     parseUUIDPtr(q.Get("assigned_to")),
		Search:         q.Get("search"),
		DueBefore:      parseTimePtr(q.Get("due_before")),
		DueAfter:       parseTimePtr(q.Get("due_after")),
		SortBy:         q.Get("sort_by"),
		Order:          q.Get("order"),
	}
	out, err := h.service.ListTasks(r.Context(), principal.CustomerID, in)
	if err != nil {
		h.writeServiceError(w, r, "list tasks", err)
		return
	}
	response.WriteJSON(w, http.StatusOK, dto.FromListTasksOutput(out))
}

func (h *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
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

	out, err := h.service.DeleteTask(r.Context(), principal.CustomerID, req.ID)
	if err != nil {
		h.writeServiceError(w, r, "delete task", err)
		return
	}
	response.WriteJSON(w, http.StatusOK, dto.FromDeleteTaskOutput(out))
}

func (h *TaskHandler) writeServiceError(w http.ResponseWriter, r *http.Request, op string, err error) {
	var verr domain.ValidationErrors
	switch {
	case errors.As(err, &verr):
		response.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "domain validation failed", domainValidationDetails(verr)...)
	case errors.Is(err, domain.ErrNotFound):
		response.WriteError(w, http.StatusNotFound, "TASK_NOT_FOUND", "task not found")
	case errors.Is(err, domain.ErrAlreadyExists):
		response.WriteError(w, http.StatusConflict, "TASK_EXISTS", "task already exists")
	default:
		h.logger.Error(op, "err", err, "request_id", mw.RequestIDFromContext(r.Context()))
		response.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	}
}
