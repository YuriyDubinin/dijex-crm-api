package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// AuthToken — авторизационный токен оператора (строка таблицы auth_tokens).
// В БД хранится только TokenHash (SHA-256 от секрета); сам секрет известен
// только клиенту, выдаётся один раз при логине.
//
// Токен привязан к тенанту (CustomerID) и к конкретному пользователю
// (UserID) — оба поля берутся из строки customer_users на момент логина.
type AuthToken struct {
	ID            uuid.UUID
	CustomerID    uuid.UUID
	UserID        uuid.UUID
	TokenHash     string
	TokenType     string // ENUM auth_token_type: ACCESS | REFRESH
	IssuedAt      time.Time
	ExpiresAt     time.Time
	LastUsedAt    *time.Time
	RevokedAt     *time.Time
	RevokedReason string

	// Сеть
	IPAddress string // строка IP (без порта); пусто = NULL в БД
	UserAgent string // сырая строка из заголовка User-Agent

	// Устройство/система клиента — best-effort, парсятся из UA
	DeviceType     string // ENUM auth_device_type: WEB|MOBILE|TABLET|DESKTOP|API|UNKNOWN
	DeviceName     string // клиент может прислать через X-Device-Name
	OS             string
	OSVersion      string
	Browser        string
	BrowserVersion string
	AppVersion     string // клиент может прислать через X-App-Version

	// Геопозиция (задел на будущее, через GeoIP)
	CountryCode string
	City        string
}

// Значения ENUM auth_token_type.
const (
	TokenTypeAccess  = "ACCESS"
	TokenTypeRefresh = "REFRESH"
)

// Значения ENUM auth_device_type.
const (
	DeviceTypeWeb     = "WEB"
	DeviceTypeMobile  = "MOBILE"
	DeviceTypeTablet  = "TABLET"
	DeviceTypeDesktop = "DESKTOP"
	DeviceTypeAPI     = "API"
	DeviceTypeUnknown = "UNKNOWN"
)

// IsActive — токен не отозван и ещё не истёк.
func (t *AuthToken) IsActive(now time.Time) bool {
	if t.RevokedAt != nil {
		return false
	}
	return now.Before(t.ExpiresAt)
}

// AuthTokenWithUser — токен вместе с актуальными данными владельца
// (роль и признак активности), полученными одним JOIN-запросом. Позволяет
// service-слою собрать Principal без повторного похода в БД.
type AuthTokenWithUser struct {
	Token      AuthToken
	UserRole   string
	UserActive bool
}

// Principal — идентификатор аутентифицированной сессии. Кладётся в контекст
// мидлварью Auth и доступен в защищённых хендлерах через PrincipalFromContext.
//
// CustomerID определяет тенант, в рамках которого работает запрос: любой
// доступ к данным скоупится по нему. TokenID нужен для logout/ротации.
type Principal struct {
	CustomerID uuid.UUID
	UserID     uuid.UUID
	Role       string
	TokenID    uuid.UUID
}

// Authenticator — узкий контракт проверки сырого токена из заголовка.
// Реализация — *service.AuthService; router зависит от интерфейса.
type Authenticator interface {
	Authenticate(ctx context.Context, rawToken string) (*Principal, error)
}

// Auth-ошибки. Мидлварь маппит их на 401 с разными кодами в payload, чтобы
// клиент различал причины отказа. ErrInvalidCredentials — единый ответ логина
// («нет email» / «неверный пароль» / «пользователь отключён»), чтобы нельзя
// было определить существование email по ответу/таймингу.
var (
	ErrUnauthenticated    = errors.New("unauthenticated")
	ErrTokenInvalid       = errors.New("token invalid")
	ErrTokenExpired       = errors.New("token expired")
	ErrTokenRevoked       = errors.New("token revoked")
	ErrUserDisabled       = errors.New("user disabled")
	ErrInvalidCredentials = errors.New("invalid credentials")
)
