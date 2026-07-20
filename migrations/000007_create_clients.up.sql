-- CLIENTS — клиенты тенанта (конечные потребители услуг).
CREATE TYPE client_status AS ENUM ('LEAD', 'ACTIVE', 'INACTIVE', 'ARCHIVED');

-- Вид субъекта: физическое или юридическое лицо.
CREATE TYPE client_subject_type AS ENUM ('INDIVIDUAL', 'LEGAL_ENTITY');

CREATE TABLE clients (
    id           UUID PRIMARY KEY,
    customer_id  UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    project_id   UUID REFERENCES projects(id) ON DELETE SET NULL,
    first_name   TEXT,
    last_name    TEXT,
    company_name TEXT,
    email        TEXT,
    phone        TEXT,
    address      JSONB,
    status       client_status NOT NULL DEFAULT 'LEAD',
    subject_type client_subject_type NOT NULL DEFAULT 'INDIVIDUAL',
    source       TEXT,
    attributes   JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at   TIMESTAMPTZ
);

CREATE INDEX idx_clients_customer ON clients (customer_id);
CREATE INDEX idx_clients_project ON clients (project_id);
CREATE INDEX idx_clients_email ON clients (customer_id, email);
CREATE INDEX idx_clients_phone ON clients (customer_id, phone);
CREATE INDEX idx_clients_attributes_gin ON clients USING GIN (attributes);

CREATE TRIGGER trg_clients_register_entity
    BEFORE INSERT ON clients
    FOR EACH ROW EXECUTE FUNCTION fn_register_entity('CLIENT');

CREATE TRIGGER set_updated_at_clients
    BEFORE UPDATE ON clients
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();
