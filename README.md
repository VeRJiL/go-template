# Go REST API Template

A production-ready Go REST API template built with Clean Architecture principles, featuring comprehensive testing, multiple database support, and enterprise-grade features.

![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)
![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Test Coverage](https://img.shields.io/badge/coverage-90%25+-green.svg)
![Build Status](https://img.shields.io/badge/build-passing-brightgreen.svg)

## 🚀 Features

### Core Architecture
- **Clean Architecture** with proper dependency injection
- **SOLID principles** implementation
- **Domain-driven design** structure
- **Repository pattern** for data access
- **Service layer** for business logic

### Database Support
- **PostgreSQL** - Primary database with advanced features
- **Redis** - Caching with automatic invalidation
- **MongoDB** - Document database support
- **Elasticsearch** - Full-text search capabilities

### Authentication & Security
- **JWT authentication** with configurable expiration
- **Rate limiting** middleware
- **CORS** configuration
- **Security headers** middleware
- **Input validation** with custom validators
- **Password hashing** with bcrypt

### Storage & File Handling
- **Pluggable storage drivers**: Local, AWS S3, Google Cloud Storage, Azure Blob
- **File upload handling** with size and type validation
- **Image processing** capabilities
- **URL generation** for stored files

### Message Broker Integration
- **Laravel-style manager pattern**
- **Multiple drivers**: RabbitMQ, Kafka, Redis Pub/Sub
- **Event system** for user lifecycle events
- **Job queues** with priority and retry logic
- **Background processing** for emails, notifications, reports

### Monitoring & Observability
- **Prometheus metrics** with custom collectors
- **Gin middleware** for HTTP metrics
- **ELK Stack integration** (Elasticsearch + Kibana)
- **Health check endpoints**
- **Request/response logging**
- **Error tracking** with structured logging

### Development Features
- **Hot reload** with Air
- **Docker & Docker Compose** ready
- **Database migrations** with golang-migrate
- **API documentation** with Swagger
- **Comprehensive testing** (90%+ coverage)
- **Graceful shutdown** handling

## 📋 Quick Start

### Prerequisites
- Go 1.21 or higher
- PostgreSQL 13+
- Redis 6+
- Docker & Docker Compose (optional)

### Installation

1. **Clone the repository**
   ```bash
   git clone https://github.com/VeRJiL/go-template.git
   cd go-template
   ```

2. **Set up environment variables**
   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

3. **Install dependencies**
   ```bash
   go mod download
   ```

4. **Run database migrations**
   ```bash
   make migrate-up
   ```

5. **Start the application**
   ```bash
   make run
   ```

The API will be available at `http://localhost:8080`

### Using Docker

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f app
```

## 🛠️ Development

### Available Commands

```bash
# Build the application
make build

# Run with hot reload
make dev

# Run tests
make test

# Run tests with coverage
make test-coverage

# Run linter
make lint

# Format code
make fmt

# Database migrations
make migrate-up
make migrate-down

# Docker operations
make docker-build
make docker-run
```

### Project Structure

```
├── cmd/
│   └── main.go                 # Application entry point
├── internal/
│   ├── api/                    # HTTP layer
│   │   ├── handlers/           # Request handlers
│   │   ├── middleware/         # HTTP middleware
│   │   ├── routes/             # Route definitions
│   │   └── validators/         # Input validation
│   ├── config/                 # Configuration management
│   ├── database/               # Database implementations
│   │   ├── postgres/           # PostgreSQL driver
│   │   └── redis/              # Redis driver
│   ├── domain/                 # Business logic layer
│   │   ├── entities/           # Domain entities
│   │   ├── repositories/       # Repository interfaces
│   │   └── services/           # Business services
│   └── pkg/                    # Internal packages
│       ├── auth/               # Authentication
│       ├── logger/             # Logging utilities
│       ├── messagebroker/      # Message broker
│       ├── monitoring/         # Metrics & monitoring
│       └── storage/            # File storage
├── migrations/                 # Database migrations
├── docs/                       # API documentation
└── tests/                      # Test files
```

## 🔧 Configuration

The application uses environment variables for configuration. Key settings include:

### Database
```env
DB_HOST=localhost
DB_PORT=5432
DB_USER=your_user
DB_PASSWORD=your_password
DB_NAME=your_database
```

### Redis Cache
```env
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_DB=0
CACHE_USER_TTL=3600
```

### JWT Authentication
```env
JWT_SECRET=your-super-secret-key
JWT_EXPIRATION=24h
```

### Storage
```env
STORAGE_PROVIDER=local
STORAGE_LOCAL_PATH=./uploads
STORAGE_LOCAL_URL_PREFIX=/files
```

## 📊 Monitoring & Logging

### Prometheus Metrics
Access metrics at `/metrics` endpoint:
- HTTP request duration and count
- Database query metrics
- Cache hit/miss rates
- Custom business metrics

### ELK Stack Integration
```bash
# Start ELK stack
docker-compose -f docker-compose.elk.yml up -d

# Access Kibana
open http://localhost:5601
```

### Health Checks
- **Basic health**: `GET /health`
- **Detailed health**: `GET /health/detailed`

## 🧪 Testing

### Run Tests
```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run specific package tests
go test ./internal/domain/services/...
```

### Test Coverage
The project maintains high test coverage:
- **Overall**: 90%+
- **Business Logic**: 95%+
- **Handlers**: 85%+
- **Repositories**: 90%+

## 🚀 Deployment

### Production Build
```bash
# Build optimized binary
make build

# Build Docker image
make docker-build

# Deploy with Docker Compose
docker-compose -f docker-compose.prod.yml up -d
```

### Environment Setup
1. Configure production environment variables
2. Set up external databases (PostgreSQL, Redis)
3. Configure monitoring and logging
4. Set up SSL/TLS certificates
5. Configure reverse proxy (Nginx/Traefik)

## 📚 API Documentation

API documentation is available via Swagger:
- **Development**: `http://localhost:8080/swagger/index.html`
- **Generate docs**: `make swagger-gen`

### Example Endpoints

```bash
# User registration
POST /api/v1/auth/register
{
  "email": "user@example.com",
  "password": "securepassword",
  "first_name": "John",
  "last_name": "Doe"
}

# User login
POST /api/v1/auth/login
{
  "email": "user@example.com",
  "password": "securepassword"
}

# Get users (authenticated)
GET /api/v1/users?page=1&limit=10
Authorization: Bearer <jwt_token>
```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Add tests for new functionality
5. Ensure all tests pass (`make test`)
6. Run linter (`make lint`)
7. Commit your changes (`git commit -m 'Add amazing feature'`)
8. Push to the branch (`git push origin feature/amazing-feature`)
9. Open a Pull Request

### Development Guidelines
- Follow Go best practices and idioms
- Maintain test coverage above 90%
- Add documentation for new features
- Use conventional commit messages
- Ensure code passes all linting checks

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [Gin Web Framework](https://github.com/gin-gonic/gin)
- [golang-migrate](https://github.com/golang-migrate/migrate)
- [testify](https://github.com/stretchr/testify) for testing utilities
- [Prometheus](https://prometheus.io/) for monitoring
- [Clean Architecture](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html) principles

## 📞 Support

If you have any questions or need help with setup, please:
1. Check the [documentation](docs/)
2. Search [existing issues](https://github.com/VeRJiL/go-template/issues)
3. Create a [new issue](https://github.com/VeRJiL/go-template/issues/new)

---

⭐ **Star this repository if you find it helpful!**