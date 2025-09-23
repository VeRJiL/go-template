package e2e

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	migratePostgres "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/stretchr/testify/require"

	"github.com/VeRJiL/go-template/internal/config"
	dbPostgres "github.com/VeRJiL/go-template/internal/database/postgres"
)

const (
	TestDBHost     = "localhost"
	TestDBPort     = "5433"
	TestDBUser     = "test_user"
	TestDBPassword = "test_password"
	TestDBName     = "go_template_test"

	TestRedisHost = "localhost"
	TestRedisPort = "6380"
	TestRedisDB   = 0
)

// TestEnvironment holds the test infrastructure
type TestEnvironment struct {
	DB     *sql.DB
	Config *config.Config
	Ctx    context.Context
	Cancel context.CancelFunc
}

// SetupTestEnvironment creates a fresh test environment
func SetupTestEnvironment(t *testing.T) *TestEnvironment {
	t.Helper()

	// Skip if running in short mode
	if testing.Short() {
		t.Skip("Skipping e2e tests in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	// Create test configuration
	cfg := createTestConfig()

	// Wait for database to be ready
	waitForDatabase(t, cfg.Database)

	// Connect to database
	db, err := dbPostgres.NewConnection(&cfg.Database)
	require.NoError(t, err, "Failed to connect to test database")

	// Drop all tables to ensure clean state
	cleanDatabase(t, db)

	// Run migrations
	runMigrations(t, cfg.Database)

	return &TestEnvironment{
		DB:     db,
		Config: cfg,
		Ctx:    ctx,
		Cancel: cancel,
	}
}

// TeardownTestEnvironment cleans up the test environment
func (env *TestEnvironment) TeardownTestEnvironment(t *testing.T) {
	t.Helper()

	if env.DB != nil {
		// Clean the database
		cleanDatabase(t, env.DB)
		env.DB.Close()
	}

	if env.Cancel != nil {
		env.Cancel()
	}
}

// createTestConfig creates a configuration for testing
func createTestConfig() *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			Host:         "localhost",
			Port:         "8080",
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		Database: config.DatabaseConfig{
			Host:         TestDBHost,
			Port:         TestDBPort,
			User:         TestDBUser,
			Password:     TestDBPassword,
			Database:     TestDBName,
			SSLMode:      "disable",
			MaxOpenConns: 10,
			MaxIdleConns: 5,
		},
		Redis: config.RedisConfig{
			Host: TestRedisHost,
			Port: TestRedisPort,
			DB:   TestRedisDB,
		},
		Auth: config.AuthConfig{
			JWT: config.JWTConfig{
				Secret:     "test-secret-key-for-testing-only",
				Expiration: 1 * time.Hour,
			},
		},
		Logging: config.LoggingConfig{
			Level:  "info",
			Format: "text",
		},
	}
}

// waitForDatabase waits for the database to be ready
func waitForDatabase(t *testing.T, cfg config.DatabaseConfig) {
	t.Helper()

	maxRetries := 30
	for i := 0; i < maxRetries; i++ {
		db, err := sql.Open("postgres", fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database, cfg.SSLMode,
		))
		if err == nil {
			if err := db.Ping(); err == nil {
				db.Close()
				return
			}
			db.Close()
		}

		time.Sleep(1 * time.Second)
	}

	t.Fatalf("Database not ready after %d seconds", maxRetries)
}

// runMigrations runs database migrations
func runMigrations(t *testing.T, cfg config.DatabaseConfig) {
	t.Helper()

	// Connect to database for migrations
	db, err := sql.Open("postgres", fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database, cfg.SSLMode,
	))
	require.NoError(t, err, "Failed to connect for migrations")
	defer db.Close()

	// Create migration driver
	driver, err := migratePostgres.WithInstance(db, &migratePostgres.Config{})
	require.NoError(t, err, "Failed to create migration driver")

	// Get project root to find migrations
	projectRoot, err := findProjectRoot()
	require.NoError(t, err, "Failed to find project root")

	migrationsPath := fmt.Sprintf("file://%s/migrations/postgres", projectRoot)

	// Create migrator
	m, err := migrate.NewWithDatabaseInstance(migrationsPath, "postgres", driver)
	require.NoError(t, err, "Failed to create migrator")

	// Run migrations
	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		require.NoError(t, err, "Failed to run migrations")
	}

	log.Printf("Migrations completed successfully")
}

// cleanDatabase drops all tables and recreates a clean database
func cleanDatabase(t *testing.T, db *sql.DB) {
	t.Helper()

	// Drop all tables and reset the database
	queries := []string{
		"DROP TABLE IF EXISTS users CASCADE",
		"DROP TABLE IF EXISTS schema_migrations CASCADE",
		"DROP FUNCTION IF EXISTS update_updated_at_column() CASCADE",
		"DROP EXTENSION IF EXISTS \"uuid-ossp\" CASCADE",
	}

	for _, query := range queries {
		_, err := db.Exec(query)
		if err != nil {
			t.Logf("Warning: Failed to execute cleanup query '%s': %v", query, err)
		}
	}

	log.Printf("Database cleaned successfully")
}

// findProjectRoot finds the project root directory
func findProjectRoot() (string, error) {
	// Start from current directory and go up until we find go.mod
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find project root (go.mod not found)")
		}
		dir = parent
	}
}

// SeedTestData creates test data in the database
func (env *TestEnvironment) SeedTestData(t *testing.T) {
	t.Helper()

	// This will be implemented in the next step with test fixtures
	log.Printf("Test data seeding placeholder - implement in fixtures")
}