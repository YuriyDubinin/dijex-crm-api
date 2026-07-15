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

type DealRepository struct {
	pool *pgxpool.Pool
}

func NewDealRepository(pool *pgxpool.Pool) *DealRepository {
	return &DealRepository{pool: pool}
}

const dealColumns = `
	id, customer_id, project_id, client_id, service_id, title,
	COALESCE(description, ''), status::text, type::text, amount::float8, currency, probability,
	expected_close_at, closed_at, assigned_to, created_by,
	attributes::text, created_at, updated_at, deleted_at
`

func scanDeal(row pgx.Row) (*domain.Deal, error) {
	var (
		d    domain.Deal
		attr []byte
	)
	err := row.Scan(
		&d.ID, &d.CustomerID, &d.ProjectID, &d.ClientID, &d.ServiceID, &d.Title,
		&d.Description, &d.Status, &d.Type, &d.Amount, &d.Currency, &d.Probability,
		&d.ExpectedCloseAt, &d.ClosedAt, &d.AssignedTo, &d.CreatedBy,
		&attr, &d.CreatedAt, &d.UpdatedAt, &d.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	d.Attributes = attr
	return &d, nil
}

// Create вставляет сделку. id задаёт сервис (uuid.New) — BEFORE INSERT триггер
// регистрирует его в entities как DEAL. created_by ставится только при создании.
func (r *DealRepository) Create(ctx context.Context, d *domain.Deal) error {
	const query = `
		INSERT INTO deals
			(id, customer_id, project_id, client_id, service_id, title, description, status, type, amount, currency,
			 probability, expected_close_at, closed_at, assigned_to, created_by, attributes, created_at, updated_at)
		VALUES
			($1, $2, $3, $4, $5, $6, NULLIF($7, ''), $8::deal_status, $9::deal_type, $10::numeric, $11,
			 $12, $13, $14, $15, $16, $17::jsonb, $18, $19)
	`
	_, err := r.pool.Exec(ctx, query,
		d.ID, d.CustomerID, d.ProjectID, d.ClientID, d.ServiceID, d.Title, d.Description, d.Status, d.Type, d.Amount, d.Currency,
		d.Probability, d.ExpectedCloseAt, d.ClosedAt, d.AssignedTo, d.CreatedBy, jsonbArg(d.Attributes), d.CreatedAt, d.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
			return fmt.Errorf("postgres: create deal: %w", domain.ErrAlreadyExists)
		}
		return fmt.Errorf("postgres: create deal: %w", err)
	}
	return nil
}

var dealSortColumns = map[string]string{
	"created_at":        "created_at",
	"updated_at":        "updated_at",
	"title":             "title",
	"status":            "status",
	"amount":            "amount",
	"expected_close_at": "expected_close_at",
}

func dealSortColumn(s string) string {
	if col, ok := dealSortColumns[s]; ok {
		return col
	}
	return "created_at"
}

func (r *DealRepository) List(ctx context.Context, customerID uuid.UUID, f domain.DealListFilter) ([]*domain.Deal, int, error) {
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
	if f.ServiceID != nil {
		args = append(args, *f.ServiceID)
		conds = append(conds, fmt.Sprintf("service_id = $%d", len(args)))
	}
	if f.Status != "" {
		args = append(args, f.Status)
		conds = append(conds, fmt.Sprintf("status = $%d::deal_status", len(args)))
	}
	if f.Type != "" {
		args = append(args, f.Type)
		conds = append(conds, fmt.Sprintf("type = $%d::deal_type", len(args)))
	}
	if f.AssignedTo != nil {
		args = append(args, *f.AssignedTo)
		conds = append(conds, fmt.Sprintf("assigned_to = $%d", len(args)))
	}
	if f.Search != "" {
		args = append(args, "%"+f.Search+"%")
		conds = append(conds, fmt.Sprintf("title ILIKE $%d", len(args)))
	}
	where := "WHERE " + strings.Join(conds, " AND ")

	var total int
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM deals "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("postgres: count deals: %w", err)
	}
	if total == 0 {
		return []*domain.Deal{}, 0, nil
	}

	args = append(args, f.Limit)
	limitPos := len(args)
	args = append(args, f.Offset)
	offsetPos := len(args)

	query := fmt.Sprintf(
		"SELECT "+dealColumns+" FROM deals %s ORDER BY %s %s, id LIMIT $%d OFFSET $%d",
		where, dealSortColumn(f.SortBy), sortDir(f.SortDesc), limitPos, offsetPos,
	)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("postgres: list deals: %w", err)
	}
	defer rows.Close()

	var items []*domain.Deal
	for rows.Next() {
		d, err := scanDeal(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("postgres: scan deal: %w", err)
		}
		items = append(items, d)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("postgres: iterate deals: %w", err)
	}
	return items, total, nil
}

func (r *DealRepository) GetByID(ctx context.Context, customerID, id uuid.UUID) (*domain.Deal, error) {
	query := "SELECT " + dealColumns + " FROM deals WHERE id = $1 AND customer_id = $2 AND deleted_at IS NULL"
	d, err := scanDeal(r.pool.QueryRow(ctx, query, id, customerID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("postgres: get deal %s: %w", id, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("postgres: get deal %s: %w", id, err)
	}
	return d, nil
}

func (r *DealRepository) Update(ctx context.Context, customerID uuid.UUID, d *domain.Deal) error {
	const query = `
		UPDATE deals SET
			project_id        = $1,
			client_id         = $2,
			service_id        = $3,
			title             = $4,
			description       = NULLIF($5, ''),
			status            = $6::deal_status,
			type              = $7::deal_type,
			amount            = $8::numeric,
			currency          = $9,
			probability       = $10,
			expected_close_at = $11,
			closed_at         = $12,
			assigned_to       = $13,
			attributes        = $14::jsonb
		WHERE id = $15 AND customer_id = $16 AND deleted_at IS NULL
		RETURNING created_at, updated_at
	`
	err := r.pool.QueryRow(ctx, query,
		d.ProjectID, d.ClientID, d.ServiceID, d.Title, d.Description, d.Status, d.Type, d.Amount, d.Currency, d.Probability,
		d.ExpectedCloseAt, d.ClosedAt, d.AssignedTo, jsonbArg(d.Attributes), d.ID, customerID,
	).Scan(&d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("postgres: update deal %s: %w", d.ID, domain.ErrNotFound)
		}
		return fmt.Errorf("postgres: update deal: %w", err)
	}
	return nil
}

func (r *DealRepository) SoftDelete(ctx context.Context, customerID, id uuid.UUID) (time.Time, error) {
	const query = `
		UPDATE deals SET deleted_at = NOW()
		WHERE id = $1 AND customer_id = $2 AND deleted_at IS NULL
		RETURNING deleted_at
	`
	var deletedAt time.Time
	if err := r.pool.QueryRow(ctx, query, id, customerID).Scan(&deletedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return time.Time{}, fmt.Errorf("postgres: soft delete deal %s: %w", id, domain.ErrNotFound)
		}
		return time.Time{}, fmt.Errorf("postgres: soft delete deal %s: %w", id, err)
	}
	return deletedAt, nil
}
