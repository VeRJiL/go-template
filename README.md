# Go REST API Template

A clean, production-ready Go REST API template following Clean Architecture principles. Perfect for quickly bootstrapping microservices and REST APIs.

## 🚀 Features

- ✅ **Clean Architecture** - Proper separation of concerns with clear layer boundaries
- ✅ **JWT Authentication** - Secure token-based authentication
- ✅ **User Management** - Complete CRUD operations with proper authorization
- ✅ **Database Layer** - PostgreSQL with repository pattern
- ✅ **Middleware** - Security headers, CORS, authentication, logging
- ✅ **Configuration** - Environment-based configuration management
- ✅ **Structured Logging** - JSON and text logging with different levels
- ✅ **Graceful Shutdown** - Proper resource cleanup on shutdown
- ✅ **Health Check** - Health monitoring endpoint
- ✅ **Password Security** - bcrypt password hashing
- ✅ **Input Validation** - Request validation and error handling
- ✅ **API Documentation** - Swagger/OpenAPI documentation with interactive UI
- ✅ **Docker Support** - Ready for containerization
- ✅ **Modular Design** - Easy to extend and maintain

## 📁 Project Structure

```
cmd/                       # Application entry points
├── main.go               # Main application entry point

internal/
├── app/                  # Application layer (dependency injection)
│   └── app.go
├── api/                  # Presentation layer
│   ├── handlers/         # HTTP handlers
│   ├── middleware/       # HTTP middleware
│   └── routes/           # Route definitions
├── domain/               # Business logic layer
│   ├── entities/         # Business entities and DTOs
│   ├── repositories/     # Repository interfaces
│   └── services/         # Business logic services
├── database/             # Data layer
│   ├── postgres/         # PostgreSQL implementation
│   └── redis/            # Redis implementation (optional)
├── config/               # Configuration management
└── pkg/                  # Internal packages
    ├── auth/             # JWT authentication
    ├── logger/           # Structured logging
    ├── storage/          # File storage (optional)
    ├── messagebroker/    # Message broker (optional)
    └── monitoring/       # Prometheus metrics (optional)

migrations/               # Database migrations
```

## 🛠️ Quick Start

### Prerequisites
- Go 1.21 or higher
- PostgreSQL database
- Redis server (for caching and optional features)
- Make (optional, for convenience)

### Setup

1. **Clone and configure**:
   ```bash
   git clone https://github.com/VeRJiL/go-template.git
   cd go-template
   cp .env.example .env
   # Edit .env with your database and Redis configuration
   ```

2. **Install Redis** (if not already installed):
   ```bash
   # Ubuntu/Debian
   sudo apt update && sudo apt install redis-server

   # macOS (using Homebrew)
   brew install redis

   # Start Redis server
   redis-server
   # Or run in background: redis-server --daemonize yes
   ```

3. **Install dependencies**:
   ```bash
   go mod download
   ```

4. **Run database migrations**:
   ```bash
   make migrate-up
   ```

5. **Build and run**:
   ```bash
   make build
   make run
   # Or directly: go run cmd/main.go
   ```

### Available Commands

```bash
make build          # Build the application
make run            # Run the application
make test           # Run tests
make test-coverage  # Run tests with coverage
make lint           # Run linter (requires golangci-lint)
make fmt            # Format code
make migrate-up     # Run database migrations up
make migrate-down   # Run database migrations down
make docker-run     # Run with Docker Compose
```

## 📋 API Endpoints

### Authentication
- `POST /api/v1/auth/register` - Register new user
- `POST /api/v1/auth/login` - Login user
- `POST /api/v1/auth/logout` - Logout user (requires auth)
- `GET /api/v1/auth/me` - Get current user profile (requires auth)

### User Management
- `GET /api/v1/users` - List all users (requires auth)
- `GET /api/v1/users/:id` - Get user by ID (requires auth)
- `PUT /api/v1/users/:id` - Update user (requires auth)
- `DELETE /api/v1/users/:id` - Delete user (requires auth)

### Health
- `GET /health` - Health check

### API Documentation
- `GET /swagger/index.html` - Interactive Swagger UI for API documentation

> 📖 **Swagger Documentation**: Once the server is running, visit `http://localhost:8080/swagger/index.html` to explore the complete API documentation with interactive testing capabilities.

## 🔧 Configuration

The application uses environment variables for configuration. Key variables:

```bash
# Server
SERVER_HOST=localhost
SERVER_PORT=8080
SERVER_MODE=development

# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=your_user
DB_PASSWORD=your_password
DB_DATABASE=your_database

# Redis (optional, for caching and message broker)
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_DB=0

# Authentication
JWT_SECRET=your-secret-key
JWT_EXPIRATION=24h

# Logging
LOG_LEVEL=info
LOG_FORMAT=text
```

## 🧪 Example Usage

### Register a user
```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "john@example.com",
    "password": "securepassword123",
    "first_name": "John",
    "last_name": "Doe",
    "role": "user"
  }'
```

### Login
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "john@example.com",
    "password": "securepassword123"
  }'
```

### Get users (with token)
```bash
curl -X GET http://localhost:8080/api/v1/users \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

## 🐳 Docker Support

### Using Docker Compose (Recommended for Development)

```bash
# Start all services (app + database)
make docker-run

# Or manually:
docker-compose up -d

# View logs
docker-compose logs -f app

# Stop services
docker-compose down
```

### Building Docker Image

```dockerfile
# Dockerfile is included in the template
docker build -t go-template .
docker run -p 8080:8080 go-template
```

## 🏗️ Architecture Principles

This template follows **Clean Architecture** principles:

1. **Independence**: Business logic doesn't depend on external concerns
2. **Testability**: Easy to test with mocked dependencies
3. **Flexibility**: Easy to swap implementations (database, external services)
4. **Separation of Concerns**: Each layer has a single responsibility

### Layer Responsibilities

- **Entities**: Core business objects and data structures
- **Use Cases/Services**: Business logic and application rules
- **Interface Adapters**: Handlers, repositories, external service adapters
- **Frameworks & Drivers**: Web framework, database, external APIs

## 🚀 Adding New Features

To add a new feature (e.g., Products):

1. **Create entity**: `internal/domain/entities/product.go`
2. **Create repository interface**: `internal/domain/repositories/product_repository.go`
3. **Implement repository**: `internal/database/postgres/product_repository.go`
4. **Create service**: `internal/domain/services/product_service.go`
5. **Create handler**: `internal/api/handlers/product_handler.go`
6. **Add routes**: Update `internal/api/routes/routes.go`
7. **Update app setup**: Add to `internal/app/app.go`

## 🤝 Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Clean Architecture by Robert C. Martin
- Go community best practices
- Gin HTTP web framework
- PostgreSQL database