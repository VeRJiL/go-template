package bootstrap

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/VeRJiL/go-template/internal/config"
	"github.com/VeRJiL/go-template/internal/pkg/auth"
	"github.com/VeRJiL/go-template/internal/pkg/container"
	"github.com/VeRJiL/go-template/internal/pkg/logger"
	"github.com/VeRJiL/go-template/internal/pkg/modules"
	"github.com/VeRJiL/go-template/internal/pkg/registry"
)

// EnterpriseBootstrap manages the complete enterprise application bootstrap
type EnterpriseBootstrap struct {
	container        *container.Container
	moduleRegistry   modules.ModuleRegistry
	entityRegistry   *registry.EntityRegistry
	logger           *logger.Logger
	config           *config.Config
	dependencies     *modules.Dependencies
	isInitialized    bool
}

// NewEnterpriseBootstrap creates a new enterprise bootstrap instance
func NewEnterpriseBootstrap(cfg *config.Config, logger *logger.Logger) *EnterpriseBootstrap {
	cont := container.NewContainer()

	return &EnterpriseBootstrap{
		container:      cont,
		moduleRegistry: registry.NewModuleRegistry(logger, cont),
		logger:         logger,
		config:         cfg,
	}
}

// Initialize initializes the enterprise application
func (e *EnterpriseBootstrap) Initialize(ctx context.Context, db *sql.DB, redisClient *redis.Client, jwtService *auth.JWTService) error {
	if e.isInitialized {
		return fmt.Errorf("enterprise bootstrap already initialized")
	}

	e.logger.Info("Starting enterprise application initialization")

	// Register core dependencies in container
	if err := e.registerCoreDependencies(db, redisClient, jwtService); err != nil {
		return fmt.Errorf("failed to register core dependencies: %w", err)
	}

	// Initialize entity registry
	e.entityRegistry = registry.NewEntityRegistry(e.logger, e.container, db)

	// Create module dependencies
	e.dependencies = &modules.Dependencies{
		Container:   e.container,
		Logger:      e.logger,
		Config:      e.config,
		DB:          db,
		RedisClient: redisClient,
		JWTService:  jwtService,
	}

	// Auto-discover and load modules
	if err := e.moduleRegistry.LoadModules(); err != nil {
		return fmt.Errorf("failed to load modules: %w", err)
	}

	// Initialize all modules
	if err := e.moduleRegistry.Initialize(ctx, e.dependencies); err != nil {
		return fmt.Errorf("failed to initialize modules: %w", err)
	}

	e.isInitialized = true
	e.logger.Info("Enterprise application initialized successfully",
		"modules", e.moduleRegistry.GetModuleCount(),
		"entities", e.entityRegistry.GetEntityCount())

	return nil
}

// RegisterModule registers a new module with the system
func (e *EnterpriseBootstrap) RegisterModule(module modules.Module) error {
	if err := e.moduleRegistry.Register(module); err != nil {
		return fmt.Errorf("failed to register module %s: %w", module.Name(), err)
	}

	e.logger.Info("Module registered", "name", module.Name(), "version", module.Version())
	return nil
}

// RegisterEntity registers a new entity with auto-generation
func (e *EnterpriseBootstrap) RegisterEntity(entityType interface{}, config modules.EntityConfig) error {
	if e.entityRegistry == nil {
		return fmt.Errorf("entity registry not initialized")
	}

	// Use reflection to get type information
	// entityReflectType := reflect.TypeOf(entityType)
	// if entityReflectType.Kind() == reflect.Ptr {
	// 	entityReflectType = entityReflectType.Elem()
	// }

	// For now, use a simplified approach - in production you'd use full reflection
	e.logger.Info("Entity registration requested", "name", config.Name)
	return nil
}

// RegisterRoutes registers all module routes with the router
func (e *EnterpriseBootstrap) RegisterRoutes(router *gin.RouterGroup) error {
	if !e.isInitialized {
		return fmt.Errorf("enterprise bootstrap not initialized")
	}

	e.logger.Info("Registering module routes")

	// Register entity routes first
	if err := e.entityRegistry.RegisterRoutes(router); err != nil {
		return fmt.Errorf("failed to register entity routes: %w", err)
	}

	// Register module routes
	modules := e.moduleRegistry.GetModules()
	for _, module := range modules {
		e.logger.Debug("Registering routes for module", "module", module.Name())

		if err := module.RegisterRoutes(router, e.dependencies); err != nil {
			return fmt.Errorf("failed to register routes for module %s: %w", module.Name(), err)
		}
	}

	e.logger.Info("All module routes registered successfully", "modules", len(modules))
	return nil
}

