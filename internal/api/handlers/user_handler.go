package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/VeRJiL/go-template/internal/domain/entities"
	"github.com/VeRJiL/go-template/internal/domain/services"
	"github.com/VeRJiL/go-template/internal/pkg/logger"
)

type UserHandler struct {
	userService *services.UserService
	logger      *logger.Logger
}

func NewUserHandler(userService *services.UserService, logger *logger.Logger) *UserHandler {
	return &UserHandler{
		userService: userService,
		logger:      logger,
	}
}

// Create godoc
// @Summary Register a new user
// @Description Register a new user with email and password
// @Tags auth
// @Accept json
// @Produce json
// @Param user body entities.CreateUserRequest true "User registration data"
// @Success 201 {object} entities.User
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /auth/register [post]
func (h *UserHandler) Create(c *gin.Context) {
	var req entities.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid request body", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Basic validation
	if req.Email == "" || req.Password == "" || req.FirstName == "" || req.LastName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email, password, first name, and last name are required"})
		return
	}

	user, err := h.userService.Create(c.Request.Context(), &req)
	if err != nil {
		if err == services.ErrUserExists {
			c.JSON(http.StatusConflict, gin.H{"error": "User already exists"})
			return
		}
		h.logger.Error("Failed to create user", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "User created successfully",
		"user":    user,
	})
}

// GetByID godoc
// @Summary Get user by ID
// @Description Get a specific user by their ID
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Success 200 {object} entities.User
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /users/{id} [get]
func (h *UserHandler) GetByID(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	user, err := h.userService.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == services.ErrUserNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}

// Update godoc
// @Summary Update user
// @Description Update user information
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Param user body entities.UpdateUserRequest true "User update data"
// @Success 200 {object} entities.User
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /users/{id} [put]
func (h *UserHandler) Update(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var req entities.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid request body", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Check if user is updating their own profile or is admin
	userID := c.MustGet("user_id").(uuid.UUID)
	userRole := c.MustGet("user_role").(string)

	if userID != id && userRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot update other users"})
		return
	}

	user, err := h.userService.Update(c.Request.Context(), id, &req)
	if err != nil {
		if err == services.ErrUserNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		h.logger.Error("Failed to update user", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "User updated successfully",
		"user":    user,
	})
}

// Delete godoc
// @Summary Delete user
// @Description Delete a user account
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /users/{id} [delete]
func (h *UserHandler) Delete(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Check if user is deleting their own account or is admin
	userID := c.MustGet("user_id").(uuid.UUID)
	userRole := c.MustGet("user_role").(string)

	if userID != id && userRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot delete other users"})
		return
	}

	if err := h.userService.Delete(c.Request.Context(), id); err != nil {
		if err == services.ErrUserNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}

// List godoc
// @Summary List all users
// @Description Get a paginated list of all users
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Number of items per page" default(10)
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /users/ [get]
func (h *UserHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	offset := (page - 1) * limit

	users, total, err := h.userService.List(c.Request.Context(), offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list users"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"users": users,
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": (total + limit - 1) / limit,
		},
	})
}

// GET /api/v1/users/search
func (h *UserHandler) Search(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Search query is required"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	offset := (page - 1) * limit

	users, total, err := h.userService.Search(c.Request.Context(), query, offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search users"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"users": users,
		"query": query,
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": (total + limit - 1) / limit,
		},
	})
}

// Login godoc
// @Summary User login
// @Description Authenticate user and return JWT token
// @Tags auth
// @Accept json
// @Produce json
// @Param credentials body entities.LoginRequest true "Login credentials"
// @Success 200 {object} entities.LoginResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /auth/login [post]
func (h *UserHandler) Login(c *gin.Context) {
	var req entities.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid request body", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Basic validation
	if req.Email == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email and password are required"})
		return
	}

	response, err := h.userService.Login(c.Request.Context(), &req)
	if err != nil {
		if err == services.ErrInvalidCredentials {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			return
		}
		h.logger.Error("Login failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Login failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Login successful",
		"token":      response.Token,
		"user":       response.User,
		"expires_at": response.ExpiresAt,
	})
}

// Logout godoc
// @Summary User logout
// @Description Logout user and invalidate token
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /auth/logout [post]
func (h *UserHandler) Logout(c *gin.Context) {
	token := c.MustGet("token").(string)

	if err := h.userService.Logout(c.Request.Context(), token); err != nil {
		h.logger.Error("Logout failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Logout failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// GetProfile godoc
// @Summary Get user profile
// @Description Get current authenticated user profile
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} entities.User
// @Failure 404 {object} map[string]string
// @Router /auth/me [get]
func (h *UserHandler) GetProfile(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	user, err := h.userService.GetByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}
