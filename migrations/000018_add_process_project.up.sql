-- Привязка процесса к проекту.
-- Отдельная (append-only) миграция: таблица processes могла быть создана в
-- 000016 ДО появления этой колонки, поэтому добавляем идемпотентно
-- (IF NOT EXISTS) — применится и на «старых» БД при рестарте, и на свежих
-- (как no-op).
ALTER TABLE processes ADD COLUMN IF NOT EXISTS project_id UUID REFERENCES projects(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_processes_project ON processes (project_id);
