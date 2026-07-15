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

type CustomerUserRepository struct {
	pool *pgxpool.Pool
}

func NewCustomerUserRepository(pool *pgxpool.Pool) *CustomerUserRepository {
	return &CustomerUserRepository{pool: pool}
}

const customerUserColumns = `
	id, customer_id, email, full_name, role, is_active, password_hash,
	attributes::text, created_at, updated_at, deleted_at
`

func scanCustomerUser(row pgx.Row) (*domain.CustomerUser, error) {
	var (
		u    domain.CustomerUser
		attr []byte
	)
	err := row.Scan(
		&u.ID, &u.CustomerID, &u.Email, &u.FullName, &u.Role, &u.IsActive, &u.PasswordHash,
		&attr, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	u.Attributes = attr
	return &u, nil
}

// FindByEmail — глобальный поиск оператора по email (для логина). Тенант
// выводится из строки. Возвращает ErrNotFound, если не найден/удалён.
func (r *CustomerUserRepository) FindByEmail(ctx context.Context, email string) (*domain.CustomerUser, error) {
	query := "SELECT " + customerUserColumns + " FROM customer_users WHERE email = $1 AND deleted_at IS NULL"
	u, err := scanCustomerUser(r.pool.QueryRow(ctx, query, email))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("postgres: find user by email: %w", domain.ErrNotFound)
		}
		return nil, fmt.Errorf("postgres: find user by email: %w", err)
	}
	return u, nil
}

// Create вставляет оператора. Конфликт уникального email → ErrAlreadyExists.
func (r *CustomerUserRepository) Create(ctx context.Context, u *domain.CustomerUser) error {
	const query = `
		INSERT INTO customer_users
			(id, customer_id, email, full_name, role, is_active, password_hash, attributes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9, $10)
	`
	_, err := r.pool.Exec(ctx, query,
		u.ID, u.CustomerID, u.Email, u.FullName, u.Role, u.IsActive, u.PasswordHash,
		jsonbArg(u.Attributes), u.CreatedAt, u.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
			return fmt.Errorf("postgres: create user: %w", domain.ErrAlreadyExists)
		}
		return fmt.Errorf("postgres: create user: %w", err)
	}
	return nil
}

var customerUserSortColumns = map[string]string{
	"created_at": "created_at",
	"updated_at": "updated_at",
	"full_name":  "full_name",
	"email":      "email",
	"role":       "role",
}

func customerUserSortColumn(s string) string {
	if col, ok := customerUserSortColumns[s]; ok {
		return col
	}
	return "created_at"
}

func (r *CustomerUserRepository) List(ctx context.Context, customerID uuid.UUID, f domain.CustomerUserListFilter) ([]*domain.CustomerUser, int, error) {
	conds := []string{"customer_id = $1", "deleted_at IS NULL"}
	args := []any{customerID}

	if f.Role != "" {
		args = append(args, f.Role)
		conds = append(conds, fmt.Sprintf("role = $%d", len(args)))
	}
	if f.IsActive != nil {
		args = append(args, *f.IsActive)
		conds = append(conds, fmt.Sprintf("is_active = $%d", len(args)))
	}
	if f.Search != "" {
		args = append(args, "%"+f.Search+"%")
		conds = append(conds, fmt.Sprintf("(full_name ILIKE $%d OR email ILIKE $%d)", len(args), len(args)))
	}
	where := "WHERE " + strings.Join(conds, " AND ")

	var total int
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM customer_users "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("postgres: count users: %w", err)
	}
	if total == 0 {
		return []*domain.CustomerUser{}, 0, nil
	}

	args = append(args, f.Limit)
	limitPos := len(args)
	args = append(args, f.Offset)
	offsetPos := len(args)

	query := fmt.Sprintf(
		"SELECT "+customerUserColumns+" FROM customer_users %s ORDER BY %s %s, id LIMIT $%d OFFSET $%d",
		where, customerUserSortColumn(f.SortBy), sortDir(f.SortDesc), limitPos, offsetPos,
	)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("postgres: list users: %w", err)
	}
	defer rows.Close()

	var items []*domain.CustomerUser
	for rows.Next() {
		u, err := scanCustomerUser(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("postgres: scan user: %w", err)
		}
		items = append(items, u)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("postgres: iterate users: %w", err)
	}
	return items, total, nil
}

func (r *CustomerUserRepository) GetByID(ctx context.Context, customerID, id uuid.UUID) (*domain.CustomerUser, error) {
	query := "SELECT " + customerUserColumns + " FROM customer_users WHERE id = $1 AND customer_id = $2 AND deleted_at IS NULL"
	u, err := scanCustomerUser(r.pool.QueryRow(ctx, query, id, customerID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("postgres: get user %s: %w", id, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("postgres: get user %s: %w", id, err)
	}
	return u, nil
}

// Update обновляет оператора. password_hash меняется только если передан
// непустым (COALESCE(NULLIF...)). Конфликт email → ErrAlreadyExists.
func (r *CustomerUserRepository) Update(ctx context.Context, customerID uuid.UUID, u *domain.CustomerUser) error {
	const query = `
		UPDATE customer_users SET
			email         = $1,
			full_name     = $2,
			role          = $3,
			is_active     = $4,
			password_hash = COALESCE(NULLIF($5, ''), password_hash),
			attributes    = $6::jsonb
		WHERE id = $7 AND customer_id = $8 AND deleted_at IS NULL
		RETURNING created_at, updated_at
	`
	err := r.pool.QueryRow(ctx, query,
		u.Email, u.FullName, u.Role, u.IsActive, u.PasswordHash, jsonbArg(u.Attributes), u.ID, customerID,
	).Scan(&u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("postgres: update user %s: %w", u.ID, domain.ErrNotFound)
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
			return fmt.Errorf("postgres: update user: %w", domain.ErrAlreadyExists)
		}
		return fmt.Errorf("postgres: update user: %w", err)
	}
	return nil
}

func (r *CustomerUserRepository) SoftDelete(ctx context.Context, customerID, id uuid.UUID) (time.Time, error) {
	const query = `
		UPDATE customer_users SET deleted_at = NOW(), is_active = FALSE
		WHERE id = $1 AND customer_id = $2 AND deleted_at IS NULL
		RETURNING deleted_at
	`
	var deletedAt time.Time
	if err := r.pool.QueryRow(ctx, query, id, customerID).Scan(&deletedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return time.Time{}, fmt.Errorf("postgres: soft delete user %s: %w", id, domain.ErrNotFound)
		}
		return time.Time{}, fmt.Errorf("postgres: soft delete user %s: %w", id, err)
	}
	return deletedAt, nil
}

func (r *CustomerUserRepository) CountOwners(ctx context.Context, customerID uuid.UUID) (int, error) {
	const query = `
		SELECT COUNT(*) FROM customer_users
		WHERE customer_id = $1 AND role = 'OWNER' AND is_active = TRUE AND deleted_at IS NULL
	`
	var n int
	if err := r.pool.QueryRow(ctx, query, customerID).Scan(&n); err != nil {
		return 0, fmt.Errorf("postgres: count owners: %w", err)
	}
	return n, nil
}
