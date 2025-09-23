package e2e

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDatabaseMigrations(t *testing.T) {
	env := SetupTestEnvironment(t)
	defer env.TeardownTestEnvironment(t)

	t.Run("should create users table with correct schema", func(t *testing.T) {
		// Verify users table exists
		var tableName string
		err := env.DB.QueryRow(`
			SELECT table_name
			FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = 'users'
		`).Scan(&tableName)

		require.NoError(t, err)
		assert.Equal(t, "users", tableName)
	})

	t.Run("should have correct column types", func(t *testing.T) {
		// Check column types
		rows, err := env.DB.Query(`
			SELECT column_name, data_type, is_nullable, column_default
			FROM information_schema.columns
			WHERE table_name = 'users'
			ORDER BY ordinal_position
		`)
		require.NoError(t, err)
		defer rows.Close()

		expectedColumns := map[string]struct {
			dataType   string
			isNullable string
			hasDefault bool
		}{
			"id":            {"uuid", "NO", true},
			"email":         {"character varying", "NO", false},
			"password_hash": {"character varying", "NO", false},
			"first_name":    {"character varying", "NO", false},
			"last_name":     {"character varying", "NO", false},
			"role":          {"character varying", "NO", true},
			"is_active":     {"boolean", "NO", true},
			"created_at":    {"timestamp with time zone", "NO", true},
			"updated_at":    {"timestamp with time zone", "NO", true},
		}

		columnCount := 0
		for rows.Next() {
			var columnName, dataType, isNullable string
			var columnDefault *string

			err := rows.Scan(&columnName, &dataType, &isNullable, &columnDefault)
			require.NoError(t, err)

			expectedCol, exists := expectedColumns[columnName]
			require.True(t, exists, "Unexpected column: %s", columnName)

			assert.Equal(t, expectedCol.dataType, dataType, "Wrong data type for column %s", columnName)
			assert.Equal(t, expectedCol.isNullable, isNullable, "Wrong nullable setting for column %s", columnName)

			if expectedCol.hasDefault {
				assert.NotNil(t, columnDefault, "Column %s should have a default value", columnName)
			}

			columnCount++
		}

		assert.Equal(t, len(expectedColumns), columnCount, "Wrong number of columns")
	})

	t.Run("should have correct indexes", func(t *testing.T) {
		// Check indexes exist
		expectedIndexes := []string{
			"users_pkey",           // Primary key
			"users_email_key",      // Unique constraint on email
			"idx_users_email",      // Email index
			"idx_users_is_active",  // Active users index
			"idx_users_created_at", // Created at index
			"idx_users_role",       // Role index
		}

		for _, indexName := range expectedIndexes {
			var exists bool
			err := env.DB.QueryRow(`
				SELECT EXISTS (
					SELECT 1 FROM pg_indexes
					WHERE tablename = 'users' AND indexname = $1
				)
			`, indexName).Scan(&exists)

			require.NoError(t, err)
			assert.True(t, exists, "Index %s should exist", indexName)
		}
	})

	t.Run("should have uuid-ossp extension", func(t *testing.T) {
		var extensionName string
		err := env.DB.QueryRow(`
			SELECT extname FROM pg_extension WHERE extname = 'uuid-ossp'
		`).Scan(&extensionName)

		require.NoError(t, err)
		assert.Equal(t, "uuid-ossp", extensionName)
	})

	t.Run("should have update trigger function", func(t *testing.T) {
		var functionName string
		err := env.DB.QueryRow(`
			SELECT routine_name
			FROM information_schema.routines
			WHERE routine_name = 'update_updated_at_column'
		`).Scan(&functionName)

		require.NoError(t, err)
		assert.Equal(t, "update_updated_at_column", functionName)
	})

	t.Run("should have update trigger", func(t *testing.T) {
		var triggerName string
		err := env.DB.QueryRow(`
			SELECT trigger_name
			FROM information_schema.triggers
			WHERE event_object_table = 'users' AND trigger_name = 'update_users_updated_at'
		`).Scan(&triggerName)

		require.NoError(t, err)
		assert.Equal(t, "update_users_updated_at", triggerName)
	})
}

