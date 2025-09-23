# End-to-End Tests

This directory contains comprehensive end-to-end tests for the go-template application.

## Overview

The e2e tests verify the complete application stack from database migrations to API endpoints. They test the entire user lifecycle and ensure the application works correctly from a fresh start.

## Test Structure

### Core Test Files

- `setup.go` - Test environment setup and database management
- `migration_test.go` - Database migration and schema validation tests
- `user_api_test.go` - User registration, authentication, and profile tests
- `user_crud_test.go` - Complete CRUD operations and authorization tests
- `fixtures.go` - Test data fixtures and population utilities
- `full_integration_test.go` - Complete integration scenarios and performance tests

### Infrastructure

- `docker-compose.test.yml` - Test database and services
- `README.md` - This documentation

## Running Tests

### Prerequisites

1. **Docker and Docker Compose** - For test databases
2. **Go 1.21+** - For running tests
3. **PostgreSQL and Redis test containers** - Started automatically

### Quick Start

```bash
# Start test infrastructure
cd tests/e2e
docker-compose -f docker-compose.test.yml up -d

# Run all e2e tests
cd ../..
go test ./tests/e2e/... -v

# Run specific test suites
go test ./tests/e2e/... -v -run TestDatabaseMigrations
go test ./tests/e2e/... -v -run TestUserRegistration
go test ./tests/e2e/... -v -run TestUserCRUDOperations
go test ./tests/e2e/... -v -run TestFullIntegration

# Clean up
cd tests/e2e
docker-compose -f docker-compose.test.yml down
```

### Using Make Commands

```bash
# Run all tests with infrastructure
make test-e2e

# Run only e2e tests (assumes infrastructure is running)
make test-e2e-only

# Start test infrastructure
make test-e2e-up

# Stop test infrastructure
make test-e2e-down

# Run with coverage
make test-e2e-coverage
```

## Test Scenarios

### 1. Database Migration Tests (`migration_test.go`)

- ✅ Verifies database schema creation
- ✅ Validates column types and constraints
- ✅ Checks indexes and triggers
- ✅ Tests constraint enforcement
- ✅ Validates default values and auto-generation

### 2. User API Tests (`user_api_test.go`)

- ✅ User registration with validation
- ✅ Login/logout functionality
- ✅ JWT token validation
- ✅ Profile management
- ✅ Error handling and edge cases

### 3. CRUD Operations (`user_crud_test.go`)

- ✅ Get user by ID
- ✅ Update user information
- ✅ Delete user (soft delete)
- ✅ List users with pagination
- ✅ Search users with filters
- ✅ Authorization and permissions

### 4. Full Integration Tests (`full_integration_test.go`)

- ✅ Complete user lifecycle
- ✅ Multi-user scenarios
- ✅ Performance with large datasets
- ✅ Database consistency
- ✅ Cross-component integration

## Test Features

### Database Management

- **Fresh Database**: Each test gets a clean database
- **Migration Testing**: Validates schema creation from scratch
- **Isolation**: Tests don't interfere with each other
- **Cleanup**: Automatic cleanup after tests

### Test Data

- **Fixtures**: Pre-populated test data for complex scenarios
- **Factories**: Utilities to create custom test data
- **Large Datasets**: Performance testing with bulk data
- **Real UUIDs**: Tests actual UUID handling

### API Testing

- **Complete HTTP Stack**: Tests through actual HTTP handlers
- **Authentication**: JWT token generation and validation
- **Authorization**: Role-based access control
- **Error Scenarios**: Comprehensive error case coverage

## UUID Testing

The tests specifically address the UUID/int issue by:

1. **Schema Validation**: Confirms UUID columns in database
2. **API Response Verification**: Checks UUID format in responses
3. **Cross-System Consistency**: Validates UUIDs work across all layers
4. **Edge Case Testing**: Tests UUID parsing and validation

## Configuration

### Test Database

- **Host**: localhost
- **Port**: 5433 (different from development)
- **Database**: go_template_test
- **User**: test_user
- **Password**: test_password

### Test Redis

- **Host**: localhost
- **Port**: 6380 (different from development)
- **Database**: 0

## Troubleshooting

### Common Issues

1. **Port Conflicts**: Ensure ports 5433 and 6380 are available
2. **Docker Issues**: Check Docker daemon is running
3. **Database Connection**: Wait for healthchecks to pass
4. **Permission Issues**: Check Docker permissions

### Debug Mode

```bash
# Run with verbose logging
go test ./tests/e2e/... -v -args -debug

# Run specific test with details
go test ./tests/e2e/... -v -run TestDatabaseMigrations -count=1

# Check test database
docker exec -it go-template-postgres-test psql -U test_user -d go_template_test
```

### Environment Variables

```bash
# Skip long-running tests
export SHORT_TESTS=true
go test ./tests/e2e/... -short

# Custom test database
export TEST_DB_HOST=custom-host
export TEST_DB_PORT=5432
```

## Best Practices

### Writing New Tests

1. **Use Test Helpers**: Leverage existing setup functions
2. **Clean State**: Each test should start with clean data
3. **Real Scenarios**: Test actual user workflows
4. **Error Cases**: Include negative test cases
5. **Documentation**: Document complex test scenarios

### Performance Considerations

1. **Parallel Tests**: Use t.Parallel() where appropriate
2. **Fixture Reuse**: Reuse test data when possible
3. **Cleanup**: Always cleanup test data
4. **Resource Limits**: Be mindful of test resource usage

## Continuous Integration

The e2e tests are designed to run in CI environments:

- **Docker Support**: Self-contained test infrastructure
- **Fast Startup**: Optimized for quick feedback
- **Reliable**: Handles timing and environment issues
- **Comprehensive**: Covers all critical paths

## Contributing

When adding new features:

1. **Add E2E Tests**: Include e2e tests for new endpoints
2. **Update Fixtures**: Add new test data as needed
3. **Document Changes**: Update this README
4. **Test Locally**: Verify tests pass locally before committing