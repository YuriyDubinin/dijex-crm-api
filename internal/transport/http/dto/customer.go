package dto

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/YuriyDubinin/bms-api/internal/service"
)

type UpdateCustomerHTTPRequest struct {
	Name      string          `json:"name"       validate:"required,min=1,max=255"`
	LegalName string          `json:"legal_name" validate:"omitempty,max=255"`
	Industry  string          `json:"industry"   validate:"omitempty,max=255"`
	Email     string          `json:"email"      validate:"required,email,max=255"`
	Phone     string          `json:"phone"      validate:"omitempty,max=50"`
	Timezone  string          `json:"timezone"   validate:"omitempty,max=100"`
	Locale    string          `json:"locale"     validate:"omitempty,max=10"`
	Settings  json.RawMessage `json:"settings"   validate:"omitempty"`
}

func (r UpdateCustomerHTTPRequest) ToServiceInput() service.UpdateCustomerInput {
	return service.UpdateCustomerInput{
		Name:      r.Name,
		LegalName: r.LegalName,
		Industry:  r.Industry,
		Email:     r.Email,
		Phone:     r.Phone,
		Timezone:  r.Timezone,
		Locale:    r.Locale,
		Settings:  r.Settings,
	}
}

type CustomerHTTPResponse struct {
	ID                 uuid.UUID       `json:"id"`
	Name               string          `json:"name"`
	LegalName          string          `json:"legal_name"`
	Industry           string          `json:"industry"`
	Email              string          `json:"email"`
	Phone              string          `json:"phone"`
	Timezone           string          `json:"timezone"`
	Locale             string          `json:"locale"`
	SubscriptionPlan   string          `json:"subscription_plan"`
	SubscriptionStatus string          `json:"subscription_status"`
	Settings           json.RawMessage `json:"settings"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
}

func FromCustomerItem(c *service.CustomerItem) CustomerHTTPResponse {
	settings := c.Settings
	if len(settings) == 0 {
		settings = json.RawMessage(`{}`)
	}
	return CustomerHTTPResponse{
		ID:                 c.ID,
		Name:               c.Name,
		LegalName:          c.LegalName,
		Industry:           c.Industry,
		Email:              c.Email,
		Phone:              c.Phone,
		Timezone:           c.Timezone,
		Locale:             c.Locale,
		SubscriptionPlan:   c.SubscriptionPlan,
		SubscriptionStatus: c.SubscriptionStatus,
		Settings:           settings,
		CreatedAt:          c.CreatedAt,
		UpdatedAt:          c.UpdatedAt,
	}
}
