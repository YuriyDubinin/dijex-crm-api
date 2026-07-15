package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/YuriyDubinin/bms-api/internal/domain"
)

type AuthTokenRepository struct {
	pool *pgxpool.Pool
}

func NewAuthTokenRepository(pool *pgxpool.Pool) *AuthTokenRepository {
	return &AuthTokenRepository{pool: pool}
}

// Create вставляет токен в auth_tokens. Опциональные текстовые поля при пустой
// строке становятся NULL через NULLIF (иначе INET-каст на "" упадёт).
func (r *AuthTokenRepository) Create(ctx context.Context, t *domain.AuthToken) error {
	const query = `
		INSERT INTO auth_tokens (
			id, customer_id, user_id, token_hash, token_type,
			issued_at, expires_at,
			ip_address, user_agent,
			device_type, device_name,
			os, os_version, browser, browser_version, app_version,
			country_code, city
		) VALUES (
			$1, $2, $3, $4, $5::auth_token_type,
			$6, $7,
			NULLIF($8, '')::inet,
			NULLIF($9, ''),
			COALESCE(NULLIF($10, ''), 'UNKNOWN')::auth_device_type,
			NULLIF($11, ''),
			NULLIF($12, ''), NULLIF($13, ''), NULLIF($14, ''), NULLIF($15, ''), NULLIF($16, ''),
			NULLIF($17, ''), NULLIF($18, '')
		)
	`
	_, err := r.pool.Exec(ctx, query,
		t.ID, t.CustomerID, t.UserID, t.TokenHash, t.TokenType,
		t.IssuedAt, t.ExpiresAt,
		t.IPAddress, t.UserAgent,
		t.DeviceType, t.DeviceName,
		t.OS, t.OSVersion, t.Browser, t.BrowserVersion, t.AppVersion,
		t.CountryCode, t.City,
	)
	if err != nil {
		return fmt.Errorf("postgres: create auth token: %w", err)
	}
	return nil
}

// FindActiveByHash подтягивает токен и актуальные role/is_active владельца
// одним JOIN. Логические проверки (истёк/отозван/отключён) — в service-слое.
func (r *AuthTokenRepository) FindActiveByHash(ctx context.Context, hash string) (*domain.AuthTokenWithUser, error) {
	const query = `
		SELECT
			t.id, t.customer_id, t.user_id, t.token_hash, t.token_type::text,
			t.issued_at, t.expires_at, t.last_used_at, t.revoked_at,
			COALESCE(t.revoked_reason, ''),
			u.role, u.is_active
		FROM auth_tokens t
		JOIN customer_users u ON u.id = t.user_id
		WHERE t.token_hash = $1
	`
	var (
		res       domain.AuthTokenWithUser
		tokenType string
	)
	err := r.pool.QueryRow(ctx, query, hash).Scan(
		&res.Token.ID,
		&res.Token.CustomerID,
		&res.Token.UserID,
		&res.Token.TokenHash,
		&tokenType,
		&res.Token.IssuedAt,
		&res.Token.ExpiresAt,
		&res.Token.LastUsedAt,
		&res.Token.RevokedAt,
		&res.Token.RevokedReason,
		&res.UserRole,
		&res.UserActive,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("postgres: find auth token by hash: %w", domain.ErrNotFound)
		}
		return nil, fmt.Errorf("postgres: find auth token by hash: %w", err)
	}
	res.Token.TokenType = tokenType
	return &res, nil
}

// TouchLastUsed — обновление last_used_at (вызывается асинхронно).
func (r *AuthTokenRepository) TouchLastUsed(ctx context.Context, id uuid.UUID, now time.Time) error {
	const query = `UPDATE auth_tokens SET last_used_at = $1 WHERE id = $2`
	if _, err := r.pool.Exec(ctx, query, now, id); err != nil {
		return fmt.Errorf("postgres: touch auth token last_used_at: %w", err)
	}
	return nil
}

// Revoke помечает токен отозванным. Условие revoked_at IS NULL защищает историю
// (в т.ч. когда токен уже отозван триггером при отключении пользователя).
func (r *AuthTokenRepository) Revoke(ctx context.Context, id uuid.UUID, reason string, now time.Time) error {
	const query = `
		UPDATE auth_tokens
		SET revoked_at = $1, revoked_reason = $2
		WHERE id = $3 AND revoked_at IS NULL
	`
	if _, err := r.pool.Exec(ctx, query, now, reason, id); err != nil {
		return fmt.Errorf("postgres: revoke auth token: %w", err)
	}
	return nil
}
