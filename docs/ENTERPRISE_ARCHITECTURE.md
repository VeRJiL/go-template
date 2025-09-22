# Enterprise Plugin Architecture

This document describes the enterprise-grade plugin architecture implemented in this Go template project. This architecture solves the scalability issues of manually adding dependencies for new entities and provides a robust, modular system for building complex applications.

## Table of Contents

1. [Overview](#overview)
2. [Architecture Components](#architecture-components)
3. [Key Features](#key-features)
4. [Getting Started](#getting-started)
5. [Adding New Entities](#adding-new-entities)
6. [Creating Custom Modules](#creating-custom-modules)
7. [Code Generation](#code-generation)
8. [Dependency Injection](#dependency-injection)
9. [Module Registry](#module-registry)
10. [Best Practices](#best-practices)
11. [Examples](#examples)

## Overview

The enterprise plugin architecture is designed to eliminate the need for manual dependency injection and route registration when adding new entities. Instead of modifying `app.go` and `routes.go` for every new entity, the system provides:

- **Auto-discovery**: Modules register themselves automatically
- **Code Generation**: Complete CRUD operations generated from entity definitions
- **Plugin System**: Modular architecture with clear separation of concerns
- **Dependency Injection**: Automatic dependency resolution with reflection
- **Generic Patterns**: Reusable repository, service, and handler patterns

## Architecture Components

### 1. Core Components

```
internal/pkg/
├── container/          # Dependency injection container
├── modules/           # Module interfaces and types
├── registry/          # Module and entity registries
├── crud/             # Generic CRUD patterns
├── generator/        # Code generation tools
└── bootstrap/        # Enterprise bootstrap system
```

### 2. Module Structure

```
internal/modules/
├── user_module.go     # User module implementation
├── product_module.go  # Product module (example)
└── order_module.go    # Order module (example)
```

### 3. Generated Code Structure

```
internal/
├── domain/
│   ├── entities/      # Entity definitions
│   └── services/      # Business logic services
├── database/
│   └── repositories/  # Data access layer
└── api/
    └── handlers/      # HTTP handlers
```

## Key Features

### 1. Zero Manual Configuration

```go
// Before: Manual dependency injection in app.go
userRepo := postgres.NewUserRepository(db)
userService := services.NewUserService(userRepo, logger)
userHandler := handlers.NewUserHandler(userService, logger)

// After: Automatic registration
registry.Register(modules.NewUserModule())
```

### 2. Auto-Generated CRUD

```go
// Entity definition triggers automatic generation
type Product struct {
    ID          uint   `json:"id" db:"id"`
    Name        string `json:"name" db:"name" validate:"required"`
    Description string `json:"description" db:"description"`
    Price       int64  `json:"price" db:"price" validate:"required,min=0"`
}

// Generates: Repository, Service, Handler, Routes, Tests
```

### 3. Plugin-Style Modules

```go
type ProductModule struct {
    name         string
    version      string
    dependencies []string
}

func (m *ProductModule) RegisterServices(container *container.Container) error {
    // Auto-register all services
}

func (m *ProductModule) RegisterRoutes(router *gin.RouterGroup, deps *modules.Dependencies) error {
    // Auto-register all routes
}
```

## Getting Started

### 1. Run the Enterprise Application

```bash
# Build the enterprise application
go build -o bin/enterprise cmd/enterprise/main.go

# Run the application
./bin/enterprise

# Or use with hot reload
air -c .air.enterprise.toml
```

### 2. Check Application Status

```bash
# Health check
curl http://localhost:8080/health

# Module information
curl http://localhost:8080/admin/modules

# Service registry
curl http://localhost:8080/admin/services
```

## Adding New Entities

### Method 1: Using Code Generator (Recommended)

```bash
# Generate complete module for Product entity
go run cmd/generator/main.go -entity=Product -all

# Generate with custom options
go run cmd/generator/main.go \
  -entity=Product \
  -table=products \
  -soft-delete \
  -all

# Generate specific components only
go run cmd/generator/main.go \
  -entity=Product \
  -gen-entity \
  -gen-repo \
  -gen-service
```

### Method 2: Manual Implementation

1. **Define Entity**:

```go
// internal/domain/entities/product.go
type Product struct {
    ID          uint   `json:"id" db:"id"`
    Name        string `json:"name" db:"name" validate:"required"`
    Description string `json:"description" db:"description"`
    Price       int64  `json:"price" db:"price" validate:"required,min=0"`
    CreatedAt   int64  `json:"created_at" db:"created_at"`
    UpdatedAt   int64  `json:"updated_at" db:"updated_at"`
}

func (p *Product) GetID() uint { return p.ID }
func (p *Product) SetID(id uint) { p.ID = id }
func (p *Product) GetTableName() string { return "products" }
func (p *Product) Validate() error {
    if p.Name == "" {
        return fmt.Errorf("name is required")
    }
    return nil
}
```

2. **Create Module**:

```go
// internal/modules/product_module.go
func NewProductModule() modules.Module {
    return &ProductModule{
        name:    "product",
        version: "1.0.0",
    }
}
```

3. **Register Module**:

```go
// cmd/enterprise/main.go
func (a *Application) registerCoreModules() error {
    // Register product module
    productModule := modules.NewProductModule()
    if err := a.bootstrap.RegisterModule(productModule); err != nil {
        return fmt.Errorf("failed to register product module: %w", err)
    }
    return nil
}
```

## Creating Custom Modules

### 1. Implement Module Interface

```go
type CustomModule struct {
    name         string
    version      string
    dependencies []string
}

func (m *CustomModule) Name() string { return m.name }
func (m *CustomModule) Version() string { return m.version }
func (m *CustomModule) Dependencies() []string { return m.dependencies }

func (m *CustomModule) RegisterServices(container *container.Container) error {
    // Register services with DI container
    container.RegisterSingleton("customService", func(c *container.Container) interface{} {
        return NewCustomService()
    })
    return nil
}

func (m *CustomModule) RegisterRoutes(router *gin.RouterGroup, deps *modules.Dependencies) error {
    // Register HTTP routes
    customGroup := router.Group("/custom")
    customGroup.GET("/endpoint", customHandler.HandleRequest)
    return nil
}

func (m *CustomModule) Migrate(db *sql.DB) error {
    // Database migrations
    return nil
}

func (m *CustomModule) Initialize(ctx context.Context) error {
    // Module initialization
    return nil
}

func (m *CustomModule) Shutdown(ctx context.Context) error {
    // Cleanup logic
    return nil
}
```

### 2. Register Module

```go
customModule := &CustomModule{
    name:    "custom",
    version: "1.0.0",
}

if err := bootstrap.RegisterModule(customModule); err != nil {
    log.Fatalf("Failed to register custom module: %v", err)
}
```

## Code Generation

### Generator Features

- **Complete CRUD**: Generate entity, repository, service, handler, module
- **Test Generation**: Unit tests for all components
- **Database Migrations**: Table creation and indexes
- **Route Registration**: REST API endpoints
- **Swagger Documentation**: API documentation annotations

### Generator Options

```bash
# Available flags
-entity string          # Entity name (required)
-table string          # Table name (defaults to snake_case)
-soft-delete           # Enable soft delete
-timestamps            # Enable timestamps (default: true)
-cache                 # Enable caching (default: true)
-all                   # Generate everything
-gen-entity            # Generate entity only
-gen-repo              # Generate repository only
-gen-service           # Generate service only
-gen-handler           # Generate handler only
-gen-module            # Generate module only
-gen-tests             # Generate tests only
-package string        # Package name
-base-path string      # Base path for generation
```

### Generated File Structure

```
internal/
├── domain/
│   ├── entities/
│   │   ├── product.go
│   │   └── product_test.go
│   └── services/
│       ├── product_service.go
│       ├── product_service_impl.go
│       └── product_service_test.go
├── database/
│   └── repositories/
│       ├── product_repository.go
│       ├── product_repository_impl.go
│       └── product_repository_test.go
├── api/
│   └── handlers/
│       ├── product_handler.go
│       └── product_handler_test.go
└── modules/
    └── product_module.go
```

## Dependency Injection

### Container Features

- **Singleton Registration**: Single instance services
- **Transient Registration**: New instance per request
- **Lazy Loading**: Services created on first access
- **Reflection-based Resolution**: Automatic dependency injection
- **Type-safe Retrieval**: Compile-time type checking

### Usage Examples

```go
// Register singleton
container.RegisterSingleton("userService", func(c *container.Container) interface{} {
    repo := c.MustGet("userRepository").(repositories.UserRepository)
    logger := c.MustGet("logger").(*logger.Logger)
    return services.NewUserService(repo, logger)
})

// Register transient
container.RegisterTransient("requestHandler", func(c *container.Container) interface{} {
    return handlers.NewRequestHandler()
})

// Retrieve service
userService := container.MustGet("userService").(services.UserService)

// Auto-resolve dependencies
type MyService struct {
    UserRepo   repositories.UserRepository `inject:"userRepository"`
    Logger     *logger.Logger              `inject:"logger"`
    RedisCache *redis.Client               `inject:"redis"`
}

var service MyService
if err := container.Resolve(&service); err != nil {
    log.Fatal(err)
}
```

## Module Registry

### Features

- **Auto-discovery**: Automatically find and register modules
- **Dependency Resolution**: Topological sorting of module dependencies
- **Lifecycle Management**: Initialize and shutdown modules in correct order
- **Health Monitoring**: Track module status and health

### Registry Operations

```go
// Register module
if err := registry.Register(module); err != nil {
    log.Fatalf("Failed to register module: %v", err)
}

// Get module
module, err := registry.GetModule("user")
if err != nil {
    log.Fatalf("Module not found: %v", err)
}

// Get all modules (in dependency order)
modules := registry.GetModules()

// Initialize all modules
deps := &modules.Dependencies{
    Container: container,
    Logger:    logger,
    Config:    config,
    DB:        db,
}

if err := registry.Initialize(ctx, deps); err != nil {
    log.Fatalf("Failed to initialize modules: %v", err)
}
```

## Best Practices

### 1. Module Design

- Keep modules focused on a single domain
- Define clear boundaries between modules
- Use interfaces for inter-module communication
- Implement proper dependency declarations

### 2. Entity Design

- Follow Go naming conventions
- Use validation tags for input validation
- Implement required interfaces (Entity, Timestampable, SoftDeletable)
- Keep entities simple and focused

### 3. Service Layer

- Put business logic in services, not handlers
- Use repository interfaces, not concrete implementations
- Implement proper error handling and logging
- Add caching where appropriate

### 4. Database Design

- Use database migrations for schema changes
- Add proper indexes for performance
- Consider soft delete for data preservation
- Use transactions for multi-table operations

### 5. Testing

- Generate tests with code generator
- Use dependency injection for mocking
- Test business logic in services
- Use integration tests for full workflows

## Examples

### Complete Entity Example

```bash
# Generate complete Product module
go run cmd/generator/main.go -entity=Product -table=products -soft-delete -all

# This generates:
# - Product entity with CRUD operations
# - ProductRepository with database operations
# - ProductService with business logic
# - ProductHandler with HTTP endpoints
# - ProductModule with module registration
# - Complete test suite
# - Database migration
```

### Custom Business Logic

```go
// Extend generated service with custom logic
func (s *productService) GetProductsByCategory(ctx context.Context, categoryID uint) ([]*entities.Product, error) {
    // Custom business logic
    filters := modules.ListFilters{
        Filters: map[string]string{
            "category_id": fmt.Sprintf("%d", categoryID),
        },
    }

    products, _, err := s.repository.List(ctx, filters)
    return products, err
}

// Add custom route in module
func (m *ProductModule) RegisterRoutes(router *gin.RouterGroup, deps *modules.Dependencies) error {
    handler := deps.Container.MustGet("productHandler").(*handlers.ProductHandler)

    productGroup := router.Group("/products")
    {
        // Standard CRUD routes (auto-generated)
        productGroup.POST("", handler.Create)
        productGroup.GET("", handler.List)
        productGroup.GET("/:id", handler.GetByID)
        productGroup.PUT("/:id", handler.Update)
        productGroup.DELETE("/:id", handler.Delete)

        // Custom routes
        productGroup.GET("/category/:categoryId", handler.GetByCategory)
        productGroup.POST("/:id/publish", handler.Publish)
        productGroup.POST("/:id/unpublish", handler.Unpublish)
    }

    return nil
}
```

### Module Dependencies

```go
// Order module depends on Product and User modules
func NewOrderModule() modules.Module {
    return &OrderModule{
        name:         "order",
        version:      "1.0.0",
        dependencies: []string{"user", "product"}, // Dependencies will be initialized first
    }
}
```

## Performance Considerations

### 1. Caching Strategy

- Use Redis for frequently accessed data
- Implement cache invalidation on updates
- Cache list queries with appropriate TTL
- Monitor cache hit rates

### 2. Database Optimization

- Use connection pooling
- Add proper database indexes
- Implement query optimization
- Use read replicas for read-heavy workloads

### 3. Module Loading

- Lazy load modules when possible
- Use singleton pattern for shared services
- Minimize startup time with parallel initialization
- Monitor module initialization performance

## Migration Guide

### From Legacy Architecture

1. **Identify Entities**: List all current entities that need CRUD operations
2. **Generate Modules**: Use code generator for each entity
3. **Migrate Business Logic**: Move custom logic to generated services
4. **Update Routes**: Replace manual route registration with module routes
5. **Test Migration**: Ensure all functionality works with new architecture
6. **Remove Old Code**: Clean up legacy dependency injection and routing

### Example Migration

```go
// Before: Manual setup in app.go
func (a *App) initializeServices() {
    // 50+ lines of manual dependency injection
    userRepo := postgres.NewUserRepository(a.db)
    productRepo := postgres.NewProductRepository(a.db)
    orderRepo := postgres.NewOrderRepository(a.db)
    // ... many more
}

// After: Automatic registration
func (a *Application) registerCoreModules() error {
    modules := []modules.Module{
        modules.NewUserModule(),
        modules.NewProductModule(),
        modules.NewOrderModule(),
    }

    for _, module := range modules {
        if err := a.bootstrap.RegisterModule(module); err != nil {
            return err
        }
    }
    return nil
}
```

This enterprise architecture provides a scalable, maintainable foundation for building complex Go applications with minimal boilerplate code and maximum developer productivity.