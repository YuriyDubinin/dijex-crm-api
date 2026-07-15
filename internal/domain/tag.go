package domain

import (
	"time"

	"github.com/google/uuid"
)

// Tag — сквозной тег тенанта, навешиваемый на любые сущности через entity_tags.
type Tag struct {
	ID         uuid.UUID
	CustomerID uuid.UUID
	Name       string
	Color      string
	CreatedAt  time.Time
}

// TagListFilter — выборка тегов внутри тенанта. Если EntityID задан —
// возвращаются только теги, навешенные на эту сущность.
type TagListFilter struct {
	EntityID *uuid.UUID
	Search   string

	Limit    int
	Offset   int
	SortDesc bool
}
