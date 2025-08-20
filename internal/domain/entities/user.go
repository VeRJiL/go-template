package entities

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID        uuid.UUID `json:"id" db:"id"`
	Email     string    `json:"email" db:"email" validate:"required,email"`
	Password  string    `json:"-" db:"password_hash" validate:"required,min=8"`
	FirstName string    `json:"first_name" db:"first_name" validate:"required,min=1"`
	LastName  string    `json:"last_name" db:"last_name" validate:"required,min=1"`
	Role      string    `json:"role" db:"role" validate:"required,oneof=admin user"`
	IsActive  bool      `json:"is_active" db:"is_active"`

	// Image fields
	Avatar         *string `json:"avatar,omitempty" db:"avatar"`                   // URL to avatar image
	AvatarPath     *string `json:"avatar_path,omitempty" db:"avatar_path"`         // Storage path
	AvatarOriginal *string `json:"avatar_original,omitempty" db:"avatar_original"` // Original filename

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type CreateUserRequest struct {
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=8"`
	FirstName string `json:"first_name" validate:"required,min=1"`
	LastName  string `json:"last_name" validate:"required,min=1"`
	Role      string `json:"role" validate:"required,oneof=admin user"`
}

type UpdateUserRequest struct {
	FirstName *string `json:"first_name,omitempty" validate:"omitempty,min=1"`
	LastName  *string `json:"last_name,omitempty" validate:"omitempty,min=1"`
	Role      *string `json:"role,omitempty" validate:"omitempty,oneof=admin user"`
	IsActive  *bool   `json:"is_active,omitempty"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type LoginResponse struct {
	Token     string    `json:"token"`
	User      User      `json:"user"`
	ExpiresAt time.Time `json:"expires_at"`
}

func (u *User) BeforeCreate() {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	now := time.Now()
	u.CreatedAt = now
	u.UpdatedAt = now
	if u.Role == "" {
		u.Role = "user"
	}
	u.IsActive = true
}

func (u *User) BeforeUpdate() {
	u.UpdatedAt = time.Now()
}

// Avatar upload response types
type AvatarUploadResponse struct {
	Avatar         string `json:"avatar"`          // Public URL
	AvatarPath     string `json:"avatar_path"`     // Storage path
	AvatarOriginal string `json:"avatar_original"` // Original filename
	Message        string `json:"message"`
}

type AvatarDeleteResponse struct {
	Message string `json:"message"`
}
