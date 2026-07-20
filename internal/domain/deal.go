package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Deal — сделка (CRM): продажа/возможность с суммой, этапом воронки и
// привязками к клиенту (ClientID), направлению (ProjectID) и ответственному
// (AssignedTo). CreatedBy — оператор, создавший сделку.
type Deal struct {
	ID              uuid.UUID
	CustomerID      uuid.UUID
	ProjectID       *uuid.UUID
	ClientID        *uuid.UUID
	ServiceID       *uuid.UUID // услуга, которую продаём/покупаем в рамках сделки
	ProcessID       *uuid.UUID // процесс, в рамках которого идёт сделка
	Title           string
	Description     string
	Status          string // ENUM deal_status
	Type            string // ENUM deal_type (INCOME/EXPENSE)
	Amount          *float64
	Currency        string
	Probability     *int
	ExpectedCloseAt *time.Time // DATE — ожидаемая дата закрытия
	ClosedAt        *time.Time // фактическая дата закрытия (WON/LOST), ставит сервис
	AssignedTo      *uuid.UUID
	CreatedBy       *uuid.UUID
	Attributes      json.RawMessage
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       *time.Time
}

// Значения ENUM deal_status — этапы воронки продаж.
const (
	DealStatusNew     = "NEW"
	DealStatusPending = "PENDING"
	DealStatusWon     = "WON"
	DealStatusLost    = "LOST"
)

func IsValidDealStatus(s string) bool {
	switch s {
	case DealStatusNew, DealStatusPending, DealStatusWon, DealStatusLost:
		return true
	default:
		return false
	}
}

// IsClosedDealStatus — терминальные этапы (сделка закрыта). Используется для
// автоматической простановки ClosedAt.
func IsClosedDealStatus(s string) bool {
	return s == DealStatusWon || s == DealStatusLost
}

// Значения ENUM deal_type — направление сделки: доход или расход.
const (
	DealTypeIncome  = "INCOME"
	DealTypeExpense = "EXPENSE"
)

func IsValidDealType(t string) bool {
	switch t {
	case DealTypeIncome, DealTypeExpense:
		return true
	default:
		return false
	}
}

// DealListFilter — параметры выборки списка сделок внутри тенанта.
type DealListFilter struct {
	ProjectID  *uuid.UUID
	ClientID   *uuid.UUID
	ServiceID  *uuid.UUID
	ProcessID  *uuid.UUID
	Status     string
	Type       string
	AssignedTo *uuid.UUID
	Search     string     // ILIKE по title
	CloseFrom  *time.Time // expected_close_at >= CloseFrom
	CloseTo    *time.Time // expected_close_at <= CloseTo

	Limit    int
	Offset   int
	SortBy   string
	SortDesc bool
}
