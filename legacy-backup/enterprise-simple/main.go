package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/VeRJiL/go-template/internal/app"
	"github.com/VeRJiL/go-template/internal/modules"
	"github.com/VeRJiL/go-template/internal/pkg/bootstrap"
)

func main() {
	// Initialize the standard app first to get all dependencies
	standardApp, err := app.New()
	if err != nil {
		log.Fatalf("Failed to create app: %v", err)
	}

	if err := standardApp.Initialize(); err != nil {
		log.Fatalf("Failed to initialize app: %v", err)
	}

	// Get the dependencies from the standard app
	config := standardApp.GetConfig()
	logger := standardApp.GetLogger()
	db := standardApp.GetDB()
	redisClient := standardApp.GetRedisClient()
	jwtService := standardApp.GetJWTService()

	// Create enterprise bootstrap
	enterpriseBootstrap := bootstrap.NewEnterpriseBootstrap(config, logger)

	// Initialize enterprise system
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := enterpriseBootstrap.Initialize(ctx, db, redisClient, jwtService); err != nil {
		log.Fatalf("Failed to initialize enterprise system: %v", err)
	}

	// Register core modules
	userModule := modules.NewUserModule()
	if err := enterpriseBootstrap.RegisterModule(userModule); err != nil {
		log.Fatalf("Failed to register user module: %v", err)
	}

	// Run migrations
	if err := enterpriseBootstrap.Migrate(ctx); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Create router
	router := gin.Default()

	// Add basic middleware
	router.Use(gin.Recovery())

	// Create API group
	apiGroup := router.Group("/api/v1")

	// Register enterprise routes
	if err := enterpriseBootstrap.RegisterRoutes(apiGroup); err != nil {
		log.Fatalf("Failed to register routes: %v", err)
	}

	// Add health check endpoints
	router.GET("/health", func(c *gin.Context) {
		health := enterpriseBootstrap.HealthCheck(c.Request.Context())
		c.JSON(200, health)
	})

	router.GET("/admin/stats", func(c *gin.Context) {
		stats := enterpriseBootstrap.GetStats()
		c.JSON(200, stats)
	})

	router.GET("/admin/modules", func(c *gin.Context) {
		modules := enterpriseBootstrap.GetModuleInfo()
		c.JSON(200, gin.H{
			"modules": modules,
			"count":   len(modules),
		})
	})

	// Start server
	logger.Info("Starting enterprise HTTP server on port 8081")

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := router.Run(":8081"); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	logger.Info("Enterprise application started successfully on port 8081")
	fmt.Println("🚀 Enterprise Application Running!")
	fmt.Println("📊 Health Check: http://localhost:8081/health")
	fmt.Println("📈 Stats: http://localhost:8081/admin/stats")
	fmt.Println("📦 Modules: http://localhost:8081/admin/modules")
	fmt.Println("👥 Users API: http://localhost:8081/api/v1/users")
	fmt.Println("🔐 Auth API: http://localhost:8081/api/v1/auth")

	// Wait for signal
	<-sigChan
	logger.Info("Shutting down enterprise application")

	// Shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := enterpriseBootstrap.Shutdown(shutdownCtx); err != nil {
		logger.Error("Failed to shutdown enterprise system", "error", err)
	}

	logger.Info("Enterprise application shutdown completed")
}