package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/VeRJiL/go-template/internal/domain/entities"
)

func TestUserCRUDOperations(t *testing.T) {
	app := SetupTestApp(t)
	defer app.TeardownTestApp(t)

	// Create admin user for privileged operations
	adminUser := &entities.CreateUserRequest{
		Email:     "admin@example.com",
		Password:  "admin123",
		FirstName: "Admin",
		LastName:  "User",
		Role:      "admin",
	}

	createdAdmin, err := app.UserService.Create(app.Environment.Ctx, adminUser)
	require.NoError(t, err)

	// Login as admin to get token
	adminLoginRequest := &entities.LoginRequest{
		Email:    adminUser.Email,
		Password: adminUser.Password,
	}

	adminLoginResponse, err := app.UserService.Login(app.Environment.Ctx, adminLoginRequest)
	require.NoError(t, err)

	// Create regular user
	regularUser := &entities.CreateUserRequest{
		Email:     "regular@example.com",
		Password:  "regular123",
		FirstName: "Regular",
		LastName:  "User",
		Role:      "user",
	}

	createdRegular, err := app.UserService.Create(app.Environment.Ctx, regularUser)
	require.NoError(t, err)

	// Login as regular user to get token
	regularLoginRequest := &entities.LoginRequest{
		Email:    regularUser.Email,
		Password: regularUser.Password,
	}

	regularLoginResponse, err := app.UserService.Login(app.Environment.Ctx, regularLoginRequest)
	require.NoError(t, err)

	t.Run("Get User by ID", func(t *testing.T) {
		t.Run("should get user by ID with admin token", func(t *testing.T) {
			req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/users/%s", createdRegular.ID), nil)
			req.Header.Set("Authorization", "Bearer "+adminLoginResponse.Token)

			w := httptest.NewRecorder()
			app.Router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Contains(t, response, "user")
			user := response["user"].(map[string]interface{})
			assert.Equal(t, createdRegular.ID.String(), user["id"])
			assert.Equal(t, createdRegular.Email, user["email"])
		})

		t.Run("should get user by ID with own token", func(t *testing.T) {
			req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/users/%s", createdRegular.ID), nil)
			req.Header.Set("Authorization", "Bearer "+regularLoginResponse.Token)

			w := httptest.NewRecorder()
			app.Router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})

		t.Run("should return error for invalid UUID", func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/api/v1/users/invalid-uuid", nil)
			req.Header.Set("Authorization", "Bearer "+adminLoginResponse.Token)

			w := httptest.NewRecorder()
			app.Router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Equal(t, "Invalid user ID", response["error"])
		})

		t.Run("should return error for non-existent user", func(t *testing.T) {
			nonExistentID := uuid.New()
			req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/users/%s", nonExistentID), nil)
			req.Header.Set("Authorization", "Bearer "+adminLoginResponse.Token)

			w := httptest.NewRecorder()
			app.Router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusNotFound, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Equal(t, "User not found", response["error"])
		})

		t.Run("should return error without authentication", func(t *testing.T) {
			req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/users/%s", createdRegular.ID), nil)

			w := httptest.NewRecorder()
			app.Router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	})

	t.Run("Update User", func(t *testing.T) {
		t.Run("should update user with admin privileges", func(t *testing.T) {
			updateRequest := entities.UpdateUserRequest{
				FirstName: stringPtr("UpdatedFirst"),
				LastName:  stringPtr("UpdatedLast"),
				Role:      stringPtr("admin"),
			}

			body, _ := json.Marshal(updateRequest)
			req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/users/%s", createdRegular.ID), bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+adminLoginResponse.Token)

			w := httptest.NewRecorder()
			app.Router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Equal(t, "User updated successfully", response["message"])
			user := response["user"].(map[string]interface{})
			assert.Equal(t, "UpdatedFirst", user["first_name"])
			assert.Equal(t, "UpdatedLast", user["last_name"])
			assert.Equal(t, "admin", user["role"])
		})

		t.Run("should update own profile", func(t *testing.T) {
			updateRequest := entities.UpdateUserRequest{
				FirstName: stringPtr("SelfUpdated"),
				LastName:  stringPtr("Name"),
			}

			body, _ := json.Marshal(updateRequest)
			req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/users/%s", createdRegular.ID), bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+regularLoginResponse.Token)

			w := httptest.NewRecorder()
			app.Router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			user := response["user"].(map[string]interface{})
			assert.Equal(t, "SelfUpdated", user["first_name"])
			assert.Equal(t, "Name", user["last_name"])
		})

		t.Run("should not allow regular user to update other users", func(t *testing.T) {
			updateRequest := entities.UpdateUserRequest{
				FirstName: stringPtr("Forbidden"),
			}

			body, _ := json.Marshal(updateRequest)
			req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/users/%s", createdAdmin.ID), bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+regularLoginResponse.Token)

			w := httptest.NewRecorder()
			app.Router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusForbidden, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Equal(t, "Cannot update other users", response["error"])
		})

		t.Run("should return error for invalid UUID", func(t *testing.T) {
			updateRequest := entities.UpdateUserRequest{
				FirstName: stringPtr("Test"),
			}

			body, _ := json.Marshal(updateRequest)
			req, _ := http.NewRequest("PUT", "/api/v1/users/invalid-uuid", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+adminLoginResponse.Token)

			w := httptest.NewRecorder()
			app.Router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	})

	t.Run("Delete User", func(t *testing.T) {
		// Create a user specifically for deletion test
		deleteUser := &entities.CreateUserRequest{
			Email:     "delete-test@example.com",
			Password:  "delete123",
			FirstName: "Delete",
			LastName:  "Test",
			Role:      "user",
		}

		createdDeleteUser, err := app.UserService.Create(app.Environment.Ctx, deleteUser)
		require.NoError(t, err)

		t.Run("should delete user with admin privileges", func(t *testing.T) {
			req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/v1/users/%s", createdDeleteUser.ID), nil)
			req.Header.Set("Authorization", "Bearer "+adminLoginResponse.Token)

			w := httptest.NewRecorder()
			app.Router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Equal(t, "User deleted successfully", response["message"])

			// Verify user is no longer accessible
			getReq, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/users/%s", createdDeleteUser.ID), nil)
			getReq.Header.Set("Authorization", "Bearer "+adminLoginResponse.Token)

			getW := httptest.NewRecorder()
			app.Router.ServeHTTP(getW, getReq)

			assert.Equal(t, http.StatusNotFound, getW.Code)
		})

		t.Run("should not allow regular user to delete other users", func(t *testing.T) {
			req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/v1/users/%s", createdAdmin.ID), nil)
			req.Header.Set("Authorization", "Bearer "+regularLoginResponse.Token)

			w := httptest.NewRecorder()
			app.Router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusForbidden, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Equal(t, "Cannot delete other users", response["error"])
		})

		t.Run("should allow user to delete their own account", func(t *testing.T) {
			// Create a user for self-deletion test
			selfDeleteUser := &entities.CreateUserRequest{
				Email:     "self-delete@example.com",
				Password:  "selfdelete123",
				FirstName: "SelfDelete",
				LastName:  "Test",
				Role:      "user",
			}

			createdSelfDeleteUser, err := app.UserService.Create(app.Environment.Ctx, selfDeleteUser)
			require.NoError(t, err)

			// Login as this user
			selfDeleteLoginRequest := &entities.LoginRequest{
				Email:    selfDeleteUser.Email,
				Password: selfDeleteUser.Password,
			}

			selfDeleteLoginResponse, err := app.UserService.Login(app.Environment.Ctx, selfDeleteLoginRequest)
			require.NoError(t, err)

			req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/v1/users/%s", createdSelfDeleteUser.ID), nil)
			req.Header.Set("Authorization", "Bearer "+selfDeleteLoginResponse.Token)

			w := httptest.NewRecorder()
			app.Router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	})

	t.Run("List Users", func(t *testing.T) {
		t.Run("should list users with pagination", func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/api/v1/users/?page=1&limit=5", nil)
			req.Header.Set("Authorization", "Bearer "+adminLoginResponse.Token)

			w := httptest.NewRecorder()
			app.Router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Contains(t, response, "users")
			assert.Contains(t, response, "pagination")

			pagination := response["pagination"].(map[string]interface{})
			assert.Equal(t, float64(1), pagination["page"])
			assert.Equal(t, float64(5), pagination["limit"])
			assert.Contains(t, pagination, "total")
			assert.Contains(t, pagination, "total_pages")

			users := response["users"].([]interface{})
			assert.True(t, len(users) >= 0)

			// Verify user structure if users exist
			if len(users) > 0 {
				user := users[0].(map[string]interface{})
				assert.Contains(t, user, "id")
				assert.Contains(t, user, "email")
				assert.Contains(t, user, "first_name")
				assert.Contains(t, user, "last_name")
				assert.Contains(t, user, "role")
				assert.Contains(t, user, "is_active")
			}
		})

		t.Run("should handle default pagination parameters", func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/api/v1/users/", nil)
			req.Header.Set("Authorization", "Bearer "+adminLoginResponse.Token)

			w := httptest.NewRecorder()
			app.Router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			pagination := response["pagination"].(map[string]interface{})
			assert.Equal(t, float64(1), pagination["page"])  // Default page
			assert.Equal(t, float64(10), pagination["limit"]) // Default limit
		})

		t.Run("should require authentication", func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/api/v1/users/", nil)

			w := httptest.NewRecorder()
			app.Router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	})
}

func TestUserSearchOperations(t *testing.T) {
	app := SetupTestApp(t)
	defer app.TeardownTestApp(t)

	// Create admin user
	adminUser := &entities.CreateUserRequest{
		Email:     "search-admin@example.com",
		Password:  "admin123",
		FirstName: "Search",
		LastName:  "Admin",
		Role:      "admin",
	}

	_, err := app.UserService.Create(app.Environment.Ctx, adminUser)
	require.NoError(t, err)

	// Login as admin
	adminLoginRequest := &entities.LoginRequest{
		Email:    adminUser.Email,
		Password: adminUser.Password,
	}

	adminLoginResponse, err := app.UserService.Login(app.Environment.Ctx, adminLoginRequest)
	require.NoError(t, err)

	// Create test users for searching
	testUsers := []*entities.CreateUserRequest{
		{
			Email:     "john.doe@example.com",
			Password:  "password123",
			FirstName: "John",
			LastName:  "Doe",
			Role:      "user",
		},
		{
			Email:     "jane.smith@example.com",
			Password:  "password123",
			FirstName: "Jane",
			LastName:  "Smith",
			Role:      "user",
		},
		{
			Email:     "alice.johnson@example.com",
			Password:  "password123",
			FirstName: "Alice",
			LastName:  "Johnson",
			Role:      "admin",
		},
	}

	for _, user := range testUsers {
		_, err := app.UserService.Create(app.Environment.Ctx, user)
		require.NoError(t, err)
	}

	t.Run("Search Users", func(t *testing.T) {
		t.Run("should search users by first name", func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/api/v1/users/search?q=john", nil)
			req.Header.Set("Authorization", "Bearer "+adminLoginResponse.Token)

			w := httptest.NewRecorder()
			app.Router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Contains(t, response, "users")
			assert.Contains(t, response, "query")
			assert.Equal(t, "john", response["query"])

			users := response["users"].([]interface{})
			// Should find John Doe and Alice Johnson (contains "john")
			assert.True(t, len(users) >= 1)
		})

		t.Run("should search users by email", func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/api/v1/users/search?q=jane.smith", nil)
			req.Header.Set("Authorization", "Bearer "+adminLoginResponse.Token)

			w := httptest.NewRecorder()
			app.Router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			users := response["users"].([]interface{})
			assert.Equal(t, 1, len(users))

			user := users[0].(map[string]interface{})
			assert.Equal(t, "jane.smith@example.com", user["email"])
		})

		t.Run("should return empty results for non-matching query", func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/api/v1/users/search?q=nonexistent", nil)
			req.Header.Set("Authorization", "Bearer "+adminLoginResponse.Token)

			w := httptest.NewRecorder()
			app.Router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			users := response["users"].([]interface{})
			assert.Equal(t, 0, len(users))

			pagination := response["pagination"].(map[string]interface{})
			assert.Equal(t, float64(0), pagination["total"])
		})

		t.Run("should require search query", func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/api/v1/users/search", nil)
			req.Header.Set("Authorization", "Bearer "+adminLoginResponse.Token)

			w := httptest.NewRecorder()
			app.Router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Equal(t, "Search query is required", response["error"])
		})

		t.Run("should support pagination in search", func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/api/v1/users/search?q=@example.com&page=1&limit=2", nil)
			req.Header.Set("Authorization", "Bearer "+adminLoginResponse.Token)

			w := httptest.NewRecorder()
			app.Router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			pagination := response["pagination"].(map[string]interface{})
			assert.Equal(t, float64(1), pagination["page"])
			assert.Equal(t, float64(2), pagination["limit"])
		})
	})
}

// Helper function to create string pointers for optional fields
func stringPtr(s string) *string {
	return &s
}

// Helper function to create bool pointers for optional fields
func boolPtr(b bool) *bool {
	return &b
}