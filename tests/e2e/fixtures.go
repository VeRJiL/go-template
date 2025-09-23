package e2e

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/VeRJiL/go-template/internal/domain/entities"
	"github.com/VeRJiL/go-template/internal/domain/services"
	"github.com/VeRJiL/go-template/internal/pkg/auth"
)

// TestFixtures holds all test data fixtures
type TestFixtures struct {
	Users  []TestUser
	Tokens map[string]string // email -> token mapping
}

// TestUser represents a test user with credentials
type TestUser struct {
	User     *entities.User
	Password string
	Token    string
}

// LoadTestFixtures creates and loads test data into the database
func LoadTestFixtures(t *testing.T, userService *services.UserService, jwtService *auth.JWTService) *TestFixtures {
	t.Helper()

	fixtures := &TestFixtures{
		Users:  make([]TestUser, 0),
		Tokens: make(map[string]string),
	}

	ctx := context.Background()

	// Create test users
	testUserData := []struct {
		email     string
		password  string
		firstName string
		lastName  string
		role      string
	}{
		{"admin@example.com", "admin123", "Admin", "User", "admin"},
		{"john.doe@example.com", "password123", "John", "Doe", "user"},
		{"jane.smith@example.com", "password123", "Jane", "Smith", "user"},
		{"alice.johnson@example.com", "password123", "Alice", "Johnson", "user"},
		{"bob.wilson@example.com", "password123", "Bob", "Wilson", "user"},
		{"charlie.brown@example.com", "password123", "Charlie", "Brown", "user"},
		{"diana.prince@example.com", "password123", "Diana", "Prince", "admin"},
		{"eve.adams@example.com", "password123", "Eve", "Adams", "user"},
		{"frank.miller@example.com", "password123", "Frank", "Miller", "user"},
		{"grace.kelly@example.com", "password123", "Grace", "Kelly", "user"},
	}

	for i, userData := range testUserData {
		t.Logf("Creating test user %d: %s", i+1, userData.email)

		// Create user request
		createRequest := &entities.CreateUserRequest{
			Email:     userData.email,
			Password:  userData.password,
			FirstName: userData.firstName,
			LastName:  userData.lastName,
			Role:      userData.role,
		}

		// Create user
		user, err := userService.Create(ctx, createRequest)
		require.NoError(t, err, "Failed to create test user: %s", userData.email)

		// Login to get token
		loginRequest := &entities.LoginRequest{
			Email:    userData.email,
			Password: userData.password,
		}

		loginResponse, err := userService.Login(ctx, loginRequest)
		require.NoError(t, err, "Failed to login test user: %s", userData.email)

		// Store test user
		testUser := TestUser{
			User:     user,
			Password: userData.password,
			Token:    loginResponse.Token,
		}

		fixtures.Users = append(fixtures.Users, testUser)
		fixtures.Tokens[userData.email] = loginResponse.Token
	}

	t.Logf("Successfully created %d test users", len(fixtures.Users))
	return fixtures
}

// GetUserByEmail returns a test user by email
func (f *TestFixtures) GetUserByEmail(email string) *TestUser {
	for _, user := range f.Users {
		if user.User.Email == email {
			return &user
		}
	}
	return nil
}

// GetUserByRole returns the first test user with the specified role
func (f *TestFixtures) GetUserByRole(role string) *TestUser {
	for _, user := range f.Users {
		if user.User.Role == role {
			return &user
		}
	}
	return nil
}

// GetAdminUser returns the first admin user
func (f *TestFixtures) GetAdminUser() *TestUser {
	return f.GetUserByRole("admin")
}

// GetRegularUser returns the first regular user
func (f *TestFixtures) GetRegularUser() *TestUser {
	return f.GetUserByRole("user")
}

// GetTokenFor returns the token for a user by email
func (f *TestFixtures) GetTokenFor(email string) string {
	return f.Tokens[email]
}

