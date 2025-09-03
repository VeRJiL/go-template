package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/VeRJiL/go-template/internal/config"
	"github.com/VeRJiL/go-template/internal/domain/entities"
)

type PostgresTestSuite struct {
	suite.Suite
	db         *sql.DB
	repository *userRepository
	testDBName string
}

func (suite *PostgresTestSuite) SetupSuite() {
	// Create test database
	suite.testDBName = "go_template_test_postgres"

	// Connect to postgres to create test database
	adminDB, err := sql.Open("postgres", "postgres://verjil:admin1234@localhost:5432/postgres?sslmode=disable")
	require.NoError(suite.T(), err)
	defer adminDB.Close()

	// Drop test database if it exists
	_, err = adminDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", suite.testDBName))
	require.NoError(suite.T(), err)

	// Create test database
	_, err = adminDB.Exec(fmt.Sprintf("CREATE DATABASE %s", suite.testDBName))
	require.NoError(suite.T(), err)

	// Connect to test database using config
	cfg := &config.DatabaseConfig{
		Host:         "localhost",
		Port:         "5432",
		User:         "verjil",
		Password:     "admin1234",
		Database:     suite.testDBName,
		SSLMode:      "disable",
		MaxOpenConns: 25,
		MaxIdleConns: 5,
	}

	suite.db, err = NewConnection(cfg)
	require.NoError(suite.T(), err)

	// Create users table
	suite.createUsersTable()

	// Initialize repository
	suite.repository = &userRepository{db: suite.db}
}

func (suite *PostgresTestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.db.Close()
	}

	// Drop test database
	adminDB, err := sql.Open("postgres", "postgres://verjil:admin1234@localhost:5432/postgres?sslmode=disable")
	require.NoError(suite.T(), err)
	defer adminDB.Close()

	_, err = adminDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", suite.testDBName))
	require.NoError(suite.T(), err)
}

func (suite *PostgresTestSuite) BeforeTest(suiteName, testName string) {
	// Clean users table before each test
	_, err := suite.db.Exec("DELETE FROM users")
	if err != nil {
		suite.T().Logf("Warning: Could not clean users table: %v", err)
	}

	// Verify table is empty
	var count int
	err = suite.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		suite.T().Logf("Warning: Could not verify table cleanup: %v", err)
	} else if count > 0 {
		suite.T().Logf("Warning: Table still has %d users after cleanup", count)
	}
}

func (suite *PostgresTestSuite) createUsersTable() {
	createTableSQL := `
		CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

		CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			email VARCHAR(255) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			first_name VARCHAR(100) NOT NULL,
			last_name VARCHAR(100) NOT NULL,
			role VARCHAR(20) NOT NULL DEFAULT 'user' CHECK (role IN ('admin', 'user')),
			is_active BOOLEAN NOT NULL DEFAULT true,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
		CREATE INDEX IF NOT EXISTS idx_users_is_active ON users(is_active);

		CREATE OR REPLACE FUNCTION update_updated_at_column()
		RETURNS TRIGGER AS $$
		BEGIN
			NEW.updated_at = NOW();
			RETURN NEW;
		END;
		$$ language 'plpgsql';

		CREATE TRIGGER update_users_updated_at
			BEFORE UPDATE ON users
			FOR EACH ROW
			EXECUTE FUNCTION update_updated_at_column();
	`

	_, err := suite.db.Exec(createTableSQL)
	require.NoError(suite.T(), err)
}

