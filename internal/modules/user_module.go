package modules

import (
	"context"
	"database/sql"

	"github.com/gin-gonic/gin"

	"github.com/VeRJiL/go-template/internal/api/handlers"
	"github.com/VeRJiL/go-template/internal/database/postgres"
	"github.com/VeRJiL/go-template/internal/database/redis"
	"github.com/VeRJiL/go-template/internal/domain/repositories"
	"github.com/VeRJiL/go-template/internal/domain/services"
	"github.com/VeRJiL/go-template/internal/pkg/auth"
	"github.com/VeRJiL/go-template/internal/pkg/container"
	"github.com/VeRJiL/go-template/internal/pkg/logger"
	"github.com/VeRJiL/go-template/internal/pkg/modules"
	redisLib "github.com/redis/go-redis/v9"
)

// UserModule implements the Module interface for user functionality
type UserModule struct {
	name         string
	version      string
	dependencies []string
	logger       *logger.Logger
}

// NewUserModule creates a new user module
func NewUserModule() modules.Module {
	return &UserModule{
		name:         "user",
		version:      "1.0.0",
		dependencies: []string{}, // No dependencies for core user module
	}
}

// Name returns the module name
func (m *UserModule) Name() string {
	return m.name
}

// Version returns the module version
func (m *UserModule) Version() string {
	return m.version
}

// Dependencies returns the module dependencies
func (m *UserModule) Dependencies() []string {
	return m.dependencies
}

// RegisterServices registers user module services with the container
func (m *UserModule) RegisterServices(c *container.Container) error {
	// Register user repository
	c.RegisterSingleton("userRepository", func(container *container.Container) interface{} {
		db := container.MustGet("db").(*sql.DB)
		return postgres.NewUserRepository(db)
	})

	// Register user cache repository (optional)
	c.RegisterSingleton("userCacheRepository", func(container *container.Container) interface{} {
		redisClient, err := container.Get("redis")
		if err != nil {
			// Redis not available, return nil
			return nil
		}
		return redis.NewUserCacheRepository(redisClient.(*redisLib.Client))
	})

	// Register user service
	c.RegisterSingleton("userService", func(container *container.Container) interface{} {
		userRepo := container.MustGet("userRepository").(repositories.UserRepository)
		jwtService := container.MustGet("jwtService").(*auth.JWTService)

		userService := services.NewUserService(userRepo, jwtService)

		// Set cache repository if available
		if cacheRepo, err := container.Get("userCacheRepository"); err == nil && cacheRepo != nil {
			userService.SetCacheRepository(cacheRepo.(repositories.UserCacheRepository))
		}

		return userService
	})

	// Register user handler
	c.RegisterSingleton("userHandler", func(container *container.Container) interface{} {
		userService := container.MustGet("userService").(*services.UserService)
		logger := container.MustGet("logger").(*logger.Logger)
		return handlers.NewUserHandler(userService, logger)
	})

	// Register user messaging service (for events and notifications)
	c.RegisterSingleton("userMessagingService", func(container *container.Container) interface{} {
		// Note: UserMessagingService doesn't exist yet, returning nil for now
		return nil
	})

	return nil
}

// RegisterRoutes registers user module routes
func (m *UserModule) RegisterRoutes(router *gin.RouterGroup, deps *modules.Dependencies) error {
	userHandler := deps.Container.MustGet("userHandler").(*handlers.UserHandler)

	// Create users group
	usersGroup := router.Group("/users")
	{
		// CRUD operations
		usersGroup.POST("", userHandler.Create)
		usersGroup.GET("", userHandler.List)
		usersGroup.GET("/:id", userHandler.GetByID)
		usersGroup.PUT("/:id", userHandler.Update)
		usersGroup.DELETE("/:id", userHandler.Delete)

		// User-specific operations
		usersGroup.POST("/login", userHandler.Login)
		usersGroup.POST("/logout", userHandler.Logout)
		usersGroup.GET("/profile", userHandler.GetProfile)

		// Search operations
		usersGroup.GET("/search", userHandler.Search)
	}

	// Auth routes (separate from users)
	authGroup := router.Group("/auth")
	{
		authGroup.POST("/login", userHandler.Login)
		authGroup.POST("/logout", userHandler.Logout)
	}

	deps.Logger.Info("User module routes registered successfully")
	return nil
}

