-- CUSTOM FIELDS — типизированный конструктор полей per тенант per тип сущности.
CREATE TYPE custom_field_type AS ENUM ('TEXT', 'NUMBER', 'BOOLEAN', 'DATE', 'DATETIME', 'JSON', 'SELECT');

CREATE TABLE custom_field_definitions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    entity_type entity_type NOT NULL,
    field_key   TEXT NOT NULL,
    label       TEXT NOT NULL,
    field_type  custom_field_type NOT NULL DEFAULT 'TEXT',
    options     JSONB,
    is_required BOOLEAN NOT NULL DEFAULT FALSE,
    sort_order  INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (customer_id, entity_type, field_key)
);

CREATE TABLE custom_field_values (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_id           UUID NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    field_definition_id UUID NOT NULL REFERENCES custom_field_definitions(id) ON DELETE CASCADE,
    value_text          TEXT,
    value_number        NUMERIC,
    value_boolean       BOOLEAN,
    value_date          DATE,
    value_datetime      TIMESTAMPTZ,
    value_json          JSONB,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (entity_id, field_definition_id)
);

CREATE INDEX idx_cfv_entity ON custom_field_values (entity_id);
CREATE INDEX idx_cfv_definition ON custom_field_values (field_definition_id);

CREATE TRIGGER set_updated_at_custom_field_values
    BEFORE UPDATE ON custom_field_values
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();
