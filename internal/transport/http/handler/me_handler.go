package handler

import (
	"net/http"

	"github.com/google/uuid"

	mw "github.com/YuriyDubinin/bms-api/internal/transport/http/middleware"
	"github.com/YuriyDubinin/bms-api/internal/transport/http/response"
)

// MeHandler отдаёт информацию о текущей аутентифицированной сессии из контекста
// (заполнен мидлварью Auth), без похода в БД.
type MeHandler struct{}

func NewMeHandler() *MeHandler {
	return &MeHandler{}
}

type meResponse struct {
	UserID     uuid.UUID `json:"user_id"`
	CustomerID uuid.UUID `json:"customer_id"`
	Role       string    `json:"role"`
}

func (h *MeHandler) Get(w http.ResponseWriter, r *http.Request) {
	principal, ok := mw.PrincipalFromContext(r.Context())
	if !ok {
		// Защитная ветка: мидлварь Auth гарантирует наличие Principal.
		response.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "principal missing in context")
		return
	}

	response.WriteJSON(w, http.StatusOK, meResponse{
		UserID:     principal.UserID,
		CustomerID: principal.CustomerID,
		Role:       principal.Role,
	})
}
