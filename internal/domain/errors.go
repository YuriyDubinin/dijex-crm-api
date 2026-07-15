package domain

import (
	"errors"
	"fmt"
	"strings"
)

// Базовые доменные ошибки. Репозитории оборачивают низкоуровневые ошибки БД
// в эти значения (через fmt.Errorf(..%w..)), а хендлеры маппят их в HTTP-коды
// через errors.Is.
var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
	ErrForbidden     = errors.New("forbidden")
	ErrCrossTenant   = errors.New("cross-tenant access")
	ErrInternal      = errors.New("internal error")
)

// ValidationError — ошибка бизнес-валидации по конкретному полю.
// Собираются в service-слое (normalize+validate) и маппятся хендлером в
// 422 VALIDATION_ERROR с массивом details[{field,message}].
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

type ValidationErrors []*ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return ""
	}
	parts := make([]string, 0, len(e))
	for _, v := range e {
		parts = append(parts, v.Error())
	}
	return strings.Join(parts, "; ")
}
