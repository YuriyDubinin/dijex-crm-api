package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/YuriyDubinin/bms-api/internal/domain"
)

// ───────────────────────── Service DTO ─────────────────────────

type CustomerItem struct {
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
}

type UpdateCustomerInput struct {
	Name      string
	LegalName string
	Industry  string
	Email     string
	Phone     string
	Timezone  string
	Locale    string
	Settings  json.RawMessage
}

// ───────────────────────── Service ─────────────────────────

type CustomerService struct {
	repo   domain.CustomerRepository
	logger *slog.Logger
}

func NewCustomerService(repo domain.CustomerRepository, logger *slog.Logger) *CustomerService {
	return &CustomerService{repo: repo, logger: logger}
}

func (s *CustomerService) GetCustomer(ctx context.Context, customerID uuid.UUID) (*CustomerItem, error) {
	c, err := s.repo.GetByID(ctx, customerID)
	if err != nil {
		return nil, err
	}
	return customerToItem(c), nil
}

func (s *CustomerService) UpdateCustomer(ctx context.Context, customerID uuid.UUID, actorRole string, in UpdateCustomerInput) (*CustomerItem, error) {
	if !domain.CanManageUsers(actorRole) {
		return nil, domain.ErrForbidden
	}
	in.Name = strings.TrimSpace(in.Name)
	in.Email = normalizeEmail(in.Email)
	in.Timezone = normalizeTimezone(in.Timezone)
	if strings.TrimSpace(in.Locale) == "" {
		in.Locale = "ru"
	}

	var errs domain.ValidationErrors
	if in.Name == "" || utf8.RuneCountInString(in.Name) > 255 {
		errs = append(errs, &domain.ValidationError{Field: "name", Message: "must be between 1 and 255 characters"})
	}
	if in.Email == "" {
		errs = append(errs, &domain.ValidationError{Field: "email", Message: "is required"})
	} else if _, err := mail.ParseAddress(in.Email); err != nil {
		errs = append(errs, &domain.ValidationError{Field: "email", Message: "is not a valid email"})
	}
	if !validAttributes(in.Settings) {
		errs = append(errs, attrError("settings"))
	}
	if len(errs) > 0 {
		return nil, errs
	}

	c := &domain.Customer{
		ID:        customerID,
		Name:      in.Name,
		LegalName: strings.TrimSpace(in.LegalName),
		Industry:  strings.TrimSpace(in.Industry),
		Email:     in.Email,
		Phone:     strings.TrimSpace(in.Phone),
		Timezone:  in.Timezone,
		Locale:    strings.TrimSpace(in.Locale),
		Settings:  in.Settings,
	}
	if err := s.repo.Update(ctx, c); err != nil {
		return nil, err
	}
	s.logger.Info("customer updated", "customer_id", customerID)

	// Дочитываем поля, которые Update не возвращает (plan/status), из БД.
	fresh, err := s.repo.GetByID(ctx, customerID)
	if err != nil {
		return nil, err
	}
	return customerToItem(fresh), nil
}

func customerToItem(c *domain.Customer) *CustomerItem {
	return &CustomerItem{
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
		Settings:           c.Settings,
		CreatedAt:          c.CreatedAt,
		UpdatedAt:          c.UpdatedAt,
	}
}
