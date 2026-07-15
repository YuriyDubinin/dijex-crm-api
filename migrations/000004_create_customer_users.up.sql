-- CUSTOMER_USERS — операторы тенанта (те, кто логинится и работает в системе).
-- password_hash (bcrypt) добавлен относительно исходной db.sql — нужен для auth.
-- Email глобально уникален (частичный UNIQUE WHERE deleted_at IS NULL): это
-- позволяет логиниться по одному email+паролю, тенант выводится из строки.
CREATE TABLE customer_users (
    id            UUID PRIMARY KEY,
    customer_id   UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    email         TEXT NOT NULL,
    full_name     TEXT NOT NULL,
    role          TEXT NOT NULL DEFAULT 'STAFF', -- OWNER | ADMIN | MANAGER | STAFF
    is_active     BOOLEAN NOT NULL DEFAULT TRUE,
    password_hash TEXT NOT NULL,
    attributes    JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMPTZ
);

CREATE UNIQUE INDEX uq_customer_users_email ON customer_users (email) WHERE deleted_at IS NULL;
CREATE INDEX idx_customer_users_customer ON customer_users (customer_id);

CREATE TRIGGER trg_customer_users_register_entity
    BEFORE INSERT ON customer_users
    FOR EACH ROW EXECUTE FUNCTION fn_register_entity('CUSTOMER_USER');

CREATE TRIGGER set_updated_at_customer_users
    BEFORE UPDATE ON customer_users
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();
