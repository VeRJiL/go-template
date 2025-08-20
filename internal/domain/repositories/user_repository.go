package repositories

import (
	"context"

	"github.com/VeRJiL/go-template/internal/domain/entities"
	"github.com/google/uuid"
)

type UserRepository interface {
	Create(ctx context.Context, user *entities.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.User, error)
	GetByEmail(ctx context.Context, email string) (*entities.User, error)
	Update(ctx context.Context, id uuid.UUID, updates *entities.UpdateUserRequest) (*entities.User, error)
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, offset, limit int) ([]*entities.User, int, error)
	Search(ctx context.Context, query string, offset, limit int) ([]*entities.User, int, error)
}

type UserCacheRepository interface {
	Set(ctx context.Context, key string, user *entities.User) error
	Get(ctx context.Context, key string) (*entities.User, error)
	Delete(ctx context.Context, key string) error
	SetJSON(ctx context.Context, key string, data interface{}) error
	GetJSON(ctx context.Context, key string, dest interface{}) error
	DeletePattern(ctx context.Context, pattern string) error
	SetSession(ctx context.Context, token string, userID uuid.UUID) error
	GetSession(ctx context.Context, token string) (uuid.UUID, error)
	DeleteSession(ctx context.Context, token string) error
}
