DROP INDEX IF EXISTS idx_processes_project;
ALTER TABLE processes DROP COLUMN IF EXISTS project_id;
