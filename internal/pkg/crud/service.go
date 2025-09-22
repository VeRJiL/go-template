package crud

import (
	"context"
	"fmt"

	"github.com/VeRJiL/go-template/internal/pkg/modules"
)

// GenericService implements the Service interface for any entity
type GenericService[T modules.Entity] struct {
	repository modules.Repository[T]
	cache      modules.CacheRepository[T]
}

// NewGenericService creates a new generic service
func NewGenericService[T modules.Entity](repository modules.Repository[T]) *GenericService[T] {
	return &GenericService[T]{
		repository: repository,
	}
}

// SetCacheRepository sets the cache repository for the service
func (s *GenericService[T]) SetCacheRepository(cache modules.CacheRepository[T]) {
	s.cache = cache
}

// Create creates a new entity
func (s *GenericService[T]) Create(ctx context.Context, entity *T) (*T, error) {
	// Validate business rules before creation
	if err := s.validateBusinessRules(ctx, entity, "create"); err != nil {
		return nil, err
	}

	// Create in repository
	if err := s.repository.Create(ctx, entity); err != nil {
		return nil, fmt.Errorf("failed to create entity: %w", err)
	}

	// Invalidate cache if available
	if s.cache != nil {
		s.invalidateEntityCache(ctx, (*entity).GetID())
	}

	// Publish domain event
	s.publishEvent(ctx, "created", *entity)

	return entity, nil
}

// GetByID retrieves an entity by its ID
func (s *GenericService[T]) GetByID(ctx context.Context, id uint) (*T, error) {
	// Try cache first if available
	if s.cache != nil {
		cacheKey := s.buildCacheKey("entity", id)
		if entity, err := s.cache.Get(ctx, cacheKey); err == nil && entity != nil {
			return entity, nil
		}
	}

	// Get from repository
	entity, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Cache the result if available
	if s.cache != nil {
		cacheKey := s.buildCacheKey("entity", id)
		s.cache.Set(ctx, cacheKey, entity, 3600) // 1 hour TTL
	}

	return entity, nil
}

// Update updates an existing entity
func (s *GenericService[T]) Update(ctx context.Context, id uint, entity *T) (*T, error) {
	// Check if entity exists
	existing, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("entity not found: %w", err)
	}

	// Set the ID to ensure we're updating the correct entity
	(*entity).SetID(id)

	// Validate business rules before update
	if err := s.validateBusinessRules(ctx, entity, "update"); err != nil {
		return nil, err
	}

	// Update in repository
	if err := s.repository.Update(ctx, entity); err != nil {
		return nil, fmt.Errorf("failed to update entity: %w", err)
	}

	// Invalidate cache if available
	if s.cache != nil {
		s.invalidateEntityCache(ctx, id)
	}

	// Publish domain event
	s.publishEvent(ctx, "updated", map[string]interface{}{
		"old": existing,
		"new": entity,
	})

	return entity, nil
}

// Delete deletes an entity by its ID
func (s *GenericService[T]) Delete(ctx context.Context, id uint) error {
	// Check if entity exists
	entity, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("entity not found: %w", err)
	}

	// Validate business rules before deletion
	if err := s.validateBusinessRules(ctx, entity, "delete"); err != nil {
		return err
	}

	// Delete from repository
	if err := s.repository.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete entity: %w", err)
	}

	// Invalidate cache if available
	if s.cache != nil {
		s.invalidateEntityCache(ctx, id)
	}

	// Publish domain event
	s.publishEvent(ctx, "deleted", entity)

	return nil
}

// List retrieves entities with filtering and pagination
func (s *GenericService[T]) List(ctx context.Context, filters modules.ListFilters) ([]*T, int64, error) {
	// Try cache first if available
	if s.cache != nil && s.isCacheableListQuery(filters) {
		cacheKey := s.buildListCacheKey(filters)
		if entities, total, err := s.cache.GetList(ctx, cacheKey); err == nil && entities != nil {
			return entities, total, nil
		}
	}

	// Get from repository
	entities, total, err := s.repository.List(ctx, filters)
	if err != nil {
		return nil, 0, err
	}

	// Cache the result if available and cacheable
	if s.cache != nil && s.isCacheableListQuery(filters) {
		cacheKey := s.buildListCacheKey(filters)
		s.cache.SetList(ctx, cacheKey, entities, total, 1800) // 30 minutes TTL
	}

	return entities, total, nil
}

