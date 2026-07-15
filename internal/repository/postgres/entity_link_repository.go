package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/YuriyDubinin/bms-api/internal/domain"
)

type EntityLinkRepository struct {
	pool *pgxpool.Pool
}

func NewEntityLinkRepository(pool *pgxpool.Pool) *EntityLinkRepository {
	return &EntityLinkRepository{pool: pool}
}

// Create вставляет связь. Принадлежность обеих сторон тенанту дополнительно
// гарантируется БД-триггером fn_validate_entity_link. Дубликат (src,dst,
// relation_type) → ErrAlreadyExists.
func (r *EntityLinkRepository) Create(ctx context.Context, l *domain.EntityLink) error {
	const query = `
		INSERT INTO entity_links
			(id, customer_id, src_entity_id, dst_entity_id, relation_type, attributes, created_by)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7)
		RETURNING created_at
	`
	err := r.pool.QueryRow(ctx, query,
		l.ID, l.CustomerID, l.SrcEntityID, l.DstEntityID, l.RelationType, jsonbArg(l.Attributes), l.CreatedBy,
	).Scan(&l.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
			return fmt.Errorf("postgres: create entity link: %w", domain.ErrAlreadyExists)
		}
		return fmt.Errorf("postgres: create entity link: %w", err)
	}
	return nil
}

// List возвращает связи тенанта. Если f.EntityID задан — используется
// неориентированное представление (сущность как любая из сторон), иначе —
// базовая таблица (каждая связь один раз: src→dst).
func (r *EntityLinkRepository) List(ctx context.Context, customerID uuid.UUID, f domain.EntityLinkListFilter) ([]*domain.LinkedEntity, int, error) {
	source := `(SELECT id, customer_id, src_entity_id AS entity_id, dst_entity_id AS linked_entity_id,
	                  relation_type, attributes, created_at FROM entity_links)`
	if f.EntityID != nil {
		source = "v_entity_links_bidirectional"
	}

	conds := []string{"l.customer_id = $1"}
	args := []any{customerID}

	if f.EntityID != nil {
		args = append(args, *f.EntityID)
		conds = append(conds, fmt.Sprintf("l.entity_id = $%d", len(args)))
	}
	if f.RelationType != "" {
		args = append(args, f.RelationType)
		conds = append(conds, fmt.Sprintf("l.relation_type = $%d", len(args)))
	}
	if f.LinkedType != "" {
		args = append(args, f.LinkedType)
		conds = append(conds, fmt.Sprintf("e.entity_type = $%d::entity_type", len(args)))
	}
	where := "WHERE " + strings.Join(conds, " AND ")

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s l JOIN entities e ON e.id = l.linked_entity_id %s", source, where)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("postgres: count entity links: %w", err)
	}
	if total == 0 {
		return []*domain.LinkedEntity{}, 0, nil
	}

	args = append(args, f.Limit)
	limitPos := len(args)
	args = append(args, f.Offset)
	offsetPos := len(args)

	// linked_label — человекочитаемое имя связанной сущности, по её типу.
	// Позволяет фронту показывать связи без дозапроса за каждой сущностью.
	query := fmt.Sprintf(`
		SELECT l.id, l.entity_id, l.linked_entity_id, e.entity_type::text,
		       COALESCE(
		           CASE e.entity_type
		               WHEN 'PROJECT'       THEN (SELECT name FROM projects WHERE id = l.linked_entity_id)
		               WHEN 'SERVICE'       THEN (SELECT name FROM services WHERE id = l.linked_entity_id)
		               WHEN 'CLIENT'        THEN (SELECT COALESCE(NULLIF(company_name, ''), NULLIF(TRIM(COALESCE(first_name,'') || ' ' || COALESCE(last_name,'')), '')) FROM clients WHERE id = l.linked_entity_id)
		               WHEN 'TASK'          THEN (SELECT title FROM tasks WHERE id = l.linked_entity_id)
		               WHEN 'DEAL'          THEN (SELECT title FROM deals WHERE id = l.linked_entity_id)
		               WHEN 'CUSTOMER_USER' THEN (SELECT full_name FROM customer_users WHERE id = l.linked_entity_id)
		           END, '') AS linked_label,
		       l.relation_type, l.attributes::text, l.created_at
		FROM %s l
		JOIN entities e ON e.id = l.linked_entity_id
		%s
		ORDER BY l.created_at %s, l.id
		LIMIT $%d OFFSET $%d
	`, source, where, sortDir(f.SortDesc), limitPos, offsetPos)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("postgres: list entity links: %w", err)
	}
	defer rows.Close()

	var items []*domain.LinkedEntity
	for rows.Next() {
		var (
			le   domain.LinkedEntity
			attr []byte
		)
		if err := rows.Scan(&le.LinkID, &le.EntityID, &le.LinkedEntityID, &le.LinkedType, &le.LinkedLabel, &le.RelationType, &attr, &le.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("postgres: scan entity link: %w", err)
		}
		le.Attributes = attr
		items = append(items, &le)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("postgres: iterate entity links: %w", err)
	}
	return items, total, nil
}

// Delete удаляет связь (жёстко — это не бизнес-запись). Тенант-скоуп по customer_id.
func (r *EntityLinkRepository) Delete(ctx context.Context, customerID, id uuid.UUID) error {
	const query = `DELETE FROM entity_links WHERE id = $1 AND customer_id = $2`
	tag, err := r.pool.Exec(ctx, query, id, customerID)
	if err != nil {
		return fmt.Errorf("postgres: delete entity link: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("postgres: delete entity link %s: %w", id, domain.ErrNotFound)
	}
	return nil
}