func TestDatabaseConstraints(t *testing.T) {
	env := SetupTestEnvironment(t)
	defer env.TeardownTestEnvironment(t)

	t.Run("should enforce email uniqueness", func(t *testing.T) {
		email := "duplicate@example.com"

		// Insert first user
		_, err := env.DB.Exec(`
			INSERT INTO users (id, email, password_hash, first_name, last_name, role)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, uuid.New(), email, "hash1", "John", "Doe", "user")
		require.NoError(t, err)

		// Try to insert second user with same email
		_, err = env.DB.Exec(`
			INSERT INTO users (id, email, password_hash, first_name, last_name, role)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, uuid.New(), email, "hash2", "Jane", "Smith", "user")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate key")
	})

	t.Run("should enforce role constraint", func(t *testing.T) {
		// Try to insert user with invalid role
		_, err := env.DB.Exec(`
			INSERT INTO users (id, email, password_hash, first_name, last_name, role)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, uuid.New(), "test@example.com", "hash", "John", "Doe", "invalid_role")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "check constraint")
	})

	t.Run("should set default values", func(t *testing.T) {
		userID := uuid.New()

		// Insert user with minimal data
		_, err := env.DB.Exec(`
			INSERT INTO users (id, email, password_hash, first_name, last_name)
			VALUES ($1, $2, $3, $4, $5)
		`, userID, "defaults@example.com", "hash", "John", "Doe")
		require.NoError(t, err)

		// Check defaults were applied
		var role string
		var isActive bool
		err = env.DB.QueryRow(`
			SELECT role, is_active FROM users WHERE id = $1
		`, userID).Scan(&role, &isActive)

		require.NoError(t, err)
		assert.Equal(t, "user", role)
		assert.True(t, isActive)
	})

	t.Run("should auto-generate UUID for id", func(t *testing.T) {
		// Insert user without specifying ID
		var generatedID uuid.UUID
		err := env.DB.QueryRow(`
			INSERT INTO users (email, password_hash, first_name, last_name)
			VALUES ($1, $2, $3, $4)
			RETURNING id
		`, "autoid@example.com", "hash", "John", "Doe").Scan(&generatedID)

		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, generatedID)
	})

	t.Run("should auto-update updated_at on changes", func(t *testing.T) {
		userID := uuid.New()

		// Insert user
		_, err := env.DB.Exec(`
			INSERT INTO users (id, email, password_hash, first_name, last_name)
			VALUES ($1, $2, $3, $4, $5)
		`, userID, "update-test@example.com", "hash", "John", "Doe")
		require.NoError(t, err)

		// Get initial timestamps
		var createdAt, updatedAt1 string
		err = env.DB.QueryRow(`
			SELECT created_at, updated_at FROM users WHERE id = $1
		`, userID).Scan(&createdAt, &updatedAt1)
		require.NoError(t, err)

		// Wait a moment to ensure timestamp difference
		// (in real tests you might use a different approach)
		_, err = env.DB.Exec("SELECT pg_sleep(0.1)")
		require.NoError(t, err)

		// Update user
		_, err = env.DB.Exec(`
			UPDATE users SET first_name = 'Jane' WHERE id = $1
		`, userID)
		require.NoError(t, err)

		// Check updated_at changed but created_at didn't
		var createdAt2, updatedAt2 string
		err = env.DB.QueryRow(`
			SELECT created_at, updated_at FROM users WHERE id = $1
		`, userID).Scan(&createdAt2, &updatedAt2)
		require.NoError(t, err)

		assert.Equal(t, createdAt, createdAt2, "created_at should not change")
		assert.NotEqual(t, updatedAt1, updatedAt2, "updated_at should change")
	})
}