// Business logic validation (override in specific services)
func (s *GenericService[T]) validateBusinessRules(ctx context.Context, entity *T, operation string) error {
	// Default implementation - can be overridden in concrete services
	return nil
}

// Cache management methods

func (s *GenericService[T]) buildCacheKey(prefix string, id uint) string {
	var entity T
	return fmt.Sprintf("%s:%s:%d", entity.GetTableName(), prefix, id)
}

func (s *GenericService[T]) buildListCacheKey(filters modules.ListFilters) string {
	var entity T
	return fmt.Sprintf("%s:list:%d:%d:%s:%s",
		entity.GetTableName(),
		filters.Offset,
		filters.Limit,
		filters.SortBy,
		filters.Search)
}

func (s *GenericService[T]) invalidateEntityCache(ctx context.Context, id uint) {
	if s.cache == nil {
		return
	}

	// Invalidate specific entity cache
	cacheKey := s.buildCacheKey("entity", id)
	s.cache.Delete(ctx, cacheKey)

	// Invalidate list caches
	var entity T
	pattern := fmt.Sprintf("%s:list:*", entity.GetTableName())
	s.cache.Clear(ctx, pattern)
}

func (s *GenericService[T]) isCacheableListQuery(filters modules.ListFilters) bool {
	// Simple heuristic - cache queries without complex filters
	return len(filters.Filters) == 0 && filters.Search == ""
}

// Event publishing (placeholder - implement based on your event system)
func (s *GenericService[T]) publishEvent(ctx context.Context, eventType string, data interface{}) {
	// Placeholder for domain event publishing
	// In a real implementation, you would publish to an event bus
	// Example:
	// event := NewDomainEvent(entityType, eventType, data)
	// eventBus.Publish(ctx, event)
}

// Advanced service methods

// BulkCreate creates multiple entities in a transaction
func (s *GenericService[T]) BulkCreate(ctx context.Context, entities []*T) ([]*T, error) {
	for _, entity := range entities {
		if err := s.validateBusinessRules(ctx, entity, "create"); err != nil {
			return nil, fmt.Errorf("validation failed for entity: %w", err)
		}
	}

	// In a real implementation, you would use database transactions
	var created []*T
	for _, entity := range entities {
		if err := s.repository.Create(ctx, entity); err != nil {
			return nil, fmt.Errorf("failed to create entity: %w", err)
		}
		created = append(created, entity)
	}

	// Invalidate cache
	if s.cache != nil {
		var entity T
		pattern := fmt.Sprintf("%s:*", entity.GetTableName())
		s.cache.Clear(ctx, pattern)
	}

	// Publish events
	for _, entity := range created {
		s.publishEvent(ctx, "created", *entity)
	}

	return created, nil
}

// BulkUpdate updates multiple entities
func (s *GenericService[T]) BulkUpdate(ctx context.Context, updates map[uint]*T) error {
	for id, entity := range updates {
		(*entity).SetID(id)
		if err := s.validateBusinessRules(ctx, entity, "update"); err != nil {
			return fmt.Errorf("validation failed for entity %d: %w", id, err)
		}
	}

	// Perform updates
	for id, entity := range updates {
		if err := s.repository.Update(ctx, entity); err != nil {
			return fmt.Errorf("failed to update entity %d: %w", id, err)
		}

		// Invalidate cache
		if s.cache != nil {
			s.invalidateEntityCache(ctx, id)
		}

		s.publishEvent(ctx, "updated", entity)
	}

	return nil
}

// Count returns the total count of entities with optional filters
func (s *GenericService[T]) Count(ctx context.Context, filters modules.ListFilters) (int64, error) {
	// Set limit to 0 to get only count
	filters.Limit = 0
	_, total, err := s.repository.List(ctx, filters)
	return total, err
}

// Exists checks if an entity exists by ID
func (s *GenericService[T]) Exists(ctx context.Context, id uint) (bool, error) {
	return s.repository.Exists(ctx, id)
}

// Refresh invalidates cache for a specific entity
func (s *GenericService[T]) Refresh(ctx context.Context, id uint) error {
	if s.cache != nil {
		s.invalidateEntityCache(ctx, id)
	}
	return nil
}

// GetMultiple retrieves multiple entities by their IDs
func (s *GenericService[T]) GetMultiple(ctx context.Context, ids []uint) ([]*T, error) {
	var entities []*T

	for _, id := range ids {
		entity, err := s.GetByID(ctx, id)
		if err != nil {
			// Continue with other entities, just log the error
			continue
		}
		entities = append(entities, entity)
	}

	return entities, nil
}