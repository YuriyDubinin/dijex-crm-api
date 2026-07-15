-- PROJECTS — бизнес-направления/проекты тенанта ("Автопарк", "Салон красоты").
CREATE TYPE project_status AS ENUM ('DRAFT', 'ACTIVE', 'ON_HOLD', 'ARCHIVED');

CREATE TABLE projects (
    id          UUID PRIMARY KEY,
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    slug        TEXT,
    direction   TEXT,
    description TEXT,
    status      project_status NOT NULL DEFAULT 'ACTIVE',
    attributes  JSONB NOT NULL DEFAULT '{}'::jsonb,
    starts_at   DATE,
    ends_at     DATE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);

CREATE UNIQUE INDEX uq_projects_slug ON projects (customer_id, slug) WHERE deleted_at IS NULL AND slug IS NOT NULL;
CREATE INDEX idx_projects_customer ON projects (customer_id);
CREATE INDEX idx_projects_attributes_gin ON projects USING GIN (attributes);

CREATE TRIGGER trg_projects_register_entity
    BEFORE INSERT ON projects
    FOR EACH ROW EXECUTE FUNCTION fn_register_entity('PROJECT');

CREATE TRIGGER set_updated_at_projects
    BEFORE UPDATE ON projects
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();
