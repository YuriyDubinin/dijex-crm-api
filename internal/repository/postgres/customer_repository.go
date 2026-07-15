package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/YuriyDubinin/bms-api/internal/domain"
)

type CustomerRepository struct {
	pool *pgxpool.Pool
}

func NewCustomerRepository(pool *pgxpool.Pool) *CustomerRepository {
	return &CustomerRepository{pool: pool}
}

const customerColumns = `
	id, name, COALESCE(legal_name, ''), COALESCE(industry, ''), email, COALESCE(phone, ''),
	timezone, locale, subscription_plan, subscription_status, settings::text,
	created_at, updated_at, deleted_at
`

func scanCustomer(row pgx.Row) (*domain.Customer, error) {
	var (
		c        domain.Customer
		settings []byte
	)
	err := row.Scan(
		&c.ID, &c.Name, &c.LegalName, &c.Industry, &c.Email, &c.Phone,
		&c.Timezone, &c.Locale, &c.SubscriptionPlan, &c.SubscriptionStatus, &settings,
		&c.CreatedAt, &c.UpdatedAt, &c.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	c.Settings = settings
	return &c, nil
}

// RegisterTenant атомарно создаёт тенант и его первого пользователя-владельца.
// Конфликт уникального email пользователя → ErrAlreadyExists.
func (r *CustomerRepository) RegisterTenant(ctx context.Context, c *domain.Customer, owner *domain.CustomerUser) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("postgres: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	const insCustomer = `INSERT INTO customers (id, name, email) VALUES ($1, $2, $3)`
	if _, err := tx.Exec(ctx, insCustomer, c.ID, c.Name, c.Email); err != nil {
		return fmt.Errorf("postgres: register tenant (customer): %w", err)
	}

	const insUser = `
		INSERT INTO customer_users (id, customer_id, email, full_name, role, is_active, password_hash)
		VALUES ($1, $2, $3, $4, $5, TRUE, $6)
	`
	_, err = tx.Exec(ctx, insUser, owner.ID, c.ID, owner.Email, owner.FullName, owner.Role, owner.PasswordHash)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
			return fmt.Errorf("postgres: register tenant (owner): %w", domain.ErrAlreadyExists)
		}
		return fmt.Errorf("postgres: register tenant (owner): %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("postgres: commit tx: %w", err)
	}
	return nil
}

func (r *CustomerRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Customer, error) {
	query := "SELECT " + customerColumns + " FROM customers WHERE id = $1 AND deleted_at IS NULL"
	c, err := scanCustomer(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("postgres: get customer %s: %w", id, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("postgres: get customer %s: %w", id, err)
	}
	return c, nil
}

func (r *CustomerRepository) Update(ctx context.Context, c *domain.Customer) error {
	const query = `
		UPDATE customers SET
			name       = $1,
			legal_name = NULLIF($2, ''),
			industry   = NULLIF($3, ''),
			email      = $4,
			phone      = NULLIF($5, ''),
			timezone   = $6,
			locale     = $7,
			settings   = $8::jsonb
		WHERE id = $9 AND deleted_at IS NULL
		RETURNING created_at, updated_at
	`
	err := r.pool.QueryRow(ctx, query,
		c.Name, c.LegalName, c.Industry, c.Email, c.Phone, c.Timezone, c.Locale, jsonbArg(c.Settings), c.ID,
	).Scan(&c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("postgres: update customer %s: %w", c.ID, domain.ErrNotFound)
		}
		return fmt.Errorf("postgres: update customer: %w", err)
	}
	return nil
}

func (r *CustomerRepository) SoftDelete(ctx context.Context, id uuid.UUID) (time.Time, error) {
	const query = `UPDATE customers SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL RETURNING deleted_at`
	var deletedAt time.Time
	if err := r.pool.QueryRow(ctx, query, id).Scan(&deletedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return time.Time{}, fmt.Errorf("postgres: soft delete customer %s: %w", id, domain.ErrNotFound)
		}
		return time.Time{}, fmt.Errorf("postgres: soft delete customer %s: %w", id, err)
	}
	return deletedAt, nil
}
