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

type ClientRepository struct {
	pool *pgxpool.Pool
}

func NewClientRepository(pool *pgxpool.Pool) *ClientRepository {
	return &ClientRepository{pool: pool}
}

const clientColumns = `
	id, customer_id, project_id,
	COALESCE(first_name, ''), COALESCE(last_name, ''), COALESCE(company_name, ''),
	COALESCE(email, ''), COALESCE(phone, ''), address::text,
	status::text, subject_type::text, COALESCE(source, ''), attributes::text, created_at, updated_at, deleted_at
`

func scanClient(row pgx.Row) (*domain.Client, error) {
	var (
		c    domain.Client
		addr []byte
		attr []byte
	)
	err := row.Scan(
		&c.ID, &c.CustomerID, &c.ProjectID,
		&c.FirstName, &c.LastName, &c.CompanyName,
		&c.Email, &c.Phone, &addr,
		&c.Status, &c.SubjectType, &c.Source, &attr, &c.CreatedAt, &c.UpdatedAt, &c.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	// address — nullable jsonb: пустое значение оставляем nil (сериализуется как
	// null), иначе пустой не-nil json.RawMessage сломал бы JSON-энкодинг.
	if len(addr) > 0 {
		c.Address = addr
	}
	c.Attributes = attr
	return &c, nil
}

func (r *ClientRepository) Create(ctx context.Context, c *domain.Client) error {
	const query = `
		INSERT INTO clients
			(id, customer_id, project_id, first_name, last_name, company_name, email, phone, address, status, subject_type, source, attributes, created_at, updated_at)
		VALUES
			($1, $2, $3, NULLIF($4, ''), NULLIF($5, ''), NULLIF($6, ''), NULLIF($7, ''), NULLIF($8, ''), $9::jsonb, $10::client_status, $11::client_subject_type, NULLIF($12, ''), $13::jsonb, $14, $15)
	`
	_, err := r.pool.Exec(ctx, query,
		c.ID, c.CustomerID, c.ProjectID, c.FirstName, c.LastName, c.CompanyName, c.Email, c.Phone,
		jsonbArgOrNil(c.Address), c.Status, c.SubjectType, c.Source, jsonbArg(c.Attributes), c.CreatedAt, c.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
			return fmt.Errorf("postgres: create client: %w", domain.ErrAlreadyExists)
		}
		return fmt.Errorf("postgres: create client: %w", err)
	}
	return nil
}

var clientSortColumns = map[string]string{
	"created_at":   "created_at",
	"updated_at":   "updated_at",
	"first_name":   "first_name",
	"last_name":    "last_name",
	"company_name": "company_name",
}

func clientSortColumn(s string) string {
	if col, ok := clientSortColumns[s]; ok {
		return col
	}
	return "created_at"
}

func (r *ClientRepository) List(ctx context.Context, customerID uuid.UUID, f domain.ClientListFilter) ([]*domain.Client, int, error) {
	conds := []string{"customer_id = $1", "deleted_at IS NULL"}
	args := []any{customerID}

	if f.ProjectID != nil {
		// Клиент относится к проекту, если это его «основной» project_id ИЛИ есть
		// членство в client_projects (M:N).
		args = append(args, *f.ProjectID)
		conds = append(conds, fmt.Sprintf(
			"(project_id = $%d OR EXISTS (SELECT 1 FROM client_projects cp WHERE cp.client_id = clients.id AND cp.project_id = $%d))",
			len(args), len(args)))
	}
	if f.Status != "" {
		args = append(args, f.Status)
		conds = append(conds, fmt.Sprintf("status = $%d::client_status", len(args)))
	}
	if f.SubjectType != "" {
		args = append(args, f.SubjectType)
		conds = append(conds, fmt.Sprintf("subject_type = $%d::client_subject_type", len(args)))
	}
	if f.Search != "" {
		args = append(args, "%"+f.Search+"%")
		conds = append(conds, fmt.Sprintf(
			"(first_name ILIKE $%d OR last_name ILIKE $%d OR company_name ILIKE $%d OR email ILIKE $%d OR phone ILIKE $%d)",
			len(args), len(args), len(args), len(args), len(args),
		))
	}
	where := "WHERE " + strings.Join(conds, " AND ")

	var total int
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM clients "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("postgres: count clients: %w", err)
	}
	if total == 0 {
		return []*domain.Client{}, 0, nil
	}

	args = append(args, f.Limit)
	limitPos := len(args)
	args = append(args, f.Offset)
	offsetPos := len(args)

	query := fmt.Sprintf(
		"SELECT "+clientColumns+" FROM clients %s ORDER BY %s %s, id LIMIT $%d OFFSET $%d",
		where, clientSortColumn(f.SortBy), sortDir(f.SortDesc), limitPos, offsetPos,
	)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("postgres: list clients: %w", err)
	}
	defer rows.Close()

	var items []*domain.Client
	for rows.Next() {
		c, err := scanClient(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("postgres: scan client: %w", err)
		}
		items = append(items, c)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("postgres: iterate clients: %w", err)
	}
	return items, total, nil
}

