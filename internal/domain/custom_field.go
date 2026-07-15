package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// CustomFieldDefinition — определение типизированного кастом-поля для
// конкретного типа сущности внутри тенанта (конструктор полей).
type CustomFieldDefinition struct {
	ID         uuid.UUID
	CustomerID uuid.UUID
	EntityType string // ENUM entity_type
	FieldKey   string
	Label      string
	FieldType  string // ENUM custom_field_type
	Options    json.RawMessage
	IsRequired bool
	SortOrder  int
	CreatedAt  time.Time
}

// Значения ENUM custom_field_type.
const (
	CustomFieldTypeText     = "TEXT"
	CustomFieldTypeNumber   = "NUMBER"
	CustomFieldTypeBoolean  = "BOOLEAN"
	CustomFieldTypeDate     = "DATE"
	CustomFieldTypeDatetime = "DATETIME"
	CustomFieldTypeJSON     = "JSON"
	CustomFieldTypeSelect   = "SELECT"
)

func IsValidCustomFieldType(t string) bool {
	switch t {
	case CustomFieldTypeText, CustomFieldTypeNumber, CustomFieldTypeBoolean,
		CustomFieldTypeDate, CustomFieldTypeDatetime, CustomFieldTypeJSON, CustomFieldTypeSelect:
		return true
	default:
		return false
	}
}

// CustomFieldValue — значение кастом-поля для конкретной сущности. Заполняется
// ровно одна value_*-колонка, соответствующая FieldType определения.
type CustomFieldValue struct {
	ID                uuid.UUID
	EntityID          uuid.UUID
	FieldDefinitionID uuid.UUID
	ValueText         *string
	ValueNumber       *float64
	ValueBoolean      *bool
	ValueDate         *time.Time
	ValueDatetime     *time.Time
	ValueJSON         json.RawMessage
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// CustomFieldDefinitionListFilter — выборка определений кастом-полей.
type CustomFieldDefinitionListFilter struct {
	EntityType string // "" = все типы

	Limit    int
	Offset   int
	SortDesc bool
}
