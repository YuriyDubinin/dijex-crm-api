package handler

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/YuriyDubinin/bms-api/internal/domain"
	mw "github.com/YuriyDubinin/bms-api/internal/transport/http/middleware"
	"github.com/YuriyDubinin/bms-api/internal/transport/http/response"
	"github.com/YuriyDubinin/bms-api/pkg/validator"
)

// maxRequestBody — лимит тела входящего запроса (защита от чрезмерных payload).
const maxRequestBody = 1 << 20 // 1 MiB

// principalOr500 достаёт Principal из контекста (кладёт мидлварь Auth). Если
// его нет — это ошибка конфигурации роутера: отвечаем 500 и false.
func principalOr500(w http.ResponseWriter, r *http.Request, logger *slog.Logger) (*domain.Principal, bool) {
	p, ok := mw.PrincipalFromContext(r.Context())
	if !ok {
		logger.Error("principal missing in context", "request_id", mw.RequestIDFromContext(r.Context()))
		response.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "principal missing in context")
		return nil, false
	}
	return p, true
}

// toResponseFieldErrors переводит ошибки структурной валидации (validator) в
// формат конверта ответа.
func toResponseFieldErrors(in []validator.FieldError) []response.FieldError {
	out := make([]response.FieldError, 0, len(in))
	for _, fe := range in {
		out = append(out, response.FieldError{Field: fe.Field, Message: fe.Message})
	}
	return out
}

// domainValidationDetails переводит доменные ошибки бизнес-валидации в формат
// конверта ответа.
func domainValidationDetails(verrs domain.ValidationErrors) []response.FieldError {
	out := make([]response.FieldError, 0, len(verrs))
	for _, ve := range verrs {
		out = append(out, response.FieldError{Field: ve.Field, Message: ve.Message})
	}
	return out
}

// atoiDefault парсит целое; при пустом/некорректном — возвращает def.
func atoiDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

// parseBoolPtr: "true"/"false" → *bool; всё прочее → nil («фильтр не задан»).
func parseBoolPtr(s string) *bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "true":
		v := true
		return &v
	case "false":
		v := false
		return &v
	default:
		return nil
	}
}

// parseUUIDPtr: валидный UUID → *uuid.UUID; пусто/мусор → nil.
func parseUUIDPtr(s string) *uuid.UUID {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return nil
	}
	return &id
}

// parseTimePtr: RFC3339 → *time.Time; пусто/мусор → nil.
func parseTimePtr(s string) *time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil
	}
	return &t
}
