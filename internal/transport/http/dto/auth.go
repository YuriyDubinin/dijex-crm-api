package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/YuriyDubinin/bms-api/internal/service"
)

// LoginHTTPRequest — тело POST /api/auth/login. Валидация пароля минимальна
// (только required/max), чтобы сообщение об ошибке не подсказывало атакующему
// причину отказа.
type LoginHTTPRequest struct {
	Email    string `json:"email"    validate:"required,email,max=255"`
	Password string `json:"password" validate:"required,max=72"`
}

func (r LoginHTTPRequest) ToServiceInput(ci service.ClientInfo) service.LoginInput {
	return service.LoginInput{
		Email:      r.Email,
		Password:   r.Password,
		ClientInfo: ci,
	}
}

// RegisterHTTPRequest — тело POST /api/auth/register: создаёт тенант + владельца.
type RegisterHTTPRequest struct {
	CompanyName string `json:"company_name" validate:"required,min=2,max=255"`
	FullName    string `json:"full_name"    validate:"required,min=2,max=255"`
	Email       string `json:"email"        validate:"required,email,max=255"`
	Password    string `json:"password"     validate:"required,min=8,max=72"`
}

func (r RegisterHTTPRequest) ToServiceInput(ci service.ClientInfo) service.RegisterInput {
	return service.RegisterInput{
		CompanyName: r.CompanyName,
		FullName:    r.FullName,
		Email:       r.Email,
		Password:    r.Password,
		ClientInfo:  ci,
	}
}

// LoginHTTPResponse — успешный ответ логина. Токен возвращается ОДИН РАЗ.
type LoginHTTPResponse struct {
	Token      string            `json:"token"`
	TokenType  string            `json:"token_type"` // "Bearer"
	ExpiresAt  time.Time         `json:"expires_at"`
	CustomerID uuid.UUID         `json:"customer_id"`
	User       LoginUserResponse `json:"user"`
}

type LoginUserResponse struct {
	ID       uuid.UUID `json:"id"`
	FullName string    `json:"full_name"`
	Email    string    `json:"email"`
	Role     string    `json:"role"`
}

func FromLoginOutput(o *service.LoginOutput) LoginHTTPResponse {
	return LoginHTTPResponse{
		Token:      o.Token,
		TokenType:  o.TokenType,
		ExpiresAt:  o.ExpiresAt,
		CustomerID: o.CustomerID,
		User: LoginUserResponse{
			ID:       o.User.ID,
			FullName: o.User.FullName,
			Email:    o.User.Email,
			Role:     o.User.Role,
		},
	}
}
