package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// EntityLink — универсальная связь "любая сущность ↔ любая сущность" внутри
// одного тенанта. Обе стороны — id из общего реестра entities. RelationType
// описывает семантику связи (свободная строка, дефолт RELATED).
type EntityLink struct {
	ID           uuid.UUID
	CustomerID   uuid.UUID
	SrcEntityID  uuid.UUID
	DstEntityID  uuid.UUID
	RelationType string
	Attributes   json.RawMessage
	CreatedBy    *uuid.UUID
	CreatedAt    time.Time
}

const RelationTypeDefault = "RELATED"

// LinkedEntity — элемент выборки связей: связанная сущность с её типом
// (JOIN entities) и параметрами самой связи.
type LinkedEntity struct {
	LinkID         uuid.UUID
	EntityID       uuid.UUID // сущность, относительно которой строим выборку
	LinkedEntityID uuid.UUID // связанная с ней сущность
	LinkedType     string    // entity_type связанной сущности
	LinkedLabel    string    // человекочитаемое имя связанной сущности (name/title/full_name)
	RelationType   string
	Attributes     json.RawMessage
	CreatedAt      time.Time
}

// EntityLinkListFilter — параметры выборки связей внутри тенанта.
// EntityID (если задан) возвращает все связи, где сущность участвует любой
// из сторон (src ИЛИ dst) — неориентированный обход.
type EntityLinkListFilter struct {
	EntityID     *uuid.UUID
	RelationType string
	LinkedType   string // фильтр по типу связанной сущности

	Limit    int
	Offset   int
	SortDesc bool
}
