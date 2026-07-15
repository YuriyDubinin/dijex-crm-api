package domain

// EntityType — тип записи в общем реестре entities (супертайп). Значения
// совпадают с ENUM entity_type в БД. Любая "линкуемая" сущность регистрируется
// в entities автоматически (BEFORE INSERT триггером), что даёт единый механизм
// связей (entity_links), тегов и кастом-полей поверх всех сущностей.
type EntityType string

const (
	EntityTypeProject       EntityType = "PROJECT"
	EntityTypeService       EntityType = "SERVICE"
	EntityTypeClient        EntityType = "CLIENT"
	EntityTypeTask          EntityType = "TASK"
	EntityTypeCalendarEvent EntityType = "CALENDAR_EVENT"
	EntityTypeDeal          EntityType = "DEAL"
	EntityTypeCustomerUser  EntityType = "CUSTOMER_USER"
)

func IsValidEntityType(t string) bool {
	switch EntityType(t) {
	case EntityTypeProject, EntityTypeService, EntityTypeClient,
		EntityTypeTask, EntityTypeCalendarEvent, EntityTypeDeal, EntityTypeCustomerUser:
		return true
	default:
		return false
	}
}
