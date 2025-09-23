package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/VeRJiL/go-template/internal/domain/entities"
)

// TestFullIntegration runs a complete end-to-end test scenario
func TestFullIntegration(t *testing.T) {
	app := SetupTestApp(t)
	defer app.TeardownTestApp(t)

	t.Run("Complete User Lifecycle", func(t *testing.T) {
		// Step 1: Health check
		t.Log("Step 1: Verifying application health")
		healthReq, _ := http.NewRequest("GET", "/health", nil)
		healthW := httptest.NewRecorder()
		app.Router.ServeHTTP(healthW, healthReq)
		require.Equal(t, http.StatusOK, healthW.Code)

		var healthResponse map[string]interface{}
		err := json.Unmarshal(healthW.Body.Bytes(), &healthResponse)
		require.NoError(t, err)
		assert.Equal(t, "ok", healthResponse["status"])

		// Step 2: Register new user
		t.Log("Step 2: Registering new user")
		newUser := entities.CreateUserRequest{
			Email:     "integration@example.com",
			Password:  "integration123",
			FirstName: "Integration",
			LastName:  "Test",
			Role:      "user",
		}

		body, _ := json.Marshal(newUser)
		registerReq, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(body))
		registerReq.Header.Set("Content-Type", "application/json")

		registerW := httptest.NewRecorder()
		app.Router.ServeHTTP(registerW, registerReq)
		require.Equal(t, http.StatusCreated, registerW.Code)

		var registerResponse map[string]interface{}
		err = json.Unmarshal(registerW.Body.Bytes(), &registerResponse)
		require.NoError(t, err)
		assert.Equal(t, "User created successfully", registerResponse["message"])

		createdUser := registerResponse["user"].(map[string]interface{})
		userID := createdUser["id"].(string)

		// Step 3: Login with new user
		t.Log("Step 3: Logging in with new user")
		loginRequest := entities.LoginRequest{
			Email:    newUser.Email,
			Password: newUser.Password,
		}

		loginBody, _ := json.Marshal(loginRequest)
		loginReq, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(loginBody))
		loginReq.Header.Set("Content-Type", "application/json")

		loginW := httptest.NewRecorder()
		app.Router.ServeHTTP(loginW, loginReq)
		require.Equal(t, http.StatusOK, loginW.Code)

		var loginResponse map[string]interface{}
		err = json.Unmarshal(loginW.Body.Bytes(), &loginResponse)
		require.NoError(t, err)
		assert.Equal(t, "Login successful", loginResponse["message"])

		userToken := loginResponse["token"].(string)
		assert.NotEmpty(t, userToken)

		// Step 4: Get user profile
		t.Log("Step 4: Getting user profile")
		profileReq, _ := http.NewRequest("GET", "/api/v1/auth/me", nil)
		profileReq.Header.Set("Authorization", "Bearer "+userToken)

		profileW := httptest.NewRecorder()
		app.Router.ServeHTTP(profileW, profileReq)
		require.Equal(t, http.StatusOK, profileW.Code)

		var profileResponse map[string]interface{}
		err = json.Unmarshal(profileW.Body.Bytes(), &profileResponse)
		require.NoError(t, err)

		profile := profileResponse["user"].(map[string]interface{})
		assert.Equal(t, userID, profile["id"])
		assert.Equal(t, newUser.Email, profile["email"])

		// Step 5: Update user profile
		t.Log("Step 5: Updating user profile")
		updateRequest := entities.UpdateUserRequest{
			FirstName: stringPtr("Updated"),
			LastName:  stringPtr("Name"),
		}

		updateBody, _ := json.Marshal(updateRequest)
		updateReq, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/users/%s", userID), bytes.NewBuffer(updateBody))
		updateReq.Header.Set("Content-Type", "application/json")
		updateReq.Header.Set("Authorization", "Bearer "+userToken)

		updateW := httptest.NewRecorder()
		app.Router.ServeHTTP(updateW, updateReq)
		require.Equal(t, http.StatusOK, updateW.Code)

		var updateResponse map[string]interface{}
		err = json.Unmarshal(updateW.Body.Bytes(), &updateResponse)
		require.NoError(t, err)

		updatedUser := updateResponse["user"].(map[string]interface{})
		assert.Equal(t, "Updated", updatedUser["first_name"])
		assert.Equal(t, "Name", updatedUser["last_name"])

		// Step 6: Create admin user for privileged operations
		t.Log("Step 6: Creating admin user")
		adminUser := entities.CreateUserRequest{
			Email:     "admin-integration@example.com",
			Password:  "admin123",
			FirstName: "Admin",
			LastName:  "Test",
			Role:      "admin",
		}

		adminBody, _ := json.Marshal(adminUser)
		adminRegisterReq, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(adminBody))
		adminRegisterReq.Header.Set("Content-Type", "application/json")

		adminRegisterW := httptest.NewRecorder()
		app.Router.ServeHTTP(adminRegisterW, adminRegisterReq)
		require.Equal(t, http.StatusCreated, adminRegisterW.Code)

		// Login as admin
		adminLoginRequest := entities.LoginRequest{
			Email:    adminUser.Email,
			Password: adminUser.Password,
		}

		adminLoginBody, _ := json.Marshal(adminLoginRequest)
		adminLoginReq, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(adminLoginBody))
		adminLoginReq.Header.Set("Content-Type", "application/json")

		adminLoginW := httptest.NewRecorder()
		app.Router.ServeHTTP(adminLoginW, adminLoginReq)
		require.Equal(t, http.StatusOK, adminLoginW.Code)

		var adminLoginResponse map[string]interface{}
		err = json.Unmarshal(adminLoginW.Body.Bytes(), &adminLoginResponse)
		require.NoError(t, err)

		adminToken := adminLoginResponse["token"].(string)

		// Step 7: List users as admin
		t.Log("Step 7: Listing users as admin")
		listReq, _ := http.NewRequest("GET", "/api/v1/users/?page=1&limit=10", nil)
		listReq.Header.Set("Authorization", "Bearer "+adminToken)

		listW := httptest.NewRecorder()
		app.Router.ServeHTTP(listW, listReq)
		require.Equal(t, http.StatusOK, listW.Code)

		var listResponse map[string]interface{}
		err = json.Unmarshal(listW.Body.Bytes(), &listResponse)
		require.NoError(t, err)

		users := listResponse["users"].([]interface{})
		assert.True(t, len(users) >= 2) // At least our 2 test users

		pagination := listResponse["pagination"].(map[string]interface{})
		assert.Equal(t, float64(1), pagination["page"])
		assert.Equal(t, float64(10), pagination["limit"])
		assert.True(t, pagination["total"].(float64) >= 2)

		// Step 8: Search users
		t.Log("Step 8: Searching users")
		searchReq, _ := http.NewRequest("GET", "/api/v1/users/search?q=integration&page=1&limit=5", nil)
		searchReq.Header.Set("Authorization", "Bearer "+adminToken)

		searchW := httptest.NewRecorder()
		app.Router.ServeHTTP(searchW, searchReq)
		require.Equal(t, http.StatusOK, searchW.Code)

		var searchResponse map[string]interface{}
		err = json.Unmarshal(searchW.Body.Bytes(), &searchResponse)
		require.NoError(t, err)

		searchUsers := searchResponse["users"].([]interface{})
		assert.True(t, len(searchUsers) >= 1)
		assert.Equal(t, "integration", searchResponse["query"])

		// Step 9: Logout user
		t.Log("Step 9: Logging out user")
		logoutReq, _ := http.NewRequest("POST", "/api/v1/auth/logout", nil)
		logoutReq.Header.Set("Authorization", "Bearer "+userToken)

		logoutW := httptest.NewRecorder()
		app.Router.ServeHTTP(logoutW, logoutReq)
		require.Equal(t, http.StatusOK, logoutW.Code)

		var logoutResponse map[string]interface{}
		err = json.Unmarshal(logoutW.Body.Bytes(), &logoutResponse)
		require.NoError(t, err)
		assert.Equal(t, "Logged out successfully", logoutResponse["message"])

		// Step 10: Verify token is invalidated
		t.Log("Step 10: Verifying token invalidation")
		invalidReq, _ := http.NewRequest("GET", "/api/v1/auth/me", nil)
		invalidReq.Header.Set("Authorization", "Bearer "+userToken)

		invalidW := httptest.NewRecorder()
		app.Router.ServeHTTP(invalidW, invalidReq)
		assert.Equal(t, http.StatusUnauthorized, invalidW.Code)

		t.Log("Full integration test completed successfully!")
	})
}

