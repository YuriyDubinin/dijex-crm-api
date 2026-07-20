package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/YuriyDubinin/bms-api/internal/domain"
)

// ───────────────────────── Client DTO ─────────────────────────

type CreateClientInput struct {
	ProjectID   *uuid.UUID
	FirstName   string
	LastName    string
	CompanyName string
	Email       string
	Phone       string
	Status      string
	SubjectType string
	Source      string
	Address     json.RawMessage
	Attributes  json.RawMessage
}

type UpdateClientInput struct {
	ID          uuid.UUID
	ProjectID   *uuid.UUID
	FirstName   string
	LastName    string
	CompanyName string
	Email       string
	Phone       string
	Status      string
	SubjectType string
	Source      string
	Address     json.RawMessage
	Attributes  json.RawMessage
}

type ListClientsInput struct {
	Page        int
	PageSize    int
	ProjectID   *uuid.UUID
	Status      string
	SubjectType string
	Search      string
	SortBy      string
	Order       string
}

type ClientItem struct {
	ID          uuid.UUID
	ProjectID   *uuid.UUID
	FirstName   string
	LastName    string
	CompanyName string
	Email       string
	Phone       string
	Status      string
	SubjectType string
	Source      string
	Address     json.RawMessage
	Attributes  json.RawMessage
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ListClientsOutput struct {
	Items      []ClientItem
	Pagination Pagination
}

type DeleteClientOutput struct {
	ID        uuid.UUID
	DeletedAt time.Time
}

// ───────────────────────── Client ─────────────────────────

type ClientService struct {
	repo     domain.ClientRepository
	projects ProjectChecker
	logger   *slog.Logger
	clock    func() time.Time
}

func NewClientService(repo domain.ClientRepository, projects ProjectChecker, logger *slog.Logger) *ClientService {
	return &ClientService{repo: repo, projects: projects, logger: logger, clock: time.Now}
}

func (s *ClientService) CreateClient(ctx context.Context, customerID uuid.UUID, in CreateClientInput) (*ClientItem, error) {
	in.FirstName = strings.TrimSpace(in.FirstName)
	in.LastName = strings.TrimSpace(in.LastName)
	in.CompanyName = strings.TrimSpace(in.CompanyName)
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	in.Phone = strings.TrimSpace(in.Phone)
	in.Source = strings.TrimSpace(in.Source)
	in.Status = normalizeStatus(in.Status, domain.ClientStatusLead)
	in.SubjectType = normalizeStatus(in.SubjectType, domain.ClientSubjectTypeIndividual)

	errs := validateClient(in.FirstName, in.LastName, in.CompanyName, in.Email, in.Status, in.SubjectType, in.Address, in.Attributes)
	if ve := ensureProjectOwned(ctx, s.projects, customerID, in.ProjectID); ve != nil {
		errs = append(errs, ve)
	}
	if len(errs) > 0 {
		return nil, errs
	}

	now := s.clock()
	client := &domain.Client{
		ID:          uuid.New(),
		CustomerID:  customerID,
		ProjectID:   in.ProjectID,
		FirstName:   in.FirstName,
		LastName:    in.LastName,
		CompanyName: in.CompanyName,
		Email:       in.Email,
		Phone:       in.Phone,
		Address:     in.Address,
		Status:      in.Status,
		SubjectType: in.SubjectType,
		Source:      in.Source,
		Attributes:  in.Attributes,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.repo.Create(ctx, client); err != nil {
		return nil, err
	}
	s.logger.Info("client created", "client_id", client.ID, "customer_id", customerID)
	return clientToItem(client), nil
}

func (s *ClientService) UpdateClient(ctx context.Context, customerID uuid.UUID, in UpdateClientInput) (*ClientItem, error) {
	in.FirstName = strings.TrimSpace(in.FirstName)
	in.LastName = strings.TrimSpace(in.LastName)
	in.CompanyName = strings.TrimSpace(in.CompanyName)
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	in.Phone = strings.TrimSpace(in.Phone)
	in.Source = strings.TrimSpace(in.Source)
	in.Status = normalizeStatus(in.Status, domain.ClientStatusLead)
	in.SubjectType = normalizeStatus(in.SubjectType, domain.ClientSubjectTypeIndividual)

	errs := validateClient(in.FirstName, in.LastName, in.CompanyName, in.Email, in.Status, in.SubjectType, in.Address, in.Attributes)
	if ve := ensureProjectOwned(ctx, s.projects, customerID, in.ProjectID); ve != nil {
		errs = append(errs, ve)
	}
	if len(errs) > 0 {
		return nil, errs
	}

	client := &domain.Client{
		ID:          in.ID,
		CustomerID:  customerID,
		ProjectID:   in.ProjectID,
		FirstName:   in.FirstName,
		LastName:    in.LastName,
		CompanyName: in.CompanyName,
		Email:       in.Email,
		Phone:       in.Phone,
		Address:     in.Address,
		Status:      in.Status,
		SubjectType: in.SubjectType,
		Source:      in.Source,
		Attributes:  in.Attributes,
	}
	if err := s.repo.Update(ctx, customerID, client); err != nil {
		return nil, err
	}
	s.logger.Info("client updated", "client_id", client.ID, "customer_id", customerID)
	return clientToItem(client), nil
}

func (s *ClientService) ListClients(ctx context.Context, customerID uuid.UUID, in ListClientsInput) (*ListClientsOutput, error) {
	status := normalizeStatus(in.Status, "")
	if status != "" && !domain.IsValidClientStatus(status) {
		return nil, domain.ValidationErrors{{Field: "status", Message: "invalid client status"}}
	}
	subjectType := normalizeStatus(in.SubjectType, "")
	if subjectType != "" && !domain.IsValidClientSubjectType(subjectType) {
		return nil, domain.ValidationErrors{{Field: "subject_type", Message: "invalid client subject type"}}
	}

	page, size, offset := clampPagination(in.Page, in.PageSize)
	filter := domain.ClientListFilter{
		ProjectID:   in.ProjectID,
		Status:      status,
		SubjectType: subjectType,
		Search:      strings.TrimSpace(in.Search),
		Limit:       size,
		Offset:      offset,
		SortBy:      in.SortBy,
		SortDesc:    strings.ToLower(in.Order) != "asc",
	}

	items, total, err := s.repo.List(ctx, customerID, filter)
	if err != nil {
		return nil, err
	}
	out := &ListClientsOutput{
		Items:      make([]ClientItem, 0, len(items)),
		Pagination: Pagination{Page: page, PageSize: size, Total: total, TotalPages: totalPages(total, size)},
	}
	for _, client := range items {
		out.Items = append(out.Items, *clientToItem(client))
	}
	return out, nil
}

func (s *ClientService) DeleteClient(ctx context.Context, customerID, id uuid.UUID) (*DeleteClientOutput, error) {
	deletedAt, err := s.repo.SoftDelete(ctx, customerID, id)
	if err != nil {
		return nil, err
	}
	s.logger.Info("client deleted", "client_id", id, "customer_id", customerID)
	return &DeleteClientOutput{ID: id, DeletedAt: deletedAt}, nil
}

// ───────────────────────── helpers ─────────────────────────

func validateClient(firstName, lastName, companyName, email, status, subjectType string, address, attrs json.RawMessage) domain.ValidationErrors {
	var errs domain.ValidationErrors
	if firstName == "" && lastName == "" && companyName == "" {
		errs = append(errs, &domain.ValidationError{Field: "name", Message: "at least one of first_name, last_name, company_name is required"})
	}
	if email != "" {
		if _, err := mail.ParseAddress(email); err != nil {
			errs = append(errs, &domain.ValidationError{Field: "email", Message: "must be a valid email address"})
		}
	}
	if !domain.IsValidClientStatus(status) {
		errs = append(errs, &domain.ValidationError{Field: "status", Message: "must be one of LEAD, ACTIVE, INACTIVE, ARCHIVED"})
	}
	if !domain.IsValidClientSubjectType(subjectType) {
		errs = append(errs, &domain.ValidationError{Field: "subject_type", Message: "must be one of INDIVIDUAL, LEGAL_ENTITY"})
	}
	if !validAttributes(address) {
		errs = append(errs, attrError("address"))
	}
	if !validAttributes(attrs) {
		errs = append(errs, attrError("attributes"))
	}
	return errs
}

func clientToItem(c *domain.Client) *ClientItem {
	return &ClientItem{
		ID:          c.ID,
		ProjectID:   c.ProjectID,
		FirstName:   c.FirstName,
		LastName:    c.LastName,
		CompanyName: c.CompanyName,
		Email:       c.Email,
		Phone:       c.Phone,
		Status:      c.Status,
		SubjectType: c.SubjectType,
		Source:      c.Source,
		Address:     c.Address,
		Attributes:  c.Attributes,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
	}
}
