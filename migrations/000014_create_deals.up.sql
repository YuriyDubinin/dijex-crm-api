-- DEALS — сделки (CRM): суммы, этапы воронки, привязка к клиенту/проекту/ответственному.

-- Регистрируем новый тип сущности в общем реестре entities — чтобы сделки тоже
-- участвовали в универсальных связях (entity_links), тегах и кастом-полях.
-- IF NOT EXISTS делает шаг идемпотентным (повторный прогон после отката не упадёт).
ALTER TYPE entity_type ADD VALUE IF NOT EXISTS 'DEAL';

-- Этапы воронки продаж.
CREATE TYPE deal_status AS ENUM ('NEW', 'PENDING', 'WON', 'LOST');

-- Тип сделки: доход или расход (для учёта денежного потока).
CREATE TYPE deal_type AS ENUM ('INCOME', 'EXPENSE');

CREATE TABLE deals (
    id                UUID PRIMARY KEY,
    customer_id       UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    project_id        UUID REFERENCES projects(id) ON DELETE SET NULL,
    client_id         UUID REFERENCES clients(id) ON DELETE SET NULL,
    service_id        UUID REFERENCES services(id) ON DELETE SET NULL,
    title             TEXT NOT NULL,
    description       TEXT,
    status            deal_status NOT NULL DEFAULT 'NEW',
    type              deal_type NOT NULL DEFAULT 'INCOME',
    amount            NUMERIC(14,2),
    currency          CHAR(3) NOT NULL DEFAULT 'USD',
    probability       INTEGER,
    expected_close_at DATE,
    closed_at         TIMESTAMPTZ,          -- фактическая дата закрытия (WON/LOST)
    assigned_to       UUID REFERENCES customer_users(id) ON DELETE SET NULL,
    created_by        UUID REFERENCES customer_users(id) ON DELETE SET NULL,
    attributes        JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at        TIMESTAMPTZ,

    CONSTRAINT chk_deals_probability CHECK (probability IS NULL OR (probability BETWEEN 0 AND 100)),
    CONSTRAINT chk_deals_amount CHECK (amount IS NULL OR amount >= 0)
);

CREATE INDEX idx_deals_customer ON deals (customer_id);
CREATE INDEX idx_deals_project ON deals (project_id);
CREATE INDEX idx_deals_client ON deals (client_id);
CREATE INDEX idx_deals_service ON deals (service_id);
CREATE INDEX idx_deals_assigned_to ON deals (assigned_to);
CREATE INDEX idx_deals_status ON deals (customer_id, status);
CREATE INDEX idx_deals_type ON deals (customer_id, type);
CREATE INDEX idx_deals_attributes_gin ON deals USING GIN (attributes);

CREATE TRIGGER trg_deals_register_entity
    BEFORE INSERT ON deals
    FOR EACH ROW EXECUTE FUNCTION fn_register_entity('DEAL');

CREATE TRIGGER set_updated_at_deals
    BEFORE UPDATE ON deals
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();