// TestWithFixtures demonstrates using test fixtures
func TestWithFixtures(t *testing.T) {
	app := SetupTestApp(t)
	defer app.TeardownTestApp(t)

	// Load test fixtures
	fixtures := LoadTestFixtures(t, app.UserService, app.JWTService)
	defer CleanupFixtures(t, app.Environment)

	t.Run("should work with pre-populated data", func(t *testing.T) {
		// Verify fixtures were loaded
		assert.True(t, len(fixtures.Users) >= 5, "Should have at least 5 test users")

		// Get admin user
		admin := fixtures.GetAdminUser()
		require.NotNil(t, admin, "Should have admin user")
		assert.Equal(t, "admin", admin.User.Role)

		// Get regular user
		user := fixtures.GetRegularUser()
		require.NotNil(t, user, "Should have regular user")
		assert.Equal(t, "user", user.User.Role)

		// Test listing users with admin token
		listReq, _ := http.NewRequest("GET", "/api/v1/users/?page=1&limit=20", nil)
		listReq.Header.Set("Authorization", "Bearer "+admin.Token)

		listW := httptest.NewRecorder()
		app.Router.ServeHTTP(listW, listReq)
		require.Equal(t, http.StatusOK, listW.Code)

		var listResponse map[string]interface{}
		err := json.Unmarshal(listW.Body.Bytes(), &listResponse)
		require.NoError(t, err)

		users := listResponse["users"].([]interface{})
		assert.True(t, len(users) >= len(fixtures.Users))

		// Test searching for specific user
		searchReq, _ := http.NewRequest("GET", "/api/v1/users/search?q=john.doe", nil)
		searchReq.Header.Set("Authorization", "Bearer "+admin.Token)

		searchW := httptest.NewRecorder()
		app.Router.ServeHTTP(searchW, searchReq)
		require.Equal(t, http.StatusOK, searchW.Code)

		var searchResponse map[string]interface{}
		err = json.Unmarshal(searchW.Body.Bytes(), &searchResponse)
		require.NoError(t, err)

		searchUsers := searchResponse["users"].([]interface{})
		assert.Equal(t, 1, len(searchUsers))

		foundUser := searchUsers[0].(map[string]interface{})
		assert.Equal(t, "john.doe@example.com", foundUser["email"])
	})
}

