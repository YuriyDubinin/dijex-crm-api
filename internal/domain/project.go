package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Project — бизнес-направление/проект тенанта ("Автопарк", "Салон красоты").
// Центральная сущность, к которой привязываются services/clients/tasks/events.
type Project struct {
	ID          uuid.UUID
	CustomerID  uuid.UUID
	Name        string
	Slug        string
	Direction   string
	Description string
	Status      string // ENUM project_status
	Attributes  json.RawMessage
	StartsAt    *time.Time
	EndsAt      *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}

// Значения ENUM project_status.
const (
	ProjectStatusDraft    = "DRAFT"
	ProjectStatusActive   = "ACTIVE"
	ProjectStatusOnHold   = "ON_HOLD"
	ProjectStatusArchived = "ARCHIVED"
)

func IsValidProjectStatus(s string) bool {
	switch s {
	case ProjectStatusDraft, ProjectStatusActive, ProjectStatusOnHold, ProjectStatusArchived:
		return true
	default:
		return false
	}
}

// ProjectListFilter — параметры выборки списка проектов внутри тенанта.
// customerID передаётся отдельным аргументом метода, не полем фильтра.
type ProjectListFilter struct {
	Status string // "" = любой
	Search string // ILIKE по name/direction

	Limit    int
	Offset   int
	SortBy   string // whitelist на стороне репозитория
	SortDesc bool
}
