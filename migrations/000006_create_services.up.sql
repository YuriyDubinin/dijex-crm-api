-- SERVICES — услуги тенанта ("установка документов", "сеанс массажа").
CREATE TYPE service_status AS ENUM ('ACTIVE', 'INACTIVE', 'ARCHIVED');

CREATE TABLE services (
    id           UUID PRIMARY KEY,
    customer_id  UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    project_id   UUID REFERENCES projects(id) ON DELETE SET NULL,
    name         TEXT NOT NULL,
    category     TEXT,
    description  TEXT,
    price        NUMERIC(12,2),
    currency     CHAR(3) NOT NULL DEFAULT 'USD',
    duration_min INTEGER,
    status       service_status NOT NULL DEFAULT 'ACTIVE',
    attributes   JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at   TIMESTAMPTZ
);

CREATE INDEX idx_services_customer ON services (customer_id);
CREATE INDEX idx_services_project ON services (project_id);
CREATE INDEX idx_services_attributes_gin ON services USING GIN (attributes);

CREATE TRIGGER trg_services_register_entity
    BEFORE INSERT ON services
    FOR EACH ROW EXECUTE FUNCTION fn_register_entity('SERVICE');

CREATE TRIGGER set_updated_at_services
    BEFORE UPDATE ON services
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();
