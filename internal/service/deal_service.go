package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/YuriyDubinin/bms-api/internal/domain"
)

// ───────────────────────── Service DTO ─────────────────────────

type CreateDealInput struct {
	ProjectID       *uuid.UUID
	ClientID        *uuid.UUID
	ServiceID       *uuid.UUID
	Title           string
	Description     string
	Status          string
	Type            string
	Amount          *float64
	Currency        string
	Probability     *int
	ExpectedCloseAt *time.Time
	AssignedTo      *uuid.UUID
	CreatedBy       uuid.UUID
	Attributes      json.RawMessage
}

type UpdateDealInput struct {
	ID              uuid.UUID
	ProjectID       *uuid.UUID
	ClientID        *uuid.UUID
	ServiceID       *uuid.UUID
	Title           string
	Description     string
	Status          string
	Type            string
	Amount          *float64
	Currency        string
	Probability     *int
	ExpectedCloseAt *time.Time
	AssignedTo      *uuid.UUID
	Attributes      json.RawMessage
}

type ListDealsInput struct {
	Page       int
	PageSize   int
	ProjectID  *uuid.UUID
	ClientID   *uuid.UUID
	ServiceID  *uuid.UUID
	Status     string
	Type       string
	AssignedTo *uuid.UUID
	Search     string
	SortBy     string
	Order      string
}

