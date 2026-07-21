package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Все контракты бизнес-репозиториев принимают customerID и скоупят по нему
// каждый SELECT/UPDATE/DELETE. Доступ к чужому тенанту неотличим от отсутствия
// записи — методы возвращают ErrNotFound (не палим существование).

// ─────────────────────────────── Auth ───────────────────────────────

// TokenRepository — хранение и поиск авторизационных токенов.
type TokenRepository interface {
	Create(ctx context.Context, t *AuthToken) error
	// FindActiveByHash находит токен по SHA-256 хэшу и возвращает его вместе с
	// ролью/активностью владельца (JOIN customer_users). ErrNotFound, если нет.
	FindActiveByHash(ctx context.Context, hash string) (*AuthTokenWithUser, error)
	TouchLastUsed(ctx context.Context, id uuid.UUID, now time.Time) error
	Revoke(ctx context.Context, id uuid.UUID, reason string, now time.Time) error
}

// CustomerUserRepository — операторы тенанта. FindByEmail — глобальный
// (email уникален), остальное скоупится по customerID.
type CustomerUserRepository interface {
	FindByEmail(ctx context.Context, email string) (*CustomerUser, error)
	Create(ctx context.Context, u *CustomerUser) error
	List(ctx context.Context, customerID uuid.UUID, f CustomerUserListFilter) ([]*CustomerUser, int, error)
	GetByID(ctx context.Context, customerID, id uuid.UUID) (*CustomerUser, error)
	Update(ctx context.Context, customerID uuid.UUID, u *CustomerUser) error
	SoftDelete(ctx context.Context, customerID, id uuid.UUID) (time.Time, error)
	// CountOwners возвращает число активных владельцев тенанта (для запрета
	// удаления/понижения последнего OWNER).
	CountOwners(ctx context.Context, customerID uuid.UUID) (int, error)
}

// CustomerRepository — сам тенант. RegisterTenant атомарно создаёт customer и
// первого пользователя-владельца (OWNER) при публичной регистрации.
type CustomerRepository interface {
	RegisterTenant(ctx context.Context, c *Customer, owner *CustomerUser) error
	GetByID(ctx context.Context, id uuid.UUID) (*Customer, error)
	Update(ctx context.Context, c *Customer) error
	SoftDelete(ctx context.Context, id uuid.UUID) (time.Time, error)
}

// ─────────────────────────── Бизнес-сущности ───────────────────────────

type ProjectRepository interface {
	Create(ctx context.Context, p *Project) error
	List(ctx context.Context, customerID uuid.UUID, f ProjectListFilter) ([]*Project, int, error)
	GetByID(ctx context.Context, customerID, id uuid.UUID) (*Project, error)
	Update(ctx context.Context, customerID uuid.UUID, p *Project) error
	SoftDelete(ctx context.Context, customerID, id uuid.UUID) (time.Time, error)
}

type ServiceRepository interface {
	Create(ctx context.Context, s *Service) error
	List(ctx context.Context, customerID uuid.UUID, f ServiceListFilter) ([]*Service, int, error)
	GetByID(ctx context.Context, customerID, id uuid.UUID) (*Service, error)
	Update(ctx context.Context, customerID uuid.UUID, s *Service) error
	SoftDelete(ctx context.Context, customerID, id uuid.UUID) (time.Time, error)
}

type ClientRepository interface {
	Create(ctx context.Context, c *Client) error
	List(ctx context.Context, customerID uuid.UUID, f ClientListFilter) ([]*Client, int, error)
	GetByID(ctx context.Context, customerID, id uuid.UUID) (*Client, error)
	Update(ctx context.Context, customerID uuid.UUID, c *Client) error
	SoftDelete(ctx context.Context, customerID, id uuid.UUID) (time.Time, error)
	// Членство клиента в проектах (M:N, поверх «основного» project_id).
	AttachProject(ctx context.Context, customerID, clientID, projectID uuid.UUID) error
	DetachProject(ctx context.Context, customerID, clientID, projectID uuid.UUID) error
	// ListProjects — все проекты клиента: «основной» project_id + членства.
	ListProjects(ctx context.Context, customerID, clientID uuid.UUID) ([]*Project, error)
}

type TaskRepository interface {
	Create(ctx context.Context, t *Task) error
	List(ctx context.Context, customerID uuid.UUID, f TaskListFilter) ([]*Task, int, error)
	GetByID(ctx context.Context, customerID, id uuid.UUID) (*Task, error)
	Update(ctx context.Context, customerID uuid.UUID, t *Task) error
	SoftDelete(ctx context.Context, customerID, id uuid.UUID) (time.Time, error)
}

type DealRepository interface {
	Create(ctx context.Context, d *Deal) error
	List(ctx context.Context, customerID uuid.UUID, f DealListFilter) ([]*Deal, int, error)
	GetByID(ctx context.Context, customerID, id uuid.UUID) (*Deal, error)
	Update(ctx context.Context, customerID uuid.UUID, d *Deal) error
	SoftDelete(ctx context.Context, customerID, id uuid.UUID) (time.Time, error)
}

