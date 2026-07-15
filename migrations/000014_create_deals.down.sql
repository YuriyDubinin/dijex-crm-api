DROP TABLE IF EXISTS deals;
DROP TYPE IF EXISTS deal_status;
DROP TYPE IF EXISTS deal_type;
-- Примечание: значение 'DEAL' в ENUM entity_type НЕ удаляется — PostgreSQL не
-- поддерживает удаление значений из enum. Осиротевшее значение безвредно
-- (ничего на него не ссылается после DROP TABLE deals).
