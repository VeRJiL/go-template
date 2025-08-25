# Migration Tests

This directory contains comprehensive tests for the database migration system.

## Test Coverage

### Migration CLI Tests (`migration_cli_test.go`)

Tests the migration functionality using the CLI binary (`./bin/migrate`):

1. **Migration Up**: Tests applying migrations and verifying database state
2. **Migration Down**: Tests rolling back migrations
3. **Version Tracking**: Tests migration version management
4. **Force Command**: Tests cleaning dirty migration states
5. **Step Commands**: Tests applying specific number of migrations
6. **Idempotency**: Tests that running migrations multiple times is safe
7. **Database Structure**: Tests that tables, indexes, and constraints are created correctly

### Test Database

Tests use a separate test database (`go_template_test_migration_cli`) to avoid interfering with development data. The test database is:
- Created automatically before tests
- Cleaned up after tests complete
- Reset between individual test cases

## Running Tests

### Run All Migration Tests
```bash
cd tests/migration
go test -v
```

### Run Tests from Project Root
```bash
make test-migration
```

### Run Tests in Short Mode (skip integration tests)
```bash
go test -short -v
```

## Test Requirements

- PostgreSQL server running
- Database credentials in .env file
- `./bin/migrate` binary available
- Test dependencies installed (`go mod download`)

## Coverage Areas

✅ **Migration Up/Down**: Verifies migrations apply and rollback correctly
✅ **Version Tracking**: Ensures proper migration state management
✅ **Database Schema**: Validates table structure, indexes, constraints
✅ **Error Handling**: Tests dirty states and force recovery
✅ **Idempotency**: Confirms safe re-running of migrations
✅ **CLI Integration**: Tests actual migrate binary used in production

## Adding New Tests

When adding new migrations:
1. Update expected table/index lists in test helper methods
2. Add specific tests for new migration features
3. Ensure test database cleanup handles new objects