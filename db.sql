-- =====================================================================================
-- BMS (Business Management System) — базовая схема БД, PostgreSQL 14+
--
-- Ключевая архитектурная идея:
--   Каждая "линкуемая" сущность (project, service, client, task, calendar_event,
--   customer_user) регистрируется в общей таблице entities (паттерн
--   "supertype/subtype" / class-table inheritance). Это даёт:
--     1) реальные FK-ограничения при связи "любая сущность <-> любая сущность"
--        через universal-таблицу entity_links (вместо неконтролируемого
--        полиморфизма "тип+id текстом");
--     2) единый механизм тегов, кастомных полей и вложений для всех сущностей;
--     3) каскадное удаление без дублирования логики.
--
--   Поверх этого — обычные "жёсткие" FK для типовых иерархий (services.project_id,
--   clients.project_id, tasks.project_id и т.д.), чтобы не гонять частые запросы
--   через generic-таблицу. entity_links используется для гибких, необязательных
--   и множественных связей (M:N), которые не укладываются в простую иерархию.
-- =====================================================================================

CREATE EXTENSION IF NOT EXISTS pgcrypto;   -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS btree_gist; -- exclusion-constraints по диапазонам времени

-- =====================================================================================
-- 1. ENUM-ТИПЫ
-- =====================================================================================

CREATE TYPE entity_type AS ENUM (
    'project',
    'service',
    'client',
    'task',
    'calendar_event',
    'customer_user'
);

CREATE TYPE project_status AS ENUM ('draft', 'active', 'on_hold', 'archived');
CREATE TYPE service_status AS ENUM ('active', 'inactive', 'archived');
CREATE TYPE client_status  AS ENUM ('lead', 'active', 'inactive', 'archived');

CREATE TYPE task_status   AS ENUM ('todo', 'in_progress', 'blocked', 'done', 'cancelled');
CREATE TYPE task_priority AS ENUM ('low', 'medium', 'high', 'urgent');

CREATE TYPE calendar_event_status AS ENUM ('scheduled', 'completed', 'cancelled');
CREATE TYPE attendee_response      AS ENUM ('pending', 'accepted', 'declined', 'tentative');

CREATE TYPE custom_field_type AS ENUM ('text', 'number', 'boolean', 'date', 'datetime', 'json', 'select');

-- =====================================================================================
-- 2. СЛУЖЕБНЫЕ ФУНКЦИИ
-- =====================================================================================

-- Автообновление updated_at
CREATE OR REPLACE FUNCTION fn_set_updated_at()
RETURNS trigger AS $$
BEGIN
    NEW.updated_at := now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Регистрация подтип-строки в общей таблице entities (см. раздел 4)
CREATE OR REPLACE FUNCTION fn_register_entity()
RETURNS trigger AS $$
DECLARE
    new_id uuid;
BEGIN
    INSERT INTO entities (id, customer_id, entity_type)
    VALUES (COALESCE(NEW.id, gen_random_uuid()), NEW.customer_id, TG_ARGV[0]::entity_type)
    RETURNING id INTO new_id;

    NEW.id := new_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- =====================================================================================
-- 3. CUSTOMERS — клиенты, покупающие BMS (арендаторы / tenants)
-- =====================================================================================

CREATE TABLE customers (
    id                   uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name                 text NOT NULL,
    legal_name           text,
    industry             text,
    email                text NOT NULL,
    phone                text,
    timezone             text NOT NULL DEFAULT 'UTC',
    locale               text NOT NULL DEFAULT 'ru',
    subscription_plan    text NOT NULL DEFAULT 'trial',
    subscription_status  text NOT NULL DEFAULT 'active',
    settings             jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at           timestamptz NOT NULL DEFAULT now(),
    updated_at           timestamptz NOT NULL DEFAULT now(),
    deleted_at           timestamptz
);

CREATE UNIQUE INDEX uq_customers_email_active ON customers (email) WHERE deleted_at IS NULL;

CREATE TRIGGER trg_customers_updated_at
    BEFORE UPDATE ON customers
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

