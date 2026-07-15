package dto

import (
	"time"

	"github.com/YuriyDubinin/bms-api/internal/service"
)

const dateLayout = "2006-01-02"

// PaginationResponse — блок пагинации в ответах списков.
type PaginationResponse struct {
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

func fromPagination(p service.Pagination) PaginationResponse {
	return PaginationResponse{
		Page:       p.Page,
		PageSize:   p.PageSize,
		Total:      p.Total,
		TotalPages: p.TotalPages,
	}
}

// parseDatePtr парсит дату "2006-01-02" в *time.Time. Формат гарантирован
// validate-тегом datetime, поэтому ошибку игнорируем (nil при пустом).
func parseDatePtr(s *string) *time.Time {
	if s == nil || *s == "" {
		return nil
	}
	t, err := time.Parse(dateLayout, *s)
	if err != nil {
		return nil
	}
	return &t
}

// formatDatePtr форматирует *time.Time обратно в *string "2006-01-02".
func formatDatePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(dateLayout)
	return &s
}
