#!/bin/bash

# End-to-End Test Runner Script
# This script sets up the test environment and runs comprehensive e2e tests

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
TEST_COMPOSE_FILE="tests/e2e/docker-compose.test.yml"
PROJECT_NAME="go-template-e2e"

# Functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

cleanup() {
    log_info "Cleaning up test environment..."
    docker-compose -f "$TEST_COMPOSE_FILE" -p "$PROJECT_NAME" down -v --remove-orphans 2>/dev/null || true
}

wait_for_services() {
    log_info "Waiting for test services to be ready..."

    # Wait for PostgreSQL
    local max_attempts=30
    local attempt=1

    while [ $attempt -le $max_attempts ]; do
        if docker-compose -f "$TEST_COMPOSE_FILE" -p "$PROJECT_NAME" exec -T postgres-test pg_isready -U test_user -d go_template_test 2>/dev/null; then
            log_success "PostgreSQL is ready"
            break
        fi

        if [ $attempt -eq $max_attempts ]; then
            log_error "PostgreSQL failed to start after $max_attempts attempts"
            return 1
        fi

        log_info "Waiting for PostgreSQL... (attempt $attempt/$max_attempts)"
        sleep 2
        ((attempt++))
    done

    # Wait for Redis
    attempt=1
    while [ $attempt -le $max_attempts ]; do
        if docker-compose -f "$TEST_COMPOSE_FILE" -p "$PROJECT_NAME" exec -T redis-test redis-cli ping 2>/dev/null | grep -q PONG; then
            log_success "Redis is ready"
            break
        fi

        if [ $attempt -eq $max_attempts ]; then
            log_error "Redis failed to start after $max_attempts attempts"
            return 1
        fi

        log_info "Waiting for Redis... (attempt $attempt/$max_attempts)"
        sleep 2
        ((attempt++))
    done

    # Additional wait to ensure services are fully initialized
    sleep 5
    log_success "All test services are ready"
}

setup_test_environment() {
    log_info "Setting up test environment..."

    # Stop any existing test containers
    cleanup

    # Start test services
    log_info "Starting test services..."
    docker-compose -f "$TEST_COMPOSE_FILE" -p "$PROJECT_NAME" up -d

    # Wait for services to be ready
    wait_for_services

    log_success "Test environment setup complete"
}

run_tests() {
    local test_pattern="$1"
    local coverage="$2"
    local verbose="$3"

    log_info "Running end-to-end tests..."

    # Build test command
    local cmd="go test ./tests/e2e/..."

    if [ "$verbose" = "true" ]; then
        cmd="$cmd -v"
    fi

    if [ -n "$test_pattern" ]; then
        cmd="$cmd -run $test_pattern"
    fi

    if [ "$coverage" = "true" ]; then
        cmd="$cmd -coverprofile=coverage-e2e.out -covermode=atomic"
    fi

    # Add timeout for long-running tests
    cmd="$cmd -timeout=10m"

    log_info "Executing: $cmd"

    # Run tests
    if eval "$cmd"; then
        log_success "All tests passed!"

        if [ "$coverage" = "true" ]; then
            log_info "Generating coverage report..."
            go tool cover -html=coverage-e2e.out -o coverage-e2e.html
            log_success "Coverage report generated: coverage-e2e.html"
        fi

        return 0
    else
        log_error "Some tests failed!"
        return 1
    fi
}

print_usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -h, --help       Show this help message"
    echo "  -s, --setup      Only setup test environment (don't run tests)"
    echo "  -c, --cleanup    Only cleanup test environment"
    echo "  -t, --test       Run specific test pattern (e.g., TestUserAPI)"
    echo "  -v, --verbose    Run tests with verbose output"
    echo "  --coverage       Generate test coverage report"
    echo "  --no-setup       Skip environment setup (assumes it's already running)"
    echo ""
    echo "Examples:"
    echo "  $0                           # Run all e2e tests"
    echo "  $0 -v --coverage            # Run with verbose output and coverage"
    echo "  $0 -t TestUserAPI           # Run only user API tests"
    echo "  $0 -s                       # Only setup test environment"
    echo "  $0 -c                       # Only cleanup test environment"
    echo "  $0 --no-setup -v            # Run tests without setup"
}

# Main script logic
main() {
    local setup_only=false
    local cleanup_only=false
    local no_setup=false
    local test_pattern=""
    local coverage=false
    local verbose=false

    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                print_usage
                exit 0
                ;;
            -s|--setup)
                setup_only=true
                shift
                ;;
            -c|--cleanup)
                cleanup_only=true
                shift
                ;;
            -t|--test)
                test_pattern="$2"
                shift 2
                ;;
            -v|--verbose)
                verbose=true
                shift
                ;;
            --coverage)
                coverage=true
                shift
                ;;
            --no-setup)
                no_setup=true
                shift
                ;;
            *)
                log_error "Unknown option: $1"
                print_usage
                exit 1
                ;;
        esac
    done

    # Cleanup only
    if [ "$cleanup_only" = "true" ]; then
        cleanup
        log_success "Test environment cleaned up"
        exit 0
    fi

    # Setup environment if not skipped
    if [ "$no_setup" = "false" ]; then
        setup_test_environment
    fi

    # Setup only
    if [ "$setup_only" = "true" ]; then
        log_success "Test environment is ready"
        log_info "To run tests manually: go test ./tests/e2e/... -v"
        log_info "To cleanup: $0 -c"
        exit 0
    fi

    # Run tests
    local exit_code=0
    if ! run_tests "$test_pattern" "$coverage" "$verbose"; then
        exit_code=1
    fi

    # Cleanup unless no-setup was specified (implies user manages environment)
    if [ "$no_setup" = "false" ]; then
        cleanup
    fi

    exit $exit_code
}

# Set trap for cleanup on script exit
trap cleanup EXIT

# Check if we're in the right directory
if [ ! -f "go.mod" ]; then
    log_error "This script must be run from the project root directory"
    exit 1
fi

# Check if Docker is available
if ! command -v docker &> /dev/null; then
    log_error "Docker is required but not installed"
    exit 1
fi

if ! command -v docker-compose &> /dev/null; then
    log_error "Docker Compose is required but not installed"
    exit 1
fi

# Run main function
main "$@"