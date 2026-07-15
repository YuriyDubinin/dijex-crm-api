-- TAGS — сквозные теги тенанта + связь many-to-many с любыми сущностями.
CREATE TABLE tags (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    color       TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (customer_id, name)
);

CREATE TABLE entity_tags (
    entity_id UUID NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    tag_id    UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (entity_id, tag_id)
);

CREATE INDEX idx_entity_tags_tag ON entity_tags (tag_id);
