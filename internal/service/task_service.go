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

// ───────────────────────── Task DTO ─────────────────────────

type CreateTaskInput struct {
	ProjectID   *uuid.UUID
	ClientID    *uuid.UUID
	DealID      *uuid.UUID
	Title       string
	Description string
	Status      string
	Priority    string
	AssignedTo  *uuid.UUID
	CreatedBy   uuid.UUID
	DueAt       *time.Time
	Attributes  json.RawMessage
}

type UpdateTaskInput struct {
	ID          uuid.UUID
	ProjectID   *uuid.UUID
	ClientID    *uuid.UUID
	DealID      *uuid.UUID
	Title       string
	Description string
	Status      string
	Priority    string
	AssignedTo  *uuid.UUID
	DueAt       *time.Time
	Attributes  json.RawMessage
}

type ListTasksInput struct {
	Page       int
	PageSize   int
	ProjectID  *uuid.UUID
	ClientID   *uuid.UUID
	DealID     *uuid.UUID
	Status     string
	Priority   string
	AssignedTo *uuid.UUID
	Search     string
	DueBefore  *time.Time
	DueAfter   *time.Time
	SortBy     string
	Order      string
}

type TaskItem struct {
	ID          uuid.UUID
	ProjectID   *uuid.UUID
	ClientID    *uuid.UUID
	DealID      *uuid.UUID
	Title       string
	Description string
	Status      string
	Priority    string
	AssignedTo  *uuid.UUID
	CreatedBy   *uuid.UUID
	DueAt       *time.Time
	CompletedAt *time.Time
	Attributes  json.RawMessage
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ListTasksOutput struct {
	Items      []TaskItem
	Pagination Pagination
}

type DeleteTaskOutput struct {
	ID        uuid.UUID
	DeletedAt time.Time
}

// ───────────────────────── Task ─────────────────────────

type TaskService struct {
	repo     domain.TaskRepository
	projects ProjectChecker
	clients  ClientChecker
	deals    DealChecker
	users    UserChecker
	logger   *slog.Logger
	clock    func() time.Time
}

func NewTaskService(repo domain.TaskRepository, projects ProjectChecker, clients ClientChecker, deals DealChecker, users UserChecker, logger *slog.Logger) *TaskService {
	return &TaskService{repo: repo, projects: projects, clients: clients, deals: deals, users: users, logger: logger, clock: time.Now}
}

// taskLinkErrors — общие проверки принадлежности связанных сущностей тенанту.
func (s *TaskService) taskLinkErrors(ctx context.Context, customerID uuid.UUID, projectID, clientID, dealID, assignedTo *uuid.UUID) domain.ValidationErrors {
	var errs domain.ValidationErrors
	if ve := ensureProjectOwned(ctx, s.projects, customerID, projectID); ve != nil {
		errs = append(errs, ve)
	}
	if ve := ensureClientOwned(ctx, s.clients, customerID, clientID); ve != nil {
		errs = append(errs, ve)
	}
	if ve := ensureDealOwned(ctx, s.deals, customerID, dealID); ve != nil {
		errs = append(errs, ve)
	}
	if ve := ensureUserOwned(ctx, s.users, customerID, assignedTo, "assigned_to"); ve != nil {
		errs = append(errs, ve)
	}
	return errs
}

func (s *TaskService) CreateTask(ctx context.Context, customerID uuid.UUID, in CreateTaskInput) (*TaskItem, error) {
	in.Title = strings.TrimSpace(in.Title)
	in.Description = strings.TrimSpace(in.Description)
	in.Status = normalizeStatus(in.Status, domain.TaskStatusTodo)
	in.Priority = normalizeStatus(in.Priority, domain.TaskPriorityMedium)

	errs := validateTask(in.Title, in.Status, in.Priority, in.Attributes)
	errs = append(errs, s.taskLinkErrors(ctx, customerID, in.ProjectID, in.ClientID, in.DealID, in.AssignedTo)...)
	if len(errs) > 0 {
		return nil, errs
	}

	now := s.clock()
	var completedAt *time.Time
	if in.Status == domain.TaskStatusDone {
		completedAt = &now
	}
	createdBy := in.CreatedBy
	task := &domain.Task{
		ID:          uuid.New(),
		CustomerID:  customerID,
		ProjectID:   in.ProjectID,
		ClientID:    in.ClientID,
		DealID:      in.DealID,
		Title:       in.Title,
		Description: in.Description,
		Status:      in.Status,
		Priority:    in.Priority,
		AssignedTo:  in.AssignedTo,
		CreatedBy:   &createdBy,
		DueAt:       in.DueAt,
		CompletedAt: completedAt,
		Attributes:  in.Attributes,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.repo.Create(ctx, task); err != nil {
		return nil, err
	}
	s.logger.Info("task created", "task_id", task.ID, "customer_id", customerID)
	return taskToItem(task), nil
}

func (s *TaskService) UpdateTask(ctx context.Context, customerID uuid.UUID, in UpdateTaskInput) (*TaskItem, error) {
	in.Title = strings.TrimSpace(in.Title)
	in.Description = strings.TrimSpace(in.Description)
	in.Status = normalizeStatus(in.Status, domain.TaskStatusTodo)
	in.Priority = normalizeStatus(in.Priority, domain.TaskPriorityMedium)

	errs := validateTask(in.Title, in.Status, in.Priority, in.Attributes)
	errs = append(errs, s.taskLinkErrors(ctx, customerID, in.ProjectID, in.ClientID, in.DealID, in.AssignedTo)...)
	if len(errs) > 0 {
		return nil, errs
	}

	var completedAt *time.Time
	if in.Status == domain.TaskStatusDone {
		now := s.clock()
		completedAt = &now
	}
	task := &domain.Task{
		ID:          in.ID,
		CustomerID:  customerID,
		ProjectID:   in.ProjectID,
		ClientID:    in.ClientID,
		DealID:      in.DealID,
		Title:       in.Title,
		Description: in.Description,
		Status:      in.Status,
		Priority:    in.Priority,
		AssignedTo:  in.AssignedTo,
		DueAt:       in.DueAt,
		CompletedAt: completedAt,
		Attributes:  in.Attributes,
	}
	if err := s.repo.Update(ctx, customerID, task); err != nil {
		return nil, err
	}
	s.logger.Info("task updated", "task_id", task.ID, "customer_id", customerID)
	return taskToItem(task), nil
}

func (s *TaskService) ListTasks(ctx context.Context, customerID uuid.UUID, in ListTasksInput) (*ListTasksOutput, error) {
	status := normalizeStatus(in.Status, "")
	if status != "" && !domain.IsValidTaskStatus(status) {
		return nil, domain.ValidationErrors{{Field: "status", Message: "invalid task status"}}
	}
	priority := normalizeStatus(in.Priority, "")
	if priority != "" && !domain.IsValidTaskPriority(priority) {
		return nil, domain.ValidationErrors{{Field: "priority", Message: "invalid task priority"}}
	}

	page, size, offset := clampPagination(in.Page, in.PageSize)
	filter := domain.TaskListFilter{
		ProjectID:  in.ProjectID,
		ClientID:   in.ClientID,
		DealID:     in.DealID,
		Status:     status,
		Priority:   priority,
		AssignedTo: in.AssignedTo,
		Search:     strings.TrimSpace(in.Search),
		DueBefore:  in.DueBefore,
		DueAfter:   in.DueAfter,
		Limit:      size,
		Offset:     offset,
		SortBy:     in.SortBy,
		SortDesc:   strings.ToLower(in.Order) != "asc",
	}

	items, total, err := s.repo.List(ctx, customerID, filter)
	if err != nil {
		return nil, err
	}
	out := &ListTasksOutput{
		Items:      make([]TaskItem, 0, len(items)),
		Pagination: Pagination{Page: page, PageSize: size, Total: total, TotalPages: totalPages(total, size)},
	}
	for _, task := range items {
		out.Items = append(out.Items, *taskToItem(task))
	}
	return out, nil
}

func (s *TaskService) DeleteTask(ctx context.Context, customerID, id uuid.UUID) (*DeleteTaskOutput, error) {
	deletedAt, err := s.repo.SoftDelete(ctx, customerID, id)
	if err != nil {
		return nil, err
	}
	s.logger.Info("task deleted", "task_id", id, "customer_id", customerID)
	return &DeleteTaskOutput{ID: id, DeletedAt: deletedAt}, nil
}

// ───────────────────────── helpers ─────────────────────────

func validateTask(title, status, priority string, attrs json.RawMessage) domain.ValidationErrors {
	var errs domain.ValidationErrors
	if title == "" {
		errs = append(errs, &domain.ValidationError{Field: "title", Message: "is required"})
	} else if l := utf8.RuneCountInString(title); l < 2 || l > 500 {
		errs = append(errs, &domain.ValidationError{Field: "title", Message: "must be between 2 and 500 characters"})
	}
	if !domain.IsValidTaskStatus(status) {
		errs = append(errs, &domain.ValidationError{Field: "status", Message: "must be one of TODO, IN_PROGRESS, BLOCKED, DONE, CANCELLED"})
	}
	if !domain.IsValidTaskPriority(priority) {
		errs = append(errs, &domain.ValidationError{Field: "priority", Message: "must be one of LOW, MEDIUM, HIGH, URGENT"})
	}
	if !validAttributes(attrs) {
		errs = append(errs, attrError("attributes"))
	}
	return errs
}

func taskToItem(t *domain.Task) *TaskItem {
	return &TaskItem{
		ID:          t.ID,
		ProjectID:   t.ProjectID,
		ClientID:    t.ClientID,
		DealID:      t.DealID,
		Title:       t.Title,
		Description: t.Description,
		Status:      t.Status,
		Priority:    t.Priority,
		AssignedTo:  t.AssignedTo,
		CreatedBy:   t.CreatedBy,
		DueAt:       t.DueAt,
		CompletedAt: t.CompletedAt,
		Attributes:  t.Attributes,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}
