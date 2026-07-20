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

type ProcessRepository struct {
	pool *pgxpool.Pool
}

func NewProcessRepository(pool *pgxpool.Pool) *ProcessRepository {
	return &ProcessRepository{pool: pool}
}

// ───────────────────────── Процессы ─────────────────────────

const processColumns = `
	id, customer_id, name, COALESCE(description, ''), status::text,
	closed_at, attributes::text, created_at, updated_at, deleted_at
`

func scanProcess(row pgx.Row) (*domain.Process, error) {
	var (
		p    domain.Process
		attr []byte
	)
	err := row.Scan(
		&p.ID, &p.CustomerID, &p.Name, &p.Description, &p.Status,
		&p.ClosedAt, &attr, &p.CreatedAt, &p.UpdatedAt, &p.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	p.Attributes = attr
	return &p, nil
}

// Create вставляет процесс. id задаёт сервис (uuid.New) — BEFORE INSERT триггер
// регистрирует его в entities как PROCESS.
func (r *ProcessRepository) Create(ctx context.Context, p *domain.Process) error {
	const query = `
		INSERT INTO processes
			(id, customer_id, name, description, status, closed_at, attributes, created_at, updated_at)
		VALUES
			($1, $2, $3, NULLIF($4, ''), $5::process_status, $6, $7::jsonb, $8, $9)
	`
	_, err := r.pool.Exec(ctx, query,
		p.ID, p.CustomerID, p.Name, p.Description, p.Status, p.ClosedAt, jsonbArg(p.Attributes), p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
			return fmt.Errorf("postgres: create process: %w", domain.ErrAlreadyExists)
		}
		return fmt.Errorf("postgres: create process: %w", err)
	}
	return nil
}

var processSortColumns = map[string]string{
	"created_at": "created_at",
	"updated_at": "updated_at",
	"name":       "name",
	"status":     "status",
}

func processSortColumn(s string) string {
	if col, ok := processSortColumns[s]; ok {
		return col
	}
	return "created_at"
}

func (r *ProcessRepository) List(ctx context.Context, customerID uuid.UUID, f domain.ProcessListFilter) ([]*domain.Process, int, error) {
	conds := []string{"customer_id = $1", "deleted_at IS NULL"}
	args := []any{customerID}

	if f.Status != "" {
		args = append(args, f.Status)
		conds = append(conds, fmt.Sprintf("status = $%d::process_status", len(args)))
	}
	if f.Search != "" {
		args = append(args, "%"+f.Search+"%")
		conds = append(conds, fmt.Sprintf("name ILIKE $%d", len(args)))
	}
	where := "WHERE " + strings.Join(conds, " AND ")

	var total int
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM processes "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("postgres: count processes: %w", err)
	}
	if total == 0 {
		return []*domain.Process{}, 0, nil
	}

	args = append(args, f.Limit)
	limitPos := len(args)
	args = append(args, f.Offset)
	offsetPos := len(args)

	query := fmt.Sprintf(
		"SELECT "+processColumns+" FROM processes %s ORDER BY %s %s, id LIMIT $%d OFFSET $%d",
		where, processSortColumn(f.SortBy), sortDir(f.SortDesc), limitPos, offsetPos,
	)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("postgres: list processes: %w", err)
	}
	defer rows.Close()

	var items []*domain.Process
	for rows.Next() {
		p, err := scanProcess(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("postgres: scan process: %w", err)
		}
		items = append(items, p)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("postgres: iterate processes: %w", err)
	}
	return items, total, nil
}

func (r *ProcessRepository) GetByID(ctx context.Context, customerID, id uuid.UUID) (*domain.Process, error) {
	query := "SELECT " + processColumns + " FROM processes WHERE id = $1 AND customer_id = $2 AND deleted_at IS NULL"
	p, err := scanProcess(r.pool.QueryRow(ctx, query, id, customerID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("postgres: get process %s: %w", id, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("postgres: get process %s: %w", id, err)
	}
	return p, nil
}

func (r *ProcessRepository) Update(ctx context.Context, customerID uuid.UUID, p *domain.Process) error {
	const query = `
		UPDATE processes SET
			name        = $1,
			description = NULLIF($2, ''),
			status      = $3::process_status,
			closed_at   = $4,
			attributes  = $5::jsonb
		WHERE id = $6 AND customer_id = $7 AND deleted_at IS NULL
		RETURNING created_at, updated_at
	`
	err := r.pool.QueryRow(ctx, query,
		p.Name, p.Description, p.Status, p.ClosedAt, jsonbArg(p.Attributes), p.ID, customerID,
	).Scan(&p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("postgres: update process %s: %w", p.ID, domain.ErrNotFound)
		}
		return fmt.Errorf("postgres: update process: %w", err)
	}
	return nil
}

func (r *ProcessRepository) SoftDelete(ctx context.Context, customerID, id uuid.UUID) (time.Time, error) {
	const query = `
		UPDATE processes SET deleted_at = NOW()
		WHERE id = $1 AND customer_id = $2 AND deleted_at IS NULL
		RETURNING deleted_at
	`
	var deletedAt time.Time
	if err := r.pool.QueryRow(ctx, query, id, customerID).Scan(&deletedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return time.Time{}, fmt.Errorf("postgres: soft delete process %s: %w", id, domain.ErrNotFound)
		}
		return time.Time{}, fmt.Errorf("postgres: soft delete process %s: %w", id, err)
	}
	return deletedAt, nil
}

