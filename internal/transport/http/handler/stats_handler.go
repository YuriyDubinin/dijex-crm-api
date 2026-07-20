package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/YuriyDubinin/bms-api/internal/domain"
	"github.com/YuriyDubinin/bms-api/internal/service"
	"github.com/YuriyDubinin/bms-api/internal/transport/http/dto"
	mw "github.com/YuriyDubinin/bms-api/internal/transport/http/middleware"
	"github.com/YuriyDubinin/bms-api/internal/transport/http/response"
)

type StatsService interface {
	Summary(ctx context.Context, customerID uuid.UUID, p service.StatsPeriod) (*domain.SummaryStats, error)
	DealStats(ctx context.Context, customerID uuid.UUID, p service.StatsPeriod) (*domain.DealStats, error)
	ClientStats(ctx context.Context, customerID uuid.UUID, p service.StatsPeriod) (*domain.ClientStats, error)
	TaskStats(ctx context.Context, customerID uuid.UUID, p service.StatsPeriod) (*domain.TaskStats, error)
	ProjectStats(ctx context.Context, customerID uuid.UUID, p service.StatsPeriod) (*domain.ProjectStats, error)
	ServiceStats(ctx context.Context, customerID uuid.UUID, p service.StatsPeriod) (*domain.ServiceStats, error)
	ProcessStats(ctx context.Context, customerID uuid.UUID, p service.StatsPeriod) (*domain.ProcessStats, error)
	TimeSeries(ctx context.Context, customerID uuid.UUID, metric, interval, currency string, p service.StatsPeriod) (*service.TimeSeriesResult, error)
}

type StatsHandler struct {
	service StatsService
	logger  *slog.Logger
}

func NewStatsHandler(svc StatsService, logger *slog.Logger) *StatsHandler {
	return &StatsHandler{service: svc, logger: logger}
}

// parseStatsBound принимает RFC3339 или дату YYYY-MM-DD; для верхней границы
// (isEnd) дата трактуется как конец дня, чтобы период был включающим.
func parseStatsBound(s string, isEnd bool) *time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return &t
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		if isEnd {
			t = t.Add(24*time.Hour - time.Second)
		}
		return &t
	}
	return nil
}

func statsPeriod(r *http.Request) service.StatsPeriod {
	q := r.URL.Query()
	return service.StatsPeriod{
		From: parseStatsBound(q.Get("from"), false),
		To:   parseStatsBound(q.Get("to"), true),
	}
}

func (h *StatsHandler) Summary(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalOr500(w, r, h.logger)
	if !ok {
		return
	}
	out, err := h.service.Summary(r.Context(), principal.CustomerID, statsPeriod(r))
	if err != nil {
		h.writeError(w, r, "stats summary", err)
		return
	}
	response.WriteJSON(w, http.StatusOK, dto.FromSummaryStats(out))
}

func (h *StatsHandler) Deals(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalOr500(w, r, h.logger)
	if !ok {
		return
	}
	out, err := h.service.DealStats(r.Context(), principal.CustomerID, statsPeriod(r))
	if err != nil {
		h.writeError(w, r, "stats deals", err)
		return
	}
	response.WriteJSON(w, http.StatusOK, dto.FromDealStats(out))
}

func (h *StatsHandler) Clients(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalOr500(w, r, h.logger)
	if !ok {
		return
	}
	out, err := h.service.ClientStats(r.Context(), principal.CustomerID, statsPeriod(r))
	if err != nil {
		h.writeError(w, r, "stats clients", err)
		return
	}
	response.WriteJSON(w, http.StatusOK, dto.FromClientStats(out))
}

func (h *StatsHandler) Tasks(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalOr500(w, r, h.logger)
	if !ok {
		return
	}
	out, err := h.service.TaskStats(r.Context(), principal.CustomerID, statsPeriod(r))
	if err != nil {
		h.writeError(w, r, "stats tasks", err)
		return
	}
	response.WriteJSON(w, http.StatusOK, dto.FromTaskStats(out))
}

func (h *StatsHandler) Projects(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalOr500(w, r, h.logger)
	if !ok {
		return
	}
	out, err := h.service.ProjectStats(r.Context(), principal.CustomerID, statsPeriod(r))
	if err != nil {
		h.writeError(w, r, "stats projects", err)
		return
	}
	response.WriteJSON(w, http.StatusOK, dto.FromProjectStats(out))
}

func (h *StatsHandler) Services(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalOr500(w, r, h.logger)
	if !ok {
		return
	}
	out, err := h.service.ServiceStats(r.Context(), principal.CustomerID, statsPeriod(r))
	if err != nil {
		h.writeError(w, r, "stats services", err)
		return
	}
	response.WriteJSON(w, http.StatusOK, dto.FromServiceStats(out))
}

func (h *StatsHandler) Processes(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalOr500(w, r, h.logger)
	if !ok {
		return
	}
	out, err := h.service.ProcessStats(r.Context(), principal.CustomerID, statsPeriod(r))
	if err != nil {
		h.writeError(w, r, "stats processes", err)
		return
	}
	response.WriteJSON(w, http.StatusOK, dto.FromProcessStats(out))
}

func (h *StatsHandler) TimeSeries(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalOr500(w, r, h.logger)
	if !ok {
		return
	}
	q := r.URL.Query()
	out, err := h.service.TimeSeries(r.Context(), principal.CustomerID, q.Get("metric"), q.Get("interval"), q.Get("currency"), statsPeriod(r))
	if err != nil {
		h.writeError(w, r, "stats timeseries", err)
		return
	}
	response.WriteJSON(w, http.StatusOK, dto.FromTimeSeries(out))
}

func (h *StatsHandler) writeError(w http.ResponseWriter, r *http.Request, op string, err error) {
	var verr domain.ValidationErrors
	if errors.As(err, &verr) {
		response.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "request validation failed", domainValidationDetails(verr)...)
		return
	}
	h.logger.Error(op, "err", err, "request_id", mw.RequestIDFromContext(r.Context()))
	response.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
}
