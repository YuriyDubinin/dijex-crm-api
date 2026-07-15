package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/YuriyDubinin/bms-api/internal/domain"
	"github.com/YuriyDubinin/bms-api/pkg/crypto"
)

const touchLastUsedTimeout = 2 * time.Second

// dummyPasswordPlaceholder — строка, по которой при инициализации считается
// bcrypt-хэш. Используется для constant-time сравнения в ветке «email не
// найден» (защита от перечисления пользователей по таймингу).
const dummyPasswordPlaceholder = "dummy-password-for-constant-time-bcrypt-comparison"

// authUserRepo — узкий контракт, нужный auth-сервису от репозитория операторов.
type authUserRepo interface {
	FindByEmail(ctx context.Context, email string) (*domain.CustomerUser, error)
}

// customerRegistrar — узкий контракт для публичной регистрации тенанта.
type customerRegistrar interface {
	RegisterTenant(ctx context.Context, c *domain.Customer, owner *domain.CustomerUser) error
}

type AuthService struct {
	tokens    domain.TokenRepository
	users     authUserRepo
	customers customerRegistrar
	passwords *crypto.PasswordHasher
	tokenTTL  time.Duration
	logger    *slog.Logger
	clock     func() time.Time

	dummyPasswordHash string
}

func NewAuthService(
	tokens domain.TokenRepository,
	users authUserRepo,
	customers customerRegistrar,
	passwords *crypto.PasswordHasher,
	tokenTTL time.Duration,
	logger *slog.Logger,
) (*AuthService, error) {
	dummyHash, err := passwords.Hash(dummyPasswordPlaceholder)
	if err != nil {
		return nil, fmt.Errorf("auth: precompute dummy password hash: %w", err)
	}
	return &AuthService{
		tokens:            tokens,
		users:             users,
		customers:         customers,
		passwords:         passwords,
		tokenTTL:          tokenTTL,
		logger:            logger,
		clock:             time.Now,
		dummyPasswordHash: dummyHash,
	}, nil
}

// Authenticate проверяет сырой Bearer-токен. Хэширует и ищет по SHA-256 хэшу,
// затем проверяет отзыв/истечение/активность владельца. Возвращает Principal
// с CustomerID тенанта либо доменную auth-ошибку.
func (s *AuthService) Authenticate(ctx context.Context, rawToken string) (*domain.Principal, error) {
	raw := strings.TrimSpace(rawToken)
	if raw == "" {
		return nil, domain.ErrUnauthenticated
	}

	hash := crypto.HashToken(raw)

	row, err := s.tokens.FindActiveByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrTokenInvalid
		}
		return nil, fmt.Errorf("auth: find token: %w", err)
	}

	now := s.clock()
	switch {
	case row.Token.RevokedAt != nil:
		return nil, domain.ErrTokenRevoked
	case !row.Token.IsActive(now):
		return nil, domain.ErrTokenExpired
	case !row.UserActive:
		// Страховка на случай, если триггер отзыва токенов не сработал.
		return nil, domain.ErrUserDisabled
	}

	s.touchLastUsedAsync(row.Token.ID, now)

	return &domain.Principal{
		CustomerID: row.Token.CustomerID,
		UserID:     row.Token.UserID,
		Role:       row.UserRole,
		TokenID:    row.Token.ID,
	}, nil
}

// Logout отзывает токен по id (идемпотентно).
func (s *AuthService) Logout(ctx context.Context, tokenID uuid.UUID) error {
	if err := s.tokens.Revoke(ctx, tokenID, "user logout", s.clock()); err != nil {
		return fmt.Errorf("auth: logout: %w", err)
	}
	s.logger.Info("logout", "token_id", tokenID)
	return nil
}

// Login проверяет email+пароль и выдаёт новый ACCESS-токен. Все негативные
// исходы возвращают единый ErrInvalidCredentials (защита от перечисления).
func (s *AuthService) Login(ctx context.Context, in LoginInput) (*LoginOutput, error) {
	email := normalizeEmail(in.Email)
	if email == "" || in.Password == "" {
		return nil, domain.ErrInvalidCredentials
	}

	u, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			_ = s.passwords.Verify(in.Password, s.dummyPasswordHash) // constant-time
			return nil, domain.ErrInvalidCredentials
		}
		return nil, fmt.Errorf("auth: find user by email: %w", err)
	}

	if err := s.passwords.Verify(in.Password, u.PasswordHash); err != nil {
		if errors.Is(err, crypto.ErrPasswordMismatch) {
			return nil, domain.ErrInvalidCredentials
		}
		return nil, fmt.Errorf("auth: verify password: %w", err)
	}

	if !u.IsActive {
		return nil, domain.ErrInvalidCredentials
	}

	rawToken, expiresAt, err := s.issueToken(ctx, u.CustomerID, u.ID, in.ClientInfo)
	if err != nil {
		return nil, err
	}

	s.logger.Info("login success", "customer_id", u.CustomerID, "user_id", u.ID)

	return &LoginOutput{
		Token:      rawToken,
		TokenType:  "Bearer",
		ExpiresAt:  expiresAt,
		CustomerID: u.CustomerID,
		User: LoginUser{
			ID:       u.ID,
			FullName: u.FullName,
			Email:    u.Email,
			Role:     u.Role,
		},
	}, nil
}

