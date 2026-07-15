package service

import (
	"time"

	"github.com/google/uuid"
)

// ───────────────────────── Авторизация / Логин ─────────────────────────

// ClientInfo — метаданные клиента, извлечённые HTTP-слоем из запроса и
// сохраняемые в auth_tokens как есть. Все поля опциональны (пусто → NULL).
type ClientInfo struct {
	IPAddress      string
	UserAgent      string
	DeviceType     string
	DeviceName     string
	OS             string
	OSVersion      string
	Browser        string
	BrowserVersion string
	AppVersion     string
}

type LoginInput struct {
	Email      string
	Password   string
	ClientInfo ClientInfo
}

// RegisterInput — публичная регистрация: создаёт тенант + владельца.
type RegisterInput struct {
	CompanyName string
	FullName    string
	Email       string
	Password    string
	ClientInfo  ClientInfo
}

// LoginUser — компактный срез оператора для ответа клиенту.
type LoginUser struct {
	ID       uuid.UUID
	FullName string
	Email    string
	Role     string
}

type LoginOutput struct {
	Token      string // сырой opaque-токен; возвращается один раз
	TokenType  string // "Bearer"
	ExpiresAt  time.Time
	CustomerID uuid.UUID
	User       LoginUser
}

// ───────────────────────── Общая пагинация ─────────────────────────

type Pagination struct {
	Page       int
	PageSize   int
	Total      int
	TotalPages int
}

const (
	defaultPageSize = 20
	maxPageSize     = 100
)

// clampPagination нормализует page/pageSize в безопасные пределы и считает
// offset и число страниц-заготовку. total подставляется вызывающим позже.
func clampPagination(page, pageSize int) (normPage, normSize, offset int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return page, pageSize, (page - 1) * pageSize
}

// totalPages — округление вверх.
func totalPages(total, pageSize int) int {
	if total <= 0 || pageSize <= 0 {
		return 0
	}
	return (total + pageSize - 1) / pageSize
}
