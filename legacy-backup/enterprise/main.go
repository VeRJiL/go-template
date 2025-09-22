package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/VeRJiL/go-template/internal/config"
	"github.com/VeRJiL/go-template/internal/database/postgres"
	redisDB "github.com/VeRJiL/go-template/internal/database/redis"
	"github.com/VeRJiL/go-template/internal/modules"
	"github.com/VeRJiL/go-template/internal/pkg/auth"
	"github.com/VeRJiL/go-template/internal/pkg/bootstrap"
	"github.com/VeRJiL/go-template/internal/pkg/logger"
)

// Application represents the enterprise application
type Application struct {
	config     *config.Config
	logger     *logger.Logger
	db         *sql.DB
	redis      *redis.Client
	jwtService *auth.JWTService
	router     *gin.Engine
	server     *http.Server
	bootstrap  *bootstrap.EnterpriseBootstrap
}

// main is the application entry point
func main() {
	app := &Application{}

	// Initialize and run application
	if err := app.Initialize(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize application: %v\n", err)
		os.Exit(1)
	}

	// Start the application
	if err := app.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start application: %v\n", err)
		os.Exit(1)
	}

	// Wait for shutdown signal
	app.WaitForShutdown()
}

// Initialize initializes all application components
func (a *Application) Initialize() error {
	var err error

	// Load configuration
	a.config, err = config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize logger
	a.logger = logger.New(
		a.config.Logging.Level,
		a.config.Logging.Format,
	)

	a.logger.Info("Starting enterprise application",
		"name", a.config.App.Name,
		"version", a.config.App.Version)

	// Initialize database
	if err := a.initializeDatabase(); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	// Initialize Redis
	if err := a.initializeRedis(); err != nil {
		return fmt.Errorf("failed to initialize Redis: %w", err)
	}

	// Initialize JWT service
	if err := a.initializeJWTService(); err != nil {
		return fmt.Errorf("failed to initialize JWT service: %w", err)
	}

	// Initialize enterprise bootstrap
	if err := a.initializeEnterprise(); err != nil {
		return fmt.Errorf("failed to initialize enterprise: %w", err)
	}

	// Initialize HTTP router
	if err := a.initializeRouter(); err != nil {
		return fmt.Errorf("failed to initialize router: %w", err)
	}

	a.logger.Info("Application initialized successfully")
	return nil
}

// Start starts the HTTP server
func (a *Application) Start() error {
	// Create HTTP server
	a.server = &http.Server{
		Addr:         a.config.Server.Host + ":" + a.config.Server.Port,
		Handler:      a.router,
		ReadTimeout:  a.config.Server.ReadTimeout,
		WriteTimeout: a.config.Server.WriteTimeout,
		IdleTimeout:  a.config.Server.IdleTimeout,
	}

	a.logger.Info("Starting HTTP server",
		"address", a.server.Addr,
		"read_timeout", a.server.ReadTimeout,
		"write_timeout", a.server.WriteTimeout)

	// Start server in goroutine
	go func() {
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			a.logger.Error("HTTP server error", "error", err)
			os.Exit(1)
		}
	}()

	a.logger.Info("HTTP server started successfully", "port", a.config.Server.Port)
	return nil
}

// WaitForShutdown waits for shutdown signals and gracefully shuts down
func (a *Application) WaitForShutdown() {
	// Create channel to receive OS signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Wait for signal
	sig := <-sigChan
	a.logger.Info("Received shutdown signal", "signal", sig.String())

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown application
	if err := a.Shutdown(ctx); err != nil {
		a.logger.Error("Failed to shutdown gracefully", "error", err)
		os.Exit(1)
	}

	a.logger.Info("Application shutdown completed")
}

// Shutdown gracefully shuts down the application
func (a *Application) Shutdown(ctx context.Context) error {
	a.logger.Info("Starting graceful shutdown")

	// Shutdown HTTP server
	if a.server != nil {
		a.logger.Info("Shutting down HTTP server")
		if err := a.server.Shutdown(ctx); err != nil {
			a.logger.Error("Failed to shutdown HTTP server", "error", err)
		}
	}

	// Shutdown enterprise components
	if a.bootstrap != nil {
		if err := a.bootstrap.Shutdown(ctx); err != nil {
			a.logger.Error("Failed to shutdown enterprise components", "error", err)
		}
	}

	// Close database connections
	if a.db != nil {
		a.logger.Info("Closing database connection")
		if err := a.db.Close(); err != nil {
			a.logger.Error("Failed to close database", "error", err)
		}
	}

	// Close Redis connection
	if a.redis != nil {
		a.logger.Info("Closing Redis connection")
		if err := a.redis.Close(); err != nil {
			a.logger.Error("Failed to close Redis", "error", err)
		}
	}

	return nil
}

// Helper initialization methods

