package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Client — клиент тенанта (конечный потребитель услуг). Address и Attributes —
// произвольный JSON.
type Client struct {
	ID          uuid.UUID
	CustomerID  uuid.UUID
	ProjectID   *uuid.UUID
	FirstName   string
	LastName    string
	CompanyName string
	Email       string
	Phone       string
	Address     json.RawMessage
	Status      string // ENUM client_status
	SubjectType string // ENUM client_subject_type (INDIVIDUAL/LEGAL_ENTITY)
	Source      string
	Attributes  json.RawMessage
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}

// Значения ENUM client_status.
const (
	ClientStatusLead     = "LEAD"
	ClientStatusActive   = "ACTIVE"
	ClientStatusInactive = "INACTIVE"
	ClientStatusArchived = "ARCHIVED"
)

func IsValidClientStatus(s string) bool {
	switch s {
	case ClientStatusLead, ClientStatusActive, ClientStatusInactive, ClientStatusArchived:
		return true
	default:
		return false
	}
}

// Значения ENUM client_subject_type — вид субъекта (физлицо/юрлицо).
const (
	ClientSubjectTypeIndividual  = "INDIVIDUAL"
	ClientSubjectTypeLegalEntity = "LEGAL_ENTITY"
)

func IsValidClientSubjectType(s string) bool {
	switch s {
	case ClientSubjectTypeIndividual, ClientSubjectTypeLegalEntity:
		return true
	default:
		return false
	}
}

// ClientListFilter — параметры выборки списка клиентов внутри тенанта.
type ClientListFilter struct {
	ProjectID   *uuid.UUID
	Status      string
	SubjectType string
	Search      string // ILIKE по first_name/last_name/company_name/email/phone

	Limit    int
	Offset   int
	SortBy   string
	SortDesc bool
}