func (suite *PostgresTestSuite) createTestUser() *entities.User {
	user := &entities.User{
		ID:        uuid.New(),
		Email:     fmt.Sprintf("test-%s@example.com", uuid.New().String()[:8]),
		Password:  "hashed_password",
		FirstName: "John",
		LastName:  "Doe",
		Role:      "user",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	return user
}

func (suite *PostgresTestSuite) createTestUserWithEmail(email string) *entities.User {
	user := &entities.User{
		ID:        uuid.New(),
		Email:     email,
		Password:  "hashed_password",
		FirstName: "John",
		LastName:  "Doe",
		Role:      "user",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	return user
}

func TestNewConnection(t *testing.T) {
	t.Run("should create database connection successfully", func(t *testing.T) {
		cfg := &config.DatabaseConfig{
			Host:         "localhost",
			Port:         "5432",
			User:         "verjil",
			Password:     "admin1234",
			Database:     "go_template",
			SSLMode:      "disable",
			MaxOpenConns: 10,
			MaxIdleConns: 2,
		}

		db, err := NewConnection(cfg)

		assert.NoError(t, err)
		assert.NotNil(t, db)

		// Test connection
		err = db.Ping()
		assert.NoError(t, err)

		db.Close()
	})

	t.Run("should return error for invalid database", func(t *testing.T) {
		cfg := &config.DatabaseConfig{
			Host:     "localhost",
			Port:     "5432",
			User:     "invalid_user",
			Password: "invalid_password",
			Database: "nonexistent_db",
			SSLMode:  "disable",
		}

		db, err := NewConnection(cfg)

		assert.Error(t, err)
		assert.Nil(t, db)
		assert.Contains(t, err.Error(), "failed to ping database")
	})

	t.Run("should set connection pool settings", func(t *testing.T) {
		cfg := &config.DatabaseConfig{
			Host:         "localhost",
			Port:         "5432",
			User:         "verjil",
			Password:     "admin1234",
			Database:     "go_template",
			SSLMode:      "disable",
			MaxOpenConns: 15,
			MaxIdleConns: 3,
		}

		db, err := NewConnection(cfg)
		require.NoError(t, err)
		defer db.Close()

		stats := db.Stats()
		assert.Equal(t, 15, stats.MaxOpenConnections)
	})
}

func TestNewUserRepository(t *testing.T) {
	t.Run("should create user repository", func(t *testing.T) {
		db := &sql.DB{} // Mock DB
		repo := NewUserRepository(db)

		assert.NotNil(t, repo)
		assert.IsType(t, &userRepository{}, repo)
	})
}

func (suite *PostgresTestSuite) TestUserRepository_Create() {
	suite.T().Run("should create user successfully", func(t *testing.T) {
		user := suite.createTestUser()

		err := suite.repository.Create(context.Background(), user)

		assert.NoError(t, err)

		// Verify user was created
		var count int
		err = suite.db.QueryRow("SELECT COUNT(*) FROM users WHERE email = $1", user.Email).Scan(&count)
		assert.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	suite.T().Run("should return error for duplicate email", func(t *testing.T) {
		email := "duplicate@example.com"
		user1 := suite.createTestUserWithEmail(email)
		user2 := suite.createTestUserWithEmail(email)
		user2.ID = uuid.New() // Different ID, same email

		// Create first user
		err := suite.repository.Create(context.Background(), user1)
		require.NoError(t, err)

		// Try to create second user with same email
		err = suite.repository.Create(context.Background(), user2)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	suite.T().Run("should handle context cancellation", func(t *testing.T) {
		user := suite.createTestUser()
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err := suite.repository.Create(ctx, user)
		assert.Error(t, err)
	})
}

func (suite *PostgresTestSuite) TestUserRepository_GetByID() {
	suite.T().Run("should get user by ID successfully", func(t *testing.T) {
		user := suite.createTestUser()

		// Create user first
		err := suite.repository.Create(context.Background(), user)
		require.NoError(t, err)

		// Get user by ID
		foundUser, err := suite.repository.GetByID(context.Background(), user.ID)

		assert.NoError(t, err)
		assert.NotNil(t, foundUser)
		assert.Equal(t, user.ID, foundUser.ID)
		assert.Equal(t, user.Email, foundUser.Email)
		assert.Equal(t, user.FirstName, foundUser.FirstName)
		assert.Equal(t, user.LastName, foundUser.LastName)
		assert.Equal(t, user.Role, foundUser.Role)
		assert.True(t, foundUser.IsActive)
	})

	suite.T().Run("should return error for non-existent user", func(t *testing.T) {
		nonExistentID := uuid.New()

		foundUser, err := suite.repository.GetByID(context.Background(), nonExistentID)

		assert.Error(t, err)
		assert.Nil(t, foundUser)
		assert.Contains(t, err.Error(), "user not found")
	})

	suite.T().Run("should not return inactive user", func(t *testing.T) {
		user := suite.createTestUserWithEmail("inactive1@example.com")
		user.IsActive = false

		// Create inactive user directly in database
		_, err := suite.db.Exec(
			"INSERT INTO users (id, email, password_hash, first_name, last_name, role, is_active, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)",
			user.ID, user.Email, user.Password, user.FirstName, user.LastName, user.Role, user.IsActive, user.CreatedAt, user.UpdatedAt,
		)
		require.NoError(t, err)

		// Try to get inactive user
		foundUser, err := suite.repository.GetByID(context.Background(), user.ID)

		assert.Error(t, err)
		assert.Nil(t, foundUser)
		assert.Contains(t, err.Error(), "user not found")
	})
}

func (suite *PostgresTestSuite) TestUserRepository_GetByEmail() {
	suite.T().Run("should get user by email successfully", func(t *testing.T) {
		user := suite.createTestUser()

		// Create user first
		err := suite.repository.Create(context.Background(), user)
		require.NoError(t, err)

		// Get user by email
		foundUser, err := suite.repository.GetByEmail(context.Background(), user.Email)

		assert.NoError(t, err)
		assert.NotNil(t, foundUser)
		assert.Equal(t, user.Email, foundUser.Email)
		assert.Equal(t, user.ID, foundUser.ID)
	})

	suite.T().Run("should return error for non-existent email", func(t *testing.T) {
		foundUser, err := suite.repository.GetByEmail(context.Background(), "nonexistent@example.com")

		assert.Error(t, err)
		assert.Nil(t, foundUser)
		assert.Contains(t, err.Error(), "user not found")
	})

	suite.T().Run("should return inactive user by email", func(t *testing.T) {
		user := suite.createTestUserWithEmail("inactive2@example.com")
		user.IsActive = false

		// Create inactive user directly in database
		_, err := suite.db.Exec(
			"INSERT INTO users (id, email, password_hash, first_name, last_name, role, is_active, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)",
			user.ID, user.Email, user.Password, user.FirstName, user.LastName, user.Role, user.IsActive, user.CreatedAt, user.UpdatedAt,
		)
		require.NoError(t, err)

		// Get user by email (should work for inactive users)
		foundUser, err := suite.repository.GetByEmail(context.Background(), user.Email)

		assert.NoError(t, err)
		assert.NotNil(t, foundUser)
		assert.False(t, foundUser.IsActive)
	})
}

func (suite *PostgresTestSuite) TestUserRepository_Update() {
	suite.T().Run("should update user successfully", func(t *testing.T) {
		user := suite.createTestUser()

		// Create user first
		err := suite.repository.Create(context.Background(), user)
		require.NoError(t, err)

		// Update user
		newFirstName := "Jane"
		newRole := "admin"
		updates := &entities.UpdateUserRequest{
			FirstName: &newFirstName,
			Role:      &newRole,
		}

		updatedUser, err := suite.repository.Update(context.Background(), user.ID, updates)

		assert.NoError(t, err)
		assert.NotNil(t, updatedUser)
		assert.Equal(t, newFirstName, updatedUser.FirstName)
		assert.Equal(t, newRole, updatedUser.Role)
		assert.Equal(t, user.LastName, updatedUser.LastName) // Unchanged
		assert.True(t, updatedUser.UpdatedAt.After(user.UpdatedAt))
	})

	suite.T().Run("should update only provided fields", func(t *testing.T) {
		user := suite.createTestUser()

		// Create user first
		err := suite.repository.Create(context.Background(), user)
		require.NoError(t, err)

		// Update only first name
		newFirstName := "Jane"
		updates := &entities.UpdateUserRequest{
			FirstName: &newFirstName,
		}

		updatedUser, err := suite.repository.Update(context.Background(), user.ID, updates)

		assert.NoError(t, err)
		assert.Equal(t, newFirstName, updatedUser.FirstName)
		assert.Equal(t, user.LastName, updatedUser.LastName) // Unchanged
		assert.Equal(t, user.Role, updatedUser.Role)         // Unchanged
		assert.Equal(t, user.IsActive, updatedUser.IsActive) // Unchanged
	})

	suite.T().Run("should return error for non-existent user", func(t *testing.T) {
		nonExistentID := uuid.New()
		newFirstName := "Jane"
		updates := &entities.UpdateUserRequest{
			FirstName: &newFirstName,
		}

		updatedUser, err := suite.repository.Update(context.Background(), nonExistentID, updates)

		assert.Error(t, err)
		assert.Nil(t, updatedUser)
		assert.Contains(t, err.Error(), "user not found")
	})
}

func (suite *PostgresTestSuite) TestUserRepository_Delete() {
	suite.T().Run("should delete user successfully", func(t *testing.T) {
		user := suite.createTestUser()

		// Create user first
		err := suite.repository.Create(context.Background(), user)
		require.NoError(t, err)

		// Delete user
		err = suite.repository.Delete(context.Background(), user.ID)

		assert.NoError(t, err)

		// Verify user was soft deleted (still exists but inactive)
		var isActive bool
		err = suite.db.QueryRow("SELECT is_active FROM users WHERE id = $1", user.ID).Scan(&isActive)
		assert.NoError(t, err)
		assert.False(t, isActive)

		// Verify user is no longer accessible via GetByID (soft delete)
		_, err = suite.repository.GetByID(context.Background(), user.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	suite.T().Run("should return error for non-existent user", func(t *testing.T) {
		nonExistentID := uuid.New()

		err := suite.repository.Delete(context.Background(), nonExistentID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})
}

func (suite *PostgresTestSuite) TestUserRepository_List() {
	suite.T().Run("should list users with pagination", func(t *testing.T) {
		// Create multiple users
		users := make([]*entities.User, 5)
		for i := 0; i < 5; i++ {
			user := suite.createTestUser()
			user.Email = fmt.Sprintf("user%d@example.com", i)
			users[i] = user

			err := suite.repository.Create(context.Background(), user)
			require.NoError(t, err)
		}

		// List users with pagination
		foundUsers, total, err := suite.repository.List(context.Background(), 0, 3)

		assert.NoError(t, err)
		assert.Len(t, foundUsers, 3)
		assert.Equal(t, 5, total)
	})

	suite.T().Run("should return empty list when no active users", func(t *testing.T) {
		// Clean table first
		_, err := suite.db.Exec("DELETE FROM users")
		require.NoError(t, err)

		// Create an inactive user to ensure the list filters properly
		user := suite.createTestUserWithEmail("inactive-list@example.com")
		user.IsActive = false
		_, err = suite.db.Exec(
			"INSERT INTO users (id, email, password_hash, first_name, last_name, role, is_active, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)",
			user.ID, user.Email, user.Password, user.FirstName, user.LastName, user.Role, user.IsActive, user.CreatedAt, user.UpdatedAt,
		)
		require.NoError(t, err)

		// List should return empty since only inactive user exists
		foundUsers, total, err := suite.repository.List(context.Background(), 0, 10)

		assert.NoError(t, err)
		assert.Empty(t, foundUsers)
		assert.Equal(t, 0, total)
	})

	suite.T().Run("should handle offset beyond total", func(t *testing.T) {
		// Clean table first
		_, err := suite.db.Exec("DELETE FROM users")
		require.NoError(t, err)

		// Create one user
		user := suite.createTestUser()
		err = suite.repository.Create(context.Background(), user)
		require.NoError(t, err)

		// Request with large offset
		foundUsers, total, err := suite.repository.List(context.Background(), 100, 10)

		assert.NoError(t, err)
		assert.Empty(t, foundUsers)
		assert.Equal(t, 1, total) // Total should still be correct
	})
}

func (suite *PostgresTestSuite) TestUserRepository_Search() {
	suite.T().Run("should search users by query", func(t *testing.T) {
		// Create users with different names
		users := []*entities.User{
			{ID: uuid.New(), Email: "john.doe@example.com", Password: "hash", FirstName: "John", LastName: "Doe", Role: "user", IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
			{ID: uuid.New(), Email: "jane.smith@example.com", Password: "hash", FirstName: "Jane", LastName: "Smith", Role: "user", IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
			{ID: uuid.New(), Email: "bob.johnson@example.com", Password: "hash", FirstName: "Bob", LastName: "Johnson", Role: "admin", IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		}

		for _, user := range users {
			err := suite.repository.Create(context.Background(), user)
			require.NoError(t, err)
		}

		// Search for "john"
		foundUsers, total, err := suite.repository.Search(context.Background(), "john", 0, 10)

		assert.NoError(t, err)
		assert.Len(t, foundUsers, 2) // John Doe and Bob Johnson
		assert.Equal(t, 2, total)
	})

	suite.T().Run("should return empty when no matches", func(t *testing.T) {
		user := suite.createTestUser()
		err := suite.repository.Create(context.Background(), user)
		require.NoError(t, err)

		foundUsers, total, err := suite.repository.Search(context.Background(), "xyz", 0, 10)

		assert.NoError(t, err)
		assert.Empty(t, foundUsers)
		assert.Equal(t, 0, total)
	})

	suite.T().Run("should search case insensitively", func(t *testing.T) {
		user := suite.createTestUserWithEmail("alice@example.com")
		user.FirstName = "Alice"
		err := suite.repository.Create(context.Background(), user)
		require.NoError(t, err)

		foundUsers, total, err := suite.repository.Search(context.Background(), "alice", 0, 10)

		assert.NoError(t, err)
		assert.Len(t, foundUsers, 1)
		assert.Equal(t, 1, total)
		assert.Equal(t, "Alice", foundUsers[0].FirstName)
	})
}

func TestPostgresTestSuite(t *testing.T) {
	// Skip if running in short mode
	if testing.Short() {
		t.Skip("Skipping postgres integration tests in short mode")
	}

	suite.Run(t, new(PostgresTestSuite))
}