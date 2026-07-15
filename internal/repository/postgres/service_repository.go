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

type ServiceRepository struct {
	pool *pgxpool.Pool
}

func NewServiceRepository(pool *pgxpool.Pool) *ServiceRepository {
	return &ServiceRepository{pool: pool}
}

const serviceColumns = `
	id, customer_id, project_id, name,
	COALESCE(category, ''), COALESCE(description, ''),
	price::float8, currency, duration_min,
	status::text, attributes::text, created_at, updated_at, deleted_at
`

func scanService(row pgx.Row) (*domain.Service, error) {
	var (
		s    domain.Service
		attr []byte
	)
	err := row.Scan(
		&s.ID, &s.CustomerID, &s.ProjectID, &s.Name,
		&s.Category, &s.Description,
		&s.Price, &s.Currency, &s.DurationMin,
		&s.Status, &attr, &s.CreatedAt, &s.UpdatedAt, &s.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	s.Attributes = attr
	return &s, nil
}

func (r *ServiceRepository) Create(ctx context.Context, s *domain.Service) error {
	const query = `
		INSERT INTO services
			(id, customer_id, project_id, name, category, description, price, currency, duration_min, status, attributes, created_at, updated_at)
		VALUES
			($1, $2, $3, $4, NULLIF($5, ''), NULLIF($6, ''), $7::numeric, $8, $9, $10::service_status, $11::jsonb, $12, $13)
	`
	_, err := r.pool.Exec(ctx, query,
		s.ID, s.CustomerID, s.ProjectID, s.Name, s.Category, s.Description,
		s.Price, s.Currency, s.DurationMin, s.Status, jsonbArg(s.Attributes), s.CreatedAt, s.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
			return fmt.Errorf("postgres: create service: %w", domain.ErrAlreadyExists)
		}
		return fmt.Errorf("postgres: create service: %w", err)
	}
	return nil
}

var serviceSortColumns = map[string]string{
	"created_at": "created_at",
	"updated_at": "updated_at",
	"name":       "name",
	"price":      "price",
	"status":     "status",
}

func serviceSortColumn(s string) string {
	if col, ok := serviceSortColumns[s]; ok {
		return col
	}
	return "created_at"
}

func (r *ServiceRepository) List(ctx context.Context, customerID uuid.UUID, f domain.ServiceListFilter) ([]*domain.Service, int, error) {
	conds := []string{"customer_id = $1", "deleted_at IS NULL"}
	args := []any{customerID}

	if f.ProjectID != nil {
		args = append(args, *f.ProjectID)
		conds = append(conds, fmt.Sprintf("project_id = $%d", len(args)))
	}
	if f.Status != "" {
		args = append(args, f.Status)
		conds = append(conds, fmt.Sprintf("status = $%d::service_status", len(args)))
	}
	if f.Category != "" {
		args = append(args, f.Category)
		conds = append(conds, fmt.Sprintf("category = $%d", len(args)))
	}
	if f.Search != "" {
		args = append(args, "%"+f.Search+"%")
		conds = append(conds, fmt.Sprintf("name ILIKE $%d", len(args)))
	}
	where := "WHERE " + strings.Join(conds, " AND ")

	var total int
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM services "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("postgres: count services: %w", err)
	}
	if total == 0 {
		return []*domain.Service{}, 0, nil
	}

	args = append(args, f.Limit)
	limitPos := len(args)
	args = append(args, f.Offset)
	offsetPos := len(args)

	query := fmt.Sprintf(
		"SELECT "+serviceColumns+" FROM services %s ORDER BY %s %s, id LIMIT $%d OFFSET $%d",
		where, serviceSortColumn(f.SortBy), sortDir(f.SortDesc), limitPos, offsetPos,
	)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("postgres: list services: %w", err)
	}
	defer rows.Close()

	var items []*domain.Service
	for rows.Next() {
		s, err := scanService(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("postgres: scan service: %w", err)
		}
		items = append(items, s)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("postgres: iterate services: %w", err)
	}
	return items, total, nil
}

func (r *ServiceRepository) GetByID(ctx context.Context, customerID, id uuid.UUID) (*domain.Service, error) {
	query := "SELECT " + serviceColumns + " FROM services WHERE id = $1 AND customer_id = $2 AND deleted_at IS NULL"
	s, err := scanService(r.pool.QueryRow(ctx, query, id, customerID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("postgres: get service %s: %w", id, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("postgres: get service %s: %w", id, err)
	}
	return s, nil
}

func (r *ServiceRepository) Update(ctx context.Context, customerID uuid.UUID, s *domain.Service) error {
	const query = `
		UPDATE services SET
			project_id   = $1,
			name         = $2,
			category     = NULLIF($3, ''),
			description  = NULLIF($4, ''),
			price        = $5::numeric,
			currency     = $6,
			duration_min = $7,
			status       = $8::service_status,
			attributes   = $9::jsonb
		WHERE id = $10 AND customer_id = $11 AND deleted_at IS NULL
		RETURNING created_at, updated_at
	`
	err := r.pool.QueryRow(ctx, query,
		s.ProjectID, s.Name, s.Category, s.Description, s.Price, s.Currency, s.DurationMin,
		s.Status, jsonbArg(s.Attributes), s.ID, customerID,
	).Scan(&s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("postgres: update service %s: %w", s.ID, domain.ErrNotFound)
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
			return fmt.Errorf("postgres: update service: %w", domain.ErrAlreadyExists)
		}
		return fmt.Errorf("postgres: update service: %w", err)
	}
	return nil
}

func (r *ServiceRepository) SoftDelete(ctx context.Context, customerID, id uuid.UUID) (time.Time, error) {
	const query = `
		UPDATE services SET deleted_at = NOW()
		WHERE id = $1 AND customer_id = $2 AND deleted_at IS NULL
		RETURNING deleted_at
	`
	var deletedAt time.Time
	if err := r.pool.QueryRow(ctx, query, id, customerID).Scan(&deletedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return time.Time{}, fmt.Errorf("postgres: soft delete service %s: %w", id, domain.ErrNotFound)
		}
		return time.Time{}, fmt.Errorf("postgres: soft delete service %s: %w", id, err)
	}
	return deletedAt, nil
}
