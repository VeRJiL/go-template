package app

import (
	"context"
	"database/sql"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/errgroup"

	"github.com/VeRJiL/go-template/internal/api/handlers"
	"github.com/VeRJiL/go-template/internal/api/middleware"
	"github.com/VeRJiL/go-template/internal/api/routes"
	"github.com/VeRJiL/go-template/internal/config"
	"github.com/VeRJiL/go-template/internal/database/postgres"
	redisRepo "github.com/VeRJiL/go-template/internal/database/redis"
	"github.com/VeRJiL/go-template/internal/domain/repositories"
	"github.com/VeRJiL/go-template/internal/domain/services"
	"github.com/VeRJiL/go-template/internal/pkg/auth"
	"github.com/VeRJiL/go-template/internal/pkg/logger"
)

type App struct {
	config      *config.Config
	db          *sql.DB
	redisClient *redis.Client
	router      *gin.Engine
	server      *http.Server
	jwtService  *auth.JWTService
	logger      *logger.Logger
}

func New() (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	log := logger.New(cfg.Logging.Level, cfg.Logging.Format)

	app := &App{
		config: cfg,
		logger: log,
	}

	// Initialize dependencies
	if err := app.initDependencies(); err != nil {
		return nil, err
	}

	app.setupRouter()

	return app, nil
}

func (a *App) initDependencies() error {
	db, err := postgres.NewConnection(&a.config.Database)
	if err != nil {
		return err
	}
	a.db = db

	a.redisClient = redis.NewClient(&redis.Options{
		Addr:         a.config.Redis.Host + ":" + a.config.Redis.Port,
		Password:     a.config.Redis.Password,
		DB:           a.config.Redis.DB,
		PoolSize:     a.config.Redis.PoolSize,
		MinIdleConns: a.config.Redis.MinIdleConns,
		DialTimeout:  a.config.Redis.DialTimeout,
		ReadTimeout:  a.config.Redis.ReadTimeout,
		WriteTimeout: a.config.Redis.WriteTimeout,
	})

	ctx := context.Background()
	if err := a.redisClient.Ping(ctx).Err(); err != nil {
		a.logger.Warn("Redis connection failed, caching will be disabled", "error", err)
		a.redisClient = nil
	} else {
		a.logger.Info("Redis connection established successfully")
	}

	a.jwtService = auth.NewJWTService(
		a.config.Auth.JWT.Secret,
		int(a.config.Auth.JWT.Expiration.Seconds()),
	)

	return nil
}

func (a *App) setupRouter() {
	if a.config.Server.Mode == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	a.router = gin.New()

	a.router.Use(gin.Recovery())
	a.router.Use(middleware.Logger(a.logger))
	a.router.Use(middleware.CORS(&a.config.Server))
	a.router.Use(middleware.Security())

	userRepo := postgres.NewUserRepository(a.db)

	var userCacheRepo repositories.UserCacheRepository
	if a.redisClient != nil {
		userCacheRepo = redisRepo.NewUserCacheRepository(a.redisClient)
	}

	userService := services.NewUserService(userRepo, a.jwtService)
	userService.SetCacheRepository(userCacheRepo)

	userHandler := handlers.NewUserHandler(userService, a.logger)

	routes.SetupRoutes(a.router, &routes.Dependencies{
		UserHandler: userHandler,
		JWTService:  a.jwtService,
		Logger:      a.logger,
		Config:      a.config,
	})
}

func (a *App) Run() error {
	a.server = &http.Server{
		Addr:         a.config.Server.Host + ":" + a.config.Server.Port,
		Handler:      a.router,
		ReadTimeout:  a.config.Server.ReadTimeout,
		WriteTimeout: a.config.Server.WriteTimeout,
		IdleTimeout:  a.config.Server.IdleTimeout,
	}

	g, ctx := errgroup.WithContext(context.Background())

	g.Go(func() error {
		a.logger.Info("Starting HTTP server", "address", a.server.Addr)
		a.logger.Info("🌐 Server running at: http://"+a.server.Addr)
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return err
		}
		return nil
	})

	g.Go(func() error {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		select {
		case sig := <-sigChan:
			a.logger.Info("Received shutdown signal", "signal", sig)
			return a.shutdown()
		case <-ctx.Done():
			return ctx.Err()
		}
	})

	return g.Wait()
}

func (a *App) shutdown() error {
	a.logger.Info("Shutting down application...")

	ctx, cancel := context.WithTimeout(context.Background(), a.config.Server.ShutdownTimeout)
	defer cancel()

	if err := a.server.Shutdown(ctx); err != nil {
		a.logger.Error("Failed to shutdown HTTP server", "error", err)
		return err
	}

	if a.db != nil {
		a.db.Close()
	}

	if a.redisClient != nil {
		a.redisClient.Close()
	}

	a.logger.Info("Application shutdown complete")
	return nil
}