// Register создаёт новый тенант вместе с первым пользователем-владельцем и
// сразу выдаёт токен (как при логине). Конфликт email → ErrAlreadyExists.
func (s *AuthService) Register(ctx context.Context, in RegisterInput) (*LoginOutput, error) {
	companyName := strings.TrimSpace(in.CompanyName)
	fullName := strings.TrimSpace(in.FullName)
	email := normalizeEmail(in.Email)

	var errs domain.ValidationErrors
	if companyName == "" {
		errs = append(errs, &domain.ValidationError{Field: "company_name", Message: "is required"})
	}
	if fullName == "" {
		errs = append(errs, &domain.ValidationError{Field: "full_name", Message: "is required"})
	}
	if email == "" {
		errs = append(errs, &domain.ValidationError{Field: "email", Message: "is required"})
	}
	if len(in.Password) < 8 {
		errs = append(errs, &domain.ValidationError{Field: "password", Message: "must be at least 8 characters"})
	}
	if len(errs) > 0 {
		return nil, errs
	}

	passwordHash, err := s.passwords.Hash(in.Password)
	if err != nil {
		return nil, fmt.Errorf("auth: hash password: %w", err)
	}

	customer := &domain.Customer{ID: uuid.New(), Name: companyName, Email: email}
	owner := &domain.CustomerUser{
		ID:           uuid.New(),
		CustomerID:   customer.ID,
		Email:        email,
		FullName:     fullName,
		Role:         domain.UserRoleOwner,
		IsActive:     true,
		PasswordHash: passwordHash,
	}
	if err := s.customers.RegisterTenant(ctx, customer, owner); err != nil {
		return nil, err // в т.ч. ErrAlreadyExists (email занят)
	}

	rawToken, expiresAt, err := s.issueToken(ctx, customer.ID, owner.ID, in.ClientInfo)
	if err != nil {
		return nil, err
	}

	s.logger.Info("tenant registered", "customer_id", customer.ID, "owner_id", owner.ID)

	return &LoginOutput{
		Token:      rawToken,
		TokenType:  "Bearer",
		ExpiresAt:  expiresAt,
		CustomerID: customer.ID,
		User:       LoginUser{ID: owner.ID, FullName: owner.FullName, Email: owner.Email, Role: owner.Role},
	}, nil
}

// issueToken генерирует opaque-токен, сохраняет его хэш и возвращает сырой
// токен клиенту (единожды) вместе со сроком истечения.
func (s *AuthService) issueToken(ctx context.Context, customerID, userID uuid.UUID, ci ClientInfo) (string, time.Time, error) {
	rawToken, err := crypto.GenerateOpaqueToken(crypto.DefaultTokenByteLength)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("auth: generate token: %w", err)
	}
	now := s.clock()
	token := &domain.AuthToken{
		ID:             uuid.New(),
		CustomerID:     customerID,
		UserID:         userID,
		TokenHash:      crypto.HashToken(rawToken),
		TokenType:      domain.TokenTypeAccess,
		IssuedAt:       now,
		ExpiresAt:      now.Add(s.tokenTTL),
		IPAddress:      ci.IPAddress,
		UserAgent:      ci.UserAgent,
		DeviceType:     ci.DeviceType,
		DeviceName:     ci.DeviceName,
		OS:             ci.OS,
		OSVersion:      ci.OSVersion,
		Browser:        ci.Browser,
		BrowserVersion: ci.BrowserVersion,
		AppVersion:     ci.AppVersion,
	}
	if err := s.tokens.Create(ctx, token); err != nil {
		return "", time.Time{}, fmt.Errorf("auth: persist token: %w", err)
	}
	return rawToken, token.ExpiresAt, nil
}

func (s *AuthService) touchLastUsedAsync(id uuid.UUID, now time.Time) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), touchLastUsedTimeout)
		defer cancel()
		if err := s.tokens.TouchLastUsed(ctx, id, now); err != nil {
			s.logger.Warn("touch auth token last_used_at", "err", err, "token_id", id)
		}
	}()
}

func normalizeEmail(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
