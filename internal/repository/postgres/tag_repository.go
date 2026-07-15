package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/YuriyDubinin/bms-api/internal/domain"
)

type TagRepository struct {
	pool *pgxpool.Pool
}

func NewTagRepository(pool *pgxpool.Pool) *TagRepository {
	return &TagRepository{pool: pool}
}

const tagColumns = `
	id, customer_id, name, COALESCE(color, ''), created_at
`

func scanTag(row pgx.Row) (*domain.Tag, error) {
	var t domain.Tag
	err := row.Scan(&t.ID, &t.CustomerID, &t.Name, &t.Color, &t.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *TagRepository) Create(ctx context.Context, t *domain.Tag) error {
	const query = `
		INSERT INTO tags (id, customer_id, name, color)
		VALUES ($1, $2, $3, NULLIF($4, ''))
		RETURNING created_at
	`
	if err := r.pool.QueryRow(ctx, query, t.ID, t.CustomerID, t.Name, t.Color).Scan(&t.CreatedAt); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
			return fmt.Errorf("postgres: create tag: %w", domain.ErrAlreadyExists)
		}
		return fmt.Errorf("postgres: create tag: %w", err)
	}
	return nil
}

func (r *TagRepository) List(ctx context.Context, customerID uuid.UUID, f domain.TagListFilter) ([]*domain.Tag, int, error) {
	conds := []string{"customer_id = $1"}
	args := []any{customerID}

	if f.EntityID != nil {
		args = append(args, *f.EntityID)
		conds = append(conds, fmt.Sprintf("id IN (SELECT tag_id FROM entity_tags WHERE entity_id = $%d)", len(args)))
	}
	if f.Search != "" {
		args = append(args, "%"+f.Search+"%")
		conds = append(conds, fmt.Sprintf("name ILIKE $%d", len(args)))
	}
	where := "WHERE " + strings.Join(conds, " AND ")

	var total int
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM tags "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("postgres: count tags: %w", err)
	}
	if total == 0 {
		return []*domain.Tag{}, 0, nil
	}

	args = append(args, f.Limit)
	limitPos := len(args)
	args = append(args, f.Offset)
	offsetPos := len(args)

	query := fmt.Sprintf(
		"SELECT "+tagColumns+" FROM tags %s ORDER BY name %s, id LIMIT $%d OFFSET $%d",
		where, sortDir(f.SortDesc), limitPos, offsetPos,
	)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("postgres: list tags: %w", err)
	}
	defer rows.Close()

	var items []*domain.Tag
	for rows.Next() {
		t, err := scanTag(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("postgres: scan tag: %w", err)
		}
		items = append(items, t)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("postgres: iterate tags: %w", err)
	}
	return items, total, nil
}

func (r *TagRepository) GetByID(ctx context.Context, customerID, id uuid.UUID) (*domain.Tag, error) {
	query := "SELECT " + tagColumns + " FROM tags WHERE id = $1 AND customer_id = $2"
	t, err := scanTag(r.pool.QueryRow(ctx, query, id, customerID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("postgres: get tag %s: %w", id, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("postgres: get tag %s: %w", id, err)
	}
	return t, nil
}

func (r *TagRepository) Update(ctx context.Context, customerID uuid.UUID, t *domain.Tag) error {
	const query = `
		UPDATE tags SET
			name  = $1,
			color = NULLIF($2, '')
		WHERE id = $3 AND customer_id = $4
		RETURNING created_at
	`
	err := r.pool.QueryRow(ctx, query, t.Name, t.Color, t.ID, customerID).Scan(&t.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("postgres: update tag %s: %w", t.ID, domain.ErrNotFound)
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
			return fmt.Errorf("postgres: update tag: %w", domain.ErrAlreadyExists)
		}
		return fmt.Errorf("postgres: update tag: %w", err)
	}
	return nil
}

func (r *TagRepository) Delete(ctx context.Context, customerID, id uuid.UUID) error {
	const query = `DELETE FROM tags WHERE id = $1 AND customer_id = $2`
	tag, err := r.pool.Exec(ctx, query, id, customerID)
	if err != nil {
		return fmt.Errorf("postgres: delete tag %s: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("postgres: delete tag %s: %w", id, domain.ErrNotFound)
	}
	return nil
}

func (r *TagRepository) Attach(ctx context.Context, customerID, tagID, entityID uuid.UUID) error {
	const query = `
		INSERT INTO entity_tags (entity_id, tag_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`
	if _, err := r.pool.Exec(ctx, query, entityID, tagID); err != nil {
		return fmt.Errorf("postgres: attach tag %s: %w", tagID, err)
	}
	return nil
}

func (r *TagRepository) Detach(ctx context.Context, customerID, tagID, entityID uuid.UUID) error {
	const query = `DELETE FROM entity_tags WHERE tag_id = $1 AND entity_id = $2`
	tag, err := r.pool.Exec(ctx, query, tagID, entityID)
	if err != nil {
		return fmt.Errorf("postgres: detach tag %s: %w", tagID, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("postgres: detach tag %s: %w", tagID, domain.ErrNotFound)
	}
	return nil
}
