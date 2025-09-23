package routes

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/VeRJiL/go-template/internal/api/handlers"
	"github.com/VeRJiL/go-template/internal/api/middleware"
	"github.com/VeRJiL/go-template/internal/config"
	"github.com/VeRJiL/go-template/internal/pkg/auth"
	"github.com/VeRJiL/go-template/internal/pkg/logger"
	_ "github.com/VeRJiL/go-template/docs/swagger"
)

type Dependencies struct {
	UserHandler *handlers.UserHandler
	JWTService  *auth.JWTService
	Logger      *logger.Logger
	Config      *config.Config
}

// SetupRoutes configures all application routes
func SetupRoutes(router *gin.Engine, deps *Dependencies) {
	// Health check endpoint
	router.GET("/health", healthCheck)

	// Swagger documentation
	if deps.Config.Server.EnableSwagger {
		router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Authentication routes (public)
		auth := v1.Group("/auth")
		{
			auth.POST("/register", deps.UserHandler.Create)
			auth.POST("/login", deps.UserHandler.Login)

			// Protected auth routes
			protected := auth.Use(middleware.AuthMiddleware(deps.JWTService))
			{
				protected.POST("/logout", deps.UserHandler.Logout)
				protected.GET("/me", deps.UserHandler.GetProfile)
			}
		}

		// User management routes (protected)
		users := v1.Group("/users").Use(middleware.AuthMiddleware(deps.JWTService))
		{
			users.GET("/", deps.UserHandler.List)         // List all users
			users.GET("/search", deps.UserHandler.Search) // Search users
			users.GET("/:id", deps.UserHandler.GetByID)   // Get user by ID
			users.PUT("/:id", deps.UserHandler.Update)    // Update user
			users.DELETE("/:id", deps.UserHandler.Delete) // Delete user
		}
	}
}

// healthCheck returns the application health status
func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"service":   "go-template",
		"version":   "1.0.0",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}
