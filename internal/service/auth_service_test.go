package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/YuriyDubinin/bms-api/internal/domain"
	"github.com/YuriyDubinin/bms-api/pkg/crypto"
)

type fakeTokenRepo struct {
	created *domain.AuthToken
	found   *domain.AuthTokenWithUser
	findErr error
	revoked uuid.UUID
	revokeN int
}

func (f *fakeTokenRepo) Create(_ context.Context, t *domain.AuthToken) error {
	f.created = t
	return nil
}
func (f *fakeTokenRepo) FindActiveByHash(_ context.Context, _ string) (*domain.AuthTokenWithUser, error) {
	return f.found, f.findErr
}
func (f *fakeTokenRepo) TouchLastUsed(_ context.Context, _ uuid.UUID, _ time.Time) error { return nil }
func (f *fakeTokenRepo) Revoke(_ context.Context, id uuid.UUID, _ string, _ time.Time) error {
	f.revoked = id
	f.revokeN++
	return nil
}

type fakeUserFinder struct {
	user *domain.CustomerUser
	err  error
}

func (f *fakeUserFinder) FindByEmail(_ context.Context, _ string) (*domain.CustomerUser, error) {
	return f.user, f.err
}

type fakeRegistrar struct {
	err     error
	gotCust *domain.Customer
	gotUser *domain.CustomerUser
}

func (f *fakeRegistrar) RegisterTenant(_ context.Context, c *domain.Customer, u *domain.CustomerUser) error {
	f.gotCust, f.gotUser = c, u
	return f.err
}

func newTestAuth(t *testing.T, tokens *fakeTokenRepo, users *fakeUserFinder, reg *fakeRegistrar) *AuthService {
	t.Helper()
	hasher, err := crypto.NewPasswordHasher(4) // минимальный cost для скорости тестов
	if err != nil {
		t.Fatalf("hasher: %v", err)
	}
	svc, err := NewAuthService(tokens, users, reg, hasher, time.Hour, testLogger())
	if err != nil {
		t.Fatalf("NewAuthService: %v", err)
	}
	return svc
}

func hashPw(t *testing.T, pw string) string {
	t.Helper()
	h, _ := crypto.NewPasswordHasher(4)
	out, err := h.Hash(pw)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	return out
}

