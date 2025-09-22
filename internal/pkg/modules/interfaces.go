package modules

import (
	"context"
	"database/sql"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/VeRJiL/go-template/internal/config"
	"github.com/VeRJiL/go-template/internal/pkg/auth"
	"github.com/VeRJiL/go-template/internal/pkg/container"
	"github.com/VeRJiL/go-template/internal/pkg/logger"
)

// Entity represents a domain entity with basic CRUD operations
type Entity interface {
	GetID() uint
	SetID(uint)
	GetTableName() string
	Validate() error
}

// SoftDeletable represents an entity that supports soft deletion
type SoftDeletable interface {
	Entity
	IsDeleted() bool
	SetDeleted(bool)
	GetDeletedAt() *int64
	SetDeletedAt(*int64)
}

// Timestampable represents an entity with timestamp fields
type Timestampable interface {
	Entity
	GetCreatedAt() int64
	SetCreatedAt(int64)
	GetUpdatedAt() int64
	SetUpdatedAt(int64)
}

// Repository represents a generic repository interface
type Repository[T Entity] interface {
	Create(ctx context.Context, entity *T) error
	GetByID(ctx context.Context, id uint) (*T, error)
	Update(ctx context.Context, entity *T) error
	Delete(ctx context.Context, id uint) error
	List(ctx context.Context, filters ListFilters) ([]*T, int64, error)
	Exists(ctx context.Context, id uint) (bool, error)
}

// CacheRepository represents a cache repository interface
type CacheRepository[T Entity] interface {
	Get(ctx context.Context, key string) (*T, error)
	Set(ctx context.Context, key string, entity *T, ttl int) error
	Delete(ctx context.Context, key string) error
	Clear(ctx context.Context, pattern string) error
	GetList(ctx context.Context, key string) ([]*T, int64, error)
	SetList(ctx context.Context, key string, entities []*T, total int64, ttl int) error
}

// Service represents a generic service interface
type Service[T Entity] interface {
	Create(ctx context.Context, entity *T) (*T, error)
	GetByID(ctx context.Context, id uint) (*T, error)
	Update(ctx context.Context, id uint, entity *T) (*T, error)
	Delete(ctx context.Context, id uint) error
	List(ctx context.Context, filters ListFilters) ([]*T, int64, error)
}

// Handler represents a generic handler interface
type Handler interface {
	Create(c *gin.Context)
	GetByID(c *gin.Context)
	Update(c *gin.Context)
	Delete(c *gin.Context)
	List(c *gin.Context)
}

// Module represents a business module
type Module interface {
	Name() string
	Version() string
	Dependencies() []string
	RegisterServices(container *container.Container) error
	RegisterRoutes(router *gin.RouterGroup, deps *Dependencies) error
	Migrate(db *sql.DB) error
	Initialize(ctx context.Context) error
	Shutdown(ctx context.Context) error
}

// ModuleInfo contains module metadata
type ModuleInfo struct {
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Description  string   `json:"description"`
	Author       string   `json:"author"`
	Dependencies []string `json:"dependencies"`
	Routes       []Route  `json:"routes"`
	Entities     []string `json:"entities"`
}

// Route represents a module route
type Route struct {
	Method      string            `json:"method"`
	Path        string            `json:"path"`
	Handler     string            `json:"handler"`
	Middleware  []string          `json:"middleware"`
	Auth        bool              `json:"auth"`
	Permissions []string          `json:"permissions"`
	Tags        []string          `json:"tags"`
	Summary     string            `json:"summary"`
	Description string            `json:"description"`
	Parameters  []RouteParameter  `json:"parameters"`
	Responses   map[string]string `json:"responses"`
}

// RouteParameter represents a route parameter
type RouteParameter struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	In          string `json:"in"` // query, path, header, body
	Required    bool   `json:"required"`
	Description string `json:"description"`
}

// Dependencies contains shared dependencies for modules
type Dependencies struct {
	Container   *container.Container
	Logger      *logger.Logger
	Config      *config.Config
	DB          *sql.DB
	RedisClient *redis.Client
	JWTService  *auth.JWTService
}

