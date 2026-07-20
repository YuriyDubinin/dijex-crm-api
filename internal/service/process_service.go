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

// ───────────────────────── Process DTO ─────────────────────────

type CreateProcessInput struct {
	ProjectID   *uuid.UUID
	Name        string
	Description string
	Status      string
	StartsAt    *time.Time
	EndsAt      *time.Time
	Attributes  json.RawMessage
}

type UpdateProcessInput struct {
	ID          uuid.UUID
	ProjectID   *uuid.UUID
	Name        string
	Description string
	Status      string
	StartsAt    *time.Time
	EndsAt      *time.Time
	Attributes  json.RawMessage
}

type ListProcessesInput struct {
	Page      int
	PageSize  int
	ProjectID *uuid.UUID
	Status    string
	Search    string
	From      *time.Time
	To        *time.Time
	SortBy    string
	Order     string
}

type ProcessItem struct {
	ID          uuid.UUID
	ProjectID   *uuid.UUID
	Name        string
	Description string
	Status      string
	ClosedAt    *time.Time
	StartsAt    *time.Time
	EndsAt      *time.Time
	Attributes  json.RawMessage
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ListProcessesOutput struct {
	Items      []ProcessItem
	Pagination Pagination
}

type DeleteProcessOutput struct {
	ID        uuid.UUID
	DeletedAt time.Time
}

// ───────────────────────── Stage DTO ─────────────────────────

type CreateProcessStageInput struct {
	ProcessID   uuid.UUID
	Name        string
	Description string
	Position    int
	Attributes  json.RawMessage
}

type UpdateProcessStageInput struct {
	ID          uuid.UUID
	Name        string
	Description string
	Position    int
	Attributes  json.RawMessage
}

type ProcessStageItem struct {
	ID          uuid.UUID
	ProcessID   uuid.UUID
	Name        string
	Description string
	Position    int
	Attributes  json.RawMessage
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type DeleteProcessStageOutput struct {
	ID        uuid.UUID
	DeletedAt time.Time
}

// ───────────────────────── Service ─────────────────────────

type ProcessService struct {
	repo     domain.ProcessRepository
	projects ProjectChecker
	logger   *slog.Logger
	clock    func() time.Time
}

func NewProcessService(repo domain.ProcessRepository, projects ProjectChecker, logger *slog.Logger) *ProcessService {
	return &ProcessService{repo: repo, projects: projects, logger: logger, clock: time.Now}
}

func (s *ProcessService) CreateProcess(ctx context.Context, customerID uuid.UUID, in CreateProcessInput) (*ProcessItem, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Description = strings.TrimSpace(in.Description)
	in.Status = normalizeStatus(in.Status, domain.ProcessStatusActive)

	errs := validateProcess(in.Name, in.Status, in.Attributes)
	if ve := ensureProjectOwned(ctx, s.projects, customerID, in.ProjectID); ve != nil {
		errs = append(errs, ve)
	}
	if len(errs) > 0 {
		return nil, errs
	}

	now := s.clock()
	var closedAt *time.Time
	if domain.IsClosedProcessStatus(in.Status) {
		closedAt = &now
	}
	p := &domain.Process{
		ID:          uuid.New(),
		CustomerID:  customerID,
		ProjectID:   in.ProjectID,
		Name:        in.Name,
		Description: in.Description,
		Status:      in.Status,
		ClosedAt:    closedAt,
		StartsAt:    in.StartsAt,
		EndsAt:      in.EndsAt,
		Attributes:  in.Attributes,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.repo.Create(ctx, p); err != nil {
		return nil, err
	}
	s.logger.Info("process created", "process_id", p.ID, "customer_id", customerID, "status", p.Status)
	return processToItem(p), nil
}

func (s *ProcessService) UpdateProcess(ctx context.Context, customerID uuid.UUID, in UpdateProcessInput) (*ProcessItem, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Description = strings.TrimSpace(in.Description)
	in.Status = normalizeStatus(in.Status, domain.ProcessStatusActive)

	errs := validateProcess(in.Name, in.Status, in.Attributes)
	if ve := ensureProjectOwned(ctx, s.projects, customerID, in.ProjectID); ve != nil {
		errs = append(errs, ve)
	}
	if len(errs) > 0 {
		return nil, errs
	}

	now := s.clock()
	var closedAt *time.Time
	if domain.IsClosedProcessStatus(in.Status) {
		closedAt = &now
	}
	p := &domain.Process{
		ID:          in.ID,
		CustomerID:  customerID,
		ProjectID:   in.ProjectID,
		Name:        in.Name,
		Description: in.Description,
		Status:      in.Status,
		ClosedAt:    closedAt,
		StartsAt:    in.StartsAt,
		EndsAt:      in.EndsAt,
		Attributes:  in.Attributes,
	}
	if err := s.repo.Update(ctx, customerID, p); err != nil {
		return nil, err
	}
	s.logger.Info("process updated", "process_id", p.ID, "customer_id", customerID)
	return processToItem(p), nil
}

func (s *ProcessService) ListProcesses(ctx context.Context, customerID uuid.UUID, in ListProcessesInput) (*ListProcessesOutput, error) {
	status := normalizeStatus(in.Status, "")
	if status != "" && !domain.IsValidProcessStatus(status) {
		return nil, domain.ValidationErrors{{Field: "status", Message: "invalid process status"}}
	}

	page, size, offset := clampPagination(in.Page, in.PageSize)
	filter := domain.ProcessListFilter{
		ProjectID: in.ProjectID,
		Status:    status,
		Search:    strings.TrimSpace(in.Search),
		From:      in.From,
		To:        in.To,
		Limit:     size,
		Offset:    offset,
		SortBy:    in.SortBy,
		SortDesc:  strings.ToLower(in.Order) != "asc",
	}

	items, total, err := s.repo.List(ctx, customerID, filter)
	if err != nil {
		return nil, err
	}
	out := &ListProcessesOutput{
		Items:      make([]ProcessItem, 0, len(items)),
		Pagination: Pagination{Page: page, PageSize: size, Total: total, TotalPages: totalPages(total, size)},
	}
	for _, p := range items {
		out.Items = append(out.Items, *processToItem(p))
	}
	return out, nil
}

func (s *ProcessService) DeleteProcess(ctx context.Context, customerID, id uuid.UUID) (*DeleteProcessOutput, error) {
	deletedAt, err := s.repo.SoftDelete(ctx, customerID, id)
	if err != nil {
		return nil, err
	}
	s.logger.Info("process deleted", "process_id", id, "customer_id", customerID)
	return &DeleteProcessOutput{ID: id, DeletedAt: deletedAt}, nil
}

// ───────────────────────── Stage methods ─────────────────────────

func (s *ProcessService) CreateStage(ctx context.Context, customerID uuid.UUID, in CreateProcessStageInput) (*ProcessStageItem, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Description = strings.TrimSpace(in.Description)

	errs := validateStage(in.Name, in.Attributes)
	// Этап можно создать только в существующем процессе тенанта.
	if _, err := s.repo.GetByID(ctx, customerID, in.ProcessID); err != nil {
		errs = append(errs, &domain.ValidationError{Field: "process_id", Message: "process not found in this tenant"})
	}
	if len(errs) > 0 {
		return nil, errs
	}

	now := s.clock()
	stage := &domain.ProcessStage{
		ID:          uuid.New(),
		CustomerID:  customerID,
		ProcessID:   in.ProcessID,
		Name:        in.Name,
		Description: in.Description,
		Position:    in.Position,
		Attributes:  in.Attributes,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.repo.CreateStage(ctx, stage); err != nil {
		return nil, err
	}
	s.logger.Info("process stage created", "stage_id", stage.ID, "process_id", stage.ProcessID, "customer_id", customerID)
	return stageToItem(stage), nil
}

// ListStages возвращает этапы процесса по порядку. ErrNotFound, если процесса
// нет (или он чужого тенанта).
func (s *ProcessService) ListStages(ctx context.Context, customerID, processID uuid.UUID) ([]ProcessStageItem, error) {
	if _, err := s.repo.GetByID(ctx, customerID, processID); err != nil {
		return nil, err
	}
	stages, err := s.repo.ListStages(ctx, customerID, processID)
	if err != nil {
		return nil, err
	}
	out := make([]ProcessStageItem, 0, len(stages))
	for _, st := range stages {
		out = append(out, *stageToItem(st))
	}
	return out, nil
}

func (s *ProcessService) UpdateStage(ctx context.Context, customerID uuid.UUID, in UpdateProcessStageInput) (*ProcessStageItem, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Description = strings.TrimSpace(in.Description)

	if errs := validateStage(in.Name, in.Attributes); len(errs) > 0 {
		return nil, errs
	}

	stage := &domain.ProcessStage{
		ID:          in.ID,
		CustomerID:  customerID,
		Name:        in.Name,
		Description: in.Description,
		Position:    in.Position,
		Attributes:  in.Attributes,
	}
	if err := s.repo.UpdateStage(ctx, customerID, stage); err != nil {
		return nil, err
	}
	s.logger.Info("process stage updated", "stage_id", stage.ID, "customer_id", customerID)
	return stageToItem(stage), nil
}

func (s *ProcessService) DeleteStage(ctx context.Context, customerID, id uuid.UUID) (*DeleteProcessStageOutput, error) {
	deletedAt, err := s.repo.SoftDeleteStage(ctx, customerID, id)
	if err != nil {
		return nil, err
	}
	s.logger.Info("process stage deleted", "stage_id", id, "customer_id", customerID)
	return &DeleteProcessStageOutput{ID: id, DeletedAt: deletedAt}, nil
}

// ───────────────────────── helpers ─────────────────────────

func validateProcess(name, status string, attrs json.RawMessage) domain.ValidationErrors {
	var errs domain.ValidationErrors
	if name == "" {
		errs = append(errs, &domain.ValidationError{Field: "name", Message: "is required"})
	} else if l := utf8.RuneCountInString(name); l < 2 || l > 255 {
		errs = append(errs, &domain.ValidationError{Field: "name", Message: "must be between 2 and 255 characters"})
	}
	if !domain.IsValidProcessStatus(status) {
		errs = append(errs, &domain.ValidationError{Field: "status", Message: "must be one of ACTIVE, COMPLETED, FAILED"})
	}
	if !validAttributes(attrs) {
		errs = append(errs, attrError("attributes"))
	}
	return errs
}

func validateStage(name string, attrs json.RawMessage) domain.ValidationErrors {
	var errs domain.ValidationErrors
	if name == "" {
		errs = append(errs, &domain.ValidationError{Field: "name", Message: "is required"})
	} else if l := utf8.RuneCountInString(name); l < 2 || l > 255 {
		errs = append(errs, &domain.ValidationError{Field: "name", Message: "must be between 2 and 255 characters"})
	}
	if !validAttributes(attrs) {
		errs = append(errs, attrError("attributes"))
	}
	return errs
}

func processToItem(p *domain.Process) *ProcessItem {
	return &ProcessItem{
		ID:          p.ID,
		ProjectID:   p.ProjectID,
		Name:        p.Name,
		Description: p.Description,
		Status:      p.Status,
		ClosedAt:    p.ClosedAt,
		StartsAt:    p.StartsAt,
		EndsAt:      p.EndsAt,
		Attributes:  p.Attributes,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

func stageToItem(s *domain.ProcessStage) *ProcessStageItem {
	return &ProcessStageItem{
		ID:          s.ID,
		ProcessID:   s.ProcessID,
		Name:        s.Name,
		Description: s.Description,
		Position:    s.Position,
		Attributes:  s.Attributes,
		CreatedAt:   s.CreatedAt,
		UpdatedAt:   s.UpdatedAt,
	}
}
