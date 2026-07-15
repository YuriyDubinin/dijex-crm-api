-- TASKS — произвольные задачи тенанта. assigned_to/created_by → customer_users.
CREATE TYPE task_status AS ENUM ('TODO', 'IN_PROGRESS', 'BLOCKED', 'DONE', 'CANCELLED');
CREATE TYPE task_priority AS ENUM ('LOW', 'MEDIUM', 'HIGH', 'URGENT');

CREATE TABLE tasks (
    id           UUID PRIMARY KEY,
    customer_id  UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    project_id   UUID REFERENCES projects(id) ON DELETE SET NULL,
    title        TEXT NOT NULL,
    description  TEXT,
    status       task_status NOT NULL DEFAULT 'TODO',
    priority     task_priority NOT NULL DEFAULT 'MEDIUM',
    assigned_to  UUID REFERENCES customer_users(id) ON DELETE SET NULL,
    created_by   UUID REFERENCES customer_users(id) ON DELETE SET NULL,
    due_at       TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    attributes   JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at   TIMESTAMPTZ
);

CREATE INDEX idx_tasks_customer ON tasks (customer_id);
CREATE INDEX idx_tasks_project ON tasks (project_id);
CREATE INDEX idx_tasks_assigned_to ON tasks (assigned_to);
CREATE INDEX idx_tasks_status_due ON tasks (customer_id, status, due_at);
CREATE INDEX idx_tasks_attributes_gin ON tasks USING GIN (attributes);

CREATE TRIGGER trg_tasks_register_entity
    BEFORE INSERT ON tasks
    FOR EACH ROW EXECUTE FUNCTION fn_register_entity('TASK');

CREATE TRIGGER set_updated_at_tasks
    BEFORE UPDATE ON tasks
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();
