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

type CustomFieldRepository struct {
	pool *pgxpool.Pool
}

func NewCustomFieldRepository(pool *pgxpool.Pool) *CustomFieldRepository {
	return &CustomFieldRepository{pool: pool}
}

// ───────────────────────── Определения ─────────────────────────

const customFieldDefColumns = `
	id, customer_id, entity_type::text, field_key, label, field_type::text,
	COALESCE(options::text, ''), is_required, sort_order, created_at
`

func scanCustomFieldDef(row pgx.Row) (*domain.CustomFieldDefinition, error) {
	var (
		d    domain.CustomFieldDefinition
		opts []byte
	)
	err := row.Scan(
		&d.ID, &d.CustomerID, &d.EntityType, &d.FieldKey, &d.Label, &d.FieldType,
		&opts, &d.IsRequired, &d.SortOrder, &d.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	if len(opts) > 0 {
		d.Options = opts
	}
	return &d, nil
}

func (r *CustomFieldRepository) CreateDefinition(ctx context.Context, d *domain.CustomFieldDefinition) error {
	const query = `
		INSERT INTO custom_field_definitions
			(id, customer_id, entity_type, field_key, label, field_type, options, is_required, sort_order)
		VALUES ($1, $2, $3::entity_type, $4, $5, $6::custom_field_type, $7::jsonb, $8, $9)
		RETURNING created_at
	`
	err := r.pool.QueryRow(ctx, query,
		d.ID, d.CustomerID, d.EntityType, d.FieldKey, d.Label, d.FieldType,
		jsonbArgOrNil(d.Options), d.IsRequired, d.SortOrder,
	).Scan(&d.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
			return fmt.Errorf("postgres: create custom field definition: %w", domain.ErrAlreadyExists)
		}
		return fmt.Errorf("postgres: create custom field definition: %w", err)
	}
	return nil
}

func (r *CustomFieldRepository) ListDefinitions(ctx context.Context, customerID uuid.UUID, f domain.CustomFieldDefinitionListFilter) ([]*domain.CustomFieldDefinition, int, error) {
	conds := []string{"customer_id = $1"}
	args := []any{customerID}
	if f.EntityType != "" {
		args = append(args, f.EntityType)
		conds = append(conds, fmt.Sprintf("entity_type = $%d::entity_type", len(args)))
	}
	where := "WHERE " + strings.Join(conds, " AND ")

	var total int
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM custom_field_definitions "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("postgres: count custom field definitions: %w", err)
	}
	if total == 0 {
		return []*domain.CustomFieldDefinition{}, 0, nil
	}

	args = append(args, f.Limit)
	limitPos := len(args)
	args = append(args, f.Offset)
	offsetPos := len(args)

	query := fmt.Sprintf(
		"SELECT "+customFieldDefColumns+" FROM custom_field_definitions %s ORDER BY sort_order %s, created_at, id LIMIT $%d OFFSET $%d",
		where, sortDir(f.SortDesc), limitPos, offsetPos,
	)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("postgres: list custom field definitions: %w", err)
	}
	defer rows.Close()

	var items []*domain.CustomFieldDefinition
	for rows.Next() {
		d, err := scanCustomFieldDef(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("postgres: scan custom field definition: %w", err)
		}
		items = append(items, d)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("postgres: iterate custom field definitions: %w", err)
	}
	return items, total, nil
}

func (r *CustomFieldRepository) GetDefinitionByID(ctx context.Context, customerID, id uuid.UUID) (*domain.CustomFieldDefinition, error) {
	query := "SELECT " + customFieldDefColumns + " FROM custom_field_definitions WHERE id = $1 AND customer_id = $2"
	d, err := scanCustomFieldDef(r.pool.QueryRow(ctx, query, id, customerID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("postgres: get custom field definition %s: %w", id, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("postgres: get custom field definition %s: %w", id, err)
	}
	return d, nil
}

func (r *CustomFieldRepository) UpdateDefinition(ctx context.Context, customerID uuid.UUID, d *domain.CustomFieldDefinition) error {
	const query = `
		UPDATE custom_field_definitions SET
			label       = $1,
			field_type  = $2::custom_field_type,
			options     = $3::jsonb,
			is_required = $4,
			sort_order  = $5
		WHERE id = $6 AND customer_id = $7
		RETURNING created_at, entity_type::text, field_key
	`
	err := r.pool.QueryRow(ctx, query,
		d.Label, d.FieldType, jsonbArgOrNil(d.Options), d.IsRequired, d.SortOrder, d.ID, customerID,
	).Scan(&d.CreatedAt, &d.EntityType, &d.FieldKey)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("postgres: update custom field definition %s: %w", d.ID, domain.ErrNotFound)
		}
		return fmt.Errorf("postgres: update custom field definition: %w", err)
	}
	return nil
}

