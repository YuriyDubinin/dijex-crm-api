package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/YuriyDubinin/bms-api/internal/service"
)

type CreateUserHTTPRequest struct {
	Email    string `json:"email"     validate:"required,email,max=255"`
	FullName string `json:"full_name" validate:"required,min=2,max=255"`
	Role     string `json:"role"      validate:"omitempty,oneof=OWNER ADMIN MANAGER STAFF"`
	Password string `json:"password"  validate:"required,min=8,max=72"`
	IsActive *bool  `json:"is_active"`
}

func (r CreateUserHTTPRequest) ToServiceInput() service.CreateUserInput {
	active := true
	if r.IsActive != nil {
		active = *r.IsActive
	}
	return service.CreateUserInput{
		Email:    r.Email,
		FullName: r.FullName,
		Role:     r.Role,
		Password: r.Password,
		IsActive: active,
	}
}

type UpdateUserHTTPRequest struct {
	ID       uuid.UUID `json:"id"        validate:"required"`
	Email    string    `json:"email"     validate:"required,email,max=255"`
	FullName string    `json:"full_name" validate:"required,min=2,max=255"`
	Role     string    `json:"role"      validate:"omitempty,oneof=OWNER ADMIN MANAGER STAFF"`
	Password string    `json:"password"  validate:"omitempty,min=8,max=72"`
	IsActive *bool     `json:"is_active"`
}

func (r UpdateUserHTTPRequest) ToServiceInput() service.UpdateUserInput {
	active := true
	if r.IsActive != nil {
		active = *r.IsActive
	}
	return service.UpdateUserInput{
		ID:       r.ID,
		Email:    r.Email,
		FullName: r.FullName,
		Role:     r.Role,
		Password: r.Password,
		IsActive: active,
	}
}

type UserHTTPResponse struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name"`
	Role      string    `json:"role"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func fromUserItem(u service.UserItem) UserHTTPResponse {
	return UserHTTPResponse{
		ID:        u.ID,
		Email:     u.Email,
		FullName:  u.FullName,
		Role:      u.Role,
		IsActive:  u.IsActive,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

func FromUserItem(u *service.UserItem) UserHTTPResponse {
	return fromUserItem(*u)
}

type ListUsersHTTPResponse struct {
	Items      []UserHTTPResponse `json:"items"`
	Pagination PaginationResponse `json:"pagination"`
}

func FromListUsersOutput(o *service.ListUsersOutput) ListUsersHTTPResponse {
	items := make([]UserHTTPResponse, 0, len(o.Items))
	for _, u := range o.Items {
		items = append(items, fromUserItem(u))
	}
	return ListUsersHTTPResponse{Items: items, Pagination: fromPagination(o.Pagination)}
}

type DeleteUserHTTPResponse struct {
	ID        uuid.UUID `json:"id"`
	DeletedAt time.Time `json:"deleted_at"`
}

func FromDeleteUserOutput(o *service.DeleteUserOutput) DeleteUserHTTPResponse {
	return DeleteUserHTTPResponse{ID: o.ID, DeletedAt: o.DeletedAt}
}
