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

type TaskRepository struct {
	pool *pgxpool.Pool
}

func NewTaskRepository(pool *pgxpool.Pool) *TaskRepository {
	return &TaskRepository{pool: pool}
}

const taskColumns = `
	id, customer_id, project_id, client_id, deal_id, title,
	COALESCE(description, ''), status::text, priority::text,
	assigned_to, created_by, due_at, completed_at,
	attributes::text, created_at, updated_at, deleted_at
`

func scanTask(row pgx.Row) (*domain.Task, error) {
	var (
		t    domain.Task
		attr []byte
	)
	err := row.Scan(
		&t.ID, &t.CustomerID, &t.ProjectID, &t.ClientID, &t.DealID, &t.Title,
		&t.Description, &t.Status, &t.Priority,
		&t.AssignedTo, &t.CreatedBy, &t.DueAt, &t.CompletedAt,
		&attr, &t.CreatedAt, &t.UpdatedAt, &t.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	t.Attributes = attr
	return &t, nil
}

func (r *TaskRepository) Create(ctx context.Context, t *domain.Task) error {
	const query = `
		INSERT INTO tasks
			(id, customer_id, project_id, client_id, deal_id, title, description, status, priority, assigned_to, created_by, due_at, completed_at, attributes, created_at, updated_at)
		VALUES
			($1, $2, $3, $4, $5, $6, NULLIF($7, ''), $8::task_status, $9::task_priority, $10, $11, $12, $13, $14::jsonb, $15, $16)
	`
	_, err := r.pool.Exec(ctx, query,
		t.ID, t.CustomerID, t.ProjectID, t.ClientID, t.DealID, t.Title, t.Description, t.Status, t.Priority,
		t.AssignedTo, t.CreatedBy, t.DueAt, t.CompletedAt, jsonbArg(t.Attributes), t.CreatedAt, t.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
			return fmt.Errorf("postgres: create task: %w", domain.ErrAlreadyExists)
		}
		return fmt.Errorf("postgres: create task: %w", err)
	}
	return nil
}

var taskSortColumns = map[string]string{
	"created_at": "created_at",
	"updated_at": "updated_at",
	"due_at":     "due_at",
	"priority":   "priority",
	"status":     "status",
	"title":      "title",
}

func taskSortColumn(s string) string {
	if col, ok := taskSortColumns[s]; ok {
		return col
	}
	return "created_at"
}

func (r *TaskRepository) List(ctx context.Context, customerID uuid.UUID, f domain.TaskListFilter) ([]*domain.Task, int, error) {
	conds := []string{"customer_id = $1", "deleted_at IS NULL"}
	args := []any{customerID}

	if f.ProjectID != nil {
		args = append(args, *f.ProjectID)
		conds = append(conds, fmt.Sprintf("project_id = $%d", len(args)))
	}
	if f.ClientID != nil {
		args = append(args, *f.ClientID)
		conds = append(conds, fmt.Sprintf("client_id = $%d", len(args)))
	}
	if f.DealID != nil {
		args = append(args, *f.DealID)
		conds = append(conds, fmt.Sprintf("deal_id = $%d", len(args)))
	}
	if f.Status != "" {
		args = append(args, f.Status)
		conds = append(conds, fmt.Sprintf("status = $%d::task_status", len(args)))
	}
	if f.Priority != "" {
		args = append(args, f.Priority)
		conds = append(conds, fmt.Sprintf("priority = $%d::task_priority", len(args)))
	}
	if f.AssignedTo != nil {
		args = append(args, *f.AssignedTo)
		conds = append(conds, fmt.Sprintf("assigned_to = $%d", len(args)))
	}
	if f.Search != "" {
		args = append(args, "%"+f.Search+"%")
		conds = append(conds, fmt.Sprintf("title ILIKE $%d", len(args)))
	}
	if f.DueBefore != nil {
		args = append(args, *f.DueBefore)
		conds = append(conds, fmt.Sprintf("due_at <= $%d", len(args)))
	}
	if f.DueAfter != nil {
		args = append(args, *f.DueAfter)
		conds = append(conds, fmt.Sprintf("due_at >= $%d", len(args)))
	}
	where := "WHERE " + strings.Join(conds, " AND ")

	var total int
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM tasks "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("postgres: count tasks: %w", err)
	}
	if total == 0 {
		return []*domain.Task{}, 0, nil
	}

	args = append(args, f.Limit)
	limitPos := len(args)
	args = append(args, f.Offset)
	offsetPos := len(args)

	query := fmt.Sprintf(
		"SELECT "+taskColumns+" FROM tasks %s ORDER BY %s %s, id LIMIT $%d OFFSET $%d",
		where, taskSortColumn(f.SortBy), sortDir(f.SortDesc), limitPos, offsetPos,
	)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("postgres: list tasks: %w", err)
	}
	defer rows.Close()

	var items []*domain.Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("postgres: scan task: %w", err)
		}
		items = append(items, t)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("postgres: iterate tasks: %w", err)
	}
	return items, total, nil
}

func (r *TaskRepository) GetByID(ctx context.Context, customerID, id uuid.UUID) (*domain.Task, error) {
	query := "SELECT " + taskColumns + " FROM tasks WHERE id = $1 AND customer_id = $2 AND deleted_at IS NULL"
	t, err := scanTask(r.pool.QueryRow(ctx, query, id, customerID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("postgres: get task %s: %w", id, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("postgres: get task %s: %w", id, err)
	}
	return t, nil
}

func (r *TaskRepository) Update(ctx context.Context, customerID uuid.UUID, t *domain.Task) error {
	const query = `
		UPDATE tasks SET
			project_id   = $1,
			client_id    = $2,
			deal_id      = $3,
			title        = $4,
			description  = NULLIF($5, ''),
			status       = $6::task_status,
			priority     = $7::task_priority,
			assigned_to  = $8,
			due_at       = $9,
			completed_at = $10,
			attributes   = $11::jsonb
		WHERE id = $12 AND customer_id = $13 AND deleted_at IS NULL
		RETURNING created_at, updated_at
	`
	err := r.pool.QueryRow(ctx, query,
		t.ProjectID, t.ClientID, t.DealID, t.Title, t.Description, t.Status, t.Priority,
		t.AssignedTo, t.DueAt, t.CompletedAt, jsonbArg(t.Attributes), t.ID, customerID,
	).Scan(&t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("postgres: update task %s: %w", t.ID, domain.ErrNotFound)
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
			return fmt.Errorf("postgres: update task: %w", domain.ErrAlreadyExists)
		}
		return fmt.Errorf("postgres: update task: %w", err)
	}
	return nil
}

func (r *TaskRepository) SoftDelete(ctx context.Context, customerID, id uuid.UUID) (time.Time, error) {
	const query = `
		UPDATE tasks SET deleted_at = NOW()
		WHERE id = $1 AND customer_id = $2 AND deleted_at IS NULL
		RETURNING deleted_at
	`
	var deletedAt time.Time
	if err := r.pool.QueryRow(ctx, query, id, customerID).Scan(&deletedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return time.Time{}, fmt.Errorf("postgres: soft delete task %s: %w", id, domain.ErrNotFound)
		}
		return time.Time{}, fmt.Errorf("postgres: soft delete task %s: %w", id, err)
	}
	return deletedAt, nil
}