func (a *Application) initializeDatabase() error {
	var err error

	a.db, err = postgres.NewConnection(&a.config.Database)

	if err != nil {
		return err
	}

	// Configure connection pool
	a.db.SetMaxOpenConns(a.config.Database.MaxOpenConns)
	a.db.SetMaxIdleConns(a.config.Database.MaxIdleConns)
	a.db.SetConnMaxLifetime(a.config.Database.MaxConnLifetime)

	a.logger.Info("Database connection established",
		"host", a.config.Database.Host,
		"port", a.config.Database.Port,
		"database", a.config.Database.Database)

	return nil
}

func (a *Application) initializeRedis() error {
	// For now, always try to connect to Redis if config exists
	var err error
	a.redis, err = redisDB.NewConnection(&a.config.Redis)

	if err != nil {
		a.logger.Warn("Failed to connect to Redis, continuing without cache", "error", err)
		return nil // Don't fail if Redis is unavailable
	}

	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := a.redis.Ping(ctx).Err(); err != nil {
		a.logger.Warn("Redis ping failed, continuing without cache", "error", err)
		a.redis = nil
		return nil
	}

	a.logger.Info("Redis connection established",
		"host", a.config.Redis.Host,
		"port", a.config.Redis.Port,
		"db", a.config.Redis.DB)

	return nil
}

func (a *Application) initializeJWTService() error {
	a.jwtService = auth.NewJWTService(
		a.config.Auth.JWT.Secret,
		int(a.config.Auth.JWT.Expiration.Seconds()),
	)

	a.logger.Info("JWT service initialized",
		"access_ttl", a.config.Auth.JWT.Expiration,
		"issuer", a.config.Auth.JWT.Issuer)

	return nil
}

func (a *Application) initializeEnterprise() error {
	// Create enterprise bootstrap
	a.bootstrap = bootstrap.NewEnterpriseBootstrap(a.config, a.logger)

	// Initialize enterprise system
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := a.bootstrap.Initialize(ctx, a.db, a.redis, a.jwtService); err != nil {
		return err
	}

	// Register core modules
	if err := a.registerCoreModules(); err != nil {
		return err
	}

	// Run migrations
	if err := a.bootstrap.Migrate(ctx); err != nil {
		return err
	}

	a.logger.Info("Enterprise system initialized successfully")
	return nil
}

func (a *Application) registerCoreModules() error {
	// Register user module
	userModule := modules.NewUserModule()
	if err := a.bootstrap.RegisterModule(userModule); err != nil {
		return fmt.Errorf("failed to register user module: %w", err)
	}

	// Register product module (generated)
	productModule := modules.NewProductModule()
	if err := a.bootstrap.RegisterModule(productModule); err != nil {
		return fmt.Errorf("failed to register product module: %w", err)
	}

	return nil
}

func (a *Application) initializeRouter() error {
	// Set Gin mode
	if a.config.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create router
	a.router = gin.New()

	// Add global middleware
	a.addGlobalMiddleware()

	// Create API group
	apiGroup := a.router.Group("/api/v1")

	// Register enterprise routes
	if err := a.bootstrap.RegisterRoutes(apiGroup); err != nil {
		return err
	}

	// Add health check endpoints
	a.addHealthCheckRoutes()

	// Add administrative endpoints
	a.addAdminRoutes()

	a.logger.Info("HTTP router initialized successfully")
	return nil
}

func (a *Application) addGlobalMiddleware() {
	// Recovery middleware
	a.router.Use(gin.Recovery())

	// Logger middleware (basic gin logger for now)
	a.router.Use(gin.Logger())

	// Basic CORS middleware
	a.router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})
}

func (a *Application) addHealthCheckRoutes() {
	// Health check endpoint
	a.router.GET("/health", func(c *gin.Context) {
		health := a.bootstrap.HealthCheck(c.Request.Context())
		c.JSON(http.StatusOK, health)
	})

	// Readiness probe
	a.router.GET("/ready", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ready",
			"time":   time.Now().Unix(),
		})
	})

	// Liveness probe
	a.router.GET("/live", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "alive",
			"time":   time.Now().Unix(),
		})
	})
}

func (a *Application) addAdminRoutes() {
	adminGroup := a.router.Group("/admin")

	// Application stats
	adminGroup.GET("/stats", func(c *gin.Context) {
		stats := a.bootstrap.GetStats()
		c.JSON(http.StatusOK, stats)
	})

	// Module information
	adminGroup.GET("/modules", func(c *gin.Context) {
		modules := a.bootstrap.GetModuleInfo()
		c.JSON(http.StatusOK, gin.H{
			"modules": modules,
			"count":   len(modules),
		})
	})

	// Container services
	adminGroup.GET("/services", func(c *gin.Context) {
		container := a.bootstrap.GetContainer()
		services := container.GetServices()
		c.JSON(http.StatusOK, gin.H{
			"services": services,
			"count":    len(services),
		})
	})
}