// ProcessRepository — процессы и их пользовательские этапы (process_stages).
// Методы этапов живут здесь же (этап — дочерняя сущность процесса).
type ProcessRepository interface {
	Create(ctx context.Context, p *Process) error
	List(ctx context.Context, customerID uuid.UUID, f ProcessListFilter) ([]*Process, int, error)
	GetByID(ctx context.Context, customerID, id uuid.UUID) (*Process, error)
	Update(ctx context.Context, customerID uuid.UUID, p *Process) error
	SoftDelete(ctx context.Context, customerID, id uuid.UUID) (time.Time, error)
	// Этапы процесса.
	CreateStage(ctx context.Context, s *ProcessStage) error
	ListStages(ctx context.Context, customerID, processID uuid.UUID) ([]*ProcessStage, error)
	GetStageByID(ctx context.Context, customerID, id uuid.UUID) (*ProcessStage, error)
	UpdateStage(ctx context.Context, customerID uuid.UUID, s *ProcessStage) error
	SoftDeleteStage(ctx context.Context, customerID, id uuid.UUID) (time.Time, error)
}

// ─────────────────────────── Кросс-сущностные ───────────────────────────

// EntityRepository — доступ к общему реестру entities (для проверок
// принадлежности/типа при построении связей, тегов, кастом-полей).
type EntityRepository interface {
	// ResolveType возвращает тип сущности, если она существует и принадлежит
	// тенанту. Иначе ErrNotFound.
	ResolveType(ctx context.Context, customerID, entityID uuid.UUID) (EntityType, error)
}

type EntityLinkRepository interface {
	Create(ctx context.Context, l *EntityLink) error
	List(ctx context.Context, customerID uuid.UUID, f EntityLinkListFilter) ([]*LinkedEntity, int, error)
	Delete(ctx context.Context, customerID, id uuid.UUID) error
}

type TagRepository interface {
	Create(ctx context.Context, t *Tag) error
	List(ctx context.Context, customerID uuid.UUID, f TagListFilter) ([]*Tag, int, error)
	GetByID(ctx context.Context, customerID, id uuid.UUID) (*Tag, error)
	Update(ctx context.Context, customerID uuid.UUID, t *Tag) error
	Delete(ctx context.Context, customerID, id uuid.UUID) error
	Attach(ctx context.Context, customerID, tagID, entityID uuid.UUID) error
	Detach(ctx context.Context, customerID, tagID, entityID uuid.UUID) error
}

// StatsRepository — агрегатные (read-only) выборки для дашборда. Все методы
// скоупятся по customerID и учитывают soft-delete; StatsFilter ограничивает
// период по created_at сущности.
type StatsRepository interface {
	Summary(ctx context.Context, customerID uuid.UUID, f StatsFilter) (*SummaryStats, error)
	DealStats(ctx context.Context, customerID uuid.UUID, f StatsFilter) (*DealStats, error)
	ClientStats(ctx context.Context, customerID uuid.UUID, f StatsFilter) (*ClientStats, error)
	TaskStats(ctx context.Context, customerID uuid.UUID, f StatsFilter) (*TaskStats, error)
	ProjectStats(ctx context.Context, customerID uuid.UUID, f StatsFilter) (*ProjectStats, error)
	ServiceStats(ctx context.Context, customerID uuid.UUID, f StatsFilter) (*ServiceStats, error)
	ProcessStats(ctx context.Context, customerID uuid.UUID, f StatsFilter) (*ProcessStats, error)
	// TimeSeries — временной ряд по метрике. metric/interval уже провалидированы
	// сервисом (whitelist); currency — опциональный фильтр валюты для денежных
	// метрик ("" = все валюты).
	TimeSeries(ctx context.Context, customerID uuid.UUID, metric, interval, currency string, f StatsFilter) ([]TimePoint, error)
}

type CustomFieldRepository interface {
	CreateDefinition(ctx context.Context, d *CustomFieldDefinition) error
	ListDefinitions(ctx context.Context, customerID uuid.UUID, f CustomFieldDefinitionListFilter) ([]*CustomFieldDefinition, int, error)
	GetDefinitionByID(ctx context.Context, customerID, id uuid.UUID) (*CustomFieldDefinition, error)
	UpdateDefinition(ctx context.Context, customerID uuid.UUID, d *CustomFieldDefinition) error
	DeleteDefinition(ctx context.Context, customerID, id uuid.UUID) error
	// SetValue делает upsert значения (ON CONFLICT entity_id+field_definition_id).
	SetValue(ctx context.Context, customerID uuid.UUID, v *CustomFieldValue) error
	ListValues(ctx context.Context, customerID, entityID uuid.UUID) ([]*CustomFieldValue, error)
	UnsetValue(ctx context.Context, customerID, entityID, fieldDefinitionID uuid.UUID) error
}
