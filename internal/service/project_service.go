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

type CreateProjectInput struct {
	Name        string
	Slug        string
	Direction   string
	Description string
	Status      string
	Attributes  json.RawMessage
	StartsAt    *time.Time
	EndsAt      *time.Time
}

type UpdateProjectInput struct {
	ID          uuid.UUID
	Name        string
	Slug        string
	Direction   string
	Description string
	Status      string
	Attributes  json.RawMessage
	StartsAt    *time.Time
	EndsAt      *time.Time
}

type ListProjectsInput struct {
	Page     int
	PageSize int
	Status   string
	Search   string
	From     *time.Time
	To       *time.Time
	SortBy   string
	Order    string // "asc" | "desc"
}

type ProjectItem struct {
	ID          uuid.UUID
	Name        string
	Slug        string
	Direction   string
	Description string
	Status      string
	Attributes  json.RawMessage
	StartsAt    *time.Time
	EndsAt      *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ListProjectsOutput struct {
	Items      []ProjectItem
	Pagination Pagination
}

type DeleteProjectOutput struct {
	ID        uuid.UUID
	DeletedAt time.Time
}

// ───────────────────────── Service ─────────────────────────

type ProjectService struct {
	repo   domain.ProjectRepository
	logger *slog.Logger
	clock  func() time.Time
}

func NewProjectService(repo domain.ProjectRepository, logger *slog.Logger) *ProjectService {
	return &ProjectService{repo: repo, logger: logger, clock: time.Now}
}

func (s *ProjectService) CreateProject(ctx context.Context, customerID uuid.UUID, in CreateProjectInput) (*ProjectItem, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Slug = strings.TrimSpace(in.Slug)
	in.Direction = strings.TrimSpace(in.Direction)
	in.Description = strings.TrimSpace(in.Description)
	in.Status = normalizeStatus(in.Status, domain.ProjectStatusActive)

	if errs := validateProject(in.Name, in.Slug, in.Status, in.Attributes); len(errs) > 0 {
		return nil, errs
	}

	now := s.clock()
	p := &domain.Project{
		ID:          uuid.New(),
		CustomerID:  customerID,
		Name:        in.Name,
		Slug:        in.Slug,
		Direction:   in.Direction,
		Description: in.Description,
		Status:      in.Status,
		Attributes:  in.Attributes,
		StartsAt:    in.StartsAt,
		EndsAt:      in.EndsAt,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.repo.Create(ctx, p); err != nil {
		return nil, err
	}
	s.logger.Info("project created", "project_id", p.ID, "customer_id", customerID)
	return projectToItem(p), nil
}

func (s *ProjectService) UpdateProject(ctx context.Context, customerID uuid.UUID, in UpdateProjectInput) (*ProjectItem, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Slug = strings.TrimSpace(in.Slug)
	in.Direction = strings.TrimSpace(in.Direction)
	in.Description = strings.TrimSpace(in.Description)
	in.Status = normalizeStatus(in.Status, domain.ProjectStatusActive)

	if errs := validateProject(in.Name, in.Slug, in.Status, in.Attributes); len(errs) > 0 {
		return nil, errs
	}

	p := &domain.Project{
		ID:          in.ID,
		CustomerID:  customerID,
		Name:        in.Name,
		Slug:        in.Slug,
		Direction:   in.Direction,
		Description: in.Description,
		Status:      in.Status,
		Attributes:  in.Attributes,
		StartsAt:    in.StartsAt,
		EndsAt:      in.EndsAt,
	}
	if err := s.repo.Update(ctx, customerID, p); err != nil {
		return nil, err
	}
	s.logger.Info("project updated", "project_id", p.ID, "customer_id", customerID)
	return projectToItem(p), nil
}

func (s *ProjectService) ListProjects(ctx context.Context, customerID uuid.UUID, in ListProjectsInput) (*ListProjectsOutput, error) {
	status := normalizeStatus(in.Status, "")
	if status != "" && !domain.IsValidProjectStatus(status) {
		return nil, domain.ValidationErrors{{Field: "status", Message: "invalid project status"}}
	}

	page, size, offset := clampPagination(in.Page, in.PageSize)
	filter := domain.ProjectListFilter{
		Status:   status,
		Search:   strings.TrimSpace(in.Search),
		From:     in.From,
		To:       in.To,
		Limit:    size,
		Offset:   offset,
		SortBy:   in.SortBy,
		SortDesc: strings.ToLower(in.Order) != "asc",
	}

	items, total, err := s.repo.List(ctx, customerID, filter)
	if err != nil {
		return nil, err
	}

	out := &ListProjectsOutput{
		Items:      make([]ProjectItem, 0, len(items)),
		Pagination: Pagination{Page: page, PageSize: size, Total: total, TotalPages: totalPages(total, size)},
	}
	for _, p := range items {
		out.Items = append(out.Items, *projectToItem(p))
	}
	return out, nil
}

func (s *ProjectService) DeleteProject(ctx context.Context, customerID, id uuid.UUID) (*DeleteProjectOutput, error) {
	deletedAt, err := s.repo.SoftDelete(ctx, customerID, id)
	if err != nil {
		return nil, err
	}
	s.logger.Info("project deleted", "project_id", id, "customer_id", customerID)
	return &DeleteProjectOutput{ID: id, DeletedAt: deletedAt}, nil
}

// ───────────────────────── helpers ─────────────────────────

func validateProject(name, slug, status string, attrs json.RawMessage) domain.ValidationErrors {
	var errs domain.ValidationErrors
	if name == "" {
		errs = append(errs, &domain.ValidationError{Field: "name", Message: "is required"})
	} else if l := utf8.RuneCountInString(name); l < 2 || l > 255 {
		errs = append(errs, &domain.ValidationError{Field: "name", Message: "must be between 2 and 255 characters"})
	}
	if slug != "" && utf8.RuneCountInString(slug) > 100 {
		errs = append(errs, &domain.ValidationError{Field: "slug", Message: "must be at most 100 characters"})
	}
	if !domain.IsValidProjectStatus(status) {
		errs = append(errs, &domain.ValidationError{Field: "status", Message: "must be one of DRAFT, ACTIVE, ON_HOLD, ARCHIVED"})
	}
	if !validAttributes(attrs) {
		errs = append(errs, attrError("attributes"))
	}
	return errs
}

func projectToItem(p *domain.Project) *ProjectItem {
	return &ProjectItem{
		ID:          p.ID,
		Name:        p.Name,
		Slug:        p.Slug,
		Direction:   p.Direction,
		Description: p.Description,
		Status:      p.Status,
		Attributes:  p.Attributes,
		StartsAt:    p.StartsAt,
		EndsAt:      p.EndsAt,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

// normalizeStatus приводит статус к верхнему регистру; пустой → def.
func normalizeStatus(s, def string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	if s == "" {
		return def
	}
	return s
}
