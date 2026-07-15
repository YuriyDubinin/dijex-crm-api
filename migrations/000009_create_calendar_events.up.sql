-- CALENDAR_EVENTS — события календаря тенанта + участники события.
CREATE TYPE calendar_event_status AS ENUM ('SCHEDULED', 'COMPLETED', 'CANCELLED');
CREATE TYPE attendee_response AS ENUM ('PENDING', 'ACCEPTED', 'DECLINED', 'TENTATIVE');

CREATE TABLE calendar_events (
    id              UUID PRIMARY KEY,
    customer_id     UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    project_id      UUID REFERENCES projects(id) ON DELETE SET NULL,
    title           TEXT NOT NULL,
    description     TEXT,
    location        TEXT,
    starts_at       TIMESTAMPTZ NOT NULL,
    ends_at         TIMESTAMPTZ NOT NULL,
    all_day         BOOLEAN NOT NULL DEFAULT FALSE,
    status          calendar_event_status NOT NULL DEFAULT 'SCHEDULED',
    recurrence_rule TEXT,
    timezone        TEXT NOT NULL DEFAULT 'UTC',
    created_by      UUID REFERENCES customer_users(id) ON DELETE SET NULL,
    attributes      JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ,

    CONSTRAINT chk_calendar_events_period CHECK (ends_at > starts_at)
);

CREATE INDEX idx_calendar_events_customer ON calendar_events (customer_id);
CREATE INDEX idx_calendar_events_project ON calendar_events (project_id);
CREATE INDEX idx_calendar_events_period ON calendar_events USING GIST (tstzrange(starts_at, ends_at));

CREATE TRIGGER trg_calendar_events_register_entity
    BEFORE INSERT ON calendar_events
    FOR EACH ROW EXECUTE FUNCTION fn_register_entity('CALENDAR_EVENT');

CREATE TRIGGER set_updated_at_calendar_events
    BEFORE UPDATE ON calendar_events
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

-- Участники события: любая зарегистрированная entity того же тенанта.
CREATE TABLE calendar_event_attendees (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id           UUID NOT NULL REFERENCES calendar_events(id) ON DELETE CASCADE,
    attendee_entity_id UUID NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    response_status    attendee_response NOT NULL DEFAULT 'PENDING',
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (event_id, attendee_entity_id)
);

CREATE INDEX idx_cea_event ON calendar_event_attendees (event_id);
CREATE INDEX idx_cea_attendee ON calendar_event_attendees (attendee_entity_id);

-- Участник обязан принадлежать тому же тенанту, что и событие.
CREATE OR REPLACE FUNCTION fn_validate_attendee()
RETURNS TRIGGER AS $$
DECLARE
    event_customer    UUID;
    attendee_customer UUID;
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