func (r *CustomFieldRepository) DeleteDefinition(ctx context.Context, customerID, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM custom_field_definitions WHERE id = $1 AND customer_id = $2", id, customerID)
	if err != nil {
		return fmt.Errorf("postgres: delete custom field definition: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("postgres: delete custom field definition %s: %w", id, domain.ErrNotFound)
	}
	return nil
}

// ───────────────────────── Значения ─────────────────────────

// SetValue делает upsert значения по (entity_id, field_definition_id).
// Ровно одна value_*-колонка заполнена сервисом; остальные NULL.
func (r *CustomFieldRepository) SetValue(ctx context.Context, customerID uuid.UUID, v *domain.CustomFieldValue) error {
	const query = `
		INSERT INTO custom_field_values
			(id, entity_id, field_definition_id, value_text, value_number, value_boolean, value_date, value_datetime, value_json)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb)
		ON CONFLICT (entity_id, field_definition_id) DO UPDATE SET
			value_text     = EXCLUDED.value_text,
			value_number   = EXCLUDED.value_number,
			value_boolean  = EXCLUDED.value_boolean,
			value_date     = EXCLUDED.value_date,
			value_datetime = EXCLUDED.value_datetime,
			value_json     = EXCLUDED.value_json
		RETURNING id, created_at, updated_at
	`
	err := r.pool.QueryRow(ctx, query,
		v.ID, v.EntityID, v.FieldDefinitionID,
		v.ValueText, v.ValueNumber, v.ValueBoolean, v.ValueDate, v.ValueDatetime, jsonbArgOrNil(v.ValueJSON),
	).Scan(&v.ID, &v.CreatedAt, &v.UpdatedAt)
	if err != nil {
		return fmt.Errorf("postgres: set custom field value: %w", err)
	}
	return nil
}

func (r *CustomFieldRepository) ListValues(ctx context.Context, customerID, entityID uuid.UUID) ([]*domain.CustomFieldValue, error) {
	const query = `
		SELECT v.id, v.entity_id, v.field_definition_id,
		       v.value_text, v.value_number::float8, v.value_boolean, v.value_date, v.value_datetime,
		       COALESCE(v.value_json::text, ''), v.created_at, v.updated_at
		FROM custom_field_values v
		JOIN custom_field_definitions d ON d.id = v.field_definition_id
		WHERE v.entity_id = $1 AND d.customer_id = $2
		ORDER BY d.sort_order, v.created_at
	`
	rows, err := r.pool.Query(ctx, query, entityID, customerID)
	if err != nil {
		return nil, fmt.Errorf("postgres: list custom field values: %w", err)
	}
	defer rows.Close()

	var items []*domain.CustomFieldValue
	for rows.Next() {
		var (
			v     domain.CustomFieldValue
			vjson []byte
		)
		if err := rows.Scan(
			&v.ID, &v.EntityID, &v.FieldDefinitionID,
			&v.ValueText, &v.ValueNumber, &v.ValueBoolean, &v.ValueDate, &v.ValueDatetime,
			&vjson, &v.CreatedAt, &v.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("postgres: scan custom field value: %w", err)
		}
		if len(vjson) > 0 {
			v.ValueJSON = vjson
		}
		items = append(items, &v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: iterate custom field values: %w", err)
	}
	return items, nil
}

func (r *CustomFieldRepository) UnsetValue(ctx context.Context, customerID, entityID, fieldDefinitionID uuid.UUID) error {
	const query = `
		DELETE FROM custom_field_values v
		USING custom_field_definitions d
		WHERE v.field_definition_id = d.id
		  AND v.entity_id = $1
		  AND v.field_definition_id = $2
		  AND d.customer_id = $3
	`
	tag, err := r.pool.Exec(ctx, query, entityID, fieldDefinitionID, customerID)
	if err != nil {
		return fmt.Errorf("postgres: unset custom field value: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("postgres: unset custom field value: %w", domain.ErrNotFound)
	}
	return nil
}