// ───────────────────────── Этапы процесса ─────────────────────────

const stageColumns = `
	id, customer_id, process_id, name, COALESCE(description, ''),
	position, attributes::text, created_at, updated_at, deleted_at
`

func scanStage(row pgx.Row) (*domain.ProcessStage, error) {
	var (
		s    domain.ProcessStage
		attr []byte
	)
	err := row.Scan(
		&s.ID, &s.CustomerID, &s.ProcessID, &s.Name, &s.Description,
		&s.Position, &attr, &s.CreatedAt, &s.UpdatedAt, &s.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	s.Attributes = attr
	return &s, nil
}

func (r *ProcessRepository) CreateStage(ctx context.Context, s *domain.ProcessStage) error {
	const query = `
		INSERT INTO process_stages
			(id, customer_id, process_id, name, description, position, attributes, created_at, updated_at)
		VALUES
			($1, $2, $3, $4, NULLIF($5, ''), $6, $7::jsonb, $8, $9)
	`
	_, err := r.pool.Exec(ctx, query,
		s.ID, s.CustomerID, s.ProcessID, s.Name, s.Description, s.Position, jsonbArg(s.Attributes), s.CreatedAt, s.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("postgres: create process stage: %w", err)
	}
	return nil
}

func (r *ProcessRepository) ListStages(ctx context.Context, customerID, processID uuid.UUID) ([]*domain.ProcessStage, error) {
	query := "SELECT " + stageColumns + " FROM process_stages WHERE customer_id = $1 AND process_id = $2 AND deleted_at IS NULL ORDER BY position, created_at, id"
	rows, err := r.pool.Query(ctx, query, customerID, processID)
	if err != nil {
		return nil, fmt.Errorf("postgres: list process stages: %w", err)
	}
	defer rows.Close()

	var items []*domain.ProcessStage
	for rows.Next() {
		s, err := scanStage(rows)
		if err != nil {
			return nil, fmt.Errorf("postgres: scan process stage: %w", err)
		}
		items = append(items, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: iterate process stages: %w", err)
	}
	return items, nil
}

func (r *ProcessRepository) GetStageByID(ctx context.Context, customerID, id uuid.UUID) (*domain.ProcessStage, error) {
	query := "SELECT " + stageColumns + " FROM process_stages WHERE id = $1 AND customer_id = $2 AND deleted_at IS NULL"
	s, err := scanStage(r.pool.QueryRow(ctx, query, id, customerID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("postgres: get process stage %s: %w", id, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("postgres: get process stage %s: %w", id, err)
	}
	return s, nil
}

func (r *ProcessRepository) UpdateStage(ctx context.Context, customerID uuid.UUID, s *domain.ProcessStage) error {
	const query = `
		UPDATE process_stages SET
			name        = $1,
			description = NULLIF($2, ''),
			position    = $3,
			attributes  = $4::jsonb
		WHERE id = $5 AND customer_id = $6 AND deleted_at IS NULL
		RETURNING process_id, created_at, updated_at
	`
	err := r.pool.QueryRow(ctx, query,
		s.Name, s.Description, s.Position, jsonbArg(s.Attributes), s.ID, customerID,
	).Scan(&s.ProcessID, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("postgres: update process stage %s: %w", s.ID, domain.ErrNotFound)
		}
		return fmt.Errorf("postgres: update process stage: %w", err)
	}
	return nil
}

func (r *ProcessRepository) SoftDeleteStage(ctx context.Context, customerID, id uuid.UUID) (time.Time, error) {
	const query = `
		UPDATE process_stages SET deleted_at = NOW()
		WHERE id = $1 AND customer_id = $2 AND deleted_at IS NULL
		RETURNING deleted_at
	`
	var deletedAt time.Time
	if err := r.pool.QueryRow(ctx, query, id, customerID).Scan(&deletedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return time.Time{}, fmt.Errorf("postgres: soft delete process stage %s: %w", id, domain.ErrNotFound)
		}
		return time.Time{}, fmt.Errorf("postgres: soft delete process stage %s: %w", id, err)
	}
	return deletedAt, nil
}
