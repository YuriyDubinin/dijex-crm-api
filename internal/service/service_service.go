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

type CreateServiceInput struct {
	ProjectID   *uuid.UUID
	Name        string
	Category    string
	Description string
	Price       *float64
	Currency    string
	DurationMin *int
	Status      string
	Attributes  json.RawMessage
}

type UpdateServiceInput struct {
	ID          uuid.UUID
	ProjectID   *uuid.UUID
	Name        string
	Category    string
	Description string
	Price       *float64
	Currency    string
	DurationMin *int
	Status      string
	Attributes  json.RawMessage
}

type ListServicesInput struct {
	Page      int
	PageSize  int
	ProjectID *uuid.UUID
	Status    string
	Category  string
	Search    string
	SortBy    string
	Order     string
}

type ServiceItem struct {
	ID          uuid.UUID
	ProjectID   *uuid.UUID
	Name        string
	Category    string
	Description string
	Price       *float64
	Currency    string
	DurationMin *int
	Status      string
	Attributes  json.RawMessage
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ListServicesOutput struct {
	Items      []ServiceItem
	Pagination Pagination
}

type DeleteServiceOutput struct {
	ID        uuid.UUID
	DeletedAt time.Time
}

// ───────────────────────── Service ─────────────────────────

type ServiceService struct {
	repo     domain.ServiceRepository
	projects ProjectChecker
	logger   *slog.Logger
	clock    func() time.Time
}

func NewServiceService(repo domain.ServiceRepository, projects ProjectChecker, logger *slog.Logger) *ServiceService {
	return &ServiceService{repo: repo, projects: projects, logger: logger, clock: time.Now}
}

func (s *ServiceService) CreateService(ctx context.Context, customerID uuid.UUID, in CreateServiceInput) (*ServiceItem, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Category = strings.TrimSpace(in.Category)
	in.Description = strings.TrimSpace(in.Description)
	in.Currency = normalizeCurrency(in.Currency)
	in.Status = normalizeStatus(in.Status, domain.ServiceStatusActive)

	errs := validateService(in.Name, in.Currency, in.Status, in.Price, in.Attributes)
	if ve := ensureProjectOwned(ctx, s.projects, customerID, in.ProjectID); ve != nil {
		errs = append(errs, ve)
	}
	if len(errs) > 0 {
		return nil, errs
	}

	now := s.clock()
	svc := &domain.Service{
		ID:          uuid.New(),
		CustomerID:  customerID,
		ProjectID:   in.ProjectID,
		Name:        in.Name,
		Category:    in.Category,
		Description: in.Description,
		Price:       in.Price,
		Currency:    in.Currency,
		DurationMin: in.DurationMin,
		Status:      in.Status,
		Attributes:  in.Attributes,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.repo.Create(ctx, svc); err != nil {
		return nil, err
	}
	s.logger.Info("service created", "service_id", svc.ID, "customer_id", customerID)
	return serviceToItem(svc), nil
}

func (s *ServiceService) UpdateService(ctx context.Context, customerID uuid.UUID, in UpdateServiceInput) (*ServiceItem, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Category = strings.TrimSpace(in.Category)
	in.Description = strings.TrimSpace(in.Description)
	in.Currency = normalizeCurrency(in.Currency)
	in.Status = normalizeStatus(in.Status, domain.ServiceStatusActive)

	errs := validateService(in.Name, in.Currency, in.Status, in.Price, in.Attributes)
	if ve := ensureProjectOwned(ctx, s.projects, customerID, in.ProjectID); ve != nil {
		errs = append(errs, ve)
	}
	if len(errs) > 0 {
		return nil, errs
	}

	svc := &domain.Service{
		ID:          in.ID,
		CustomerID:  customerID,
		ProjectID:   in.ProjectID,
		Name:        in.Name,
		Category:    in.Category,
		Description: in.Description,
		Price:       in.Price,
		Currency:    in.Currency,
		DurationMin: in.DurationMin,
		Status:      in.Status,
		Attributes:  in.Attributes,
	}
	if err := s.repo.Update(ctx, customerID, svc); err != nil {
		return nil, err
	}
	s.logger.Info("service updated", "service_id", svc.ID, "customer_id", customerID)
	return serviceToItem(svc), nil
}

func (s *ServiceService) ListServices(ctx context.Context, customerID uuid.UUID, in ListServicesInput) (*ListServicesOutput, error) {
	status := normalizeStatus(in.Status, "")
	if status != "" && !domain.IsValidServiceStatus(status) {
		return nil, domain.ValidationErrors{{Field: "status", Message: "invalid service status"}}
	}

	page, size, offset := clampPagination(in.Page, in.PageSize)
	filter := domain.ServiceListFilter{
		ProjectID: in.ProjectID,
		Status:    status,
		Category:  strings.TrimSpace(in.Category),
		Search:    strings.TrimSpace(in.Search),
		Limit:     size,
		Offset:    offset,
		SortBy:    in.SortBy,
		SortDesc:  strings.ToLower(in.Order) != "asc",
	}

	items, total, err := s.repo.List(ctx, customerID, filter)
	if err != nil {
		return nil, err
	}
	out := &ListServicesOutput{
		Items:      make([]ServiceItem, 0, len(items)),
		Pagination: Pagination{Page: page, PageSize: size, Total: total, TotalPages: totalPages(total, size)},
	}
	for _, svc := range items {
		out.Items = append(out.Items, *serviceToItem(svc))
	}
	return out, nil
}

func (s *ServiceService) DeleteService(ctx context.Context, customerID, id uuid.UUID) (*DeleteServiceOutput, error) {
	deletedAt, err := s.repo.SoftDelete(ctx, customerID, id)
	if err != nil {
		return nil, err
	}
	s.logger.Info("service deleted", "service_id", id, "customer_id", customerID)
	return &DeleteServiceOutput{ID: id, DeletedAt: deletedAt}, nil
}

// ───────────────────────── helpers ─────────────────────────

func validateService(name, currency, status string, price *float64, attrs json.RawMessage) domain.ValidationErrors {
	var errs domain.ValidationErrors
	if name == "" {
		errs = append(errs, &domain.ValidationError{Field: "name", Message: "is required"})
	} else if l := utf8.RuneCountInString(name); l < 2 || l > 255 {
		errs = append(errs, &domain.ValidationError{Field: "name", Message: "must be between 2 and 255 characters"})
	}
	if len(currency) != 3 {
		errs = append(errs, &domain.ValidationError{Field: "currency", Message: "must be a 3-letter ISO code"})
	}
	if price != nil && *price < 0 {
		errs = append(errs, &domain.ValidationError{Field: "price", Message: "must be >= 0"})
	}
	if !domain.IsValidServiceStatus(status) {
		errs = append(errs, &domain.ValidationError{Field: "status", Message: "must be one of ACTIVE, INACTIVE, ARCHIVED"})
	}
	if !validAttributes(attrs) {
		errs = append(errs, attrError("attributes"))
	}
	return errs
}

func serviceToItem(s *domain.Service) *ServiceItem {
	return &ServiceItem{
		ID:          s.ID,
		ProjectID:   s.ProjectID,
		Name:        s.Name,
		Category:    s.Category,
		Description: s.Description,
		Price:       s.Price,
		Currency:    s.Currency,
		DurationMin: s.DurationMin,
		Status:      s.Status,
		Attributes:  s.Attributes,
		CreatedAt:   s.CreatedAt,
		UpdatedAt:   s.UpdatedAt,
	}
}

func normalizeCurrency(c string) string {
	c = strings.ToUpper(strings.TrimSpace(c))
	if c == "" {
		return "USD"
	}
	return c
}
