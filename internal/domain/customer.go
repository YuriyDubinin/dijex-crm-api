package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Customer — тенант (клиент, купивший BMS). Верхнеуровневый владелец всех
// остальных сущностей: projects, services, clients, tasks и т.д.
type Customer struct {
	ID                 uuid.UUID
	Name               string
	LegalName          string
	Industry           string
	Email              string
	Phone              string
	Timezone           string
	Locale             string
	SubscriptionPlan   string
	SubscriptionStatus string
	Settings           json.RawMessage
	CreatedAt          time.Time
	UpdatedAt          time.Time
	DeletedAt          *time.Time
}
