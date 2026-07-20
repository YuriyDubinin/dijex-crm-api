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

type ProcessService interface {
	CreateProcess(ctx context.Context, customerID uuid.UUID, in service.CreateProcessInput) (*service.ProcessItem, error)
	UpdateProcess(ctx context.Context, customerID uuid.UUID, in service.UpdateProcessInput) (*service.ProcessItem, error)
	ListProcesses(ctx context.Context, customerID uuid.UUID, in service.ListProcessesInput) (*service.ListProcessesOutput, error)
	DeleteProcess(ctx context.Context, customerID, id uuid.UUID) (*service.DeleteProcessOutput, error)
	CreateStage(ctx context.Context, customerID uuid.UUID, in service.CreateProcessStageInput) (*service.ProcessStageItem, error)
	ListStages(ctx context.Context, customerID, processID uuid.UUID) ([]service.ProcessStageItem, error)
	UpdateStage(ctx context.Context, customerID uuid.UUID, in service.UpdateProcessStageInput) (*service.ProcessStageItem, error)
	DeleteStage(ctx context.Context, customerID, id uuid.UUID) (*service.DeleteProcessStageOutput, error)
}

type ProcessHandler struct {
	service   ProcessService
	validator *validator.Validator
	logger    *slog.Logger
}

func NewProcessHandler(svc ProcessService, v *validator.Validator, logger *slog.Logger) *ProcessHandler {
	return &ProcessHandler{service: svc, validator: v, logger: logger}
}

// ───────────────────────── Процессы ─────────────────────────

func (h *ProcessHandler) Create(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalOr500(w, r, h.logger)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)

	var req dto.CreateProcessHTTPRequest
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

	out, err := h.service.CreateProcess(r.Context(), principal.CustomerID, req.ToServiceInput())
	if err != nil {
		h.writeProcessError(w, r, "create process", err)
		return
	}
	response.WriteJSON(w, http.StatusCreated, dto.FromProcessItem(out))
}

func (h *ProcessHandler) Update(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalOr500(w, r, h.logger)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)

	var req dto.UpdateProcessHTTPRequest
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

	out, err := h.service.UpdateProcess(r.Context(), principal.CustomerID, req.ToServiceInput())
	if err != nil {
		h.writeProcessError(w, r, "update process", err)
		return
	}
	response.WriteJSON(w, http.StatusOK, dto.FromProcessItem(out))
}

func (h *ProcessHandler) List(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalOr500(w, r, h.logger)
	if !ok {
		return
	}
	q := r.URL.Query()
	in := service.ListProcessesInput{
		Page:     atoiDefault(q.Get("page"), 0),
		PageSize: atoiDefault(q.Get("page_size"), 0),
		Status:   q.Get("status"),
		Search:   q.Get("search"),
		SortBy:   q.Get("sort_by"),
		Order:    q.Get("order"),
	}
	out, err := h.service.ListProcesses(r.Context(), principal.CustomerID, in)
	if err != nil {
		h.writeProcessError(w, r, "list processes", err)
		return
	}
	response.WriteJSON(w, http.StatusOK, dto.FromListProcessesOutput(out))
}

func (h *ProcessHandler) Delete(w http.ResponseWriter, r *http.Request) {
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

	out, err := h.service.DeleteProcess(r.Context(), principal.CustomerID, req.ID)
	if err != nil {
		h.writeProcessError(w, r, "delete process", err)
		return
	}
	response.WriteJSON(w, http.StatusOK, dto.FromDeleteProcessOutput(out))
}

// ───────────────────────── Этапы ─────────────────────────

func (h *ProcessHandler) CreateStage(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalOr500(w, r, h.logger)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)

	var req dto.CreateProcessStageHTTPRequest
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

	out, err := h.service.CreateStage(r.Context(), principal.CustomerID, req.ToServiceInput())
	if err != nil {
		h.writeStageError(w, r, "create process stage", err)
		return
	}
	response.WriteJSON(w, http.StatusCreated, dto.FromProcessStageItem(out))
}

func (h *ProcessHandler) ListStages(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalOr500(w, r, h.logger)
	if !ok {
		return
	}
	processID := parseUUIDPtr(r.URL.Query().Get("process_id"))
	if processID == nil {
		response.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "process_id is required")
		return
	}
	out, err := h.service.ListStages(r.Context(), principal.CustomerID, *processID)
	if err != nil {
		h.writeStageError(w, r, "list process stages", err)
		return
	}
	response.WriteJSON(w, http.StatusOK, dto.FromProcessStageItems(out))
}

func (h *ProcessHandler) UpdateStage(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalOr500(w, r, h.logger)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)

	var req dto.UpdateProcessStageHTTPRequest
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

	out, err := h.service.UpdateStage(r.Context(), principal.CustomerID, req.ToServiceInput())
	if err != nil {
		h.writeStageError(w, r, "update process stage", err)
		return
	}
	response.WriteJSON(w, http.StatusOK, dto.FromProcessStageItem(out))
}

func (h *ProcessHandler) DeleteStage(w http.ResponseWriter, r *http.Request) {
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

	out, err := h.service.DeleteStage(r.Context(), principal.CustomerID, req.ID)
	if err != nil {
		h.writeStageError(w, r, "delete process stage", err)
		return
	}
	response.WriteJSON(w, http.StatusOK, dto.FromDeleteProcessStageOutput(out))
}

// ───────────────────────── ошибки ─────────────────────────

func (h *ProcessHandler) writeProcessError(w http.ResponseWriter, r *http.Request, op string, err error) {
	var verr domain.ValidationErrors
	switch {
	case errors.As(err, &verr):
		response.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "domain validation failed", domainValidationDetails(verr)...)
	case errors.Is(err, domain.ErrNotFound):
		response.WriteError(w, http.StatusNotFound, "PROCESS_NOT_FOUND", "process not found")
	case errors.Is(err, domain.ErrAlreadyExists):
		response.WriteError(w, http.StatusConflict, "PROCESS_EXISTS", "process already exists")
	default:
		h.logger.Error(op, "err", err, "request_id", mw.RequestIDFromContext(r.Context()))
		response.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	}
}

func (h *ProcessHandler) writeStageError(w http.ResponseWriter, r *http.Request, op string, err error) {
	var verr domain.ValidationErrors
	switch {
	case errors.As(err, &verr):
		response.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "domain validation failed", domainValidationDetails(verr)...)
	case errors.Is(err, domain.ErrNotFound):
		response.WriteError(w, http.StatusNotFound, "PROCESS_STAGE_NOT_FOUND", "process stage not found")
	default:
		h.logger.Error(op, "err", err, "request_id", mw.RequestIDFromContext(r.Context()))
		response.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	}
}
