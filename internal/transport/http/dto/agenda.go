package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/YuriyDubinin/bms-api/internal/service"
)

// AgendaItemResponse — нормализованный элемент календаря. start/end в RFC3339
// (для all_day-элементов время игнорируется), null — если границы нет.
type AgendaItemResponse struct {
	Type   string     `json:"type"` // event | task | deal | project
	ID     uuid.UUID  `json:"id"`
	Title  string     `json:"title"`
	Start  *time.Time `json:"start"`
	End    *time.Time `json:"end"`
	AllDay bool       `json:"all_day"`
	Status string     `json:"status"`
}

type AgendaResponse struct {
	Items []AgendaItemResponse `json:"items"`
}

func FromAgenda(items []service.AgendaItem) AgendaResponse {
	out := make([]AgendaItemResponse, 0, len(items))
	for _, it := range items {
		out = append(out, AgendaItemResponse{
			Type:   it.Type,
			ID:     it.ID,
			Title:  it.Title,
			Start:  it.Start,
			End:    it.End,
			AllDay: it.AllDay,
			Status: it.Status,
		})
	}
	return AgendaResponse{Items: out}
}
