package migration

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type MigrationCLITestSuite struct {
	suite.Suite
	db         *sql.DB
	testDBName string
	projectRoot string
	migratePath string
	dbURL      string
}

func (suite *MigrationCLITestSuite) SetupSuite() {
	// Create test database
	suite.testDBName = "go_template_test_migration_cli"

	// Get project root
	wd, err := os.Getwd()
	require.NoError(suite.T(), err)
	suite.projectRoot = filepath.Join(wd, "../..")
	suite.migratePath = filepath.Join(suite.projectRoot, "bin", "migrate")

	// Verify migrate binary exists
	_, err = os.Stat(suite.migratePath)
	require.NoError(suite.T(), err, "migrate binary should exist at %s", suite.migratePath)

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

	// Setup database connection and URL
	suite.dbURL = fmt.Sprintf("postgres://verjil:admin1234@localhost:5432/%s?sslmode=disable", suite.testDBName)
	suite.db, err = sql.Open("postgres", suite.dbURL)
	require.NoError(suite.T(), err)

	// Test connection
	err = suite.db.Ping()
	require.NoError(suite.T(), err)
}

func (suite *MigrationCLITestSuite) TearDownSuite() {
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

func (suite *MigrationCLITestSuite) SetupTest() {
	// Reset database state before each test
	suite.runMigrateCommand("drop", "-f")
}

func (suite *MigrationCLITestSuite) TestMigrateCLIUp() {
	// Test migrating up
	err := suite.runMigrateCommand("up")
	assert.NoError(suite.T(), err)

	// Verify migration version
	version := suite.getMigrationVersion()
	assert.Equal(suite.T(), "1", version)

	// Verify tables exist
	suite.verifyUsersTableExists()
	suite.verifySchemaTableExists()
}

func (suite *MigrationCLITestSuite) TestMigrateCLIDown() {
	// First migrate up
	err := suite.runMigrateCommand("up")
	require.NoError(suite.T(), err)

	// Then migrate down one step
	err = suite.runMigrateCommand("down", "1")
	assert.NoError(suite.T(), err)

	// Verify users table doesn't exist
	suite.verifyUsersTableNotExists()

	// Verify schema table still exists (it's not dropped by down migration)
	suite.verifySchemaTableExists()
}

func (suite *MigrationCLITestSuite) TestMigrationVersionCommand() {
	// Initially should have no version or error
	version := suite.getMigrationVersion()
	assert.Equal(suite.T(), "no migration", version)

	// After migration up, version should be 1
	err := suite.runMigrateCommand("up")
	assert.NoError(suite.T(), err)

	version = suite.getMigrationVersion()
	assert.Equal(suite.T(), "1", version)
}

func (suite *MigrationCLITestSuite) TestMigrationForceCommand() {
	// First migrate up
	err := suite.runMigrateCommand("up")
	require.NoError(suite.T(), err)

	// Manually set migration to dirty state
	_, err = suite.db.Exec("UPDATE schema_migrations SET dirty = true")
	require.NoError(suite.T(), err)

	// Version should show as dirty
	version := suite.getMigrationVersion()
	assert.Contains(suite.T(), version, "dirty")

	// Force clean the migration
	err = suite.runMigrateCommand("force", "1")
	assert.NoError(suite.T(), err)

	// Version should be clean now
	version = suite.getMigrationVersion()
	assert.Equal(suite.T(), "1", version)
}

func (suite *MigrationCLITestSuite) TestMigrationSteps() {
	// Test migrating up by steps
	err := suite.runMigrateCommand("up", "1")
	assert.NoError(suite.T(), err)

	version := suite.getMigrationVersion()
	assert.Equal(suite.T(), "1", version)

	// Test migrating down by steps
	err = suite.runMigrateCommand("down", "1")
	assert.NoError(suite.T(), err)

	version = suite.getMigrationVersion()
	assert.Equal(suite.T(), "no migration", version)
}

func (suite *MigrationCLITestSuite) TestUsersTableStructureAfterMigration() {
	// Migrate up first
	err := suite.runMigrateCommand("up")
	require.NoError(suite.T(), err)

	// Test table structure
	suite.verifyUsersTableStructure()
	suite.verifyUsersTableIndexes()
	suite.verifyUsersTableConstraints()
}

func (suite *MigrationCLITestSuite) TestMigrationIdempotency() {
	// Run migration up twice - second should be no-op
	err := suite.runMigrateCommand("up")
	assert.NoError(suite.T(), err)

	// Second up should not change anything
	err = suite.runMigrateCommand("up")
	assert.NoError(suite.T(), err) // Should not error, just no change

	// Version should still be 1
	version := suite.getMigrationVersion()
	assert.Equal(suite.T(), "1", version)
}

// Helper methods

func (suite *MigrationCLITestSuite) runMigrateCommand(args ...string) error {
	migrationDir := filepath.Join(suite.projectRoot, "migrations", "postgres")

	cmdArgs := []string{
		"-path", migrationDir,
		"-database", suite.dbURL,
	}
	cmdArgs = append(cmdArgs, args...)

	cmd := exec.Command(suite.migratePath, cmdArgs...)
	cmd.Dir = suite.projectRoot

	output, err := cmd.CombinedOutput()
	if err != nil {
		suite.T().Logf("Migration command failed: %s\nOutput: %s", err, string(output))
	}

	return err
}

func (suite *MigrationCLITestSuite) getMigrationVersion() string {
	migrationDir := filepath.Join(suite.projectRoot, "migrations", "postgres")

	cmd := exec.Command(suite.migratePath,
		"-path", migrationDir,
		"-database", suite.dbURL,
		"version")
	cmd.Dir = suite.projectRoot

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "no migration"
	}

	return strings.TrimSpace(string(output))
}

