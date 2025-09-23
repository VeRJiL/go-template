package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/VeRJiL/go-template/internal/api/handlers"
	"github.com/VeRJiL/go-template/internal/api/routes"
	"github.com/VeRJiL/go-template/internal/database/postgres"
	"github.com/VeRJiL/go-template/internal/database/redis"
	"github.com/VeRJiL/go-template/internal/domain/entities"
	"github.com/VeRJiL/go-template/internal/domain/repositories"
	"github.com/VeRJiL/go-template/internal/domain/services"
	"github.com/VeRJiL/go-template/internal/pkg/auth"
	"github.com/VeRJiL/go-template/internal/pkg/logger"
)

type TestApp struct {
	Router      *gin.Engine
	Environment *TestEnvironment
	JWTService  *auth.JWTService
	UserService *services.UserService
}

func SetupTestApp(t *testing.T) *TestApp {
	env := SetupTestEnvironment(t)

	// Set gin to test mode
	gin.SetMode(gin.TestMode)

	// Initialize logger
	log := logger.New(env.Config.Logging.Level, env.Config.Logging.Format)

	// Initialize JWT service
	jwtService := auth.NewJWTService(env.Config.Auth.JWT.Secret, int(env.Config.Auth.JWT.Expiration.Minutes()))

	// Initialize repositories
	userRepo := postgres.NewUserRepository(env.DB)

	// Initialize Redis (optional for tests)
	var cacheRepo repositories.UserCacheRepository
	if env.Config.Redis.Host != "" {
		redisClient, err := redis.NewConnection(&env.Config.Redis)
		if err == nil {
			cacheRepo = redis.NewUserCacheRepository(redisClient)
		}
	}

	// Initialize services
	userService := services.NewUserService(userRepo, jwtService)
	if cacheRepo != nil {
		userService.SetCacheRepository(cacheRepo)
	}

	// Initialize handlers
	userHandler := handlers.NewUserHandler(userService, log)

	// Setup router
	router := gin.New()

	// Setup routes
	deps := &routes.Dependencies{
		UserHandler: userHandler,
		JWTService:  jwtService,
		Logger:      log,
		Config:      env.Config,
	}
	routes.SetupRoutes(router, deps)

	return &TestApp{
		Router:      router,
		Environment: env,
		JWTService:  jwtService,
		UserService: userService,
	}
}

func (app *TestApp) TeardownTestApp(t *testing.T) {
	app.Environment.TeardownTestEnvironment(t)
}

func TestUserRegistration(t *testing.T) {
	app := SetupTestApp(t)
	defer app.TeardownTestApp(t)

	t.Run("should register user successfully", func(t *testing.T) {
		requestBody := entities.CreateUserRequest{
			Email:     "test@example.com",
			Password:  "password123",
			FirstName: "John",
			LastName:  "Doe",
			Role:      "user",
		}

		body, _ := json.Marshal(requestBody)
		req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		app.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "User created successfully", response["message"])
		assert.Contains(t, response, "user")

		user := response["user"].(map[string]interface{})
		assert.Equal(t, requestBody.Email, user["email"])
		assert.Equal(t, requestBody.FirstName, user["first_name"])
		assert.Equal(t, requestBody.LastName, user["last_name"])
		assert.Equal(t, requestBody.Role, user["role"])
		assert.True(t, user["is_active"].(bool))
		assert.NotEmpty(t, user["id"])

		// Verify UUID format
		_, err = uuid.Parse(user["id"].(string))
		assert.NoError(t, err, "User ID should be a valid UUID")
	})

	t.Run("should return error for duplicate email", func(t *testing.T) {
		email := "duplicate@example.com"

		// Create first user
		requestBody1 := entities.CreateUserRequest{
			Email:     email,
			Password:  "password123",
			FirstName: "John",
			LastName:  "Doe",
			Role:      "user",
		}

		body1, _ := json.Marshal(requestBody1)
		req1, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(body1))
		req1.Header.Set("Content-Type", "application/json")

		w1 := httptest.NewRecorder()
		app.Router.ServeHTTP(w1, req1)

		assert.Equal(t, http.StatusCreated, w1.Code)

		// Try to create second user with same email
		requestBody2 := entities.CreateUserRequest{
			Email:     email,
			Password:  "different123",
			FirstName: "Jane",
			LastName:  "Smith",
			Role:      "user",
		}

		body2, _ := json.Marshal(requestBody2)
		req2, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(body2))
		req2.Header.Set("Content-Type", "application/json")

		w2 := httptest.NewRecorder()
		app.Router.ServeHTTP(w2, req2)

		assert.Equal(t, http.StatusConflict, w2.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w2.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "User already exists", response["error"])
	})

	t.Run("should validate required fields", func(t *testing.T) {
		testCases := []struct {
			name         string
			requestBody  entities.CreateUserRequest
			expectedCode int
			expectedMsg  string
		}{
			{
				name: "missing email",
				requestBody: entities.CreateUserRequest{
					Password:  "password123",
					FirstName: "John",
					LastName:  "Doe",
					Role:      "user",
				},
				expectedCode: http.StatusBadRequest,
				expectedMsg:  "Email, password, first name, and last name are required",
			},
			{
				name: "missing password",
				requestBody: entities.CreateUserRequest{
					Email:     "test@example.com",
					FirstName: "John",
					LastName:  "Doe",
					Role:      "user",
				},
				expectedCode: http.StatusBadRequest,
				expectedMsg:  "Email, password, first name, and last name are required",
			},
			{
				name: "missing first name",
				requestBody: entities.CreateUserRequest{
					Email:    "test@example.com",
					Password: "password123",
					LastName: "Doe",
					Role:     "user",
				},
				expectedCode: http.StatusBadRequest,
				expectedMsg:  "Email, password, first name, and last name are required",
			},
			{
				name: "missing last name",
				requestBody: entities.CreateUserRequest{
					Email:     "test@example.com",
					Password:  "password123",
					FirstName: "John",
					Role:      "user",
				},
				expectedCode: http.StatusBadRequest,
				expectedMsg:  "Email, password, first name, and last name are required",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				body, _ := json.Marshal(tc.requestBody)
				req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")

				w := httptest.NewRecorder()
				app.Router.ServeHTTP(w, req)

				assert.Equal(t, tc.expectedCode, w.Code)

				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.Equal(t, tc.expectedMsg, response["error"])
			})
		}
	})

	t.Run("should handle invalid JSON", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		app.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "Invalid request body", response["error"])
	})
}

