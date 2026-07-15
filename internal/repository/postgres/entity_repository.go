package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/YuriyDubinin/bms-api/internal/domain"
)

// EntityRepository — доступ к общему реестру entities (супертайп).
type EntityRepository struct {
	pool *pgxpool.Pool
}

func NewEntityRepository(pool *pgxpool.Pool) *EntityRepository {
	return &EntityRepository{pool: pool}
}

// ResolveType возвращает тип сущности, если она существует и принадлежит
// тенанту. Иначе ErrNotFound. Используется для проверки принадлежности при
// построении связей, тегов и кастом-полей.
func (r *EntityRepository) ResolveType(ctx context.Context, customerID, entityID uuid.UUID) (domain.EntityType, error) {
	const query = `SELECT entity_type::text FROM entities WHERE id = $1 AND customer_id = $2`
	var t string
	if err := r.pool.QueryRow(ctx, query, entityID, customerID).Scan(&t); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", fmt.Errorf("postgres: resolve entity %s: %w", entityID, domain.ErrNotFound)
		}
		return "", fmt.Errorf("postgres: resolve entity %s: %w", entityID, err)
	}
	return domain.EntityType(t), nil
}