// ListFilters represents common list filtering options
type ListFilters struct {
	Offset    int               `json:"offset"`
	Limit     int               `json:"limit"`
	Search    string            `json:"search"`
	SortBy    string            `json:"sort_by"`
	SortOrder string            `json:"sort_order"`
	Filters   map[string]string `json:"filters"`
}

// PaginationResponse represents a paginated response
type PaginationResponse struct {
	Data       interface{} `json:"data"`
	Total      int64       `json:"total"`
	Offset     int         `json:"offset"`
	Limit      int         `json:"limit"`
	TotalPages int         `json:"total_pages"`
}

// EntityConfig represents entity configuration
type EntityConfig struct {
	Name        string            `json:"name"`
	TableName   string            `json:"table_name"`
	SoftDelete  bool              `json:"soft_delete"`
	Timestamps  bool              `json:"timestamps"`
	Cache       CacheConfig       `json:"cache"`
	Validation  ValidationConfig  `json:"validation"`
	Permissions PermissionConfig  `json:"permissions"`
	Routes      []Route           `json:"routes"`
}

// CacheConfig represents cache configuration
type CacheConfig struct {
	Enabled bool   `json:"enabled"`
	TTL     string `json:"ttl"`
	Prefix  string `json:"prefix"`
}

// ValidationConfig represents validation configuration
type ValidationConfig struct {
	Required []string          `json:"required"`
	Rules    map[string]string `json:"rules"`
}

// PermissionConfig represents permission configuration
type PermissionConfig struct {
	Create []string `json:"create"`
	Read   []string `json:"read"`
	Update []string `json:"update"`
	Delete []string `json:"delete"`
	List   []string `json:"list"`
}

// Middleware represents middleware interface
type Middleware interface {
	Name() string
	Handler() gin.HandlerFunc
	Priority() int
}

// ModuleRegistry manages module registration and discovery
type ModuleRegistry interface {
	Register(module Module) error
	GetModule(name string) (Module, error)
	GetModules() []Module
	GetModuleInfo() []ModuleInfo
	GetModuleCount() int
	LoadModules() error
	Initialize(ctx context.Context, deps *Dependencies) error
	Shutdown(ctx context.Context) error
}

// Generator represents code generation interface
type Generator interface {
	GenerateEntity(config EntityConfig) error
	GenerateRepository(config EntityConfig) error
	GenerateService(config EntityConfig) error
	GenerateHandler(config EntityConfig) error
	GenerateModule(config EntityConfig) error
	GenerateTests(config EntityConfig) error
}

// EventPublisher represents event publishing interface
type EventPublisher interface {
	Publish(ctx context.Context, event Event) error
	Subscribe(eventType string, handler EventHandler) error
}

// Event represents a domain event
type Event interface {
	Type() string
	AggregateID() string
	Data() interface{}
	Timestamp() int64
}

// EventHandler represents an event handler function
type EventHandler func(ctx context.Context, event Event) error

// Specification represents a specification pattern interface
type Specification[T Entity] interface {
	IsSatisfiedBy(entity *T) bool
	And(spec Specification[T]) Specification[T]
	Or(spec Specification[T]) Specification[T]
	Not() Specification[T]
}

// QueryBuilder represents a query builder interface
type QueryBuilder interface {
	Select(columns ...string) QueryBuilder
	From(table string) QueryBuilder
	Where(condition string, args ...interface{}) QueryBuilder
	Join(joinType, table, condition string) QueryBuilder
	OrderBy(column, direction string) QueryBuilder
	Limit(limit int) QueryBuilder
	Offset(offset int) QueryBuilder
	Build() (string, []interface{})
}

// Validator represents validation interface
type Validator interface {
	Validate(entity interface{}) error
	ValidateField(fieldName string, value interface{}, rules string) error
}

// Serializer represents serialization interface
type Serializer interface {
	Serialize(data interface{}) ([]byte, error)
	Deserialize(data []byte, target interface{}) error
	ContentType() string
}

// Cache represents caching interface
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl int) error
	Delete(ctx context.Context, key string) error
	Clear(ctx context.Context, pattern string) error
	Exists(ctx context.Context, key string) (bool, error)
}

// Transaction represents database transaction interface
type Transaction interface {
	Commit() error
	Rollback() error
	Exec(query string, args ...interface{}) error
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
}