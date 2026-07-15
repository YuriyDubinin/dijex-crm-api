package service

import (
	"context"
	"log/slog"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/YuriyDubinin/bms-api/internal/domain"
	"github.com/YuriyDubinin/bms-api/pkg/crypto"
)

// ───────────────────────── Service DTO ─────────────────────────

type CreateUserInput struct {
	Email    string
	FullName string
	Role     string
	Password string
	IsActive bool
}

type UpdateUserInput struct {
	ID       uuid.UUID
	Email    string
	FullName string
	Role     string
	IsActive bool
	Password string // пусто → пароль не менять
}

type ListUsersInput struct {
	Page     int
	PageSize int
	Role     string
	IsActive *bool
	Search   string
	SortBy   string
	Order    string
}

type UserItem struct {
	ID        uuid.UUID
	Email     string
	FullName  string
	Role      string
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type ListUsersOutput struct {
	Items      []UserItem
	Pagination Pagination
}

type DeleteUserOutput struct {
	ID        uuid.UUID
	DeletedAt time.Time
}

// ───────────────────────── Service ─────────────────────────

type CustomerUserService struct {
	repo      domain.CustomerUserRepository
	passwords *crypto.PasswordHasher
	logger    *slog.Logger
	clock     func() time.Time
}

func NewCustomerUserService(repo domain.CustomerUserRepository, passwords *crypto.PasswordHasher, logger *slog.Logger) *CustomerUserService {
	return &CustomerUserService{repo: repo, passwords: passwords, logger: logger, clock: time.Now}
}

func (s *CustomerUserService) CreateUser(ctx context.Context, customerID uuid.UUID, actorRole string, in CreateUserInput) (*UserItem, error) {
	if !domain.CanManageUsers(actorRole) {
		return nil, domain.ErrForbidden
	}
	in.Email = normalizeEmail(in.Email)
	in.FullName = strings.TrimSpace(in.FullName)
	in.Role = normalizeStatus(in.Role, domain.UserRoleStaff)

	errs := validateUser(in.Email, in.FullName, in.Role)
	if len(in.Password) < 8 {
		errs = append(errs, &domain.ValidationError{Field: "password", Message: "must be at least 8 characters"})
	}
	if len(errs) > 0 {
		return nil, errs
	}

	hash, err := s.passwords.Hash(in.Password)
	if err != nil {
		return nil, err
	}

	now := s.clock()
	u := &domain.CustomerUser{
		ID:           uuid.New(),
		CustomerID:   customerID,
		Email:        in.Email,
		FullName:     in.FullName,
		Role:         in.Role,
		IsActive:     in.IsActive,
		PasswordHash: hash,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.repo.Create(ctx, u); err != nil {
		return nil, err
	}
	s.logger.Info("user created", "user_id", u.ID, "customer_id", customerID, "role", u.Role)
	return userToItem(u), nil
}

func (s *CustomerUserService) UpdateUser(ctx context.Context, customerID uuid.UUID, actorRole string, in UpdateUserInput) (*UserItem, error) {
	if !domain.CanManageUsers(actorRole) {
		return nil, domain.ErrForbidden
	}
	in.Email = normalizeEmail(in.Email)
	in.FullName = strings.TrimSpace(in.FullName)
	in.Role = normalizeStatus(in.Role, domain.UserRoleStaff)

	errs := validateUser(in.Email, in.FullName, in.Role)
	if in.Password != "" && len(in.Password) < 8 {
		errs = append(errs, &domain.ValidationError{Field: "password", Message: "must be at least 8 characters"})
	}
	if len(errs) > 0 {
		return nil, errs
	}

	// Защита последнего владельца: нельзя понизить/деактивировать, если он один.
	target, err := s.repo.GetByID(ctx, customerID, in.ID)
	if err != nil {
		return nil, err
	}
	if target.Role == domain.UserRoleOwner && (in.Role != domain.UserRoleOwner || !in.IsActive) {
		owners, err := s.repo.CountOwners(ctx, customerID)
		if err != nil {
			return nil, err
		}
		if owners <= 1 {
			return nil, domain.ErrForbidden
		}
	}

	hash := ""
	if in.Password != "" {
		if hash, err = s.passwords.Hash(in.Password); err != nil {
			return nil, err
		}
	}

	u := &domain.CustomerUser{
		ID:           in.ID,
		CustomerID:   customerID,
		Email:        in.Email,
		FullName:     in.FullName,
		Role:         in.Role,
		IsActive:     in.IsActive,
		PasswordHash: hash, // пусто → репозиторий сохраняет текущий (COALESCE(NULLIF))
	}
	if err := s.repo.Update(ctx, customerID, u); err != nil {
		return nil, err
	}
	s.logger.Info("user updated", "user_id", u.ID, "customer_id", customerID)
	return userToItem(u), nil
}

func (s *CustomerUserService) ListUsers(ctx context.Context, customerID uuid.UUID, in ListUsersInput) (*ListUsersOutput, error) {
	role := normalizeStatus(in.Role, "")
	if role != "" && !domain.IsValidUserRole(role) {
		return nil, domain.ValidationErrors{{Field: "role", Message: "invalid role"}}
	}
	page, size, offset := clampPagination(in.Page, in.PageSize)
	filter := domain.CustomerUserListFilter{
		Role:     role,
		IsActive: in.IsActive,
		Search:   strings.TrimSpace(in.Search),
		Limit:    size,
		Offset:   offset,
		SortBy:   in.SortBy,
		SortDesc: strings.ToLower(in.Order) != "asc",
	}
	items, total, err := s.repo.List(ctx, customerID, filter)
	if err != nil {
		return nil, err
	}
	out := &ListUsersOutput{
		Items:      make([]UserItem, 0, len(items)),
		Pagination: Pagination{Page: page, PageSize: size, Total: total, TotalPages: totalPages(total, size)},
	}
	for _, u := range items {
		out.Items = append(out.Items, *userToItem(u))
	}
	return out, nil
}

func (s *CustomerUserService) DeleteUser(ctx context.Context, customerID uuid.UUID, actorRole string, id uuid.UUID) (*DeleteUserOutput, error) {
	if !domain.CanManageUsers(actorRole) {
		return nil, domain.ErrForbidden
	}
	target, err := s.repo.GetByID(ctx, customerID, id)
	if err != nil {
		return nil, err
	}
	if target.Role == domain.UserRoleOwner {
		owners, err := s.repo.CountOwners(ctx, customerID)
		if err != nil {
			return nil, err
		}
		if owners <= 1 {
			return nil, domain.ErrForbidden
		}
	}
	deletedAt, err := s.repo.SoftDelete(ctx, customerID, id)
	if err != nil {
		return nil, err
	}
	s.logger.Info("user deleted", "user_id", id, "customer_id", customerID)
	return &DeleteUserOutput{ID: id, DeletedAt: deletedAt}, nil
}

// ───────────────────────── helpers ─────────────────────────

func validateUser(email, fullName, role string) domain.ValidationErrors {
	var errs domain.ValidationErrors
	if email == "" {
		errs = append(errs, &domain.ValidationError{Field: "email", Message: "is required"})
	} else if _, err := mail.ParseAddress(email); err != nil {
		errs = append(errs, &domain.ValidationError{Field: "email", Message: "is not a valid email"})
	}
	if fullName == "" {
		errs = append(errs, &domain.ValidationError{Field: "full_name", Message: "is required"})
	} else if utf8.RuneCountInString(fullName) > 255 {
		errs = append(errs, &domain.ValidationError{Field: "full_name", Message: "must be at most 255 characters"})
	}
	if !domain.IsValidUserRole(role) {
		errs = append(errs, &domain.ValidationError{Field: "role", Message: "must be one of OWNER, ADMIN, MANAGER, STAFF"})
	}
	return errs
}

func userToItem(u *domain.CustomerUser) *UserItem {
	return &UserItem{
		ID:        u.ID,
		Email:     u.Email,
		FullName:  u.FullName,
		Role:      u.Role,
		IsActive:  u.IsActive,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}
