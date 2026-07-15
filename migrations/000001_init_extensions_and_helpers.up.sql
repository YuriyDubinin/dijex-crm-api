-- Расширения и общие помощники, используемые всеми последующими миграциями.
CREATE EXTENSION IF NOT EXISTS pgcrypto;   -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS btree_gist; -- GiST-индексы по диапазонам времени (календарь)

-- Общая триггер-функция автообновления updated_at. Переиспользуется на всех
-- таблицах, где есть колонка updated_at.
CREATE OR REPLACE FUNCTION trigger_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
