-- PROCESSES — бизнес-процессы тенанта с пользовательскими этапами.
-- Процесс агрегирует сделки и задачи; статус фиксирует завершение и исход
-- (COMPLETED — успех, FAILED — провал). Этапы (process_stages) задаёт сам
-- пользователь: произвольные название/описание и порядок (position).

-- Регистрируем новый тип сущности, чтобы процессы участвовали в универсальных
-- связях (entity_links), тегах и кастом-полях. IF NOT EXISTS — идемпотентность.
ALTER TYPE entity_type ADD VALUE IF NOT EXISTS 'PROCESS';

-- Статусы процесса: активен / завершён успешно / завершён неудачно (провал).
CREATE TYPE process_status AS ENUM ('ACTIVE', 'COMPLETED', 'FAILED');

CREATE TABLE processes (
    id          UUID PRIMARY KEY,
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    project_id  UUID REFERENCES projects(id) ON DELETE SET NULL,
    name        TEXT NOT NULL,
    description TEXT,
    status      process_status NOT NULL DEFAULT 'ACTIVE',
    closed_at   TIMESTAMPTZ,          -- фактический момент завершения (COMPLETED/FAILED)
    attributes  JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);

CREATE INDEX idx_processes_customer ON processes (customer_id);
CREATE INDEX idx_processes_project ON processes (project_id);
CREATE INDEX idx_processes_status ON processes (customer_id, status);
CREATE INDEX idx_processes_attributes_gin ON processes USING GIN (attributes);

CREATE TRIGGER trg_processes_register_entity
    BEFORE INSERT ON processes
    FOR EACH ROW EXECUTE FUNCTION fn_register_entity('PROCESS');

CREATE TRIGGER set_updated_at_processes
    BEFORE UPDATE ON processes
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

-- PROCESS_STAGES — пользовательские этапы процесса. Дочерняя таблица (в общий
-- реестр entities НЕ регистрируется): задачи ссылаются на этап напрямую через
-- FK. Порядок этапов задаётся через position.
CREATE TABLE process_stages (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    process_id  UUID NOT NULL REFERENCES processes(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    description TEXT,
    position    INTEGER NOT NULL DEFAULT 0,
    attributes  JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);

CREATE INDEX idx_process_stages_customer ON process_stages (customer_id);
CREATE INDEX idx_process_stages_process ON process_stages (process_id, position);

CREATE TRIGGER set_updated_at_process_stages
    BEFORE UPDATE ON process_stages
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();