// Migrate runs database migrations for the user module
func (m *UserModule) Migrate(db *sql.DB) error {
	// User table migration
	userTableSQL := `
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			email VARCHAR(255) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			first_name VARCHAR(100) NOT NULL,
			last_name VARCHAR(100) NOT NULL,
			is_active BOOLEAN DEFAULT true,
			is_verified BOOLEAN DEFAULT false,
			last_login_at BIGINT,
			created_at BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
			updated_at BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
			deleted_at BIGINT,

			-- Add indexes
			CONSTRAINT users_email_check CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$')
		);

		-- Create indexes for better performance
		CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
		CREATE INDEX IF NOT EXISTS idx_users_active ON users(is_active);
		CREATE INDEX IF NOT EXISTS idx_users_created_at ON users(created_at);
		CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at);
	`

	if _, err := db.Exec(userTableSQL); err != nil {
		return err
	}

	// User sessions table for JWT token management
	sessionTableSQL := `
		CREATE TABLE IF NOT EXISTS user_sessions (
			id SERIAL PRIMARY KEY,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			token_hash VARCHAR(255) NOT NULL,
			refresh_token_hash VARCHAR(255) NOT NULL,
			user_agent TEXT,
			ip_address INET,
			expires_at BIGINT NOT NULL,
			created_at BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),

			UNIQUE(token_hash),
			UNIQUE(refresh_token_hash)
		);

		-- Create indexes for sessions
		CREATE INDEX IF NOT EXISTS idx_user_sessions_user_id ON user_sessions(user_id);
		CREATE INDEX IF NOT EXISTS idx_user_sessions_token_hash ON user_sessions(token_hash);
		CREATE INDEX IF NOT EXISTS idx_user_sessions_expires_at ON user_sessions(expires_at);
	`

	if _, err := db.Exec(sessionTableSQL); err != nil {
		return err
	}

	// Password reset tokens table
	resetTokenTableSQL := `
		CREATE TABLE IF NOT EXISTS password_reset_tokens (
			id SERIAL PRIMARY KEY,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			token_hash VARCHAR(255) NOT NULL UNIQUE,
			expires_at BIGINT NOT NULL,
			used_at BIGINT,
			created_at BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),

			-- Ensure only one active token per user
			UNIQUE(user_id, token_hash)
		);

		-- Create indexes for reset tokens
		CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_user_id ON password_reset_tokens(user_id);
		CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_token_hash ON password_reset_tokens(token_hash);
		CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_expires_at ON password_reset_tokens(expires_at);
	`

	if _, err := db.Exec(resetTokenTableSQL); err != nil {
		return err
	}

	// Email verification tokens table
	verificationTableSQL := `
		CREATE TABLE IF NOT EXISTS email_verification_tokens (
			id SERIAL PRIMARY KEY,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			token_hash VARCHAR(255) NOT NULL UNIQUE,
			expires_at BIGINT NOT NULL,
			verified_at BIGINT,
			created_at BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())
		);

		-- Create indexes for verification tokens
		CREATE INDEX IF NOT EXISTS idx_email_verification_tokens_user_id ON email_verification_tokens(user_id);
		CREATE INDEX IF NOT EXISTS idx_email_verification_tokens_token_hash ON email_verification_tokens(token_hash);
	`

	if _, err := db.Exec(verificationTableSQL); err != nil {
		return err
	}

	return nil
}

// Initialize initializes the user module
func (m *UserModule) Initialize(ctx context.Context) error {
	m.logger = logger.New("info", "json")
	m.logger.Info("User module initialized", "version", m.version)

	// Add any module-specific initialization logic here
	// For example: connecting to external services, warming up caches, etc.

	return nil
}

// Shutdown gracefully shuts down the user module
func (m *UserModule) Shutdown(ctx context.Context) error {
	if m.logger != nil {
		m.logger.Info("User module shutting down")
	}

	// Add any module-specific cleanup logic here
	// For example: closing connections, flushing caches, etc.

	return nil
}

// GetModuleInfo returns detailed information about the user module
func (m *UserModule) GetModuleInfo() modules.ModuleInfo {
	return modules.ModuleInfo{
		Name:         m.name,
		Version:      m.version,
		Description:  "Core user management module with authentication, authorization, and profile management",
		Author:       "Go Template Team",
		Dependencies: m.dependencies,
		Routes: []modules.Route{
			{Method: "POST", Path: "/users", Handler: "Create", Auth: true, Permissions: []string{"admin"}},
			{Method: "GET", Path: "/users", Handler: "List", Auth: true, Permissions: []string{"admin", "user"}},
			{Method: "GET", Path: "/users/:id", Handler: "GetByID", Auth: true, Permissions: []string{"admin", "user"}},
			{Method: "PUT", Path: "/users/:id", Handler: "Update", Auth: true, Permissions: []string{"admin", "user"}},
			{Method: "DELETE", Path: "/users/:id", Handler: "Delete", Auth: true, Permissions: []string{"admin"}},
			{Method: "POST", Path: "/auth/register", Handler: "Register", Auth: false},
			{Method: "POST", Path: "/auth/login", Handler: "Login", Auth: false},
			{Method: "POST", Path: "/auth/logout", Handler: "Logout", Auth: true},
			{Method: "GET", Path: "/users/profile", Handler: "GetProfile", Auth: true},
		},
		Entities: []string{"User", "UserSession", "PasswordResetToken", "EmailVerificationToken"},
	}
}