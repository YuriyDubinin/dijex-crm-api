package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/YuriyDubinin/bms-api/internal/domain"
)

// ── фейки ──

type fakeDealRepo struct {
	created   *domain.Deal
	createErr error
	updateErr error
	listItems []*domain.Deal
	listTotal int
	deletedAt time.Time
}

func (f *fakeDealRepo) Create(_ context.Context, d *domain.Deal) error {
	f.created = d
	return f.createErr
}
func (f *fakeDealRepo) List(_ context.Context, _ uuid.UUID, _ domain.DealListFilter) ([]*domain.Deal, int, error) {
	return f.listItems, f.listTotal, nil
}
func (f *fakeDealRepo) GetByID(_ context.Context, _, _ uuid.UUID) (*domain.Deal, error) {
	return nil, domain.ErrNotFound
}
func (f *fakeDealRepo) Update(_ context.Context, _ uuid.UUID, _ *domain.Deal) error {
	return f.updateErr
}
func (f *fakeDealRepo) SoftDelete(_ context.Context, _, _ uuid.UUID) (time.Time, error) {
	return f.deletedAt, nil
}

// Чекеры принадлежности с настраиваемой ошибкой.
type stubProjectChecker struct{ err error }

func (c stubProjectChecker) GetByID(_ context.Context, _, _ uuid.UUID) (*domain.Project, error) {
	return &domain.Project{}, c.err
}

type stubClientChecker struct{ err error }

func (c stubClientChecker) GetByID(_ context.Context, _, _ uuid.UUID) (*domain.Client, error) {
	return &domain.Client{}, c.err
}

type stubServiceChecker struct{ err error }

func (c stubServiceChecker) GetByID(_ context.Context, _, _ uuid.UUID) (*domain.Service, error) {
	return &domain.Service{}, c.err
}

type stubProcessChecker struct{ err error }

func (c stubProcessChecker) GetByID(_ context.Context, _, _ uuid.UUID) (*domain.Process, error) {
	return &domain.Process{}, c.err
}

type stubUserChecker struct{ err error }

func (c stubUserChecker) GetByID(_ context.Context, _, _ uuid.UUID) (*domain.CustomerUser, error) {
	return &domain.CustomerUser{}, c.err
}

func newDealSvc(repo *fakeDealRepo, projErr, cliErr, usrErr error) *DealService {
	return NewDealService(repo, stubProjectChecker{projErr}, stubClientChecker{cliErr}, stubServiceChecker{}, stubProcessChecker{}, stubUserChecker{usrErr}, testLogger())
}

