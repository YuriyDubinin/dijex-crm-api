package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Process — бизнес-процесс тенанта. Агрегирует сделки и задачи, состоит из
// пользовательских этапов (ProcessStage). Терминальные статусы
// (COMPLETED/FAILED) проставляют ClosedAt — по аналогии с закрытием сделки.
type Process struct {
	ID          uuid.UUID
	CustomerID  uuid.UUID
	ProjectID   *uuid.UUID
	Name        string
	Description string
	Status      string // ENUM process_status
	ClosedAt    *time.Time
	Attributes  json.RawMessage
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}

// Значения ENUM process_status.
//   - ACTIVE    — процесс в работе (по умолчанию).
//   - COMPLETED — процесс завершён успешно.
//   - FAILED    — процесс завершён неудачно (провал).
const (
	ProcessStatusActive    = "ACTIVE"
	ProcessStatusCompleted = "COMPLETED"
	ProcessStatusFailed    = "FAILED"
)

func IsValidProcessStatus(s string) bool {
	switch s {
	case ProcessStatusActive, ProcessStatusCompleted, ProcessStatusFailed:
		return true
	default:
		return false
	}
}

// IsClosedProcessStatus — терминальные статусы (процесс завершён, удачно или
// нет). Используется для автоматической простановки ClosedAt.
func IsClosedProcessStatus(s string) bool {
	return s == ProcessStatusCompleted || s == ProcessStatusFailed
}

// ProcessStage — пользовательский этап процесса. Порядок задаётся через
// Position. Дочерняя сущность процесса (в реестр entities не входит).
type ProcessStage struct {
	ID          uuid.UUID
	CustomerID  uuid.UUID
	ProcessID   uuid.UUID
	Name        string
	Description string
	Position    int
	Attributes  json.RawMessage
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}

// ProcessListFilter — параметры выборки списка процессов внутри тенанта.
type ProcessListFilter struct {
	ProjectID *uuid.UUID
	Status    string
	Search    string // ILIKE по name

	Limit    int
	Offset   int
	SortBy   string
	SortDesc bool
}
