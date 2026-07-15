package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/YuriyDubinin/bms-api/internal/domain"
)

// fakeProjectRepo — конфигурируемый фейк domain.ProjectRepository.
type fakeProjectRepo struct {
	created    *domain.Project
	createErr  error
	updateErr  error
	listItems  []*domain.Project
	listTotal  int
	listErr    error
	lastFilter domain.ProjectListFilter
	lastCustID uuid.UUID
	deletedAt  time.Time
	deleteErr  error
}

func (f *fakeProjectRepo) Create(_ context.Context, p *domain.Project) error {
	f.created = p
	return f.createErr
}
func (f *fakeProjectRepo) List(_ context.Context, customerID uuid.UUID, flt domain.ProjectListFilter) ([]*domain.Project, int, error) {
	f.lastCustID = customerID
	f.lastFilter = flt
	return f.listItems, f.listTotal, f.listErr
}
func (f *fakeProjectRepo) GetByID(_ context.Context, _, _ uuid.UUID) (*domain.Project, error) {
	return nil, domain.ErrNotFound
}
func (f *fakeProjectRepo) Update(_ context.Context, _ uuid.UUID, _ *domain.Project) error {
	return f.updateErr
}
func (f *fakeProjectRepo) SoftDelete(_ context.Context, _, _ uuid.UUID) (time.Time, error) {
	return f.deletedAt, f.deleteErr
}

func testLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func fixedClock(t time.Time) func() time.Time { return func() time.Time { return t } }

