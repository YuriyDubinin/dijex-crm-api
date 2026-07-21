-- CLIENT_PROJECTS — членство клиента в проектах (M:N). Позволяет одному клиенту
-- участвовать в нескольких проектах (в дополнение к «основному» clients.project_id).
-- Append-only миграция: применится на рестарте без сноса базы.
CREATE TABLE client_projects (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    client_id   UUID NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (client_id, project_id)
);

CREATE INDEX idx_client_projects_customer ON client_projects (customer_id);
CREATE INDEX idx_client_projects_client ON client_projects (client_id);
CREATE INDEX idx_client_projects_project ON client_projects (project_id);
