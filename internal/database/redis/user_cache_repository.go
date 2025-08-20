package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/VeRJiL/go-template/internal/domain/entities"
	"github.com/VeRJiL/go-template/internal/domain/repositories"
)

type userCacheRepository struct {
	client *redis.Client
	ttl    time.Duration
}

func NewUserCacheRepository(client *redis.Client) repositories.UserCacheRepository {
	return &userCacheRepository{
		client: client,
		ttl:    time.Hour * 24,
	}
}

func (r *userCacheRepository) Set(ctx context.Context, key string, user *entities.User) error {
	data, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("failed to marshal user: %w", err)
	}

	return r.client.Set(ctx, key, data, r.ttl).Err()
}

func (r *userCacheRepository) Get(ctx context.Context, key string) (*entities.User, error) {
	data, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("user not found in cache")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user from cache: %w", err)
	}

	var user entities.User
	if err := json.Unmarshal([]byte(data), &user); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user: %w", err)
	}

	return &user, nil
}

func (r *userCacheRepository) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}

func (r *userCacheRepository) SetSession(ctx context.Context, token string, userID uuid.UUID) error {
	key := fmt.Sprintf("session:%s", token)
	return r.client.Set(ctx, key, userID.String(), time.Hour*24).Err()
}

func (r *userCacheRepository) GetSession(ctx context.Context, token string) (uuid.UUID, error) {
	key := fmt.Sprintf("session:%s", token)
	userIDStr, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return uuid.Nil, fmt.Errorf("session not found")
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to get session: %w", err)
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid user ID in session: %w", err)
	}

	return userID, nil
}

func (r *userCacheRepository) DeleteSession(ctx context.Context, token string) error {
	key := fmt.Sprintf("session:%s", token)
	return r.client.Del(ctx, key).Err()
}

func (r *userCacheRepository) SetJSON(ctx context.Context, key string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	return r.client.Set(ctx, key, jsonData, r.ttl).Err()
}

func (r *userCacheRepository) GetJSON(ctx context.Context, key string, dest interface{}) error {
	data, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return fmt.Errorf("data not found in cache")
	}
	if err != nil {
		return fmt.Errorf("failed to get data from cache: %w", err)
	}

	if err := json.Unmarshal([]byte(data), dest); err != nil {
		return fmt.Errorf("failed to unmarshal data: %w", err)
	}

	return nil
}

func (r *userCacheRepository) DeletePattern(ctx context.Context, pattern string) error {
	keys, err := r.client.Keys(ctx, pattern).Result()
	if err != nil {
		return fmt.Errorf("failed to find keys with pattern %s: %w", pattern, err)
	}

	if len(keys) == 0 {
		return nil
	}

	return r.client.Del(ctx, keys...).Err()
}
