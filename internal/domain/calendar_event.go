package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// CalendarEvent — событие календаря тенанта. Интервал [StartsAt, EndsAt)
// в БД защищён CHECK (ends_at > starts_at) и проиндексирован GiST по tstzrange.
type CalendarEvent struct {
	ID             uuid.UUID
	CustomerID     uuid.UUID
	ProjectID      *uuid.UUID
	Title          string
	Description    string
	Location       string
	StartsAt       time.Time
	EndsAt         time.Time
	AllDay         bool
	Status         string // ENUM calendar_event_status
	RecurrenceRule string // iCal RRULE
	Timezone       string
	CreatedBy      *uuid.UUID
	Attributes     json.RawMessage
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      *time.Time
}

// Значения ENUM calendar_event_status.
const (
	CalendarEventStatusScheduled = "SCHEDULED"
	CalendarEventStatusCompleted = "COMPLETED"
	CalendarEventStatusCancelled = "CANCELLED"
)

func IsValidCalendarEventStatus(s string) bool {
	switch s {
	case CalendarEventStatusScheduled, CalendarEventStatusCompleted, CalendarEventStatusCancelled:
		return true
	default:
		return false
	}
}

// CalendarEventAttendee — участник события (любая зарегистрированная entity
// того же тенанта: клиент или оператор).
type CalendarEventAttendee struct {
	ID               uuid.UUID
	EventID          uuid.UUID
	AttendeeEntityID uuid.UUID
	ResponseStatus   string // ENUM attendee_response
	CreatedAt        time.Time
}

// Значения ENUM attendee_response.
const (
	AttendeeResponsePending   = "PENDING"
	AttendeeResponseAccepted  = "ACCEPTED"
	AttendeeResponseDeclined  = "DECLINED"
	AttendeeResponseTentative = "TENTATIVE"
)

func IsValidAttendeeResponse(s string) bool {
	switch s {
	case AttendeeResponsePending, AttendeeResponseAccepted, AttendeeResponseDeclined, AttendeeResponseTentative:
		return true
	default:
		return false
	}
}

// CalendarEventListFilter — параметры выборки событий внутри тенанта.
// From/To задают окно пересечения: событие попадает в выборку, если его
// интервал пересекается с [From, To].
type CalendarEventListFilter struct {
	ProjectID *uuid.UUID
	Status    string
	Search    string     // ILIKE по title
	From      *time.Time // нижняя граница окна пересечения
	To        *time.Time // верхняя граница окна пересечения

	Limit    int
	Offset   int
	SortBy   string
	SortDesc bool
}