type DealItem struct {
	ID              uuid.UUID
	ProjectID       *uuid.UUID
	ClientID        *uuid.UUID
	ServiceID       *uuid.UUID
	Title           string
	Description     string
	Status          string
	Type            string
	Amount          *float64
	Currency        string
	Probability     *int
	ExpectedCloseAt *time.Time
	ClosedAt        *time.Time
	AssignedTo      *uuid.UUID
	CreatedBy       *uuid.UUID
	Attributes      json.RawMessage
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type ListDealsOutput struct {
	Items      []DealItem
	Pagination Pagination
}

type DeleteDealOutput struct {
	ID        uuid.UUID
	DeletedAt time.Time
}

// ───────────────────────── Service ─────────────────────────

type DealService struct {
	repo     domain.DealRepository
	projects ProjectChecker
	clients  ClientChecker
	services ServiceChecker
	users    UserChecker
	logger   *slog.Logger
	clock    func() time.Time
}

func NewDealService(repo domain.DealRepository, projects ProjectChecker, clients ClientChecker, services ServiceChecker, users UserChecker, logger *slog.Logger) *DealService {
	return &DealService{repo: repo, projects: projects, clients: clients, services: services, users: users, logger: logger, clock: time.Now}
}

func (s *DealService) CreateDeal(ctx context.Context, customerID uuid.UUID, in CreateDealInput) (*DealItem, error) {
	in.Title = strings.TrimSpace(in.Title)
	in.Description = strings.TrimSpace(in.Description)
	in.Status = normalizeStatus(in.Status, domain.DealStatusNew)
	in.Type = normalizeStatus(in.Type, domain.DealTypeIncome)
	in.Currency = normalizeCurrency(in.Currency)

	errs := validateDeal(in.Title, in.Status, in.Type, in.Currency, in.Amount, in.Probability, in.Attributes)
	errs = append(errs, dealOwnershipErrors(ctx, s, customerID, in.ProjectID, in.ClientID, in.ServiceID, in.AssignedTo)...)
	if len(errs) > 0 {
		return nil, errs
	}

	now := s.clock()
	var closedAt *time.Time
	if domain.IsClosedDealStatus(in.Status) {
		closedAt = &now
	}
	createdBy := in.CreatedBy
	d := &domain.Deal{
		ID:              uuid.New(),
		CustomerID:      customerID,
		ProjectID:       in.ProjectID,
		ClientID:        in.ClientID,
		ServiceID:       in.ServiceID,
		Title:           in.Title,
		Description:     in.Description,
		Status:          in.Status,
		Type:            in.Type,
		Amount:          in.Amount,
		Currency:        in.Currency,
		Probability:     in.Probability,
		ExpectedCloseAt: in.ExpectedCloseAt,
		ClosedAt:        closedAt,
		AssignedTo:      in.AssignedTo,
		CreatedBy:       &createdBy,
		Attributes:      in.Attributes,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := s.repo.Create(ctx, d); err != nil {
		return nil, err
	}
	s.logger.Info("deal created", "deal_id", d.ID, "customer_id", customerID, "status", d.Status)
	return dealToItem(d), nil
}

func (s *DealService) UpdateDeal(ctx context.Context, customerID uuid.UUID, in UpdateDealInput) (*DealItem, error) {
	in.Title = strings.TrimSpace(in.Title)
	in.Description = strings.TrimSpace(in.Description)
	in.Status = normalizeStatus(in.Status, domain.DealStatusNew)
	in.Type = normalizeStatus(in.Type, domain.DealTypeIncome)
	in.Currency = normalizeCurrency(in.Currency)

	errs := validateDeal(in.Title, in.Status, in.Type, in.Currency, in.Amount, in.Probability, in.Attributes)
	errs = append(errs, dealOwnershipErrors(ctx, s, customerID, in.ProjectID, in.ClientID, in.ServiceID, in.AssignedTo)...)
	if len(errs) > 0 {
		return nil, errs
	}

	now := s.clock()
	var closedAt *time.Time
	if domain.IsClosedDealStatus(in.Status) {
		closedAt = &now
	}
	d := &domain.Deal{
		ID:              in.ID,
		CustomerID:      customerID,
		ProjectID:       in.ProjectID,
		ClientID:        in.ClientID,
		ServiceID:       in.ServiceID,
		Title:           in.Title,
		Description:     in.Description,
		Status:          in.Status,
		Type:            in.Type,
		Amount:          in.Amount,
		Currency:        in.Currency,
		Probability:     in.Probability,
		ExpectedCloseAt: in.ExpectedCloseAt,
		ClosedAt:        closedAt,
		AssignedTo:      in.AssignedTo,
		Attributes:      in.Attributes,
	}
	if err := s.repo.Update(ctx, customerID, d); err != nil {
		return nil, err
	}
	s.logger.Info("deal updated", "deal_id", d.ID, "customer_id", customerID)
	return dealToItem(d), nil
}

func (s *DealService) ListDeals(ctx context.Context, customerID uuid.UUID, in ListDealsInput) (*ListDealsOutput, error) {
	status := normalizeStatus(in.Status, "")
	if status != "" && !domain.IsValidDealStatus(status) {
		return nil, domain.ValidationErrors{{Field: "status", Message: "invalid deal status"}}
	}
	dealType := normalizeStatus(in.Type, "")
	if dealType != "" && !domain.IsValidDealType(dealType) {
		return nil, domain.ValidationErrors{{Field: "type", Message: "invalid deal type"}}
	}

	page, size, offset := clampPagination(in.Page, in.PageSize)
	filter := domain.DealListFilter{
		ProjectID:  in.ProjectID,
		ClientID:   in.ClientID,
		ServiceID:  in.ServiceID,
		Status:     status,
		Type:       dealType,
		AssignedTo: in.AssignedTo,
		Search:     strings.TrimSpace(in.Search),
		Limit:      size,
		Offset:     offset,
		SortBy:     in.SortBy,
		SortDesc:   strings.ToLower(in.Order) != "asc",
	}

	items, total, err := s.repo.List(ctx, customerID, filter)
	if err != nil {
		return nil, err
	}
	out := &ListDealsOutput{
		Items:      make([]DealItem, 0, len(items)),
		Pagination: Pagination{Page: page, PageSize: size, Total: total, TotalPages: totalPages(total, size)},
	}
	for _, d := range items {
		out.Items = append(out.Items, *dealToItem(d))
	}
	return out, nil
}

func (s *DealService) DeleteDeal(ctx context.Context, customerID, id uuid.UUID) (*DeleteDealOutput, error) {
	deletedAt, err := s.repo.SoftDelete(ctx, customerID, id)
	if err != nil {
		return nil, err
	}
	s.logger.Info("deal deleted", "deal_id", id, "customer_id", customerID)
	return &DeleteDealOutput{ID: id, DeletedAt: deletedAt}, nil
}

// ───────────────────────── helpers ─────────────────────────

func validateDeal(title, status, dealType, currency string, amount *float64, probability *int, attrs json.RawMessage) domain.ValidationErrors {
	var errs domain.ValidationErrors
	if title == "" {
		errs = append(errs, &domain.ValidationError{Field: "title", Message: "is required"})
	} else if l := utf8.RuneCountInString(title); l < 2 || l > 255 {
		errs = append(errs, &domain.ValidationError{Field: "title", Message: "must be between 2 and 255 characters"})
	}
	if !domain.IsValidDealStatus(status) {
		errs = append(errs, &domain.ValidationError{Field: "status", Message: "must be one of NEW, PENDING, WON, LOST"})
	}
	if !domain.IsValidDealType(dealType) {
		errs = append(errs, &domain.ValidationError{Field: "type", Message: "must be one of INCOME, EXPENSE"})
	}
	if len(currency) != 3 {
		errs = append(errs, &domain.ValidationError{Field: "currency", Message: "must be a 3-letter ISO code"})
	}
	if amount != nil && *amount < 0 {
		errs = append(errs, &domain.ValidationError{Field: "amount", Message: "must be >= 0"})
	}
	if probability != nil && (*probability < 0 || *probability > 100) {
		errs = append(errs, &domain.ValidationError{Field: "probability", Message: "must be between 0 and 100"})
	}
	if !validAttributes(attrs) {
		errs = append(errs, attrError("attributes"))
	}
	return errs
}

// dealOwnershipErrors собирает ошибки принадлежности связанных сущностей тенанту.
func dealOwnershipErrors(ctx context.Context, s *DealService, customerID uuid.UUID, projectID, clientID, serviceID, assignedTo *uuid.UUID) domain.ValidationErrors {
	var errs domain.ValidationErrors
	if ve := ensureProjectOwned(ctx, s.projects, customerID, projectID); ve != nil {
		errs = append(errs, ve)
	}
	if ve := ensureClientOwned(ctx, s.clients, customerID, clientID); ve != nil {
		errs = append(errs, ve)
	}
	if ve := ensureServiceOwned(ctx, s.services, customerID, serviceID); ve != nil {
		errs = append(errs, ve)
	}
	if ve := ensureUserOwned(ctx, s.users, customerID, assignedTo, "assigned_to"); ve != nil {
		errs = append(errs, ve)
	}
	return errs
}

func dealToItem(d *domain.Deal) *DealItem {
	return &DealItem{
		ID:              d.ID,
		ProjectID:       d.ProjectID,
		ClientID:        d.ClientID,
		ServiceID:       d.ServiceID,
		Title:           d.Title,
		Description:     d.Description,
		Status:          d.Status,
		Type:            d.Type,
		Amount:          d.Amount,
		Currency:        d.Currency,
		Probability:     d.Probability,
		ExpectedCloseAt: d.ExpectedCloseAt,
		ClosedAt:        d.ClosedAt,
		AssignedTo:      d.AssignedTo,
		CreatedBy:       d.CreatedBy,
		Attributes:      d.Attributes,
		CreatedAt:       d.CreatedAt,
		UpdatedAt:       d.UpdatedAt,
	}
}
