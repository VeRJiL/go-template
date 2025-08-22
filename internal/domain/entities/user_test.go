package entities

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUser_BeforeCreate(t *testing.T) {
	t.Run("should set ID if not provided", func(t *testing.T) {
		user := &User{}
		user.BeforeCreate()

		assert.NotEqual(t, uuid.Nil, user.ID)
		assert.NotEmpty(t, user.ID.String())
	})

	t.Run("should not overwrite existing ID", func(t *testing.T) {
		existingID := uuid.New()
		user := &User{ID: existingID}
		user.BeforeCreate()

		assert.Equal(t, existingID, user.ID)
	})

	t.Run("should set timestamps", func(t *testing.T) {
		user := &User{}
		beforeTime := time.Now()

		user.BeforeCreate()

		afterTime := time.Now()

		assert.True(t, user.CreatedAt.After(beforeTime) || user.CreatedAt.Equal(beforeTime))
		assert.True(t, user.CreatedAt.Before(afterTime) || user.CreatedAt.Equal(afterTime))
		assert.Equal(t, user.CreatedAt, user.UpdatedAt)
	})

	t.Run("should set default role if empty", func(t *testing.T) {
		user := &User{}
		user.BeforeCreate()

		assert.Equal(t, "user", user.Role)
	})

	t.Run("should not overwrite existing role", func(t *testing.T) {
		user := &User{Role: "admin"}
		user.BeforeCreate()

		assert.Equal(t, "admin", user.Role)
	})

	t.Run("should set IsActive to true", func(t *testing.T) {
		user := &User{}
		user.BeforeCreate()

		assert.True(t, user.IsActive)
	})

	t.Run("should set IsActive to true even if previously false", func(t *testing.T) {
		user := &User{IsActive: false}
		user.BeforeCreate()

		assert.True(t, user.IsActive)
	})
}

func TestUser_BeforeUpdate(t *testing.T) {
	t.Run("should update UpdatedAt timestamp", func(t *testing.T) {
		user := &User{
			CreatedAt: time.Now().Add(-1 * time.Hour),
			UpdatedAt: time.Now().Add(-1 * time.Hour),
		}

		beforeTime := time.Now()
		user.BeforeUpdate()
		afterTime := time.Now()

		assert.True(t, user.UpdatedAt.After(beforeTime) || user.UpdatedAt.Equal(beforeTime))
		assert.True(t, user.UpdatedAt.Before(afterTime) || user.UpdatedAt.Equal(afterTime))
	})

	t.Run("should not modify CreatedAt", func(t *testing.T) {
		originalCreatedAt := time.Now().Add(-1 * time.Hour)
		user := &User{
			CreatedAt: originalCreatedAt,
			UpdatedAt: time.Now().Add(-30 * time.Minute),
		}

		user.BeforeUpdate()

		assert.Equal(t, originalCreatedAt, user.CreatedAt)
		assert.NotEqual(t, originalCreatedAt, user.UpdatedAt)
	})
}