func TestUserLogin(t *testing.T) {
	app := SetupTestApp(t)
	defer app.TeardownTestApp(t)

	// Create a test user first
	testUser := &entities.CreateUserRequest{
		Email:     "login-test@example.com",
		Password:  "password123",
		FirstName: "Login",
		LastName:  "Test",
		Role:      "user",
	}

	createdUser, err := app.UserService.Create(app.Environment.Ctx, testUser)
	require.NoError(t, err)

	t.Run("should login successfully with valid credentials", func(t *testing.T) {
		loginRequest := entities.LoginRequest{
			Email:    testUser.Email,
			Password: testUser.Password,
		}

		body, _ := json.Marshal(loginRequest)
		req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		app.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "Login successful", response["message"])
		assert.NotEmpty(t, response["token"])
		assert.Contains(t, response, "user")
		assert.Contains(t, response, "expires_at")

		// Verify user data
		user := response["user"].(map[string]interface{})
		assert.Equal(t, createdUser.ID.String(), user["id"])
		assert.Equal(t, createdUser.Email, user["email"])

		// Verify token is valid
		token := response["token"].(string)
		claims, err := app.JWTService.ValidateToken(token)
		assert.NoError(t, err)
		assert.Equal(t, createdUser.ID.String(), claims.UserID)
	})

	t.Run("should return error for invalid credentials", func(t *testing.T) {
		testCases := []struct {
			name          string
			email         string
			password      string
			expectedError string
		}{
			{
				name:          "wrong password",
				email:         testUser.Email,
				password:      "wrongpassword",
				expectedError: "Invalid credentials",
			},
			{
				name:          "non-existent email",
				email:         "nonexistent@example.com",
				password:      testUser.Password,
				expectedError: "Invalid credentials",
			},
			{
				name:          "empty email",
				email:         "",
				password:      testUser.Password,
				expectedError: "Email and password are required",
			},
			{
				name:          "empty password",
				email:         testUser.Email,
				password:      "",
				expectedError: "Email and password are required",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				loginRequest := entities.LoginRequest{
					Email:    tc.email,
					Password: tc.password,
				}

				body, _ := json.Marshal(loginRequest)
				req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")

				w := httptest.NewRecorder()
				app.Router.ServeHTTP(w, req)

				if tc.expectedError == "Email and password are required" {
					assert.Equal(t, http.StatusBadRequest, w.Code)
				} else {
					assert.Equal(t, http.StatusUnauthorized, w.Code)
				}

				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.Equal(t, tc.expectedError, response["error"])
			})
		}
	})
}

func TestUserProfile(t *testing.T) {
	app := SetupTestApp(t)
	defer app.TeardownTestApp(t)

	// Create a test user and get their token
	testUser := &entities.CreateUserRequest{
		Email:     "profile-test@example.com",
		Password:  "password123",
		FirstName: "Profile",
		LastName:  "Test",
		Role:      "user",
	}

	createdUser, err := app.UserService.Create(app.Environment.Ctx, testUser)
	require.NoError(t, err)

	// Login to get token
	loginRequest := &entities.LoginRequest{
		Email:    testUser.Email,
		Password: testUser.Password,
	}

	loginResponse, err := app.UserService.Login(app.Environment.Ctx, loginRequest)
	require.NoError(t, err)

	t.Run("should get user profile with valid token", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/auth/me", nil)
		req.Header.Set("Authorization", "Bearer "+loginResponse.Token)

		w := httptest.NewRecorder()
		app.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Contains(t, response, "user")
		user := response["user"].(map[string]interface{})
		assert.Equal(t, createdUser.ID.String(), user["id"])
		assert.Equal(t, createdUser.Email, user["email"])
		assert.Equal(t, createdUser.FirstName, user["first_name"])
		assert.Equal(t, createdUser.LastName, user["last_name"])
	})

	t.Run("should return error without token", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/auth/me", nil)

		w := httptest.NewRecorder()
		app.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("should return error with invalid token", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/auth/me", nil)
		req.Header.Set("Authorization", "Bearer invalid_token")

		w := httptest.NewRecorder()
		app.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestHealthCheck(t *testing.T) {
	app := SetupTestApp(t)
	defer app.TeardownTestApp(t)

	t.Run("should return health status", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/health", nil)

		w := httptest.NewRecorder()
		app.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "ok", response["status"])
		assert.Equal(t, "go-template", response["service"])
		assert.Equal(t, "1.0.0", response["version"])
		assert.Contains(t, response, "timestamp")

		// Verify timestamp format
		timestamp := response["timestamp"].(string)
		_, err = time.Parse(time.RFC3339, timestamp)
		assert.NoError(t, err, "Timestamp should be in RFC3339 format")
	})
}