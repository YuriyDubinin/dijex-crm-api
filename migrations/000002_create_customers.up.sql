-- CUSTOMERS — тенанты (клиенты, купившие BMS). Верхнеуровневый владелец всего.
CREATE TABLE customers (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                TEXT NOT NULL,
    legal_name          TEXT,
    industry            TEXT,
    email               TEXT NOT NULL,
    phone               TEXT,
    timezone            TEXT NOT NULL DEFAULT 'UTC',
    locale              TEXT NOT NULL DEFAULT 'ru',
    subscription_plan   TEXT NOT NULL DEFAULT 'TRIAL',
    subscription_status TEXT NOT NULL DEFAULT 'ACTIVE',
    settings            JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at          TIMESTAMPTZ
);

CREATE UNIQUE INDEX uq_customers_email_active ON customers (email) WHERE deleted_at IS NULL;

CREATE TRIGGER set_updated_at_customers
    BEFORE UPDATE ON customers
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();
