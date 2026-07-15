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

type EntityLinkService interface {
	CreateLink(ctx context.Context, customerID uuid.UUID, in service.CreateLinkInput) (*service.LinkItem, error)
	ListLinks(ctx context.Context, customerID uuid.UUID, in service.ListLinksInput) (*service.ListLinksOutput, error)
	DeleteLink(ctx context.Context, customerID, id uuid.UUID) (*service.DeleteLinkOutput, error)
}

type EntityLinkHandler struct {
	service   EntityLinkService
	validator *validator.Validator
	logger    *slog.Logger
}

func NewEntityLinkHandler(svc EntityLinkService, v *validator.Validator, logger *slog.Logger) *EntityLinkHandler {
	return &EntityLinkHandler{service: svc, validator: v, logger: logger}
}

func (h *EntityLinkHandler) Create(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalOr500(w, r, h.logger)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)

	var req dto.CreateLinkHTTPRequest
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
	out, err := h.service.CreateLink(r.Context(), principal.CustomerID, in)
	if err != nil {
		h.writeLinkError(w, r, "create entity link", err)
		return
	}
	response.WriteJSON(w, http.StatusCreated, dto.FromLinkItem(out))
}

func (h *EntityLinkHandler) List(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalOr500(w, r, h.logger)
	if !ok {
		return
	}
	q := r.URL.Query()
	in := service.ListLinksInput{
		Page:         atoiDefault(q.Get("page"), 0),
		PageSize:     atoiDefault(q.Get("page_size"), 0),
		EntityID:     parseUUIDPtr(q.Get("entity_id")),
		RelationType: q.Get("relation_type"),
		LinkedType:   q.Get("linked_type"),
		Order:        q.Get("order"),
	}
	out, err := h.service.ListLinks(r.Context(), principal.CustomerID, in)
	if err != nil {
		h.writeLinkError(w, r, "list entity links", err)
		return
	}
	response.WriteJSON(w, http.StatusOK, dto.FromListLinksOutput(out))
}

func (h *EntityLinkHandler) Delete(w http.ResponseWriter, r *http.Request) {
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

	out, err := h.service.DeleteLink(r.Context(), principal.CustomerID, req.ID)
	if err != nil {
		h.writeLinkError(w, r, "delete entity link", err)
		return
	}
	response.WriteJSON(w, http.StatusOK, dto.FromDeleteLinkOutput(out))
}

func (h *EntityLinkHandler) writeLinkError(w http.ResponseWriter, r *http.Request, op string, err error) {
	var verr domain.ValidationErrors
	switch {
	case errors.As(err, &verr):
		response.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "domain validation failed", domainValidationDetails(verr)...)
	case errors.Is(err, domain.ErrNotFound):
		response.WriteError(w, http.StatusNotFound, "LINK_NOT_FOUND", "entity link not found")
	case errors.Is(err, domain.ErrAlreadyExists):
		response.WriteError(w, http.StatusConflict, "LINK_EXISTS", "entity link already exists")
	default:
		h.logger.Error(op, "err", err, "request_id", mw.RequestIDFromContext(r.Context()))
		response.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	}
}
