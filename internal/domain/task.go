package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Task — произвольная задача тенанта, привязываемая к любым сущностям через
// entity_links. AssignedTo/CreatedBy ссылаются на customer_users.
type Task struct {
	ID          uuid.UUID
	CustomerID  uuid.UUID
	ProjectID   *uuid.UUID
	ClientID    *uuid.UUID
	DealID      *uuid.UUID
	Title       string
	Description string
	Status      string // ENUM task_status
	Priority    string // ENUM task_priority
	AssignedTo  *uuid.UUID
	CreatedBy   *uuid.UUID
	DueAt       *time.Time
	CompletedAt *time.Time
	Attributes  json.RawMessage
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}

// Значения ENUM task_status.
const (
	TaskStatusTodo       = "TODO"
	TaskStatusInProgress = "IN_PROGRESS"
	TaskStatusBlocked    = "BLOCKED"
	TaskStatusDone       = "DONE"
	TaskStatusCancelled  = "CANCELLED"
)

func IsValidTaskStatus(s string) bool {
	switch s {
	case TaskStatusTodo, TaskStatusInProgress, TaskStatusBlocked, TaskStatusDone, TaskStatusCancelled:
		return true
	default:
		return false
	}
}

// Значения ENUM task_priority.
const (
	TaskPriorityLow    = "LOW"
	TaskPriorityMedium = "MEDIUM"
	TaskPriorityHigh   = "HIGH"
	TaskPriorityUrgent = "URGENT"
)

func IsValidTaskPriority(p string) bool {
	switch p {
	case TaskPriorityLow, TaskPriorityMedium, TaskPriorityHigh, TaskPriorityUrgent:
		return true
	default:
		return false
	}
}

// TaskListFilter — параметры выборки списка задач внутри тенанта.
type TaskListFilter struct {
	ProjectID  *uuid.UUID
	ClientID   *uuid.UUID
	DealID     *uuid.UUID
	Status     string
	Priority   string
	AssignedTo *uuid.UUID
	Search     string     // ILIKE по title
	DueBefore  *time.Time // due_at <= DueBefore
	DueAfter   *time.Time // due_at >= DueAfter

	Limit    int
	Offset   int
	SortBy   string
	SortDesc bool
}
