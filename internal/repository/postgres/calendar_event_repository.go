package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/YuriyDubinin/bms-api/internal/domain"
)

type CalendarEventRepository struct {
	pool *pgxpool.Pool
}

func NewCalendarEventRepository(pool *pgxpool.Pool) *CalendarEventRepository {
	return &CalendarEventRepository{pool: pool}
}

const calendarEventColumns = `
	id, customer_id, project_id, title,
	COALESCE(description, ''), COALESCE(location, ''),
	starts_at, ends_at, all_day, status::text,
	COALESCE(recurrence_rule, ''), timezone, created_by,
	attributes::text, created_at, updated_at, deleted_at
`

func scanCalendarEvent(row pgx.Row) (*domain.CalendarEvent, error) {
	var (
		e    domain.CalendarEvent
		attr []byte
	)
	err := row.Scan(
		&e.ID, &e.CustomerID, &e.ProjectID, &e.Title,
		&e.Description, &e.Location,
		&e.StartsAt, &e.EndsAt, &e.AllDay, &e.Status,
		&e.RecurrenceRule, &e.Timezone, &e.CreatedBy,
		&attr, &e.CreatedAt, &e.UpdatedAt, &e.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	e.Attributes = attr
	return &e, nil
}

func (r *CalendarEventRepository) Create(ctx context.Context, e *domain.CalendarEvent) error {
	const query = `
		INSERT INTO calendar_events
			(id, customer_id, project_id, title, description, location, starts_at, ends_at, all_day,
			 status, recurrence_rule, timezone, created_by, attributes, created_at, updated_at)
		VALUES
			($1, $2, $3, $4, NULLIF($5, ''), NULLIF($6, ''), $7, $8, $9,
			 $10::calendar_event_status, NULLIF($11, ''), $12, $13, $14::jsonb, $15, $16)
	`
	_, err := r.pool.Exec(ctx, query,
		e.ID, e.CustomerID, e.ProjectID, e.Title, e.Description, e.Location, e.StartsAt, e.EndsAt, e.AllDay,
		e.Status, e.RecurrenceRule, e.Timezone, e.CreatedBy, jsonbArg(e.Attributes), e.CreatedAt, e.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
			return fmt.Errorf("postgres: create calendar event: %w", domain.ErrAlreadyExists)
		}
		return fmt.Errorf("postgres: create calendar event: %w", err)
	}
	return nil
}

var calendarEventSortColumns = map[string]string{
	"starts_at":  "starts_at",
	"created_at": "created_at",
	"updated_at": "updated_at",
	"title":      "title",
	"status":     "status",
}

func calendarEventSortColumn(s string) string {
	if col, ok := calendarEventSortColumns[s]; ok {
		return col
	}
	return "starts_at"
}

func (r *CalendarEventRepository) List(ctx context.Context, customerID uuid.UUID, f domain.CalendarEventListFilter) ([]*domain.CalendarEvent, int, error) {
	conds := []string{"customer_id = $1", "deleted_at IS NULL"}
	args := []any{customerID}

	if f.ProjectID != nil {
		args = append(args, *f.ProjectID)
		conds = append(conds, fmt.Sprintf("project_id = $%d", len(args)))
	}
	if f.Status != "" {
		args = append(args, f.Status)
		conds = append(conds, fmt.Sprintf("status = $%d::calendar_event_status", len(args)))
	}
	if f.Search != "" {
		args = append(args, "%"+f.Search+"%")
		conds = append(conds, fmt.Sprintf("title ILIKE $%d", len(args)))
	}
	// Пересечение с окном [From, To]: NULL-границы = безграничны.
	if f.From != nil || f.To != nil {
		args = append(args, f.From)
		fromPos := len(args)
		args = append(args, f.To)
		toPos := len(args)
		conds = append(conds, fmt.Sprintf("tstzrange(starts_at, ends_at) && tstzrange($%d, $%d)", fromPos, toPos))
	}
	where := "WHERE " + strings.Join(conds, " AND ")

	var total int
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM calendar_events "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("postgres: count calendar events: %w", err)
	}
	if total == 0 {
		return []*domain.CalendarEvent{}, 0, nil
	}

	args = append(args, f.Limit)
	limitPos := len(args)
	args = append(args, f.Offset)
	offsetPos := len(args)

	query := fmt.Sprintf(
		"SELECT "+calendarEventColumns+" FROM calendar_events %s ORDER BY %s %s, id LIMIT $%d OFFSET $%d",
		where, calendarEventSortColumn(f.SortBy), sortDir(f.SortDesc), limitPos, offsetPos,
	)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("postgres: list calendar events: %w", err)
	}
	defer rows.Close()

	var items []*domain.CalendarEvent
	for rows.Next() {
		e, err := scanCalendarEvent(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("postgres: scan calendar event: %w", err)
		}
		items = append(items, e)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("postgres: iterate calendar events: %w", err)
	}
	return items, total, nil
}

