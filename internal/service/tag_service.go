package service

import (
	"context"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/YuriyDubinin/bms-api/internal/domain"
)

// ───────────────────────── Tag DTO ─────────────────────────

type CreateTagInput struct {
	Name  string
	Color string
}

type UpdateTagInput struct {
	ID    uuid.UUID
	Name  string
	Color string
}

type ListTagsInput struct {
	Page     int
	PageSize int
	EntityID *uuid.UUID
	Search   string
	Order    string
}

type TagItem struct {
	ID        uuid.UUID
	Name      string
	Color     string
	CreatedAt time.Time
}

type ListTagsOutput struct {
	Items      []TagItem
	Pagination Pagination
}

type DeleteTagOutput struct {
	ID uuid.UUID
}

// ───────────────────────── Service ─────────────────────────

type TagService struct {
	repo     domain.TagRepository
	entities EntityResolver
	logger   *slog.Logger
	clock    func() time.Time
}

func NewTagService(repo domain.TagRepository, entities EntityResolver, logger *slog.Logger) *TagService {
	return &TagService{repo: repo, entities: entities, logger: logger, clock: time.Now}
}

func (s *TagService) CreateTag(ctx context.Context, customerID uuid.UUID, in CreateTagInput) (*TagItem, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Color = strings.TrimSpace(in.Color)

	if errs := validateTag(in.Name); len(errs) > 0 {
		return nil, errs
	}

	tag := &domain.Tag{
		ID:         uuid.New(),
		CustomerID: customerID,
		Name:       in.Name,
		Color:      in.Color,
	}
	if err := s.repo.Create(ctx, tag); err != nil {
		return nil, err
	}
	s.logger.Info("tag created", "tag_id", tag.ID, "customer_id", customerID)
	return tagToItem(tag), nil
}

func (s *TagService) UpdateTag(ctx context.Context, customerID uuid.UUID, in UpdateTagInput) (*TagItem, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Color = strings.TrimSpace(in.Color)

	if errs := validateTag(in.Name); len(errs) > 0 {
		return nil, errs
	}

	tag := &domain.Tag{
		ID:         in.ID,
		CustomerID: customerID,
		Name:       in.Name,
		Color:      in.Color,
	}
	if err := s.repo.Update(ctx, customerID, tag); err != nil {
		return nil, err
	}
	s.logger.Info("tag updated", "tag_id", tag.ID, "customer_id", customerID)
	return tagToItem(tag), nil
}

func (s *TagService) ListTags(ctx context.Context, customerID uuid.UUID, in ListTagsInput) (*ListTagsOutput, error) {
	page, size, offset := clampPagination(in.Page, in.PageSize)
	filter := domain.TagListFilter{
		EntityID: in.EntityID,
		Search:   strings.TrimSpace(in.Search),
		Limit:    size,
		Offset:   offset,
		SortDesc: strings.ToLower(in.Order) != "asc",
	}

	items, total, err := s.repo.List(ctx, customerID, filter)
	if err != nil {
		return nil, err
	}
	out := &ListTagsOutput{
		Items:      make([]TagItem, 0, len(items)),
		Pagination: Pagination{Page: page, PageSize: size, Total: total, TotalPages: totalPages(total, size)},
	}
	for _, tag := range items {
		out.Items = append(out.Items, *tagToItem(tag))
	}
	return out, nil
}

func (s *TagService) DeleteTag(ctx context.Context, customerID, id uuid.UUID) (*DeleteTagOutput, error) {
	if err := s.repo.Delete(ctx, customerID, id); err != nil {
		return nil, err
	}
	s.logger.Info("tag deleted", "tag_id", id, "customer_id", customerID)
	return &DeleteTagOutput{ID: id}, nil
}

func (s *TagService) AttachTag(ctx context.Context, customerID, tagID, entityID uuid.UUID) error {
	if _, err := s.repo.GetByID(ctx, customerID, tagID); err != nil {
		return err
	}
	if _, err := s.entities.ResolveType(ctx, customerID, entityID); err != nil {
		return domain.ValidationErrors{{Field: "entity_id", Message: "entity not found in this tenant"}}
	}
	if err := s.repo.Attach(ctx, customerID, tagID, entityID); err != nil {
		return err
	}
	s.logger.Info("tag attached", "tag_id", tagID, "entity_id", entityID, "customer_id", customerID)
	return nil
}

func (s *TagService) DetachTag(ctx context.Context, customerID, tagID, entityID uuid.UUID) error {
	if _, err := s.repo.GetByID(ctx, customerID, tagID); err != nil {
		return err
	}
	if err := s.repo.Detach(ctx, customerID, tagID, entityID); err != nil {
		return err
	}
	s.logger.Info("tag detached", "tag_id", tagID, "entity_id", entityID, "customer_id", customerID)
	return nil
}

// ───────────────────────── helpers ─────────────────────────

func validateTag(name string) domain.ValidationErrors {
	var errs domain.ValidationErrors
	if name == "" {
		errs = append(errs, &domain.ValidationError{Field: "name", Message: "is required"})
	} else if l := utf8.RuneCountInString(name); l < 1 || l > 100 {
		errs = append(errs, &domain.ValidationError{Field: "name", Message: "must be between 1 and 100 characters"})
	}
	return errs
}

func tagToItem(t *domain.Tag) *TagItem {
	return &TagItem{
		ID:        t.ID,
		Name:      t.Name,
		Color:     t.Color,
		CreatedAt: t.CreatedAt,
	}
}