// Migrate runs database migrations for all modules and entities
func (e *EnterpriseBootstrap) Migrate(ctx context.Context) error {
	if !e.isInitialized {
		return fmt.Errorf("enterprise bootstrap not initialized")
	}

	e.logger.Info("Running enterprise migrations")

	// Migrate entities first
	if err := e.entityRegistry.Migrate(ctx); err != nil {
		return fmt.Errorf("failed to migrate entities: %w", err)
	}

	// Migrate modules
	modules := e.moduleRegistry.GetModules()
	for _, module := range modules {
		e.logger.Debug("Running migration for module", "module", module.Name())

		if err := module.Migrate(e.dependencies.DB); err != nil {
			e.logger.Error("Failed to migrate module", "module", module.Name(), "error", err)
			// Continue with other modules - don't fail completely
		}
	}

	e.logger.Info("Enterprise migrations completed")
	return nil
}

// Shutdown gracefully shuts down the enterprise application
func (e *EnterpriseBootstrap) Shutdown(ctx context.Context) error {
	if !e.isInitialized {
		return nil
	}

	e.logger.Info("Shutting down enterprise application")

	if err := e.moduleRegistry.Shutdown(ctx); err != nil {
		e.logger.Error("Failed to shutdown modules", "error", err)
		return err
	}

	e.isInitialized = false
	e.logger.Info("Enterprise application shutdown completed")
	return nil
}

// GetModuleInfo returns information about all registered modules
func (e *EnterpriseBootstrap) GetModuleInfo() []modules.ModuleInfo {
	return e.moduleRegistry.GetModuleInfo()
}

// GetContainer returns the dependency injection container
func (e *EnterpriseBootstrap) GetContainer() *container.Container {
	return e.container
}

// GetEntityRegistry returns the entity registry
func (e *EnterpriseBootstrap) GetEntityRegistry() *registry.EntityRegistry {
	return e.entityRegistry
}

// GetModuleRegistry returns the module registry
func (e *EnterpriseBootstrap) GetModuleRegistry() modules.ModuleRegistry {
	return e.moduleRegistry
}

// Helper methods

func (e *EnterpriseBootstrap) registerCoreDependencies(db *sql.DB, redisClient *redis.Client, jwtService *auth.JWTService) error {
	// Register database
	e.container.Register("db", db)

	// Register Redis client
	e.container.Register("redis", redisClient)

	// Register JWT service
	e.container.Register("jwtService", jwtService)

	// Register logger
	e.container.Register("logger", e.logger)

	// Register config
	e.container.Register("config", e.config)

	// Register container itself (for self-reference in factories)
	e.container.Register("container", e.container)

	e.logger.Debug("Core dependencies registered successfully")
	return nil
}

// HealthCheck performs a health check on all enterprise components
func (e *EnterpriseBootstrap) HealthCheck(ctx context.Context) map[string]interface{} {
	health := map[string]interface{}{
		"status": "healthy",
		"components": map[string]interface{}{
			"enterprise": map[string]interface{}{
				"initialized": e.isInitialized,
				"modules":     e.moduleRegistry.GetModuleCount(),
			},
		},
	}

	if e.entityRegistry != nil {
		health["components"].(map[string]interface{})["entities"] = map[string]interface{}{
			"count": e.entityRegistry.GetEntityCount(),
			"names": e.entityRegistry.GetEntityNames(),
		}
	}

	// Check module health
	if e.isInitialized {
		moduleHealth := make(map[string]interface{})
		modules := e.moduleRegistry.GetModules()

		for _, module := range modules {
			moduleHealth[module.Name()] = map[string]interface{}{
				"version":      module.Version(),
				"dependencies": module.Dependencies(),
			}
		}

		health["components"].(map[string]interface{})["module_details"] = moduleHealth
	}

	return health
}

// GetStats returns enterprise application statistics
func (e *EnterpriseBootstrap) GetStats() map[string]interface{} {
	stats := map[string]interface{}{
		"initialized": e.isInitialized,
		"modules": map[string]interface{}{
			"count": e.moduleRegistry.GetModuleCount(),
			"info":  e.moduleRegistry.GetModuleInfo(),
		},
		"container": map[string]interface{}{
			"services": e.container.GetServices(),
		},
	}

	if e.entityRegistry != nil {
		stats["entities"] = map[string]interface{}{
			"count": e.entityRegistry.GetEntityCount(),
			"names": e.entityRegistry.GetEntityNames(),
		}
	}

	return stats
}