func TestCreateProject_ValidationAndNormalization(t *testing.T) {
	t.Parallel()
	cust := uuid.New()
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	t.Run("empty name → validation error", func(t *testing.T) {
		t.Parallel()
		repo := &fakeProjectRepo{}
		svc := NewProjectService(repo, testLogger())
		svc.clock = fixedClock(now)
		_, err := svc.CreateProject(context.Background(), cust, CreateProjectInput{Name: "  "})
		var verr domain.ValidationErrors
		if !errors.As(err, &verr) {
			t.Fatalf("want ValidationErrors, got %v", err)
		}
		if repo.created != nil {
			t.Fatal("repo.Create must not be called on invalid input")
		}
	})

	t.Run("invalid status → validation error", func(t *testing.T) {
		t.Parallel()
		svc := NewProjectService(&fakeProjectRepo{}, testLogger())
		svc.clock = fixedClock(now)
		_, err := svc.CreateProject(context.Background(), cust, CreateProjectInput{Name: "Auto", Status: "BOGUS"})
		var verr domain.ValidationErrors
		if !errors.As(err, &verr) {
			t.Fatalf("want ValidationErrors, got %v", err)
		}
	})

	t.Run("normalizes status/name, sets tenant+timestamps", func(t *testing.T) {
		t.Parallel()
		repo := &fakeProjectRepo{}
		svc := NewProjectService(repo, testLogger())
		svc.clock = fixedClock(now)
		out, err := svc.CreateProject(context.Background(), cust, CreateProjectInput{
			Name: "  Автопарк  ", Status: "active", Attributes: json.RawMessage(`{"n":1}`),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo.created == nil {
			t.Fatal("repo.Create not called")
		}
		if repo.created.Name != "Автопарк" {
			t.Errorf("name not trimmed: %q", repo.created.Name)
		}
		if repo.created.Status != domain.ProjectStatusActive {
			t.Errorf("status not normalized to ACTIVE: %q", repo.created.Status)
		}
		if repo.created.CustomerID != cust {
			t.Errorf("customer_id not set")
		}
		if repo.created.ID == uuid.Nil {
			t.Errorf("id not generated")
		}
		if !repo.created.CreatedAt.Equal(now) || !repo.created.UpdatedAt.Equal(now) {
			t.Errorf("timestamps not from clock")
		}
		if out.Status != domain.ProjectStatusActive {
			t.Errorf("output status: %q", out.Status)
		}
	})

	t.Run("default status is ACTIVE when omitted", func(t *testing.T) {
		t.Parallel()
		repo := &fakeProjectRepo{}
		svc := NewProjectService(repo, testLogger())
		svc.clock = fixedClock(now)
		if _, err := svc.CreateProject(context.Background(), cust, CreateProjectInput{Name: "X Corp"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo.created.Status != domain.ProjectStatusActive {
			t.Errorf("default status: %q", repo.created.Status)
		}
	})

	t.Run("repo ErrAlreadyExists is propagated", func(t *testing.T) {
		t.Parallel()
		repo := &fakeProjectRepo{createErr: domain.ErrAlreadyExists}
		svc := NewProjectService(repo, testLogger())
		svc.clock = fixedClock(now)
		_, err := svc.CreateProject(context.Background(), cust, CreateProjectInput{Name: "Dup", Slug: "dup"})
		if !errors.Is(err, domain.ErrAlreadyExists) {
			t.Fatalf("want ErrAlreadyExists, got %v", err)
		}
	})
}

func TestListProjects_PaginationClampAndFilter(t *testing.T) {
	t.Parallel()
	cust := uuid.New()
	repo := &fakeProjectRepo{listItems: []*domain.Project{{ID: uuid.New(), Name: "A"}}, listTotal: 41}
	svc := NewProjectService(repo, testLogger())

	out, err := svc.ListProjects(context.Background(), cust, ListProjectsInput{Page: 0, PageSize: 1000, Search: "  a  "})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.lastCustID != cust {
		t.Errorf("customer id not passed to repo")
	}
	if repo.lastFilter.Limit != maxPageSize {
		t.Errorf("page size not clamped to %d: %d", maxPageSize, repo.lastFilter.Limit)
	}
	if repo.lastFilter.Offset != 0 {
		t.Errorf("offset for page 1: %d", repo.lastFilter.Offset)
	}
	if repo.lastFilter.Search != "a" {
		t.Errorf("search not trimmed: %q", repo.lastFilter.Search)
	}
	if !repo.lastFilter.SortDesc {
		t.Errorf("default order should be desc")
	}
	if out.Pagination.Page != 1 || out.Pagination.PageSize != maxPageSize {
		t.Errorf("pagination echo wrong: %+v", out.Pagination)
	}
	if out.Pagination.Total != 41 || out.Pagination.TotalPages != 1 {
		t.Errorf("total pages calc wrong: %+v", out.Pagination)
	}
}

func TestListProjects_InvalidStatus(t *testing.T) {
	t.Parallel()
	svc := NewProjectService(&fakeProjectRepo{}, testLogger())
	_, err := svc.ListProjects(context.Background(), uuid.New(), ListProjectsInput{Status: "nope"})
	var verr domain.ValidationErrors
	if !errors.As(err, &verr) {
		t.Fatalf("want ValidationErrors, got %v", err)
	}
}

func TestUpdateAndDeleteProject_ErrorPassthrough(t *testing.T) {
	t.Parallel()
	cust := uuid.New()

	svcNotFound := NewProjectService(&fakeProjectRepo{updateErr: domain.ErrNotFound}, testLogger())
	_, err := svcNotFound.UpdateProject(context.Background(), cust, UpdateProjectInput{ID: uuid.New(), Name: "Valid"})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("update: want ErrNotFound, got %v", err)
	}

	when := time.Date(2026, 5, 6, 7, 8, 9, 0, time.UTC)
	repo := &fakeProjectRepo{deletedAt: when}
	svc := NewProjectService(repo, testLogger())
	out, err := svc.DeleteProject(context.Background(), cust, uuid.New())
	if err != nil {
		t.Fatalf("delete: unexpected error: %v", err)
	}
	if !out.DeletedAt.Equal(when) {
		t.Errorf("deleted_at not returned: %v", out.DeletedAt)
	}
}