func (r *CalendarEventRepository) GetByID(ctx context.Context, customerID, id uuid.UUID) (*domain.CalendarEvent, error) {
	query := "SELECT " + calendarEventColumns + " FROM calendar_events WHERE id = $1 AND customer_id = $2 AND deleted_at IS NULL"
	e, err := scanCalendarEvent(r.pool.QueryRow(ctx, query, id, customerID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("postgres: get calendar event %s: %w", id, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("postgres: get calendar event %s: %w", id, err)
	}
	return e, nil
}

func (r *CalendarEventRepository) Update(ctx context.Context, customerID uuid.UUID, e *domain.CalendarEvent) error {
	const query = `
		UPDATE calendar_events SET
			project_id      = $1,
			title           = $2,
			description     = NULLIF($3, ''),
			location        = NULLIF($4, ''),
			starts_at       = $5,
			ends_at         = $6,
			all_day         = $7,
			status          = $8::calendar_event_status,
			recurrence_rule = NULLIF($9, ''),
			timezone        = $10,
			attributes      = $11::jsonb
		WHERE id = $12 AND customer_id = $13 AND deleted_at IS NULL
		RETURNING created_at, updated_at
	`
	err := r.pool.QueryRow(ctx, query,
		e.ProjectID, e.Title, e.Description, e.Location, e.StartsAt, e.EndsAt, e.AllDay,
		e.Status, e.RecurrenceRule, e.Timezone, jsonbArg(e.Attributes), e.ID, customerID,
	).Scan(&e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("postgres: update calendar event %s: %w", e.ID, domain.ErrNotFound)
		}
		return fmt.Errorf("postgres: update calendar event: %w", err)
	}
	return nil
}

func (r *CalendarEventRepository) SoftDelete(ctx context.Context, customerID, id uuid.UUID) (time.Time, error) {
	const query = `
		UPDATE calendar_events SET deleted_at = NOW()
		WHERE id = $1 AND customer_id = $2 AND deleted_at IS NULL
		RETURNING deleted_at
	`
	var deletedAt time.Time
	if err := r.pool.QueryRow(ctx, query, id, customerID).Scan(&deletedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return time.Time{}, fmt.Errorf("postgres: soft delete calendar event %s: %w", id, domain.ErrNotFound)
		}
		return time.Time{}, fmt.Errorf("postgres: soft delete calendar event %s: %w", id, err)
	}
	return deletedAt, nil
}

// ───────────────────────── Участники события ─────────────────────────

// AddAttendee вставляет участника. Вызывающий (сервис) уже проверил, что и
// событие, и attendee-entity принадлежат тенанту. Дубликат → ErrAlreadyExists.
func (r *CalendarEventRepository) AddAttendee(ctx context.Context, customerID uuid.UUID, a *domain.CalendarEventAttendee) error {
	const query = `
		INSERT INTO calendar_event_attendees (id, event_id, attendee_entity_id, response_status)
		VALUES ($1, $2, $3, $4::attendee_response)
		RETURNING created_at
	`
	err := r.pool.QueryRow(ctx, query, a.ID, a.EventID, a.AttendeeEntityID, a.ResponseStatus).Scan(&a.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
			return fmt.Errorf("postgres: add attendee: %w", domain.ErrAlreadyExists)
		}
		return fmt.Errorf("postgres: add attendee: %w", err)
	}
	return nil
}

// RemoveAttendee удаляет участника, скоупясь по тенанту через JOIN события.
func (r *CalendarEventRepository) RemoveAttendee(ctx context.Context, customerID, eventID, attendeeEntityID uuid.UUID) error {
	const query = `
		DELETE FROM calendar_event_attendees a
		USING calendar_events e
		WHERE a.event_id = e.id
		  AND a.event_id = $1
		  AND a.attendee_entity_id = $2
		  AND e.customer_id = $3
	`
	tag, err := r.pool.Exec(ctx, query, eventID, attendeeEntityID, customerID)
	if err != nil {
		return fmt.Errorf("postgres: remove attendee: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("postgres: remove attendee: %w", domain.ErrNotFound)
	}
	return nil
}

func (r *CalendarEventRepository) ListAttendees(ctx context.Context, customerID, eventID uuid.UUID) ([]*domain.CalendarEventAttendee, error) {
	const query = `
		SELECT a.id, a.event_id, a.attendee_entity_id, a.response_status::text, a.created_at
		FROM calendar_event_attendees a
		JOIN calendar_events e ON e.id = a.event_id
		WHERE a.event_id = $1 AND e.customer_id = $2
		ORDER BY a.created_at
	`
	rows, err := r.pool.Query(ctx, query, eventID, customerID)
	if err != nil {
		return nil, fmt.Errorf("postgres: list attendees: %w", err)
	}
	defer rows.Close()

	var items []*domain.CalendarEventAttendee
	for rows.Next() {
		var a domain.CalendarEventAttendee
		if err := rows.Scan(&a.ID, &a.EventID, &a.AttendeeEntityID, &a.ResponseStatus, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("postgres: scan attendee: %w", err)
		}
		items = append(items, &a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: iterate attendees: %w", err)
	}
	return items, nil
}