// TestPerformanceWithLargeDataset tests with large amounts of data
func TestPerformanceWithLargeDataset(t *testing.T) {
	app := SetupTestApp(t)
	defer app.TeardownTestApp(t)

	// Create admin user
	_, adminToken := CreateTestUserWithCustomData(t, app.UserService,
		"perf-admin@example.com", "admin123", "Perf", "Admin", "admin")

	// Create large dataset
	userIDs := CreateLargeDataset(t, app.Environment, 1000)
	defer CleanupFixtures(t, app.Environment)

	t.Run("should handle large dataset pagination", func(t *testing.T) {
		// Test pagination with large dataset
		listReq, _ := http.NewRequest("GET", "/api/v1/users/?page=1&limit=50", nil)
		listReq.Header.Set("Authorization", "Bearer "+adminToken)

		listW := httptest.NewRecorder()
		app.Router.ServeHTTP(listW, listReq)
		require.Equal(t, http.StatusOK, listW.Code)

		var listResponse map[string]interface{}
		err := json.Unmarshal(listW.Body.Bytes(), &listResponse)
		require.NoError(t, err)

		users := listResponse["users"].([]interface{})
		assert.Equal(t, 50, len(users)) // Should return exactly 50 users

		pagination := listResponse["pagination"].(map[string]interface{})
		assert.Equal(t, float64(1), pagination["page"])
		assert.Equal(t, float64(50), pagination["limit"])
		assert.True(t, pagination["total"].(float64) >= 1000) // At least 1000 users + admin
	})

	t.Run("should handle search in large dataset", func(t *testing.T) {
		// Search for specific pattern in large dataset
		searchReq, _ := http.NewRequest("GET", "/api/v1/users/search?q=bulk500&page=1&limit=10", nil)
		searchReq.Header.Set("Authorization", "Bearer "+adminToken)

		searchW := httptest.NewRecorder()
		app.Router.ServeHTTP(searchW, searchReq)
		require.Equal(t, http.StatusOK, searchW.Code)

		var searchResponse map[string]interface{}
		err := json.Unmarshal(searchW.Body.Bytes(), &searchResponse)
		require.NoError(t, err)

		searchUsers := searchResponse["users"].([]interface{})
		assert.Equal(t, 1, len(searchUsers)) // Should find exactly one user

		foundUser := searchUsers[0].(map[string]interface{})
		assert.Contains(t, foundUser["email"].(string), "bulk500")
	})

	t.Logf("Performance test completed successfully with %d users", len(userIDs))
}

// TestDatabaseConsistency verifies database operations maintain consistency
func TestDatabaseConsistency(t *testing.T) {
	app := SetupTestApp(t)
	defer app.TeardownTestApp(t)

	t.Run("should maintain consistency across operations", func(t *testing.T) {
		// Verify initial state
		VerifyDatabaseState(t, app.Environment, 0)

		// Create users through API
		users := make([]string, 5)
		for i := 0; i < 5; i++ {
			userReq := entities.CreateUserRequest{
				Email:     fmt.Sprintf("consistency%d@example.com", i),
				Password:  "password123",
				FirstName: fmt.Sprintf("User%d", i),
				LastName:  "Test",
				Role:      "user",
			}

			body, _ := json.Marshal(userReq)
			req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			app.Router.ServeHTTP(w, req)
			require.Equal(t, http.StatusCreated, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			user := response["user"].(map[string]interface{})
			users[i] = user["id"].(string)
		}

		// Verify all users were created
		VerifyDatabaseState(t, app.Environment, 5)

		// Create admin and delete some users
		_, adminToken := CreateTestUserWithCustomData(t, app.UserService,
			"consistency-admin@example.com", "admin123", "Admin", "Test", "admin")

		// Delete 2 users
		for i := 0; i < 2; i++ {
			deleteReq, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/v1/users/%s", users[i]), nil)
			deleteReq.Header.Set("Authorization", "Bearer "+adminToken)

			deleteW := httptest.NewRecorder()
			app.Router.ServeHTTP(deleteW, deleteReq)
			require.Equal(t, http.StatusOK, deleteW.Code)
		}

		// Verify final state (3 users + 1 admin remaining)
		VerifyDatabaseState(t, app.Environment, 4)

		t.Log("Database consistency verified successfully")
	})
}