func TestUser_JSONSerialization(t *testing.T) {
	t.Run("should serialize user to JSON", func(t *testing.T) {
		user := &User{
			ID:        uuid.New(),
			Email:     "test@example.com",
			FirstName: "John",
			LastName:  "Doe",
			Role:      "user",
			IsActive:  true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		jsonData, err := json.Marshal(user)
		require.NoError(t, err)
		assert.Contains(t, string(jsonData), `"email":"test@example.com"`)
		assert.Contains(t, string(jsonData), `"first_name":"John"`)
		assert.Contains(t, string(jsonData), `"last_name":"Doe"`)
		assert.Contains(t, string(jsonData), `"role":"user"`)
		assert.Contains(t, string(jsonData), `"is_active":true`)
	})

	t.Run("should not include password in JSON", func(t *testing.T) {
		user := &User{
			Email:    "test@example.com",
			Password: "secret-password",
		}

		jsonData, err := json.Marshal(user)
		require.NoError(t, err)
		assert.NotContains(t, string(jsonData), "secret-password")
		assert.NotContains(t, string(jsonData), "password")
	})

	t.Run("should deserialize JSON to user", func(t *testing.T) {
		jsonData := `{
			"id": "123e4567-e89b-12d3-a456-426614174000",
			"email": "test@example.com",
			"first_name": "John",
			"last_name": "Doe",
			"role": "admin",
			"is_active": false
		}`

		var user User
		err := json.Unmarshal([]byte(jsonData), &user)
		require.NoError(t, err)

		assert.Equal(t, "test@example.com", user.Email)
		assert.Equal(t, "John", user.FirstName)
		assert.Equal(t, "Doe", user.LastName)
		assert.Equal(t, "admin", user.Role)
		assert.False(t, user.IsActive)
	})
}

func TestUser_AvatarFields(t *testing.T) {
	t.Run("should handle nil avatar fields", func(t *testing.T) {
		user := &User{
			Email: "test@example.com",
		}

		jsonData, err := json.Marshal(user)
		require.NoError(t, err)

		// Should not include omitempty fields when nil
		assert.NotContains(t, string(jsonData), "avatar")
	})

	t.Run("should include avatar fields when set", func(t *testing.T) {
		avatarURL := "https://example.com/avatar.jpg"
		avatarPath := "/storage/avatars/123.jpg"
		avatarOriginal := "user-avatar.jpg"

		user := &User{
			Email:          "test@example.com",
			Avatar:         &avatarURL,
			AvatarPath:     &avatarPath,
			AvatarOriginal: &avatarOriginal,
		}

		jsonData, err := json.Marshal(user)
		require.NoError(t, err)

		assert.Contains(t, string(jsonData), avatarURL)
		assert.Contains(t, string(jsonData), avatarPath)
		assert.Contains(t, string(jsonData), avatarOriginal)
	})
}

func TestCreateUserRequest(t *testing.T) {
	t.Run("should serialize and deserialize correctly", func(t *testing.T) {
		request := CreateUserRequest{
			Email:     "test@example.com",
			Password:  "password123",
			FirstName: "John",
			LastName:  "Doe",
			Role:      "user",
		}

		jsonData, err := json.Marshal(request)
		require.NoError(t, err)

		var decoded CreateUserRequest
		err = json.Unmarshal(jsonData, &decoded)
		require.NoError(t, err)

		assert.Equal(t, request.Email, decoded.Email)
		assert.Equal(t, request.Password, decoded.Password)
		assert.Equal(t, request.FirstName, decoded.FirstName)
		assert.Equal(t, request.LastName, decoded.LastName)
		assert.Equal(t, request.Role, decoded.Role)
	})
}

func TestUpdateUserRequest(t *testing.T) {
	t.Run("should handle nil fields correctly", func(t *testing.T) {
		request := UpdateUserRequest{}

		jsonData, err := json.Marshal(request)
		require.NoError(t, err)

		// Should not include omitempty fields when nil
		jsonStr := string(jsonData)
		assert.Equal(t, "{}", jsonStr)
	})

	t.Run("should include non-nil fields", func(t *testing.T) {
		firstName := "Jane"
		role := "admin"
		isActive := false

		request := UpdateUserRequest{
			FirstName: &firstName,
			Role:      &role,
			IsActive:  &isActive,
		}

		jsonData, err := json.Marshal(request)
		require.NoError(t, err)

		jsonStr := string(jsonData)
		assert.Contains(t, jsonStr, `"first_name":"Jane"`)
		assert.Contains(t, jsonStr, `"role":"admin"`)
		assert.Contains(t, jsonStr, `"is_active":false`)
		assert.NotContains(t, jsonStr, "last_name")
	})

	t.Run("should deserialize partial updates", func(t *testing.T) {
		jsonData := `{"first_name": "Updated", "is_active": true}`

		var request UpdateUserRequest
		err := json.Unmarshal([]byte(jsonData), &request)
		require.NoError(t, err)

		require.NotNil(t, request.FirstName)
		assert.Equal(t, "Updated", *request.FirstName)

		require.NotNil(t, request.IsActive)
		assert.True(t, *request.IsActive)

		assert.Nil(t, request.LastName)
		assert.Nil(t, request.Role)
	})
}

func TestLoginRequest(t *testing.T) {
	t.Run("should serialize and deserialize correctly", func(t *testing.T) {
		request := LoginRequest{
			Email:    "test@example.com",
			Password: "password123",
		}

		jsonData, err := json.Marshal(request)
		require.NoError(t, err)

		var decoded LoginRequest
		err = json.Unmarshal(jsonData, &decoded)
		require.NoError(t, err)

		assert.Equal(t, request.Email, decoded.Email)
		assert.Equal(t, request.Password, decoded.Password)
	})
}

func TestLoginResponse(t *testing.T) {
	t.Run("should serialize and deserialize correctly", func(t *testing.T) {
		user := User{
			ID:        uuid.New(),
			Email:     "test@example.com",
			FirstName: "John",
			LastName:  "Doe",
			Role:      "user",
			IsActive:  true,
		}

		response := LoginResponse{
			Token:     "jwt-token-here",
			User:      user,
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}

		jsonData, err := json.Marshal(response)
		require.NoError(t, err)

		var decoded LoginResponse
		err = json.Unmarshal(jsonData, &decoded)
		require.NoError(t, err)

		assert.Equal(t, response.Token, decoded.Token)
		assert.Equal(t, response.User.Email, decoded.User.Email)
		assert.Equal(t, response.User.FirstName, decoded.User.FirstName)
		assert.True(t, response.ExpiresAt.Equal(decoded.ExpiresAt))
	})
}

func TestAvatarUploadResponse(t *testing.T) {
	t.Run("should serialize and deserialize correctly", func(t *testing.T) {
		response := AvatarUploadResponse{
			Avatar:         "https://example.com/avatar.jpg",
			AvatarPath:     "/storage/avatars/123.jpg",
			AvatarOriginal: "user-photo.jpg",
			Message:        "Avatar uploaded successfully",
		}

		jsonData, err := json.Marshal(response)
		require.NoError(t, err)

		var decoded AvatarUploadResponse
		err = json.Unmarshal(jsonData, &decoded)
		require.NoError(t, err)

		assert.Equal(t, response.Avatar, decoded.Avatar)
		assert.Equal(t, response.AvatarPath, decoded.AvatarPath)
		assert.Equal(t, response.AvatarOriginal, decoded.AvatarOriginal)
		assert.Equal(t, response.Message, decoded.Message)
	})
}

func TestAvatarDeleteResponse(t *testing.T) {
	t.Run("should serialize and deserialize correctly", func(t *testing.T) {
		response := AvatarDeleteResponse{
			Message: "Avatar deleted successfully",
		}

		jsonData, err := json.Marshal(response)
		require.NoError(t, err)

		var decoded AvatarDeleteResponse
		err = json.Unmarshal(jsonData, &decoded)
		require.NoError(t, err)

		assert.Equal(t, response.Message, decoded.Message)
	})
}

func TestUserEntityIntegration(t *testing.T) {
	t.Run("should create user with lifecycle methods", func(t *testing.T) {
		user := &User{
			Email:     "integration@example.com",
			Password:  "hashedpassword",
			FirstName: "Integration",
			LastName:  "Test",
		}

		// Simulate create
		user.BeforeCreate()

		assert.NotEqual(t, uuid.Nil, user.ID)
		assert.Equal(t, "user", user.Role)
		assert.True(t, user.IsActive)
		assert.False(t, user.CreatedAt.IsZero())
		assert.False(t, user.UpdatedAt.IsZero())

		originalUpdatedAt := user.UpdatedAt
		time.Sleep(1 * time.Millisecond) // Ensure time difference

		// Simulate update
		user.BeforeUpdate()

		assert.True(t, user.UpdatedAt.After(originalUpdatedAt))
		assert.Equal(t, user.CreatedAt, user.CreatedAt) // CreatedAt unchanged
	})

	t.Run("should work with complete user lifecycle", func(t *testing.T) {
		// Create request
		createReq := CreateUserRequest{
			Email:     "lifecycle@example.com",
			Password:  "securepassword",
			FirstName: "Life",
			LastName:  "Cycle",
			Role:      "admin",
		}

		// Convert to entity
		user := &User{
			Email:     createReq.Email,
			Password:  "hashed-" + createReq.Password,
			FirstName: createReq.FirstName,
			LastName:  createReq.LastName,
			Role:      createReq.Role,
		}

		user.BeforeCreate()

		// Update request
		newFirstName := "Updated"
		updateReq := UpdateUserRequest{
			FirstName: &newFirstName,
		}

		if updateReq.FirstName != nil {
			user.FirstName = *updateReq.FirstName
		}

		user.BeforeUpdate()

		// Verify final state
		assert.Equal(t, "Updated", user.FirstName)
		assert.Equal(t, "Cycle", user.LastName)
		assert.Equal(t, "admin", user.Role)
		assert.True(t, user.IsActive)
		assert.NotEqual(t, uuid.Nil, user.ID)
	})
}