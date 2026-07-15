package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Service — услуга тенанта ("установка документов на автомобиль", "сеанс
// массажа"). Может быть привязана к "домашнему" проекту (ProjectID).
type Service struct {
	ID          uuid.UUID
	CustomerID  uuid.UUID
	ProjectID   *uuid.UUID
	Name        string
	Category    string
	Description string
	Price       *float64 // NUMERIC(12,2); nil = не задана
	Currency    string   // ISO-4217, 3 буквы
	DurationMin *int
	Status      string // ENUM service_status
	Attributes  json.RawMessage
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}

// Значения ENUM service_status.
const (
	ServiceStatusActive   = "ACTIVE"
	ServiceStatusInactive = "INACTIVE"
	ServiceStatusArchived = "ARCHIVED"
)

func IsValidServiceStatus(s string) bool {
	switch s {
	case ServiceStatusActive, ServiceStatusInactive, ServiceStatusArchived:
		return true
	default:
		return false
	}
}

// ServiceListFilter — параметры выборки списка услуг внутри тенанта.
type ServiceListFilter struct {
	ProjectID *uuid.UUID // nil = любой проект
	Status    string     // "" = любой
	Category  string     // "" = любая
	Search    string     // ILIKE по name

	Limit    int
	Offset   int
	SortBy   string
	SortDesc bool
}