func TestCreateDeal(t *testing.T) {
	t.Parallel()
	cust, actor := uuid.New(), uuid.New()
	now := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)

	t.Run("success: defaults, created_by, no closed_at for NEW", func(t *testing.T) {
		t.Parallel()
		repo := &fakeDealRepo{}
		svc := newDealSvc(repo, nil, nil, nil)
		svc.clock = fixedClock(now)
		out, err := svc.CreateDeal(context.Background(), cust, CreateDealInput{Title: "  Продажа  ", CreatedBy: actor})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo.created.Title != "Продажа" {
			t.Errorf("title not trimmed: %q", repo.created.Title)
		}
		if repo.created.Status != domain.DealStatusNew {
			t.Errorf("default status: %q", repo.created.Status)
		}
		if repo.created.Currency != "USD" {
			t.Errorf("default currency: %q", repo.created.Currency)
		}
		if repo.created.ClosedAt != nil {
			t.Errorf("closed_at must be nil for NEW")
		}
		if repo.created.CreatedBy == nil || *repo.created.CreatedBy != actor {
			t.Errorf("created_by not set from actor")
		}
		if repo.created.CustomerID != cust || repo.created.ID == uuid.Nil {
			t.Errorf("tenant/id not set")
		}
		if out.Status != domain.DealStatusNew {
			t.Errorf("output status: %q", out.Status)
		}
	})

	t.Run("WON sets closed_at", func(t *testing.T) {
		t.Parallel()
		repo := &fakeDealRepo{}
		svc := newDealSvc(repo, nil, nil, nil)
		svc.clock = fixedClock(now)
		if _, err := svc.CreateDeal(context.Background(), cust, CreateDealInput{Title: "Deal", Status: "won", CreatedBy: actor}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo.created.ClosedAt == nil || !repo.created.ClosedAt.Equal(now) {
			t.Errorf("closed_at must be set to now for WON")
		}
	})

	t.Run("validation errors", func(t *testing.T) {
		t.Parallel()
		bad := -5.0
		prob := 150
		cases := map[string]CreateDealInput{
			"empty title":     {Title: " "},
			"bad currency":    {Title: "Deal", Currency: "RUBLE"},
			"negative amount": {Title: "Deal", Amount: &bad},
			"bad probability": {Title: "Deal", Probability: &prob},
		}
		for name, in := range cases {
			in := in
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				repo := &fakeDealRepo{}
				svc := newDealSvc(repo, nil, nil, nil)
				svc.clock = fixedClock(now)
				_, err := svc.CreateDeal(context.Background(), cust, in)
				var verr domain.ValidationErrors
				if !errors.As(err, &verr) {
					t.Fatalf("want ValidationErrors, got %v", err)
				}
				if repo.created != nil {
					t.Error("repo.Create must not be called on invalid input")
				}
			})
		}
	})

	t.Run("client_id not in tenant → validation error", func(t *testing.T) {
		t.Parallel()
		repo := &fakeDealRepo{}
		svc := newDealSvc(repo, nil, domain.ErrNotFound, nil) // client checker returns NotFound
		svc.clock = fixedClock(now)
		cid := uuid.New()
		_, err := svc.CreateDeal(context.Background(), cust, CreateDealInput{Title: "Deal", ClientID: &cid, CreatedBy: actor})
		var verr domain.ValidationErrors
		if !errors.As(err, &verr) {
			t.Fatalf("want ValidationErrors, got %v", err)
		}
		if repo.created != nil {
			t.Error("repo.Create must not be called when client ownership fails")
		}
	})

	t.Run("service_id not in tenant → validation error", func(t *testing.T) {
		t.Parallel()
		repo := &fakeDealRepo{}
		// service checker returns NotFound
		svc := NewDealService(repo, stubProjectChecker{}, stubClientChecker{}, stubServiceChecker{domain.ErrNotFound}, stubProcessChecker{}, stubUserChecker{}, testLogger())
		svc.clock = fixedClock(now)
		sid := uuid.New()
		_, err := svc.CreateDeal(context.Background(), cust, CreateDealInput{Title: "Deal", ServiceID: &sid, CreatedBy: actor})
		var verr domain.ValidationErrors
		if !errors.As(err, &verr) {
			t.Fatalf("want ValidationErrors, got %v", err)
		}
		if repo.created != nil {
			t.Error("repo.Create must not be called when service ownership fails")
		}
	})

	t.Run("default type is INCOME, custom type EXPENSE preserved", func(t *testing.T) {
		t.Parallel()
		repo := &fakeDealRepo{}
		svc := newDealSvc(repo, nil, nil, nil)
		svc.clock = fixedClock(now)
		if _, err := svc.CreateDeal(context.Background(), cust, CreateDealInput{Title: "Deal", CreatedBy: actor}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo.created.Type != domain.DealTypeIncome {
			t.Errorf("default type: %q, want INCOME", repo.created.Type)
		}

		repo2 := &fakeDealRepo{}
		svc2 := newDealSvc(repo2, nil, nil, nil)
		svc2.clock = fixedClock(now)
		if _, err := svc2.CreateDeal(context.Background(), cust, CreateDealInput{Title: "Deal", Type: "expense", CreatedBy: actor}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo2.created.Type != domain.DealTypeExpense {
			t.Errorf("type not normalized to EXPENSE: %q", repo2.created.Type)
		}
	})
}

func TestListDeals_ClampAndInvalidStatus(t *testing.T) {
	t.Parallel()
	repo := &fakeDealRepo{listItems: []*domain.Deal{{ID: uuid.New(), Title: "A"}}, listTotal: 25}
	svc := newDealSvc(repo, nil, nil, nil)

	out, err := svc.ListDeals(context.Background(), uuid.New(), ListDealsInput{Page: 0, PageSize: 1000})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Pagination.PageSize != maxPageSize || out.Pagination.Page != 1 {
		t.Errorf("pagination not clamped: %+v", out.Pagination)
	}
	if out.Pagination.Total != 25 || out.Pagination.TotalPages != 1 {
		t.Errorf("total pages wrong: %+v", out.Pagination)
	}

	if _, err := svc.ListDeals(context.Background(), uuid.New(), ListDealsInput{Status: "nope"}); err == nil {
		t.Error("invalid status must be rejected")
	}
}

func TestDeleteDeal_Passthrough(t *testing.T) {
	t.Parallel()
	when := time.Date(2026, 8, 9, 10, 11, 12, 0, time.UTC)
	repo := &fakeDealRepo{deletedAt: when}
	svc := newDealSvc(repo, nil, nil, nil)
	out, err := svc.DeleteDeal(context.Background(), uuid.New(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out.DeletedAt.Equal(when) {
		t.Errorf("deleted_at not returned")
	}
}
