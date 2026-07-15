-- ENTITIES — общий реестр "линкуемых" сущностей (супертайп). Каждая бизнес-
-- сущность (project/service/client/task/calendar_event/customer_user) при
-- вставке автоматически регистрируется здесь через BEFORE INSERT триггер
-- fn_register_entity. Это даёт единый механизм связей (entity_links), тегов и
-- кастом-полей поверх всех сущностей, с реальными FK-ограничениями.

CREATE TYPE entity_type AS ENUM (
    'PROJECT',
    'SERVICE',
    'CLIENT',
    'TASK',
    'CALENDAR_EVENT',
    'CUSTOMER_USER'
);

CREATE TABLE entities (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    entity_type entity_type NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_entities_customer ON entities (customer_id);
CREATE INDEX idx_entities_type ON entities (entity_type);

-- Регистрирует подтип-строку в entities. Тип передаётся аргументом триггера
-- (TG_ARGV[0]). id согласован с бизнес-строкой: если приложение задало NEW.id —
-- используем его, иначе генерируем и возвращаем в NEW.id.
CREATE OR REPLACE FUNCTION fn_register_entity()
RETURNS TRIGGER AS $$
DECLARE
    new_id UUID;
BEGIN
    INSERT INTO entities (id, customer_id, entity_type)
    VALUES (COALESCE(NEW.id, gen_random_uuid()), NEW.customer_id, TG_ARGV[0]::entity_type)
    RETURNING id INTO new_id;

    NEW.id := new_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
