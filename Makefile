# Go Template Makefile
# This file helps with building, testing, and code generation

# Variables
BINARY_NAME=go-template
BINARY_UNIX=$(BINARY_NAME)_unix
BINARY_WINDOWS=$(BINARY_NAME)_windows.exe
MAIN_PATH=./cmd/main.go

# Protobuf and gRPC variables
PROTO_DIR=api/proto
GENERATED_DIR=internal/grpc/gen
THIRD_PARTY=third_party

# Colors for output
RED=\033[0;31m
GREEN=\033[0;32m
YELLOW=\033[1;33m
BLUE=\033[0;34m
NC=\033[0m # No Color

.PHONY: help build build-linux build-windows run test clean proto swagger install-tools deps tidy fmt lint docker-build docker-run

# Default target
all: build

# Help
help: ## Show this help message
	@echo '$(BLUE)Go Template Makefile$(NC)'
	@echo '$(BLUE)===================$(NC)'
	@awk 'BEGIN {FS = ":.*##"; printf "\n$(YELLOW)Available targets:$(NC)\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  $(GREEN)%-15s$(NC) %s\n", $$1, $$2 } /^##@/ { printf "\n$(YELLOW)%s$(NC)\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Build
build: ## Build the application
	@echo "$(BLUE)Building application...$(NC)"
	go build -o bin/$(BINARY_NAME) $(MAIN_PATH)
	@echo "$(GREEN)✅ Build completed: bin/$(BINARY_NAME)$(NC)"

build-linux: ## Build for Linux
	@echo "$(BLUE)Building for Linux...$(NC)"
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/$(BINARY_UNIX) $(MAIN_PATH)
	@echo "$(GREEN)✅ Linux build completed: bin/$(BINARY_UNIX)$(NC)"

build-windows: ## Build for Windows
	@echo "$(BLUE)Building for Windows...$(NC)"
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o bin/$(BINARY_WINDOWS) $(MAIN_PATH)
	@echo "$(GREEN)✅ Windows build completed: bin/$(BINARY_WINDOWS)$(NC)"

##@ Development
run: ## Run the application
	@echo "$(BLUE)Running application...$(NC)"
	go run $(MAIN_PATH)

run-dev: ## Run with development environment
	@echo "$(BLUE)Running in development mode...$(NC)"
	SERVER_MODE=development go run $(MAIN_PATH)

test: ## Run tests
	@echo "$(BLUE)Running tests...$(NC)"
	go test -v ./...

test-coverage: ## Run tests with coverage
	@echo "$(BLUE)Running tests with coverage...$(NC)"
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)✅ Coverage report generated: coverage.html$(NC)"

benchmark: ## Run benchmarks
	@echo "$(BLUE)Running benchmarks...$(NC)"
	go test -bench=. -benchmem ./...

##@ Code Quality
fmt: ## Format code
	@echo "$(BLUE)Formatting code...$(NC)"
	go fmt ./...
	@echo "$(GREEN)✅ Code formatted$(NC)"

lint: ## Run linter
	@echo "$(BLUE)Running linter...$(NC)"
	golangci-lint run
	@echo "$(GREEN)✅ Linting completed$(NC)"

tidy: ## Tidy dependencies
	@echo "$(BLUE)Tidying dependencies...$(NC)"
	go mod tidy
	@echo "$(GREEN)✅ Dependencies tidied$(NC)"

deps: ## Download dependencies
	@echo "$(BLUE)Downloading dependencies...$(NC)"
	go mod download
	@echo "$(GREEN)✅ Dependencies downloaded$(NC)"

##@ Code Generation
proto: ## Generate gRPC code from protobuf files
	@echo "$(BLUE)Generating gRPC code...$(NC)"
	@if [ ! -d "$(GENERATED_DIR)" ]; then mkdir -p $(GENERATED_DIR); fi
	@find $(PROTO_DIR) -name "*.proto" -exec echo "Processing: {}" \;

	# Generate Go code
	protoc --proto_path=$(PROTO_DIR) \
		--proto_path=$(THIRD_PARTY) \
		--go_out=$(GENERATED_DIR) \
		--go_opt=paths=source_relative \
		--go-grpc_out=$(GENERATED_DIR) \
		--go-grpc_opt=paths=source_relative \
		$(PROTO_DIR)/user/v1/*.proto

	# Generate gRPC Gateway code
	protoc --proto_path=$(PROTO_DIR) \
		--proto_path=$(THIRD_PARTY) \
		--grpc-gateway_out=$(GENERATED_DIR) \
		--grpc-gateway_opt=paths=source_relative \
		$(PROTO_DIR)/user/v1/*.proto

	@echo "$(GREEN)✅ gRPC code generation completed$(NC)"

swagger: ## Generate Swagger documentation
	@echo "$(BLUE)Generating Swagger documentation...$(NC)"
	@if [ ! -d "docs/swagger" ]; then mkdir -p docs/swagger; fi

	# Generate OpenAPI spec from protobuf
	protoc --proto_path=$(PROTO_DIR) \
		--proto_path=$(THIRD_PARTY) \
		--openapiv2_out=docs/swagger \
		--openapiv2_opt=logtostderr=true \
		--openapiv2_opt=json_names_for_fields=false \
		$(PROTO_DIR)/user/v1/*.proto

	# Generate Swagger from Go annotations (for REST endpoints)
	swag init -g $(MAIN_PATH) -o docs/swagger --parseDependency

	@echo "$(GREEN)✅ Swagger documentation generated in docs/swagger/$(NC)"

install-tools: ## Install required development tools
	@echo "$(BLUE)Installing development tools...$(NC)"

	# Install protobuf compiler
	@echo "Installing protobuf compiler..."
	@if ! command -v protoc &> /dev/null; then \
		echo "$(YELLOW)Please install protobuf compiler manually:$(NC)"; \
		echo "  - macOS: brew install protobuf"; \
		echo "  - Ubuntu: apt-get install protobuf-compiler"; \
		echo "  - Or download from: https://github.com/protocolbuffers/protobuf/releases"; \
	else \
		echo "$(GREEN)✅ protoc already installed$(NC)"; \
	fi

	# Install Go protobuf plugins
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest

	# Install Swagger
	go install github.com/swaggo/swag/cmd/swag@latest

	# Install linter
	@if ! command -v golangci-lint &> /dev/null; then \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.54.2; \
	else \
		echo "$(GREEN)✅ golangci-lint already installed$(NC)"; \
	fi

	# Install Air for hot reload
	go install github.com/cosmtrek/air@latest

	@echo "$(GREEN)✅ All tools installed$(NC)"

setup-proto-deps: ## Setup third-party protobuf dependencies
	@echo "$(BLUE)Setting up protobuf dependencies...$(NC)"
	@if [ ! -d "$(THIRD_PARTY)" ]; then mkdir -p $(THIRD_PARTY); fi

	# Download googleapis
	@if [ ! -d "$(THIRD_PARTY)/google" ]; then \
		echo "Downloading googleapis..."; \
		curl -L https://github.com/googleapis/googleapis/archive/master.zip -o /tmp/googleapis.zip; \
		unzip -q /tmp/googleapis.zip -d /tmp/; \
		cp -r /tmp/googleapis-master/google $(THIRD_PARTY)/; \
		rm -rf /tmp/googleapis*; \
	fi

	# Download protoc-gen-openapiv2 options
	@if [ ! -d "$(THIRD_PARTY)/protoc-gen-openapiv2" ]; then \
		echo "Downloading protoc-gen-openapiv2 options..."; \
		curl -L https://github.com/grpc-ecosystem/grpc-gateway/archive/v2.18.0.zip -o /tmp/grpc-gateway.zip; \
		unzip -q /tmp/grpc-gateway.zip -d /tmp/; \
		cp -r /tmp/grpc-gateway-2.18.0/protoc-gen-openapiv2 $(THIRD_PARTY)/; \
		rm -rf /tmp/grpc-gateway*; \
	fi

	@echo "$(GREEN)✅ Protobuf dependencies setup completed$(NC)"

##@ Docker
docker-build: ## Build Docker image
	@echo "$(BLUE)Building Docker image...$(NC)"
	docker build -t $(BINARY_NAME):latest .
	@echo "$(GREEN)✅ Docker image built: $(BINARY_NAME):latest$(NC)"

docker-run: ## Run Docker container
	@echo "$(BLUE)Running Docker container...$(NC)"
	docker run -p 8080:8080 -p 9090:9090 --env-file .env $(BINARY_NAME):latest

##@ Database
migrate-up: ## Run database migrations
	@echo "$(BLUE)Running database migrations...$(NC)"
	go run cmd/migrate/main.go up
	@echo "$(GREEN)✅ Database migrations completed$(NC)"

migrate-down: ## Rollback database migrations
	@echo "$(BLUE)Rolling back database migrations...$(NC)"
	go run cmd/migrate/main.go down
	@echo "$(GREEN)✅ Database rollback completed$(NC)"

migrate-create: ## Create new migration (usage: make migrate-create NAME=migration_name)
	@echo "$(BLUE)Creating new migration: $(NAME)$(NC)"
	go run cmd/migrate/main.go create $(NAME)
	@echo "$(GREEN)✅ Migration created$(NC)"

##@ Storage
storage-start: ## Start storage services (MinIO)
	@echo "$(BLUE)Starting storage services...$(NC)"
	docker-compose -f docker-compose.storage.yml up -d
	@echo "$(GREEN)✅ Storage services started$(NC)"

storage-stop: ## Stop storage services
	@echo "$(BLUE)Stopping storage services...$(NC)"
	docker-compose -f docker-compose.storage.yml down
	@echo "$(GREEN)✅ Storage services stopped$(NC)"

##@ Monitoring
monitoring-start: ## Start monitoring stack
	@echo "$(BLUE)Starting monitoring stack...$(NC)"
	docker-compose -f docker-compose.monitoring.yml up -d
	@echo "$(GREEN)✅ Monitoring stack started$(NC)"

monitoring-stop: ## Stop monitoring stack
	@echo "$(BLUE)Stopping monitoring stack...$(NC)"
	docker-compose -f docker-compose.monitoring.yml down
	@echo "$(GREEN)✅ Monitoring stack stopped$(NC)"

##@ Cleanup
clean: ## Clean build artifacts
	@echo "$(BLUE)Cleaning build artifacts...$(NC)"
	go clean
	rm -rf bin/
	rm -f coverage.out coverage.html
	@echo "$(GREEN)✅ Clean completed$(NC)"

clean-proto: ## Clean generated protobuf code
	@echo "$(BLUE)Cleaning generated protobuf code...$(NC)"
	rm -rf $(GENERATED_DIR)
	@echo "$(GREEN)✅ Generated code cleaned$(NC)"

clean-all: clean clean-proto ## Clean everything
	@echo "$(GREEN)✅ Everything cleaned$(NC)"

##@ Setup
init: install-tools setup-proto-deps proto swagger ## Initialize project (install tools and generate code)
	@echo "$(GREEN)✅ Project initialization completed$(NC)"
	@echo ""
	@echo "$(YELLOW)Next steps:$(NC)"
	@echo "1. Copy .env.example to .env and configure your environment"
	@echo "2. Run 'make storage-start' to start MinIO"
	@echo "3. Run 'make monitoring-start' to start monitoring stack"
	@echo "4. Run 'make run' to start the application"

setup: deps proto swagger ## Setup project for development
	@echo "$(GREEN)✅ Development setup completed$(NC)"

# Hot reload for development
dev: ## Run with hot reload
	air