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

type TagService interface {
	CreateTag(ctx context.Context, customerID uuid.UUID, in service.CreateTagInput) (*service.TagItem, error)
	UpdateTag(ctx context.Context, customerID uuid.UUID, in service.UpdateTagInput) (*service.TagItem, error)
	ListTags(ctx context.Context, customerID uuid.UUID, in service.ListTagsInput) (*service.ListTagsOutput, error)
	DeleteTag(ctx context.Context, customerID, id uuid.UUID) (*service.DeleteTagOutput, error)
	AttachTag(ctx context.Context, customerID, tagID, entityID uuid.UUID) error
	DetachTag(ctx context.Context, customerID, tagID, entityID uuid.UUID) error
}

type TagHandler struct {
	service   TagService
	validator *validator.Validator
	logger    *slog.Logger
}

func NewTagHandler(svc TagService, v *validator.Validator, logger *slog.Logger) *TagHandler {
	return &TagHandler{service: svc, validator: v, logger: logger}
}

func (h *TagHandler) Create(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalOr500(w, r, h.logger)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)

	var req dto.CreateTagHTTPRequest
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

	out, err := h.service.CreateTag(r.Context(), principal.CustomerID, req.ToServiceInput())
	if err != nil {
		h.writeTagError(w, r, "create tag", err)
		return
	}
	response.WriteJSON(w, http.StatusCreated, dto.FromTagItem(out))
}

func (h *TagHandler) Update(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalOr500(w, r, h.logger)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)

	var req dto.UpdateTagHTTPRequest
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

	out, err := h.service.UpdateTag(r.Context(), principal.CustomerID, req.ToServiceInput())
	if err != nil {
		h.writeTagError(w, r, "update tag", err)
		return
	}
	response.WriteJSON(w, http.StatusOK, dto.FromTagItem(out))
}

func (h *TagHandler) List(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalOr500(w, r, h.logger)
	if !ok {
		return
	}
	q := r.URL.Query()
	in := service.ListTagsInput{
		Page:     atoiDefault(q.Get("page"), 0),
		PageSize: atoiDefault(q.Get("page_size"), 0),
		EntityID: parseUUIDPtr(q.Get("entity_id")),
		Search:   q.Get("search"),
		Order:    q.Get("order"),
	}
	out, err := h.service.ListTags(r.Context(), principal.CustomerID, in)
	if err != nil {
		h.writeTagError(w, r, "list tags", err)
		return
	}
	response.WriteJSON(w, http.StatusOK, dto.FromListTagsOutput(out))
}

func (h *TagHandler) Delete(w http.ResponseWriter, r *http.Request) {
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

	out, err := h.service.DeleteTag(r.Context(), principal.CustomerID, req.ID)
	if err != nil {
		h.writeTagError(w, r, "delete tag", err)
		return
	}
	response.WriteJSON(w, http.StatusOK, dto.FromDeleteTagOutput(out))
}

func (h *TagHandler) Attach(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalOr500(w, r, h.logger)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)

	var req dto.TagEntityHTTPRequest
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

	if err := h.service.AttachTag(r.Context(), principal.CustomerID, req.TagID, req.EntityID); err != nil {
		h.writeTagLinkError(w, r, "attach tag", err)
		return
	}
	response.WriteJSON(w, http.StatusOK, map[string]string{"status": "ATTACHED"})
}

func (h *TagHandler) Detach(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalOr500(w, r, h.logger)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)

	var req dto.TagEntityHTTPRequest
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

	if err := h.service.DetachTag(r.Context(), principal.CustomerID, req.TagID, req.EntityID); err != nil {
		h.writeTagLinkError(w, r, "detach tag", err)
		return
	}
	response.WriteJSON(w, http.StatusOK, map[string]string{"status": "DETACHED"})
}

func (h *TagHandler) writeTagError(w http.ResponseWriter, r *http.Request, op string, err error) {
	var verr domain.ValidationErrors
	switch {
	case errors.As(err, &verr):
		response.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "domain validation failed", domainValidationDetails(verr)...)
	case errors.Is(err, domain.ErrNotFound):
		response.WriteError(w, http.StatusNotFound, "TAG_NOT_FOUND", "tag not found")
	case errors.Is(err, domain.ErrAlreadyExists):
		response.WriteError(w, http.StatusConflict, "TAG_EXISTS", "tag already exists")
	default:
		h.logger.Error(op, "err", err, "request_id", mw.RequestIDFromContext(r.Context()))
		response.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	}
}

func (h *TagHandler) writeTagLinkError(w http.ResponseWriter, r *http.Request, op string, err error) {
	var verr domain.ValidationErrors
	switch {
	case errors.As(err, &verr):
		response.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "domain validation failed", domainValidationDetails(verr)...)
	case errors.Is(err, domain.ErrNotFound):
		response.WriteError(w, http.StatusNotFound, "TAG_NOT_FOUND", "tag not found")
	default:
		h.logger.Error(op, "err", err, "request_id", mw.RequestIDFromContext(r.Context()))
		response.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	}
}
