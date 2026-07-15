package service

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"

	"github.com/YuriyDubinin/bms-api/internal/domain"
)

// validAttributes — атрибуты либо пусты, либо валидный JSON.
func validAttributes(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return true
	}
	return json.Valid(raw)
}

// attrError — стандартная ошибка валидации поля attributes.
func attrError(field string) *domain.ValidationError {
	return &domain.ValidationError{Field: field, Message: "must be a valid JSON object"}
}

// ───────────────────── Проверки принадлежности тенанту ─────────────────────
// Ссылки на другие сущности (project_id, assigned_to) обязаны указывать на
// запись ТОГО ЖЕ тенанта. FK гарантирует лишь существование строки, но не
// тенанта, поэтому проверяем явно в сервисе через узкие контракты-checker'ы.

// ProjectChecker — узкий контракт: реализуется *postgres.ProjectRepository.
type ProjectChecker interface {
	GetByID(ctx context.Context, customerID, id uuid.UUID) (*domain.Project, error)
}

// UserChecker — узкий контракт: реализуется *postgres.CustomerUserRepository.
type UserChecker interface {
	GetByID(ctx context.Context, customerID, id uuid.UUID) (*domain.CustomerUser, error)
}

// ClientChecker — узкий контракт: реализуется *postgres.ClientRepository.
type ClientChecker interface {
	GetByID(ctx context.Context, customerID, id uuid.UUID) (*domain.Client, error)
}

// DealChecker — узкий контракт: реализуется *postgres.DealRepository.
type DealChecker interface {
	GetByID(ctx context.Context, customerID, id uuid.UUID) (*domain.Deal, error)
}

// ServiceChecker — узкий контракт: реализуется *postgres.ServiceRepository.
type ServiceChecker interface {
	GetByID(ctx context.Context, customerID, id uuid.UUID) (*domain.Service, error)
}

// EntityResolver — узкий контракт: реализуется *postgres.EntityRepository.
type EntityResolver interface {
	ResolveType(ctx context.Context, customerID, entityID uuid.UUID) (domain.EntityType, error)
}

// ensureProjectOwned проверяет, что projectID (если задан) принадлежит тенанту.
// Возвращает ValidationError по полю project_id, если запись не найдена.
func ensureProjectOwned(ctx context.Context, checker ProjectChecker, customerID uuid.UUID, projectID *uuid.UUID) *domain.ValidationError {
	if projectID == nil {
		return nil
	}
	if _, err := checker.GetByID(ctx, customerID, *projectID); err != nil {
		return &domain.ValidationError{Field: "project_id", Message: "project not found in this tenant"}
	}
	return nil
}

// ensureUserOwned проверяет, что userID (если задан) принадлежит тенанту.
func ensureUserOwned(ctx context.Context, checker UserChecker, customerID uuid.UUID, userID *uuid.UUID, field string) *domain.ValidationError {
	if userID == nil {
		return nil
	}
	if _, err := checker.GetByID(ctx, customerID, *userID); err != nil {
		return &domain.ValidationError{Field: field, Message: "user not found in this tenant"}
	}
	return nil
}

// ensureClientOwned проверяет, что clientID (если задан) принадлежит тенанту.
func ensureClientOwned(ctx context.Context, checker ClientChecker, customerID uuid.UUID, clientID *uuid.UUID) *domain.ValidationError {
	if clientID == nil {
		return nil
	}
	if _, err := checker.GetByID(ctx, customerID, *clientID); err != nil {
		return &domain.ValidationError{Field: "client_id", Message: "client not found in this tenant"}
	}
	return nil
}

// ensureDealOwned проверяет, что dealID (если задан) принадлежит тенанту.
func ensureDealOwned(ctx context.Context, checker DealChecker, customerID uuid.UUID, dealID *uuid.UUID) *domain.ValidationError {
	if dealID == nil {
		return nil
	}
	if _, err := checker.GetByID(ctx, customerID, *dealID); err != nil {
		return &domain.ValidationError{Field: "deal_id", Message: "deal not found in this tenant"}
	}
	return nil
}

// ensureServiceOwned проверяет, что serviceID (если задан) принадлежит тенанту.
func ensureServiceOwned(ctx context.Context, checker ServiceChecker, customerID uuid.UUID, serviceID *uuid.UUID) *domain.ValidationError {
	if serviceID == nil {
		return nil
	}
	if _, err := checker.GetByID(ctx, customerID, *serviceID); err != nil {
		return &domain.ValidationError{Field: "service_id", Message: "service not found in this tenant"}
	}
	return nil
}