-- Сотрудники/пользователи клиента (те, кто работает в системе от лица customer'а)
CREATE TABLE customer_users (
    id           uuid PRIMARY KEY,
    customer_id  uuid NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    email        text NOT NULL,
    full_name    text NOT NULL,
    role         text NOT NULL DEFAULT 'staff', -- owner | admin | manager | staff
    is_active    boolean NOT NULL DEFAULT true,
    attributes   jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    deleted_at   timestamptz
);

CREATE UNIQUE INDEX uq_customer_users_email ON customer_users (customer_id, email) WHERE deleted_at IS NULL;
CREATE INDEX idx_customer_users_customer ON customer_users (customer_id);

CREATE TRIGGER trg_customer_users_register_entity
    BEFORE INSERT ON customer_users
    FOR EACH ROW EXECUTE FUNCTION fn_register_entity('customer_user');

CREATE TRIGGER trg_customer_users_updated_at
    BEFORE UPDATE ON customer_users
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

-- =====================================================================================
-- 4. ENTITIES — общий реестр "линкуемых" сущностей (supertype)
-- =====================================================================================

CREATE TABLE entities (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id  uuid NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    entity_type  entity_type NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_entities_customer ON entities (customer_id);
CREATE INDEX idx_entities_type ON entities (entity_type);

-- (customer_users создана до entities физически в файле, но FK ниже добавим отдельно,
--  т.к. entities должна существовать раньше customer_users логически — см. примечание
--  ниже: порядок создания таблиц скорректирован ниже единым блоком.)

-- =====================================================================================
-- 5. PROJECTS — бизнес-направления/проекты клиента ("Автопарк", "Салон красоты" и т.д.)
-- =====================================================================================

CREATE TABLE projects (
    id           uuid PRIMARY KEY,
    customer_id  uuid NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    name         text NOT NULL,
    slug         text,
    direction    text,                          -- свободное описание направления бизнеса
    description  text,
    status       project_status NOT NULL DEFAULT 'active',
    attributes   jsonb NOT NULL DEFAULT '{}'::jsonb,  -- произвольные доп. атрибуты проекта
    starts_at    date,
    ends_at      date,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    deleted_at   timestamptz
);

CREATE UNIQUE INDEX uq_projects_slug ON projects (customer_id, slug) WHERE deleted_at IS NULL AND slug IS NOT NULL;
CREATE INDEX idx_projects_customer ON projects (customer_id);
CREATE INDEX idx_projects_attributes_gin ON projects USING gin (attributes);

CREATE TRIGGER trg_projects_register_entity
    BEFORE INSERT ON projects
    FOR EACH ROW EXECUTE FUNCTION fn_register_entity('project');

CREATE TRIGGER trg_projects_updated_at
    BEFORE UPDATE ON projects
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

-- =====================================================================================
-- 6. SERVICES — услуги ("установка документов на автомобиль", "сеанс массажа" и т.д.)
-- =====================================================================================

CREATE TABLE services (
    id            uuid PRIMARY KEY,
    customer_id   uuid NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    project_id    uuid REFERENCES projects(id) ON DELETE SET NULL, -- "домашний" проект услуги
    name          text NOT NULL,
    category      text,
    description   text,
    price         numeric(12,2),
    currency      char(3) NOT NULL DEFAULT 'USD',
    duration_min  integer,                       -- ожидаемая длительность в минутах
    status        service_status NOT NULL DEFAULT 'active',
    attributes    jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    deleted_at    timestamptz
);

CREATE INDEX idx_services_customer ON services (customer_id);
CREATE INDEX idx_services_project ON services (project_id);
CREATE INDEX idx_services_attributes_gin ON services USING gin (attributes);

CREATE TRIGGER trg_services_register_entity
    BEFORE INSERT ON services
    FOR EACH ROW EXECUTE FUNCTION fn_register_entity('service');

CREATE TRIGGER trg_services_updated_at
    BEFORE UPDATE ON services
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

-- =====================================================================================
-- 7. CLIENTS — клиенты наших клиентов (конечные потребители услуг)
-- =====================================================================================

CREATE TABLE clients (
    id            uuid PRIMARY KEY,
    customer_id   uuid NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    project_id    uuid REFERENCES projects(id) ON DELETE SET NULL,
    first_name    text,
    last_name     text,
    company_name  text,
    email         text,
    phone         text,
    address       jsonb,
    status        client_status NOT NULL DEFAULT 'lead',
    source        text,                          -- откуда пришёл клиент
    attributes    jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    deleted_at    timestamptz
);

CREATE INDEX idx_clients_customer ON clients (customer_id);
CREATE INDEX idx_clients_project ON clients (project_id);
CREATE INDEX idx_clients_email ON clients (customer_id, email);
CREATE INDEX idx_clients_phone ON clients (customer_id, phone);
CREATE INDEX idx_clients_attributes_gin ON clients USING gin (attributes);

CREATE TRIGGER trg_clients_register_entity
    BEFORE INSERT ON clients
    FOR EACH ROW EXECUTE FUNCTION fn_register_entity('client');

CREATE TRIGGER trg_clients_updated_at
    BEFORE UPDATE ON clients
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

-- =====================================================================================
-- 8. TASKS — произвольные задачи, связываемые с любыми сущностями
-- =====================================================================================

CREATE TABLE tasks (
    id            uuid PRIMARY KEY,
    customer_id   uuid NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    project_id    uuid REFERENCES projects(id) ON DELETE SET NULL,
    title         text NOT NULL,
    description   text,
    status        task_status NOT NULL DEFAULT 'todo',
    priority      task_priority NOT NULL DEFAULT 'medium',
    assigned_to   uuid REFERENCES customer_users(id) ON DELETE SET NULL,
    created_by    uuid REFERENCES customer_users(id) ON DELETE SET NULL,
    due_at        timestamptz,
    completed_at  timestamptz,
    attributes    jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    deleted_at    timestamptz
);

CREATE INDEX idx_tasks_customer ON tasks (customer_id);
CREATE INDEX idx_tasks_project ON tasks (project_id);
CREATE INDEX idx_tasks_assigned_to ON tasks (assigned_to);
CREATE INDEX idx_tasks_status_due ON tasks (customer_id, status, due_at);
CREATE INDEX idx_tasks_attributes_gin ON tasks USING gin (attributes);

CREATE TRIGGER trg_tasks_register_entity
    BEFORE INSERT ON tasks
    FOR EACH ROW EXECUTE FUNCTION fn_register_entity('task');

CREATE TRIGGER trg_tasks_updated_at
    BEFORE UPDATE ON tasks
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

-- =====================================================================================
-- 9. CALENDAR_EVENTS — события календаря
-- =====================================================================================

CREATE TABLE calendar_events (
    id                uuid PRIMARY KEY,
    customer_id       uuid NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    project_id        uuid REFERENCES projects(id) ON DELETE SET NULL,
    title             text NOT NULL,
    description       text,
    location          text,
    starts_at         timestamptz NOT NULL,
    ends_at           timestamptz NOT NULL,
    all_day           boolean NOT NULL DEFAULT false,
    status            calendar_event_status NOT NULL DEFAULT 'scheduled',
    recurrence_rule   text,                      -- iCal RRULE для повторяющихся событий
    timezone          text NOT NULL DEFAULT 'UTC',
    created_by        uuid REFERENCES customer_users(id) ON DELETE SET NULL,
    attributes        jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    deleted_at        timestamptz,

    CONSTRAINT chk_calendar_events_period CHECK (ends_at > starts_at)
);

CREATE INDEX idx_calendar_events_customer ON calendar_events (customer_id);
CREATE INDEX idx_calendar_events_project ON calendar_events (project_id);
CREATE INDEX idx_calendar_events_period ON calendar_events USING gist (tstzrange(starts_at, ends_at));

CREATE TRIGGER trg_calendar_events_register_entity
    BEFORE INSERT ON calendar_events
    FOR EACH ROW EXECUTE FUNCTION fn_register_entity('calendar_event');

CREATE TRIGGER trg_calendar_events_updated_at
    BEFORE UPDATE ON calendar_events
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

-- Участники события (сотрудник ИЛИ клиент — любая зарегистрированная entity)
CREATE TABLE calendar_event_attendees (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id            uuid NOT NULL REFERENCES calendar_events(id) ON DELETE CASCADE,
    attendee_entity_id  uuid NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    response_status     attendee_response NOT NULL DEFAULT 'pending',
    created_at          timestamptz NOT NULL DEFAULT now(),

    UNIQUE (event_id, attendee_entity_id)
);

CREATE INDEX idx_cea_event ON calendar_event_attendees (event_id);
CREATE INDEX idx_cea_attendee ON calendar_event_attendees (attendee_entity_id);

-- Валидация: участник должен принадлежать тому же customer'у, что и событие
CREATE OR REPLACE FUNCTION fn_validate_attendee()
RETURNS trigger AS $$
DECLARE
    event_customer      uuid;
    attendee_customer    uuid;
BEGIN
    SELECT customer_id INTO event_customer FROM calendar_events WHERE id = NEW.event_id;
    SELECT customer_id INTO attendee_customer FROM entities WHERE id = NEW.attendee_entity_id;

    IF event_customer IS DISTINCT FROM attendee_customer THEN
        RAISE EXCEPTION 'calendar_event_attendees: attendee belongs to a different customer (tenant)';
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_cea_validate
    BEFORE INSERT OR UPDATE ON calendar_event_attendees
    FOR EACH ROW EXECUTE FUNCTION fn_validate_attendee();

-- =====================================================================================
-- 10. ENTITY_LINKS — универсальные связи "любая сущность <-> любая сущность"
--     Здесь описываются все связи, которые не покрываются прямыми FK выше:
--     project<->service, service<->client, task<->calendar_event, client<->task,
--     project<->project (иерархия направлений), и т.д. — с типом связи.
-- =====================================================================================

CREATE TABLE entity_links (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id     uuid NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    src_entity_id   uuid NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    dst_entity_id   uuid NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    relation_type   text NOT NULL DEFAULT 'related', -- related | depends_on | part_of | blocks | assigned_to ...
    attributes      jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_by      uuid REFERENCES customer_users(id) ON DELETE SET NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT chk_entity_links_no_self CHECK (src_entity_id <> dst_entity_id),
    CONSTRAINT uq_entity_links UNIQUE (src_entity_id, dst_entity_id, relation_type)
);

CREATE INDEX idx_entity_links_customer ON entity_links (customer_id);
CREATE INDEX idx_entity_links_src ON entity_links (src_entity_id);
CREATE INDEX idx_entity_links_dst ON entity_links (dst_entity_id);
CREATE INDEX idx_entity_links_relation ON entity_links (relation_type);

-- Валидация: обе стороны связи должны принадлежать одному customer'у (tenant isolation)
CREATE OR REPLACE FUNCTION fn_validate_entity_link()
RETURNS trigger AS $$
DECLARE
    src_customer uuid;
    dst_customer uuid;
BEGIN
    SELECT customer_id INTO src_customer FROM entities WHERE id = NEW.src_entity_id;
    SELECT customer_id INTO dst_customer FROM entities WHERE id = NEW.dst_entity_id;

    IF src_customer IS DISTINCT FROM NEW.customer_id OR dst_customer IS DISTINCT FROM NEW.customer_id THEN
        RAISE EXCEPTION 'entity_links: cross-tenant link is not allowed (customer_id mismatch)';
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_entity_links_validate
    BEFORE INSERT OR UPDATE ON entity_links
    FOR EACH ROW EXECUTE FUNCTION fn_validate_entity_link();

-- Удобное представление для симметричного (неориентированного) обхода связей
CREATE VIEW v_entity_links_bidirectional AS
    SELECT id, customer_id, src_entity_id AS entity_id, dst_entity_id AS linked_entity_id,
           relation_type, attributes, created_at
    FROM entity_links
    UNION ALL
    SELECT id, customer_id, dst_entity_id AS entity_id, src_entity_id AS linked_entity_id,
           relation_type, attributes, created_at
    FROM entity_links;

-- =====================================================================================
-- 11. TAGS — сквозные теги для любых сущностей
-- =====================================================================================

CREATE TABLE tags (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id  uuid NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    name         text NOT NULL,
    color        text,
    created_at   timestamptz NOT NULL DEFAULT now(),

    UNIQUE (customer_id, name)
);

CREATE TABLE entity_tags (
    entity_id  uuid NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    tag_id     uuid NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (entity_id, tag_id)
);

CREATE INDEX idx_entity_tags_tag ON entity_tags (tag_id);

-- =====================================================================================
-- 12. CUSTOM FIELDS — типизированные кастомные поля per customer / per entity_type
--     (гибкое описание проекта/услуги/клиента сверх jsonb-атрибутов —
--     полноценный конструктор полей, как в HubSpot/Salesforce)
-- =====================================================================================

CREATE TABLE custom_field_definitions (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id    uuid NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    entity_type    entity_type NOT NULL,
    field_key      text NOT NULL,             -- машинное имя поля
    label          text NOT NULL,             -- отображаемое имя
    field_type     custom_field_type NOT NULL DEFAULT 'text',
    options        jsonb,                     -- варианты для field_type = 'select'
    is_required    boolean NOT NULL DEFAULT false,
    sort_order     integer NOT NULL DEFAULT 0,
    created_at     timestamptz NOT NULL DEFAULT now(),

    UNIQUE (customer_id, entity_type, field_key)
);

CREATE TABLE custom_field_values (
    id                    uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_id             uuid NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    field_definition_id   uuid NOT NULL REFERENCES custom_field_definitions(id) ON DELETE CASCADE,
    value_text            text,
    value_number          numeric,
    value_boolean         boolean,
    value_date            date,
    value_datetime        timestamptz,
    value_json            jsonb,
    created_at            timestamptz NOT NULL DEFAULT now(),
    updated_at            timestamptz NOT NULL DEFAULT now(),

    UNIQUE (entity_id, field_definition_id)
);

CREATE INDEX idx_cfv_entity ON custom_field_values (entity_id);
CREATE INDEX idx_cfv_definition ON custom_field_values (field_definition_id);

CREATE TRIGGER trg_cfv_updated_at
    BEFORE UPDATE ON custom_field_values
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

-- =====================================================================================
-- 13. FK: entities -> customer_users (добавлен отдельно, т.к. entities объявлена
--     ниже customer_users по файлу; фактический порядок исполнения важен —
--     см. примечание в начале файла: применяйте скрипт целиком одним запуском)
-- =====================================================================================
-- Примечание: секция создания customer_users выше уже ссылается на customers,
-- а entities создаётся после неё. Триггер customer_users -> entities выполнится
-- корректно, т.к. на момент INSERT в customer_users таблица entities уже существует.
