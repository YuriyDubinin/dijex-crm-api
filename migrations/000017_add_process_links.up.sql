-- Привязка сделок и задач к процессам (и задач — к конкретному этапу процесса).

-- Сделку можно отнести к процессу.
ALTER TABLE deals ADD COLUMN process_id UUID REFERENCES processes(id) ON DELETE SET NULL;
CREATE INDEX idx_deals_process ON deals (process_id);

-- Задачу можно создать на услугу, на процесс и на конкретный этап процесса
-- (в дополнение к существующим project_id/client_id/deal_id). Все привязки
-- опциональны: задача может существовать в отрыве от всего.
ALTER TABLE tasks ADD COLUMN service_id       UUID REFERENCES services(id) ON DELETE SET NULL;
ALTER TABLE tasks ADD COLUMN process_id       UUID REFERENCES processes(id) ON DELETE SET NULL;
ALTER TABLE tasks ADD COLUMN process_stage_id UUID REFERENCES process_stages(id) ON DELETE SET NULL;

CREATE INDEX idx_tasks_service ON tasks (service_id);
CREATE INDEX idx_tasks_process ON tasks (process_id);
CREATE INDEX idx_tasks_process_stage ON tasks (process_stage_id);
