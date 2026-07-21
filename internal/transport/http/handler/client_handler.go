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

type ClientService interface {
	CreateClient(ctx context.Context, customerID uuid.UUID, in service.CreateClientInput) (*service.ClientItem, error)
	UpdateClient(ctx context.Context, customerID uuid.UUID, in service.UpdateClientInput) (*service.ClientItem, error)
	ListClients(ctx context.Context, customerID uuid.UUID, in service.ListClientsInput) (*service.ListClientsOutput, error)
	DeleteClient(ctx context.Context, customerID, id uuid.UUID) (*service.DeleteClientOutput, error)
	AttachProject(ctx context.Context, customerID, clientID, projectID uuid.UUID) error
	DetachProject(ctx context.Context, customerID, clientID, projectID uuid.UUID) error
	ListClientProjects(ctx context.Context, customerID, clientID uuid.UUID) ([]service.ProjectItem, error)
}

type ClientHandler struct {
	service   ClientService
	validator *validator.Validator
	logger    *slog.Logger
}

func NewClientHandler(svc ClientService, v *validator.Validator, logger *slog.Logger) *ClientHandler {
	return &ClientHandler{service: svc, validator: v, logger: logger}
}

func (h *ClientHandler) Create(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalOr500(w, r, h.logger)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)

	var req dto.CreateClientHTTPRequest
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

	out, err := h.service.CreateClient(r.Context(), principal.CustomerID, req.ToServiceInput())
	if err != nil {
		h.writeServiceError(w, r, "create client", err)
		return
	}
	response.WriteJSON(w, http.StatusCreated, dto.FromClientItem(out))
}

func (h *ClientHandler) Update(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalOr500(w, r, h.logger)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)

	var req dto.UpdateClientHTTPRequest
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

	out, err := h.service.UpdateClient(r.Context(), principal.CustomerID, req.ToServiceInput())
	if err != nil {
		h.writeServiceError(w, r, "update client", err)
		return
	}
	response.WriteJSON(w, http.StatusOK, dto.FromClientItem(out))
}

func (h *ClientHandler) List(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalOr500(w, r, h.logger)
	if !ok {
		return
	}
	q := r.URL.Query()
	in := service.ListClientsInput{
		Page:        atoiDefault(q.Get("page"), 0),
		PageSize:    atoiDefault(q.Get("page_size"), 0),
		ProjectID:   parseUUIDPtr(q.Get("project_id")),
		Status:      q.Get("status"),
		SubjectType: q.Get("subject_type"),
		Search:      q.Get("search"),
		SortBy:      q.Get("sort_by"),
		Order:       q.Get("order"),
	}
	out, err := h.service.ListClients(r.Context(), principal.CustomerID, in)
	if err != nil {
		h.writeServiceError(w, r, "list clients", err)
		return
	}
	response.WriteJSON(w, http.StatusOK, dto.FromListClientsOutput(out))
}

func (h *ClientHandler) Delete(w http.ResponseWriter, r *http.Request) {
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

	out, err := h.service.DeleteClient(r.Context(), principal.CustomerID, req.ID)
	if err != nil {
		h.writeServiceError(w, r, "delete client", err)
		return
	}
	response.WriteJSON(w, http.StatusOK, dto.FromDeleteClientOutput(out))
}

// AttachProject — POST /clients/projects/attach: добавить клиента в проект (M:N).
func (h *ClientHandler) AttachProject(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalOr500(w, r, h.logger)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)

	var req dto.ClientProjectHTTPRequest
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
	if err := h.service.AttachProject(r.Context(), principal.CustomerID, req.ClientID, req.ProjectID); err != nil {
		h.writeServiceError(w, r, "attach client project", err)
		return
	}
	response.WriteJSON(w, http.StatusOK, map[string]string{"status": "ATTACHED"})
}

// DetachProject — DELETE /clients/projects/detach: убрать клиента из проекта.
func (h *ClientHandler) DetachProject(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalOr500(w, r, h.logger)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)

	var req dto.ClientProjectHTTPRequest
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
	if err := h.service.DetachProject(r.Context(), principal.CustomerID, req.ClientID, req.ProjectID); err != nil {
		h.writeServiceError(w, r, "detach client project", err)
		return
	}
	response.WriteJSON(w, http.StatusOK, map[string]string{"status": "DETACHED"})
}

// ListProjects — GET /clients/projects/list?client_id=: проекты клиента.
func (h *ClientHandler) ListProjects(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalOr500(w, r, h.logger)
	if !ok {
		return
	}
	clientID := parseUUIDPtr(r.URL.Query().Get("client_id"))
	if clientID == nil {
		response.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "client_id is required")
		return
	}
	out, err := h.service.ListClientProjects(r.Context(), principal.CustomerID, *clientID)
	if err != nil {
		h.writeServiceError(w, r, "list client projects", err)
		return
	}
	response.WriteJSON(w, http.StatusOK, dto.FromClientProjects(out))
}

func (h *ClientHandler) writeServiceError(w http.ResponseWriter, r *http.Request, op string, err error) {
	var verr domain.ValidationErrors
	switch {
	case errors.As(err, &verr):
		response.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "domain validation failed", domainValidationDetails(verr)...)
	case errors.Is(err, domain.ErrNotFound):
		response.WriteError(w, http.StatusNotFound, "CLIENT_NOT_FOUND", "client not found")
	case errors.Is(err, domain.ErrAlreadyExists):
		response.WriteError(w, http.StatusConflict, "CLIENT_EXISTS", "client already exists")
	default:
		h.logger.Error(op, "err", err, "request_id", mw.RequestIDFromContext(r.Context()))
		response.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	}
}
