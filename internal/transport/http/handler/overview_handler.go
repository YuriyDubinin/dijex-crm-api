package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/YuriyDubinin/bms-api/internal/domain"
	"github.com/YuriyDubinin/bms-api/internal/service"
	"github.com/YuriyDubinin/bms-api/internal/transport/http/dto"
	mw "github.com/YuriyDubinin/bms-api/internal/transport/http/middleware"
	"github.com/YuriyDubinin/bms-api/internal/transport/http/response"
)

type OverviewService interface {
	DealOverview(ctx context.Context, customerID, dealID uuid.UUID) (*service.DealOverviewOutput, error)
	ClientOverview(ctx context.Context, customerID, clientID uuid.UUID) (*service.ClientOverviewOutput, error)
}

type OverviewHandler struct {
	service OverviewService
	logger  *slog.Logger
}

func NewOverviewHandler(svc OverviewService, logger *slog.Logger) *OverviewHandler {
	return &OverviewHandler{service: svc, logger: logger}
}

// DealOverview обрабатывает GET /api/deals/overview?id=<uuid>.
func (h *OverviewHandler) DealOverview(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalOr500(w, r, h.logger)
	if !ok {
		return
	}
	id := parseUUIDPtr(r.URL.Query().Get("id"))
	if id == nil {
		response.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "id is required")
		return
	}

	out, err := h.service.DealOverview(r.Context(), principal.CustomerID, *id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			response.WriteError(w, http.StatusNotFound, "DEAL_NOT_FOUND", "deal not found")
			return
		}
		h.logger.Error("deal overview", "err", err, "request_id", mw.RequestIDFromContext(r.Context()))
		response.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}
	response.WriteJSON(w, http.StatusOK, dto.FromDealOverview(out))
}

// ClientOverview обрабатывает GET /api/clients/overview?id=<uuid>.
func (h *OverviewHandler) ClientOverview(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalOr500(w, r, h.logger)
	if !ok {
		return
	}
	id := parseUUIDPtr(r.URL.Query().Get("id"))
	if id == nil {
		response.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "id is required")
		return
	}

	out, err := h.service.ClientOverview(r.Context(), principal.CustomerID, *id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			response.WriteError(w, http.StatusNotFound, "CLIENT_NOT_FOUND", "client not found")
			return
		}
		h.logger.Error("client overview", "err", err, "request_id", mw.RequestIDFromContext(r.Context()))
		response.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}
	response.WriteJSON(w, http.StatusOK, dto.FromClientOverview(out))
}
