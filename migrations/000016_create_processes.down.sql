DROP TABLE IF EXISTS process_stages;
DROP TABLE IF EXISTS processes;
DROP TYPE IF EXISTS process_status;
-- Примечание: значение 'PROCESS' в ENUM entity_type НЕ удаляется — PostgreSQL
-- не поддерживает удаление значений из enum. Осиротевшее значение безвредно
-- (ничего на него не ссылается после DROP TABLE processes).