func TestLogin(t *testing.T) {
	t.Parallel()
	cust, uid := uuid.New(), uuid.New()

	t.Run("success issues token", func(t *testing.T) {
		t.Parallel()
		users := &fakeUserFinder{user: &domain.CustomerUser{
			ID: uid, CustomerID: cust, Email: "o@ex.com", FullName: "Owner", Role: domain.UserRoleOwner,
			IsActive: true, PasswordHash: hashPw(t, "secret12"),
		}}
		tokens := &fakeTokenRepo{}
		svc := newTestAuth(t, tokens, users, &fakeRegistrar{})
		out, err := svc.Login(context.Background(), LoginInput{Email: "O@ex.com", Password: "secret12"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if out.Token == "" || out.CustomerID != cust || out.User.ID != uid {
			t.Errorf("bad output: %+v", out)
		}
		if tokens.created == nil || tokens.created.CustomerID != cust || tokens.created.UserID != uid {
			t.Errorf("token not persisted with tenant/user")
		}
	})

	t.Run("wrong password → invalid credentials", func(t *testing.T) {
		t.Parallel()
		users := &fakeUserFinder{user: &domain.CustomerUser{ID: uid, CustomerID: cust, IsActive: true, PasswordHash: hashPw(t, "secret12")}}
		svc := newTestAuth(t, &fakeTokenRepo{}, users, &fakeRegistrar{})
		_, err := svc.Login(context.Background(), LoginInput{Email: "o@ex.com", Password: "WRONG123"})
		if !errors.Is(err, domain.ErrInvalidCredentials) {
			t.Fatalf("want ErrInvalidCredentials, got %v", err)
		}
	})

	t.Run("unknown email → invalid credentials, no token", func(t *testing.T) {
		t.Parallel()
		tokens := &fakeTokenRepo{}
		svc := newTestAuth(t, tokens, &fakeUserFinder{err: domain.ErrNotFound}, &fakeRegistrar{})
		_, err := svc.Login(context.Background(), LoginInput{Email: "no@ex.com", Password: "secret12"})
		if !errors.Is(err, domain.ErrInvalidCredentials) {
			t.Fatalf("want ErrInvalidCredentials, got %v", err)
		}
		if tokens.created != nil {
			t.Error("no token should be created for unknown email")
		}
	})

	t.Run("disabled user → invalid credentials", func(t *testing.T) {
		t.Parallel()
		users := &fakeUserFinder{user: &domain.CustomerUser{ID: uid, CustomerID: cust, IsActive: false, PasswordHash: hashPw(t, "secret12")}}
		svc := newTestAuth(t, &fakeTokenRepo{}, users, &fakeRegistrar{})
		_, err := svc.Login(context.Background(), LoginInput{Email: "o@ex.com", Password: "secret12"})
		if !errors.Is(err, domain.ErrInvalidCredentials) {
			t.Fatalf("want ErrInvalidCredentials, got %v", err)
		}
	})
}

func TestAuthenticate(t *testing.T) {
	t.Parallel()
	base := func(active bool, revokedAt *time.Time, expires time.Time) *domain.AuthTokenWithUser {
		return &domain.AuthTokenWithUser{
			Token: domain.AuthToken{
				ID: uuid.New(), CustomerID: uuid.New(), UserID: uuid.New(),
				ExpiresAt: expires, RevokedAt: revokedAt,
			},
			UserRole:   domain.UserRoleAdmin,
			UserActive: active,
		}
	}
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	t.Run("valid → principal", func(t *testing.T) {
		t.Parallel()
		tok := base(true, nil, now.Add(time.Hour))
		svc := newTestAuth(t, &fakeTokenRepo{found: tok}, &fakeUserFinder{}, &fakeRegistrar{})
		svc.clock = fixedClock(now)
		p, err := svc.Authenticate(context.Background(), "raw")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.CustomerID != tok.Token.CustomerID || p.UserID != tok.Token.UserID || p.Role != domain.UserRoleAdmin {
			t.Errorf("bad principal: %+v", p)
		}
	})

	t.Run("revoked → ErrTokenRevoked", func(t *testing.T) {
		t.Parallel()
		rev := now.Add(-time.Minute)
		svc := newTestAuth(t, &fakeTokenRepo{found: base(true, &rev, now.Add(time.Hour))}, &fakeUserFinder{}, &fakeRegistrar{})
		svc.clock = fixedClock(now)
		if _, err := svc.Authenticate(context.Background(), "raw"); !errors.Is(err, domain.ErrTokenRevoked) {
			t.Fatalf("want ErrTokenRevoked, got %v", err)
		}
	})

	t.Run("expired → ErrTokenExpired", func(t *testing.T) {
		t.Parallel()
		svc := newTestAuth(t, &fakeTokenRepo{found: base(true, nil, now.Add(-time.Hour))}, &fakeUserFinder{}, &fakeRegistrar{})
		svc.clock = fixedClock(now)
		if _, err := svc.Authenticate(context.Background(), "raw"); !errors.Is(err, domain.ErrTokenExpired) {
			t.Fatalf("want ErrTokenExpired, got %v", err)
		}
	})

	t.Run("disabled user → ErrUserDisabled", func(t *testing.T) {
		t.Parallel()
		svc := newTestAuth(t, &fakeTokenRepo{found: base(false, nil, now.Add(time.Hour))}, &fakeUserFinder{}, &fakeRegistrar{})
		svc.clock = fixedClock(now)
		if _, err := svc.Authenticate(context.Background(), "raw"); !errors.Is(err, domain.ErrUserDisabled) {
			t.Fatalf("want ErrUserDisabled, got %v", err)
		}
	})

	t.Run("unknown hash → ErrTokenInvalid", func(t *testing.T) {
		t.Parallel()
		svc := newTestAuth(t, &fakeTokenRepo{findErr: domain.ErrNotFound}, &fakeUserFinder{}, &fakeRegistrar{})
		if _, err := svc.Authenticate(context.Background(), "raw"); !errors.Is(err, domain.ErrTokenInvalid) {
			t.Fatalf("want ErrTokenInvalid, got %v", err)
		}
	})

	t.Run("empty token → ErrUnauthenticated", func(t *testing.T) {
		t.Parallel()
		svc := newTestAuth(t, &fakeTokenRepo{}, &fakeUserFinder{}, &fakeRegistrar{})
		if _, err := svc.Authenticate(context.Background(), "   "); !errors.Is(err, domain.ErrUnauthenticated) {
			t.Fatalf("want ErrUnauthenticated, got %v", err)
		}
	})
}

func TestRegister(t *testing.T) {
	t.Parallel()

	t.Run("success creates OWNER tenant and issues token", func(t *testing.T) {
		t.Parallel()
		reg := &fakeRegistrar{}
		tokens := &fakeTokenRepo{}
		svc := newTestAuth(t, tokens, &fakeUserFinder{}, reg)
		out, err := svc.Register(context.Background(), RegisterInput{
			CompanyName: "Acme", FullName: "Jane Owner", Email: "jane@acme.com", Password: "supersecret",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if reg.gotUser == nil || reg.gotUser.Role != domain.UserRoleOwner {
			t.Errorf("owner not created with OWNER role")
		}
		if reg.gotCust == nil || reg.gotUser.CustomerID != reg.gotCust.ID {
			t.Errorf("owner not linked to new tenant")
		}
		if out.Token == "" || tokens.created == nil {
			t.Errorf("token not issued on register")
		}
	})

	t.Run("weak password → validation error", func(t *testing.T) {
		t.Parallel()
		svc := newTestAuth(t, &fakeTokenRepo{}, &fakeUserFinder{}, &fakeRegistrar{})
		_, err := svc.Register(context.Background(), RegisterInput{CompanyName: "A", FullName: "B C", Email: "a@b.com", Password: "short"})
		var verr domain.ValidationErrors
		if !errors.As(err, &verr) {
			t.Fatalf("want ValidationErrors, got %v", err)
		}
	})

	t.Run("duplicate email → ErrAlreadyExists", func(t *testing.T) {
		t.Parallel()
		svc := newTestAuth(t, &fakeTokenRepo{}, &fakeUserFinder{}, &fakeRegistrar{err: domain.ErrAlreadyExists})
		_, err := svc.Register(context.Background(), RegisterInput{CompanyName: "Acme", FullName: "Jane Owner", Email: "dup@acme.com", Password: "supersecret"})
		if !errors.Is(err, domain.ErrAlreadyExists) {
			t.Fatalf("want ErrAlreadyExists, got %v", err)
		}
	})
}
