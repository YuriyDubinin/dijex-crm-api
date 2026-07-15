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

type ProjectRepository struct {
	pool *pgxpool.Pool
}

func NewProjectRepository(pool *pgxpool.Pool) *ProjectRepository {
	return &ProjectRepository{pool: pool}
}

const projectColumns = `
	id, customer_id, name,
	COALESCE(slug, ''), COALESCE(direction, ''), COALESCE(description, ''),
	status::text, attributes::text, starts_at, ends_at,
	created_at, updated_at, deleted_at
`

func scanProject(row pgx.Row) (*domain.Project, error) {
	var (
		p    domain.Project
		attr []byte
	)
	err := row.Scan(
		&p.ID, &p.CustomerID, &p.Name,
		&p.Slug, &p.Direction, &p.Description,
		&p.Status, &attr, &p.StartsAt, &p.EndsAt,
		&p.CreatedAt, &p.UpdatedAt, &p.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	p.Attributes = attr
	return &p, nil
}

// Create вставляет проект. id задаёт сервис (uuid.New) — BEFORE INSERT триггер
// регистрирует его в entities. Конфликт slug в рамках тенанта → ErrAlreadyExists.
func (r *ProjectRepository) Create(ctx context.Context, p *domain.Project) error {
	const query = `
		INSERT INTO projects
			(id, customer_id, name, slug, direction, description, status, attributes, starts_at, ends_at, created_at, updated_at)
		VALUES
			($1, $2, $3, NULLIF($4, ''), NULLIF($5, ''), NULLIF($6, ''), $7::project_status, $8::jsonb, $9, $10, $11, $12)
	`
	_, err := r.pool.Exec(ctx, query,
		p.ID, p.CustomerID, p.Name, p.Slug, p.Direction, p.Description,
		p.Status, jsonbArg(p.Attributes), p.StartsAt, p.EndsAt, p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
			return fmt.Errorf("postgres: create project: %w", domain.ErrAlreadyExists)
		}
		return fmt.Errorf("postgres: create project: %w", err)
	}
	return nil
}

var projectSortColumns = map[string]string{
	"created_at": "created_at",
	"updated_at": "updated_at",
	"name":       "name",
	"status":     "status",
}

func projectSortColumn(s string) string {
	if col, ok := projectSortColumns[s]; ok {
		return col
	}
	return "created_at"
}

func (r *ProjectRepository) List(ctx context.Context, customerID uuid.UUID, f domain.ProjectListFilter) ([]*domain.Project, int, error) {
	conds := []string{"customer_id = $1", "deleted_at IS NULL"}
	args := []any{customerID}

	if f.Status != "" {
		args = append(args, f.Status)
		conds = append(conds, fmt.Sprintf("status = $%d::project_status", len(args)))
	}
	if f.Search != "" {
		args = append(args, "%"+f.Search+"%")
		conds = append(conds, fmt.Sprintf("(name ILIKE $%d OR direction ILIKE $%d)", len(args), len(args)))
	}
	where := "WHERE " + strings.Join(conds, " AND ")

	var total int
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM projects "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("postgres: count projects: %w", err)
	}
	if total == 0 {
		return []*domain.Project{}, 0, nil
	}

	args = append(args, f.Limit)
	limitPos := len(args)
	args = append(args, f.Offset)
	offsetPos := len(args)

	query := fmt.Sprintf(
		"SELECT "+projectColumns+" FROM projects %s ORDER BY %s %s, id LIMIT $%d OFFSET $%d",
		where, projectSortColumn(f.SortBy), sortDir(f.SortDesc), limitPos, offsetPos,
	)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("postgres: list projects: %w", err)
	}
	defer rows.Close()

	var items []*domain.Project
	for rows.Next() {
		p, err := scanProject(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("postgres: scan project: %w", err)
		}
		items = append(items, p)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("postgres: iterate projects: %w", err)
	}
	return items, total, nil
}

func (r *ProjectRepository) GetByID(ctx context.Context, customerID, id uuid.UUID) (*domain.Project, error) {
	query := "SELECT " + projectColumns + " FROM projects WHERE id = $1 AND customer_id = $2 AND deleted_at IS NULL"
	p, err := scanProject(r.pool.QueryRow(ctx, query, id, customerID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("postgres: get project %s: %w", id, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("postgres: get project %s: %w", id, err)
	}
	return p, nil
}

func (r *ProjectRepository) Update(ctx context.Context, customerID uuid.UUID, p *domain.Project) error {
	const query = `
		UPDATE projects SET
			name        = $1,
			slug        = NULLIF($2, ''),
			direction   = NULLIF($3, ''),
			description = NULLIF($4, ''),
			status      = $5::project_status,
			attributes  = $6::jsonb,
			starts_at   = $7,
			ends_at     = $8
		WHERE id = $9 AND customer_id = $10 AND deleted_at IS NULL
		RETURNING created_at, updated_at
	`
	err := r.pool.QueryRow(ctx, query,
		p.Name, p.Slug, p.Direction, p.Description, p.Status, jsonbArg(p.Attributes),
		p.StartsAt, p.EndsAt, p.ID, customerID,
	).Scan(&p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("postgres: update project %s: %w", p.ID, domain.ErrNotFound)
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
			return fmt.Errorf("postgres: update project: %w", domain.ErrAlreadyExists)
		}
		return fmt.Errorf("postgres: update project: %w", err)
	}
	return nil
}

func (r *ProjectRepository) SoftDelete(ctx context.Context, customerID, id uuid.UUID) (time.Time, error) {
	const query = `
		UPDATE projects SET deleted_at = NOW()
		WHERE id = $1 AND customer_id = $2 AND deleted_at IS NULL
		RETURNING deleted_at
	`
	var deletedAt time.Time
	if err := r.pool.QueryRow(ctx, query, id, customerID).Scan(&deletedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return time.Time{}, fmt.Errorf("postgres: soft delete project %s: %w", id, domain.ErrNotFound)
		}
		return time.Time{}, fmt.Errorf("postgres: soft delete project %s: %w", id, err)
	}
	return deletedAt, nil
}
