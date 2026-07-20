package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/YuriyDubinin/bms-api/internal/domain"
)

// agendaLimit — сколько записей каждого типа тянуть в окно календаря. Для окна
// (месяц/неделя) этого с запасом; при необходимости — постраничные /list.
const agendaLimit = 1000

// AgendaItem — нормализованный элемент календаря (любая датированная сущность,
// приведённая к общему виду для отрисовки).
type AgendaItem struct {
	Type   string // "event" | "task" | "deal" | "project"
	ID     uuid.UUID
	Title  string
	Start  *time.Time
	End    *time.Time
	AllDay bool
	Status string
}

// AgendaInput — окно календаря и (опциональный) фильтр типов.
type AgendaInput struct {
	From  *time.Time
	To    *time.Time
	Types map[string]bool // пусто = все типы
}

func (in AgendaInput) want(t string) bool {
	return len(in.Types) == 0 || in.Types[t]
}

// AgendaService — read-only агрегатор датированных сущностей для календаря.
// Ничего не пишет; переиспользует List-методы репозиториев с оконными фильтрами.
type AgendaService struct {
	tasks     domain.TaskRepository
	deals     domain.DealRepository
	projects  domain.ProjectRepository
	processes domain.ProcessRepository
	logger    *slog.Logger
}

func NewAgendaService(tasks domain.TaskRepository, deals domain.DealRepository, projects domain.ProjectRepository, processes domain.ProcessRepository, logger *slog.Logger) *AgendaService {
	return &AgendaService{tasks: tasks, deals: deals, projects: projects, processes: processes, logger: logger}
}

// Agenda собирает все датированные сущности, попадающие в окно [From, To].
func (s *AgendaService) Agenda(ctx context.Context, customerID uuid.UUID, in AgendaInput) ([]AgendaItem, error) {
	out := []AgendaItem{}

	// Задачи: по сроку (due_at) в окне.
	if in.want("task") {
		tasks, _, err := s.tasks.List(ctx, customerID, domain.TaskListFilter{DueAfter: in.From, DueBefore: in.To, Limit: agendaLimit})
		if err != nil {
			return nil, err
		}
		for _, t := range tasks {
			if t.DueAt == nil {
				continue // без срока — на календаре не размещается
			}
			out = append(out, AgendaItem{Type: "task", ID: t.ID, Title: t.Title, Start: t.DueAt, AllDay: false, Status: t.Status})
		}
	}

	// Сделки: по ожидаемой дате закрытия (expected_close_at) в окне.
	if in.want("deal") {
		deals, _, err := s.deals.List(ctx, customerID, domain.DealListFilter{CloseFrom: in.From, CloseTo: in.To, Limit: agendaLimit})
		if err != nil {
			return nil, err
		}
		for _, d := range deals {
			if d.ExpectedCloseAt == nil {
				continue
			}
			out = append(out, AgendaItem{Type: "deal", ID: d.ID, Title: d.Title, Start: d.ExpectedCloseAt, AllDay: true, Status: d.Status})
		}
	}

	// Проекты: интервал [starts_at, ends_at], пересекающий окно (многодневная полоса).
	if in.want("project") {
		projects, _, err := s.projects.List(ctx, customerID, domain.ProjectListFilter{From: in.From, To: in.To, Limit: agendaLimit})
		if err != nil {
			return nil, err
		}
		for _, p := range projects {
			if p.StartsAt == nil && p.EndsAt == nil {
				continue // без дат — на календаре не размещается
			}
			out = append(out, AgendaItem{Type: "project", ID: p.ID, Title: p.Name, Start: p.StartsAt, End: p.EndsAt, AllDay: true, Status: p.Status})
		}
	}

	// Процессы: интервал [starts_at, ends_at], пересекающий окно (многодневная полоса).
	if in.want("process") {
		processes, _, err := s.processes.List(ctx, customerID, domain.ProcessListFilter{From: in.From, To: in.To, Limit: agendaLimit})
		if err != nil {
			return nil, err
		}
		for _, p := range processes {
			if p.StartsAt == nil && p.EndsAt == nil {
				continue // без дат — на календаре не размещается
			}
			out = append(out, AgendaItem{Type: "process", ID: p.ID, Title: p.Name, Start: p.StartsAt, End: p.EndsAt, AllDay: true, Status: p.Status})
		}
	}

	return out, nil
}
