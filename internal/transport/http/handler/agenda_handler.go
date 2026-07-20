package handler

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/YuriyDubinin/bms-api/internal/service"
	"github.com/YuriyDubinin/bms-api/internal/transport/http/dto"
	mw "github.com/YuriyDubinin/bms-api/internal/transport/http/middleware"
	"github.com/YuriyDubinin/bms-api/internal/transport/http/response"
)

type AgendaService interface {
	Agenda(ctx context.Context, customerID uuid.UUID, in service.AgendaInput) ([]service.AgendaItem, error)
}

type AgendaHandler struct {
	service AgendaService
	logger  *slog.Logger
}

func NewAgendaHandler(svc AgendaService, logger *slog.Logger) *AgendaHandler {
	return &AgendaHandler{service: svc, logger: logger}
}

var knownAgendaTypes = map[string]bool{"task": true, "deal": true, "project": true, "process": true}

// Agenda обрабатывает GET /api/calendar/agenda?from=&to=&types=.
func (h *AgendaHandler) Agenda(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalOr500(w, r, h.logger)
	if !ok {
		return
	}
	q := r.URL.Query()
	in := service.AgendaInput{
		From:  parseStatsBound(q.Get("from"), false),
		To:    parseStatsBound(q.Get("to"), true),
		Types: parseAgendaTypes(q.Get("types")),
	}
	items, err := h.service.Agenda(r.Context(), principal.CustomerID, in)
	if err != nil {
		h.logger.Error("calendar agenda", "err", err, "request_id", mw.RequestIDFromContext(r.Context()))
		response.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}
	response.WriteJSON(w, http.StatusOK, dto.FromAgenda(items))
}

// parseAgendaTypes разбирает CSV вида "event,task"; неизвестные значения
// игнорируются, пустая строка → nil (= все типы).
func parseAgendaTypes(s string) map[string]bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	out := map[string]bool{}
	for _, part := range strings.Split(s, ",") {
		t := strings.ToLower(strings.TrimSpace(part))
		if knownAgendaTypes[t] {
			out[t] = true
		}
	}
	return out
}
