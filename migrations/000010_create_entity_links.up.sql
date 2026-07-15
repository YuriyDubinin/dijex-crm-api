-- ENTITY_LINKS — универсальные связи "любая сущность ↔ любая сущность" внутри
-- тенанта. Обе стороны — id из реестра entities. Всё, что не покрыто прямыми FK.
CREATE TABLE entity_links (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id   UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    src_entity_id UUID NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    dst_entity_id UUID NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    relation_type TEXT NOT NULL DEFAULT 'RELATED',
    attributes    JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_by    UUID REFERENCES customer_users(id) ON DELETE SET NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_entity_links_no_self CHECK (src_entity_id <> dst_entity_id),
    CONSTRAINT uq_entity_links UNIQUE (src_entity_id, dst_entity_id, relation_type)
);

CREATE INDEX idx_entity_links_customer ON entity_links (customer_id);
CREATE INDEX idx_entity_links_src ON entity_links (src_entity_id);
CREATE INDEX idx_entity_links_dst ON entity_links (dst_entity_id);
CREATE INDEX idx_entity_links_relation ON entity_links (relation_type);

-- Обе стороны связи обязаны принадлежать одному тенанту (tenant isolation).
CREATE OR REPLACE FUNCTION fn_validate_entity_link()
RETURNS TRIGGER AS $$
DECLARE
    src_customer UUID;
    dst_customer UUID;
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

-- Неориентированный обход связей (обе стороны как entity_id → linked_entity_id).
CREATE VIEW v_entity_links_bidirectional AS
    SELECT id, customer_id, src_entity_id AS entity_id, dst_entity_id AS linked_entity_id,
           relation_type, attributes, created_at
    FROM entity_links
    UNION ALL
    SELECT id, customer_id, dst_entity_id AS entity_id, src_entity_id AS linked_entity_id,
           relation_type, attributes, created_at
    FROM entity_links;