func (r *ClientRepository) GetByID(ctx context.Context, customerID, id uuid.UUID) (*domain.Client, error) {
	query := "SELECT " + clientColumns + " FROM clients WHERE id = $1 AND customer_id = $2 AND deleted_at IS NULL"
	c, err := scanClient(r.pool.QueryRow(ctx, query, id, customerID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("postgres: get client %s: %w", id, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("postgres: get client %s: %w", id, err)
	}
	return c, nil
}

func (r *ClientRepository) Update(ctx context.Context, customerID uuid.UUID, c *domain.Client) error {
	const query = `
		UPDATE clients SET
			project_id   = $1,
			first_name   = NULLIF($2, ''),
			last_name    = NULLIF($3, ''),
			company_name = NULLIF($4, ''),
			email        = NULLIF($5, ''),
			phone        = NULLIF($6, ''),
			address      = $7::jsonb,
			status       = $8::client_status,
			subject_type = $9::client_subject_type,
			source       = NULLIF($10, ''),
			attributes   = $11::jsonb
		WHERE id = $12 AND customer_id = $13 AND deleted_at IS NULL
		RETURNING created_at, updated_at
	`
	err := r.pool.QueryRow(ctx, query,
		c.ProjectID, c.FirstName, c.LastName, c.CompanyName, c.Email, c.Phone,
		jsonbArgOrNil(c.Address), c.Status, c.SubjectType, c.Source, jsonbArg(c.Attributes), c.ID, customerID,
	).Scan(&c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("postgres: update client %s: %w", c.ID, domain.ErrNotFound)
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
			return fmt.Errorf("postgres: update client: %w", domain.ErrAlreadyExists)
		}
		return fmt.Errorf("postgres: update client: %w", err)
	}
	return nil
}

func (r *ClientRepository) SoftDelete(ctx context.Context, customerID, id uuid.UUID) (time.Time, error) {
	const query = `
		UPDATE clients SET deleted_at = NOW()
		WHERE id = $1 AND customer_id = $2 AND deleted_at IS NULL
		RETURNING deleted_at
	`
	var deletedAt time.Time
	if err := r.pool.QueryRow(ctx, query, id, customerID).Scan(&deletedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return time.Time{}, fmt.Errorf("postgres: soft delete client %s: %w", id, domain.ErrNotFound)
		}
		return time.Time{}, fmt.Errorf("postgres: soft delete client %s: %w", id, err)
	}
	return deletedAt, nil
}

// AttachProject добавляет клиента в проект (M:N). Идемпотентно: повторная
// привязка — не ошибка.
func (r *ClientRepository) AttachProject(ctx context.Context, customerID, clientID, projectID uuid.UUID) error {
	const query = `
		INSERT INTO client_projects (customer_id, client_id, project_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (client_id, project_id) DO NOTHING
	`
	if _, err := r.pool.Exec(ctx, query, customerID, clientID, projectID); err != nil {
		return fmt.Errorf("postgres: attach client %s to project %s: %w", clientID, projectID, err)
	}
	return nil
}

// DetachProject убирает клиента из проекта. Идемпотентно: отсутствие связи — не ошибка.
func (r *ClientRepository) DetachProject(ctx context.Context, customerID, clientID, projectID uuid.UUID) error {
	const query = `DELETE FROM client_projects WHERE customer_id = $1 AND client_id = $2 AND project_id = $3`
	if _, err := r.pool.Exec(ctx, query, customerID, clientID, projectID); err != nil {
		return fmt.Errorf("postgres: detach client %s from project %s: %w", clientID, projectID, err)
	}
	return nil
}

// ListProjects возвращает все проекты клиента: «основной» project_id + членства
// (client_projects), без дублей, только не удалённые.
func (r *ClientRepository) ListProjects(ctx context.Context, customerID, clientID uuid.UUID) ([]*domain.Project, error) {
	query := "SELECT " + projectColumns + ` FROM projects
		WHERE customer_id = $1 AND deleted_at IS NULL
		  AND (
		    id = (SELECT project_id FROM clients WHERE id = $2 AND customer_id = $1)
		    OR id IN (SELECT project_id FROM client_projects WHERE client_id = $2 AND customer_id = $1)
		  )
		ORDER BY created_at, id`
	rows, err := r.pool.Query(ctx, query, customerID, clientID)
	if err != nil {
		return nil, fmt.Errorf("postgres: list client projects: %w", err)
	}
	defer rows.Close()

	var items []*domain.Project
	for rows.Next() {
		p, err := scanProject(rows)
		if err != nil {
			return nil, fmt.Errorf("postgres: scan client project: %w", err)
		}
		items = append(items, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: iterate client projects: %w", err)
	}
	return items, nil
}
