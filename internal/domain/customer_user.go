package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// CustomerUser — оператор системы, работающий от лица тенанта (customer).
// Аналог "сотрудника": именно customer_users логинятся и управляют данными
// своего тенанта. PasswordHash — bcrypt (см. pkg/crypto.PasswordHasher).
//
// Email глобально уникален (частичный UNIQUE WHERE deleted_at IS NULL), что
// позволяет логиниться по одному лишь email+паролю: тенант выводится из строки.
type CustomerUser struct {
	ID           uuid.UUID
	CustomerID   uuid.UUID
	Email        string
	FullName     string
	Role         string // ENUM-подобная строка: OWNER | ADMIN | MANAGER | STAFF
	IsActive     bool
	PasswordHash string
	Attributes   json.RawMessage
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
}

// Значения роли оператора.
const (
	UserRoleOwner   = "OWNER"
	UserRoleAdmin   = "ADMIN"
	UserRoleManager = "MANAGER"
	UserRoleStaff   = "STAFF"
)

func IsValidUserRole(r string) bool {
	switch r {
	case UserRoleOwner, UserRoleAdmin, UserRoleManager, UserRoleStaff:
		return true
	default:
		return false
	}
}

// CanManageUsers — право заводить/менять/отключать операторов и сам тенант.
func CanManageUsers(role string) bool {
	return role == UserRoleOwner || role == UserRoleAdmin
}

// CustomerUserListFilter — параметры выборки операторов внутри тенанта.
type CustomerUserListFilter struct {
	Role     string // "" = любая
	IsActive *bool  // nil = любой
	Search   string // ILIKE по full_name/email

	Limit    int
	Offset   int
	SortBy   string
	SortDesc bool
}
