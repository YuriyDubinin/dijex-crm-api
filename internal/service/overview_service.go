package service

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"github.com/YuriyDubinin/bms-api/internal/domain"
)

// overviewChildLimit — сколько связанных записей каждого типа тянуть в overview.
// Для «карточки» этого с запасом; при необходимости постранично — обычные /list.
const overviewChildLimit = 200

// ───────────────────────── Output ─────────────────────────

// DealOverviewOutput — сделка со всеми связанными сущностями одним ответом:
// проект, клиент, услуга (по service_id), задачи (по deal_id) и универсальные
// связи (прочее).
type DealOverviewOutput struct {
	Deal    DealItem
	Project *ProjectItem
	Client  *ClientItem
	Service *ServiceItem
	Tasks   []TaskItem
	Links   []LinkItem
}

// ClientOverviewOutput — клиент со всем связанным: проект, его сделки, задачи
// (по client_id) и универсальные связи (предоставленные услуги и прочее).
type ClientOverviewOutput struct {
	Client  ClientItem
	Project *ProjectItem
	Deals   []DealItem
	Tasks   []TaskItem
	Links   []LinkItem
}

// ───────────────────────── Service ─────────────────────────

// OverviewService — агрегатор поверх существующих репозиториев. Ничего не
// пишет, только собирает связанные данные в разрезе тенанта.
type OverviewService struct {
	deals    domain.DealRepository
	clients  domain.ClientRepository
	projects domain.ProjectRepository
	tasks    domain.TaskRepository
	services domain.ServiceRepository
	links    domain.EntityLinkRepository
	logger   *slog.Logger
}

func NewOverviewService(
	deals domain.DealRepository,
	clients domain.ClientRepository,
	projects domain.ProjectRepository,
	tasks domain.TaskRepository,
	services domain.ServiceRepository,
	links domain.EntityLinkRepository,
	logger *slog.Logger,
) *OverviewService {
	return &OverviewService{deals: deals, clients: clients, projects: projects, tasks: tasks, services: services, links: links, logger: logger}
}

// DealOverview собирает полную картину по сделке. ErrNotFound, если сделки нет
// (или она чужого тенанта).
func (s *OverviewService) DealOverview(ctx context.Context, customerID, dealID uuid.UUID) (*DealOverviewOutput, error) {
	deal, err := s.deals.GetByID(ctx, customerID, dealID)
	if err != nil {
		return nil, err
	}

	out := &DealOverviewOutput{Deal: *dealToItem(deal)}
	out.Project = s.projectOf(ctx, customerID, deal.ProjectID)
	out.Client = s.clientOf(ctx, customerID, deal.ClientID)
	out.Service = s.serviceOf(ctx, customerID, deal.ServiceID)

	tasks, _, err := s.tasks.List(ctx, customerID, domain.TaskListFilter{DealID: &dealID, Limit: overviewChildLimit})
	if err != nil {
		return nil, err
	}
	out.Tasks = tasksToItems(tasks)

	out.Links, err = s.linksOf(ctx, customerID, dealID)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ClientOverview собирает полную картину по клиенту.
func (s *OverviewService) ClientOverview(ctx context.Context, customerID, clientID uuid.UUID) (*ClientOverviewOutput, error) {
	client, err := s.clients.GetByID(ctx, customerID, clientID)
	if err != nil {
		return nil, err
	}

	out := &ClientOverviewOutput{Client: *clientToItem(client)}
	out.Project = s.projectOf(ctx, customerID, client.ProjectID)

	deals, _, err := s.deals.List(ctx, customerID, domain.DealListFilter{ClientID: &clientID, Limit: overviewChildLimit})
	if err != nil {
		return nil, err
	}
	out.Deals = make([]DealItem, 0, len(deals))
	for _, d := range deals {
		out.Deals = append(out.Deals, *dealToItem(d))
	}

	tasks, _, err := s.tasks.List(ctx, customerID, domain.TaskListFilter{ClientID: &clientID, Limit: overviewChildLimit})
	if err != nil {
		return nil, err
	}
	out.Tasks = tasksToItems(tasks)

	out.Links, err = s.linksOf(ctx, customerID, clientID)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ───────────────────────── helpers ─────────────────────────

// projectOf возвращает проект по (опциональному) id; отсутствие/ошибку глушит в nil.
func (s *OverviewService) projectOf(ctx context.Context, customerID uuid.UUID, id *uuid.UUID) *ProjectItem {
	if id == nil {
		return nil
	}
	p, err := s.projects.GetByID(ctx, customerID, *id)
	if err != nil {
		return nil
	}
	return projectToItem(p)
}

func (s *OverviewService) clientOf(ctx context.Context, customerID uuid.UUID, id *uuid.UUID) *ClientItem {
	if id == nil {
		return nil
	}
	c, err := s.clients.GetByID(ctx, customerID, *id)
	if err != nil {
		return nil
	}
	return clientToItem(c)
}

func (s *OverviewService) serviceOf(ctx context.Context, customerID uuid.UUID, id *uuid.UUID) *ServiceItem {
	if id == nil {
		return nil
	}
	svc, err := s.services.GetByID(ctx, customerID, *id)
	if err != nil {
		return nil
	}
	return serviceToItem(svc)
}

func (s *OverviewService) linksOf(ctx context.Context, customerID, entityID uuid.UUID) ([]LinkItem, error) {
	linked, _, err := s.links.List(ctx, customerID, domain.EntityLinkListFilter{EntityID: &entityID, Limit: overviewChildLimit})
	if err != nil {
		return nil, err
	}
	out := make([]LinkItem, 0, len(linked))
	for _, le := range linked {
		out = append(out, LinkItem{
			LinkID:         le.LinkID,
			EntityID:       le.EntityID,
			LinkedEntityID: le.LinkedEntityID,
			LinkedType:     le.LinkedType,
			LinkedLabel:    le.LinkedLabel,
			RelationType:   le.RelationType,
			Attributes:     le.Attributes,
			CreatedAt:      le.CreatedAt,
		})
	}
	return out, nil
}

func tasksToItems(tasks []*domain.Task) []TaskItem {
	out := make([]TaskItem, 0, len(tasks))
	for _, t := range tasks {
		out = append(out, *taskToItem(t))
	}
	return out
}