func (suite *MigrationCLITestSuite) verifyUsersTableExists() {
	var exists bool
	err := suite.db.QueryRow(`
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_schema = 'public'
			AND table_name = 'users'
		)`).Scan(&exists)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), exists, "users table should exist")
}

func (suite *MigrationCLITestSuite) verifyUsersTableNotExists() {
	var exists bool
	err := suite.db.QueryRow(`
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_schema = 'public'
			AND table_name = 'users'
		)`).Scan(&exists)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), exists, "users table should not exist")
}

func (suite *MigrationCLITestSuite) verifySchemaTableExists() {
	var exists bool
	err := suite.db.QueryRow(`
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_schema = 'public'
			AND table_name = 'schema_migrations'
		)`).Scan(&exists)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), exists, "schema_migrations table should exist")
}

func (suite *MigrationCLITestSuite) verifyUsersTableStructure() {
	// Check required columns exist
	expectedColumns := []string{
		"id", "email", "password_hash", "first_name",
		"last_name", "role", "is_active", "created_at", "updated_at",
	}

	for _, columnName := range expectedColumns {
		var exists bool
		err := suite.db.QueryRow(`
			SELECT EXISTS (
				SELECT FROM information_schema.columns
				WHERE table_name = 'users'
				AND column_name = $1
			)`, columnName).Scan(&exists)
		assert.NoError(suite.T(), err)
		assert.True(suite.T(), exists, "Column %s should exist", columnName)
	}
}

func (suite *MigrationCLITestSuite) verifyUsersTableIndexes() {
	expectedIndexes := []string{
		"users_pkey",
		"users_email_key",
		"idx_users_email",
		"idx_users_is_active",
		"idx_users_created_at",
		"idx_users_role",
	}

	for _, indexName := range expectedIndexes {
		var exists bool
		err := suite.db.QueryRow(`
			SELECT EXISTS (
				SELECT FROM pg_indexes
				WHERE tablename = 'users'
				AND indexname = $1
			)`, indexName).Scan(&exists)
		assert.NoError(suite.T(), err)
		assert.True(suite.T(), exists, "Index %s should exist", indexName)
	}
}

func (suite *MigrationCLITestSuite) verifyUsersTableConstraints() {
	// Verify role check constraint
	var constraintCount int
	err := suite.db.QueryRow(`
		SELECT COUNT(*)
		FROM information_schema.check_constraints cc
		JOIN information_schema.constraint_column_usage ccu
			ON cc.constraint_name = ccu.constraint_name
		WHERE ccu.table_name = 'users'
		AND cc.constraint_name = 'users_role_check'`).Scan(&constraintCount)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 1, constraintCount, "users_role_check constraint should exist")

	// Verify email unique constraint
	var uniqueConstraintCount int
	err = suite.db.QueryRow(`
		SELECT COUNT(*)
		FROM information_schema.table_constraints
		WHERE table_name = 'users'
		AND constraint_type = 'UNIQUE'
		AND constraint_name = 'users_email_key'`).Scan(&uniqueConstraintCount)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 1, uniqueConstraintCount, "users email unique constraint should exist")
}

func TestMigrationCLISuite(t *testing.T) {
	// Skip if running in short mode
	if testing.Short() {
		t.Skip("Skipping migration CLI tests in short mode")
	}

	suite.Run(t, new(MigrationCLITestSuite))
}