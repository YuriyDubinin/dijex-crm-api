-- AUTH_TOKENS — авторизационные токены операторов. Хранится только SHA-256
-- хэш секрета. Токен привязан к тенанту (customer_id) и пользователю (user_id).
CREATE TYPE auth_token_type AS ENUM ('ACCESS', 'REFRESH');
CREATE TYPE auth_device_type AS ENUM ('WEB', 'MOBILE', 'TABLET', 'DESKTOP', 'API', 'UNKNOWN');

CREATE TABLE auth_tokens (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id     UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    user_id         UUID NOT NULL REFERENCES customer_users(id) ON DELETE CASCADE,
    token_hash      TEXT NOT NULL UNIQUE,
    token_type      auth_token_type NOT NULL DEFAULT 'ACCESS',
    issued_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ NOT NULL,
    last_used_at    TIMESTAMPTZ,
    revoked_at      TIMESTAMPTZ,
    revoked_reason  TEXT,
    ip_address      INET,
    user_agent      TEXT,
    device_type     auth_device_type NOT NULL DEFAULT 'UNKNOWN',
    device_name     VARCHAR(255),
    os              VARCHAR(100),
    os_version      VARCHAR(50),
    browser         VARCHAR(100),
    browser_version VARCHAR(50),
    app_version     VARCHAR(50),
    country_code    CHAR(2),
    city            VARCHAR(100),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_auth_tokens_user ON auth_tokens (user_id);
CREATE INDEX idx_auth_tokens_customer ON auth_tokens (customer_id);
CREATE INDEX idx_auth_tokens_expires_at ON auth_tokens (expires_at);
CREATE INDEX idx_auth_tokens_active ON auth_tokens (user_id) WHERE revoked_at IS NULL;

CREATE TRIGGER set_updated_at_auth_tokens
    BEFORE UPDATE ON auth_tokens
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

-- При отключении/soft-delete пользователя отзываем все его активные токены.
CREATE OR REPLACE FUNCTION trigger_revoke_tokens_on_user_disable()
RETURNS TRIGGER AS $$
BEGIN
    IF (NEW.is_active = FALSE OR NEW.deleted_at IS NOT NULL)
       AND (OLD.is_active = TRUE AND OLD.deleted_at IS NULL) THEN
        UPDATE auth_tokens
        SET revoked_at = NOW(), revoked_reason = 'user disabled'
        WHERE user_id = NEW.id AND revoked_at IS NULL;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_revoke_tokens_on_user_disable
    AFTER UPDATE ON customer_users
    FOR EACH ROW EXECUTE FUNCTION trigger_revoke_tokens_on_user_disable();
