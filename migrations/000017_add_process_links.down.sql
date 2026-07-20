DROP INDEX IF EXISTS idx_tasks_process_stage;
DROP INDEX IF EXISTS idx_tasks_process;
DROP INDEX IF EXISTS idx_tasks_service;
ALTER TABLE tasks DROP COLUMN IF EXISTS process_stage_id;
ALTER TABLE tasks DROP COLUMN IF EXISTS process_id;
ALTER TABLE tasks DROP COLUMN IF EXISTS service_id;

DROP INDEX IF EXISTS idx_deals_process;
ALTER TABLE deals DROP COLUMN IF EXISTS process_id;