// SeedDatabaseDirectly seeds data directly into the database (bypassing service layer)
func SeedDatabaseDirectly(t *testing.T, env *TestEnvironment) *DirectFixtures {
	t.Helper()

	fixtures := &DirectFixtures{
		UserIDs: make([]uuid.UUID, 0),
	}

	// Insert users directly into database
	directUsers := []struct {
		email     string
		firstName string
		lastName  string
		role      string
	}{
		{"direct1@example.com", "Direct", "User1", "user"},
		{"direct2@example.com", "Direct", "User2", "admin"},
		{"direct3@example.com", "Direct", "User3", "user"},
	}

	for _, userData := range directUsers {
		userID := uuid.New()

		_, err := env.DB.Exec(`
			INSERT INTO users (id, email, password_hash, first_name, last_name, role, is_active, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		`, userID, userData.email, "hashed_password", userData.firstName, userData.lastName, userData.role, true)

		require.NoError(t, err, "Failed to insert direct user: %s", userData.email)
		fixtures.UserIDs = append(fixtures.UserIDs, userID)
	}

	t.Logf("Successfully seeded %d users directly", len(fixtures.UserIDs))
	return fixtures
}

// DirectFixtures holds fixtures created directly in the database
type DirectFixtures struct {
	UserIDs []uuid.UUID
}

// CreateLargeDataset creates a large dataset for performance testing
func CreateLargeDataset(t *testing.T, env *TestEnvironment, count int) []uuid.UUID {
	t.Helper()

	userIDs := make([]uuid.UUID, 0, count)

	// Batch insert for better performance
	batchSize := 100
	for i := 0; i < count; i += batchSize {
		endIdx := i + batchSize
		if endIdx > count {
			endIdx = count
		}

		// Build batch insert query
		query := `INSERT INTO users (id, email, password_hash, first_name, last_name, role, is_active, created_at, updated_at) VALUES `
		values := make([]interface{}, 0)
		placeholders := make([]string, 0)

		for j := i; j < endIdx; j++ {
			userID := uuid.New()
			userIDs = append(userIDs, userID)

			placeholder := fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d, NOW(), NOW())",
				len(values)+1, len(values)+2, len(values)+3, len(values)+4,
				len(values)+5, len(values)+6, len(values)+7)
			placeholders = append(placeholders, placeholder)

			values = append(values,
				userID,
				fmt.Sprintf("bulk%d@example.com", j),
				"hashed_password",
				fmt.Sprintf("Bulk%d", j),
				fmt.Sprintf("User%d", j),
				"user",
				true,
			)
		}

		finalQuery := query + fmt.Sprintf("%s", placeholders[0])
		for k := 1; k < len(placeholders); k++ {
			finalQuery += ", " + placeholders[k]
		}

		_, err := env.DB.Exec(finalQuery, values...)
		require.NoError(t, err, "Failed to insert batch of users")
	}

	t.Logf("Successfully created %d bulk users", count)
	return userIDs
}

// CleanupFixtures removes all test data from the database
func CleanupFixtures(t *testing.T, env *TestEnvironment) {
	t.Helper()

	// Delete all users
	_, err := env.DB.Exec("DELETE FROM users")
	if err != nil {
		t.Logf("Warning: Failed to cleanup test users: %v", err)
	}

	// Verify cleanup
	var count int
	err = env.DB.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		t.Logf("Warning: Failed to verify cleanup: %v", err)
	} else if count > 0 {
		t.Logf("Warning: %d users remain after cleanup", count)
	} else {
		t.Logf("Successfully cleaned up all test data")
	}
}

// VerifyDatabaseState verifies the database is in expected state
func VerifyDatabaseState(t *testing.T, env *TestEnvironment, expectedUserCount int) {
	t.Helper()

	var actualCount int
	err := env.DB.QueryRow("SELECT COUNT(*) FROM users WHERE is_active = true").Scan(&actualCount)
	require.NoError(t, err, "Failed to count active users")

	require.Equal(t, expectedUserCount, actualCount,
		"Expected %d active users, found %d", expectedUserCount, actualCount)

	t.Logf("Database state verified: %d active users", actualCount)
}

// CreateTestUserWithCustomData creates a user with custom data for specific tests
func CreateTestUserWithCustomData(t *testing.T, userService *services.UserService,
	email, password, firstName, lastName, role string) (*entities.User, string) {
	t.Helper()

	ctx := context.Background()

	// Create user
	createRequest := &entities.CreateUserRequest{
		Email:     email,
		Password:  password,
		FirstName: firstName,
		LastName:  lastName,
		Role:      role,
	}

	user, err := userService.Create(ctx, createRequest)
	require.NoError(t, err, "Failed to create custom test user")

	// Login to get token
	loginRequest := &entities.LoginRequest{
		Email:    email,
		Password: password,
	}

	loginResponse, err := userService.Login(ctx, loginRequest)
	require.NoError(t, err, "Failed to login custom test user")

	return user, loginResponse.Token
}