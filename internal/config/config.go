package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	App           AppConfig
	Server        ServerConfig
	Database      DatabaseConfig
	Redis         RedisConfig
	MongoDB       MongoDBConfig
	Elastic       ElasticConfig
	Auth          AuthConfig
	Security      SecurityConfig
	Logging       LoggingConfig
	Email         EmailConfig
	Storage       StorageConfig
	External      ExternalConfig
	Monitoring    MonitoringConfig
	Features      FeatureConfig
	Development   DevelopmentConfig
	Performance   PerformanceConfig
	Backup        BackupConfig
	Localization  LocalizationConfig
	Custom        CustomConfig
	MessageBroker MessageBrokerConfig
	ELK           ELKConfig
	GRPC          GRPCConfig
}

type AppConfig struct {
	Name        string
	Version     string
	Description string
	Author      string
	License     string
}

type ServerConfig struct {
	Host            string
	Port            string
	Mode            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
	MaxBodySize     int64
	EnablePprof     bool
	EnableMetrics   bool
	EnableSwagger   bool
	EnableCORS      bool
}

type DatabaseConfig struct {
	Driver          string
	Host            string
	Port            string
	User            string
	Password        string
	Database        string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	MaxConnLifetime time.Duration
	QueryTimeout    time.Duration
	AutoMigrate     bool
	MigrationPath   string
}

type RedisConfig struct {
	Host         string
	Port         string
	Password     string
	DB           int
	PoolSize     int
	MinIdleConns int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	UserTTL      time.Duration
	SessionTTL   time.Duration
	DefaultTTL   time.Duration
}

type MongoDBConfig struct {
	URI                    string
	Database               string
	Username               string
	Password               string
	AuthSource             string
	MaxPoolSize            int
	MinPoolSize            int
	ConnectTimeout         time.Duration
	ServerSelectionTimeout time.Duration
}

type ElasticConfig struct {
	URLs          []string
	Username      string
	Password      string
	IndexPrefix   string
	APIKey        string
	MaxRetries    int
	Compress      bool
	DiscoverNodes bool
	Sniff         bool
}

type AuthConfig struct {
	JWT      JWTConfig
	Session  SessionConfig
	Password PasswordConfig
	Account  AccountConfig
}

type JWTConfig struct {
	Secret            string
	Expiration        time.Duration
	RefreshExpiration time.Duration
	Issuer            string
	Algorithm         string
}

type SessionConfig struct {
	Secret   string
	MaxAge   time.Duration
	Secure   bool
	HTTPOnly bool
	SameSite string
}

type PasswordConfig struct {
	MinLength        int
	RequireUppercase bool
	RequireLowercase bool
	RequireNumbers   bool
	RequireSpecial   bool
	MaxAge           time.Duration
}

type AccountConfig struct {
	MaxLoginAttempts          int
	LockoutDuration           time.Duration
	PasswordResetExpiry       time.Duration
	EmailVerificationRequired bool
}

type SecurityConfig struct {
	RateLimit RateLimitConfig
	IP        IPSecurityConfig
	Headers   SecurityHeadersConfig
	CSRF      CSRFConfig
}

type RateLimitConfig struct {
	Global int
	Auth   int
	API    int
	Public int
}

type IPSecurityConfig struct {
	EnableWhitelist bool
	WhitelistedIPs  []string
	BlockedIPs      []string
}

type SecurityHeadersConfig struct {
	Enable     bool
	EnableHSTS bool
	HSTSMaxAge int
	CSPPolicy  string
}

type CSRFConfig struct {
	Key      string
	Secure   bool
	HTTPOnly bool
}

type LoggingConfig struct {
	Level       string
	Format      string
	Output      string
	FilePath    string
	MaxSizeMB   int
	MaxBackups  int
	MaxAgeDays  int
	Compress    bool
	LogRequests bool
	LogHeaders  bool
	LogBody     bool
}

type EmailConfig struct {
	Provider    string
	FromAddress string
	FromName    string
	SMTP        SMTPConfig
	SendGrid    SendGridConfig
	Mailgun     MailgunConfig
	AWSSES      AWSSESConfig
	TemplateDir string
}

type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	UseTLS   bool
}

type SendGridConfig struct {
	APIKey string
}

type MailgunConfig struct {
	APIKey  string
	Domain  string
	BaseURL string
}

type AWSSESConfig struct {
	Region    string
	AccessKey string
	SecretKey string
}

type StorageConfig struct {
	Provider         string
	Local            LocalStorageConfig
	S3               S3Config
	MinIO            MinIOConfig
	CloudflareR2     CloudflareR2Config
	BackblazeB2      BackblazeB2Config
	GCS              GCSConfig
	Azure            AzureConfig
	MaxUploadSizeMB  int
	AllowedFileTypes []string
	UploadPath       string
}

type LocalStorageConfig struct {
	Path      string
	URLPrefix string
}

type S3Config struct {
	Region         string
	Bucket         string
	AccessKey      string
	SecretKey      string
	UseSSL         bool
	ForcePathStyle bool
}

type MinIOConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
	PublicURL string
}

type CloudflareR2Config struct {
	AccountID string
	AccessKey string
	SecretKey string
	Bucket    string
	PublicURL string
}

type BackblazeB2Config struct {
	Region    string
	KeyID     string
	KeySecret string
	Bucket    string
	PublicURL string
}

type GCSConfig struct {
	Bucket          string
	ProjectID       string
	CredentialsFile string
}

type AzureConfig struct {
	Account   string
	Key       string
	Container string
}

type ExternalConfig struct {
	Stripe       StripeConfig
	Google       GoogleConfig
	Social       SocialConfig
	Notification NotificationConfig
}

type StripeConfig struct {
	PublicKey     string
	SecretKey     string
	WebhookSecret string
	SuccessURL    string
	CancelURL     string
}

type GoogleConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	AnalyticsID  string
}

type SocialConfig struct {
	TwitterAPIKey    string
	TwitterAPISecret string
	FacebookAppID    string
	FacebookSecret   string
}

type NotificationConfig struct {
	FirebaseKey   string
	PusherAppID   string
	PusherKey     string
	PusherSecret  string
	PusherCluster string
}

type MonitoringConfig struct {
	Enable     bool
	Provider   string
	Prometheus PrometheusConfig
	DataDog    DataDogConfig
	NewRelic   NewRelicConfig
	Sentry     SentryConfig
}

type PrometheusConfig struct {
	Namespace   string
	MetricsPath string
}

type DataDogConfig struct {
	APIKey string
	AppKey string
	Site   string
}

type NewRelicConfig struct {
	LicenseKey string
	AppName    string
}

type SentryConfig struct {
	DSN         string
	Environment string
	Release     string
	Debug       bool
}

type FeatureConfig struct {
	UserRegistration  bool
	EmailVerification bool
	TwoFactorAuth     bool
	SocialLogin       bool
	APIRateLimiting   bool
	MaintenanceMode   bool
	Payments          bool
	Subscriptions     bool
	Invoicing         bool
	FileUpload        bool
	ImageProcessing   bool
	ContentModeration bool
}

type DevelopmentConfig struct {
	EnableDebug     bool
	EnableHotReload bool
	EnableQueryLog  bool
	EnableProfiling bool
	TestDatabaseURL string
	TestRedisURL    string
	ParallelTests   bool
	TestTimeout     time.Duration
	Swagger         SwaggerConfig
}

type SwaggerConfig struct {
	Title    string
	Version  string
	Host     string
	BasePath string
	Schemes  []string
}

type PerformanceConfig struct {
	ResponseCaching    bool
	CacheStrategy      string
	CacheDuration      time.Duration
	QueryCache         bool
	ConnectionPooling  bool
	PreparedStatements bool
	AssetCacheDuration time.Duration
	GzipCompression    bool
	AssetMinification  bool
}

type BackupConfig struct {
	EnableAuto      bool
	Schedule        string
	RetentionDays   int
	StoragePath     string
	MaintenanceMode bool
	MaintenanceMsg  string
	AllowedIPs      []string
	HealthCheck     HealthCheckConfig
}

type HealthCheckConfig struct {
	Interval time.Duration
	Timeout  time.Duration
	Retries  int
}

type LocalizationConfig struct {
	DefaultLanguage    string
	SupportedLanguages []string
	Timezone           string
	DateFormat         string
	TimeFormat         string
	Currency           string
}

type CustomConfig struct {
	CompanyName          string
	CompanyEmail         string
	CompanyPhone         string
	CompanyAddress       string
	MaxUsersPerOrg       int
	DefaultUserRole      string
	TrialPeriodDays      int
	MaxAPIRequestsPerDay int
}

// MessageBrokerConfig holds configuration for message brokers
type MessageBrokerConfig struct {
	Enabled  bool               `json:"enabled" mapstructure:"enabled"`
	Driver   string             `json:"driver" mapstructure:"driver"`
	RabbitMQ *RabbitMQConfig    `json:"rabbitmq,omitempty" mapstructure:"rabbitmq"`
	Kafka    *KafkaConfig       `json:"kafka,omitempty" mapstructure:"kafka"`
	Redis    *RedisPubSubConfig `json:"redis,omitempty" mapstructure:"redis"`
	Retry    *RetryConfig       `json:"retry,omitempty" mapstructure:"retry"`
}

// RabbitMQConfig holds RabbitMQ-specific configuration
type RabbitMQConfig struct {
	URL               string        `json:"url" mapstructure:"url"`
	Host              string        `json:"host" mapstructure:"host"`
	Port              int           `json:"port" mapstructure:"port"`
	Username          string        `json:"username" mapstructure:"username"`
	Password          string        `json:"password" mapstructure:"password"`
	VHost             string        `json:"vhost" mapstructure:"vhost"`
	Exchange          string        `json:"exchange" mapstructure:"exchange"`
	ExchangeType      string        `json:"exchange_type" mapstructure:"exchange_type"`
	ConnectionTimeout time.Duration `json:"connection_timeout" mapstructure:"connection_timeout"`
	HeartbeatInterval time.Duration `json:"heartbeat_interval" mapstructure:"heartbeat_interval"`
	PrefetchCount     int           `json:"prefetch_count" mapstructure:"prefetch_count"`
	Durable           bool          `json:"durable" mapstructure:"durable"`
	AutoDelete        bool          `json:"auto_delete" mapstructure:"auto_delete"`
}

// KafkaConfig holds Kafka-specific configuration
type KafkaConfig struct {
	Brokers            []string      `json:"brokers" mapstructure:"brokers"`
	GroupID            string        `json:"group_id" mapstructure:"group_id"`
	ClientID           string        `json:"client_id" mapstructure:"client_id"`
	Version            string        `json:"version" mapstructure:"version"`
	ConnectTimeout     time.Duration `json:"connect_timeout" mapstructure:"connect_timeout"`
	SessionTimeout     time.Duration `json:"session_timeout" mapstructure:"session_timeout"`
	HeartbeatInterval  time.Duration `json:"heartbeat_interval" mapstructure:"heartbeat_interval"`
	RebalanceTimeout   time.Duration `json:"rebalance_timeout" mapstructure:"rebalance_timeout"`
	ReturnSuccesses    bool          `json:"return_successes" mapstructure:"return_successes"`
	RequiredAcks       int           `json:"required_acks" mapstructure:"required_acks"`
	CompressionType    string        `json:"compression" mapstructure:"compression"`
	FlushFrequency     time.Duration `json:"flush_frequency" mapstructure:"flush_frequency"`
	EnableAutoCommit   bool          `json:"enable_auto_commit" mapstructure:"enable_auto_commit"`
	AutoCommitInterval time.Duration `json:"auto_commit_interval" mapstructure:"auto_commit_interval"`
	InitialOffset      string        `json:"initial_offset" mapstructure:"initial_offset"`
	SASL               *SASLConfig   `json:"sasl,omitempty" mapstructure:"sasl"`
	TLS                *TLSConfig    `json:"tls,omitempty" mapstructure:"tls"`
}

// RedisPubSubConfig holds Redis Pub/Sub configuration
type RedisPubSubConfig struct {
	Host           string        `json:"host" mapstructure:"host"`
	Port           int           `json:"port" mapstructure:"port"`
	Password       string        `json:"password" mapstructure:"password"`
	DB             int           `json:"db" mapstructure:"db"`
	PoolSize       int           `json:"pool_size" mapstructure:"pool_size"`
	MinIdleConns   int           `json:"min_idle_conns" mapstructure:"min_idle_conns"`
	MaxRetries     int           `json:"max_retries" mapstructure:"max_retries"`
	ConnectTimeout time.Duration `json:"connect_timeout" mapstructure:"connect_timeout"`
	ReadTimeout    time.Duration `json:"read_timeout" mapstructure:"read_timeout"`
	WriteTimeout   time.Duration `json:"write_timeout" mapstructure:"write_timeout"`
	IdleTimeout    time.Duration `json:"idle_timeout" mapstructure:"idle_timeout"`
	TLS            *TLSConfig    `json:"tls,omitempty" mapstructure:"tls"`
}

// RetryConfig holds retry configuration for failed messages/jobs
type RetryConfig struct {
	MaxRetries      int           `json:"max_retries" mapstructure:"max_retries"`
	InitialInterval time.Duration `json:"initial_interval" mapstructure:"initial_interval"`
	MaxInterval     time.Duration `json:"max_interval" mapstructure:"max_interval"`
	Multiplier      float64       `json:"multiplier" mapstructure:"multiplier"`
	RandomFactor    float64       `json:"random_factor" mapstructure:"random_factor"`
}

// SASLConfig holds SASL authentication configuration for Kafka
type SASLConfig struct {
	Enable    bool   `json:"enable" mapstructure:"enable"`
	Mechanism string `json:"mechanism" mapstructure:"mechanism"`
	Username  string `json:"username" mapstructure:"username"`
	Password  string `json:"password" mapstructure:"password"`
}

// TLSConfig holds TLS configuration
type TLSConfig struct {
	Enable             bool   `json:"enable" mapstructure:"enable"`
	CertFile           string `json:"cert_file" mapstructure:"cert_file"`
	KeyFile            string `json:"key_file" mapstructure:"key_file"`
	CAFile             string `json:"ca_file" mapstructure:"ca_file"`
	InsecureSkipVerify bool   `json:"insecure_skip_verify" mapstructure:"insecure_skip_verify"`
}

// EnhancedLoggingConfig holds comprehensive logging configuration
type EnhancedLoggingConfig struct {
	Level          string `json:"level" mapstructure:"level"`
	Format         string `json:"format" mapstructure:"format"`
	Output         string `json:"output" mapstructure:"output"`
	FilePath       string `json:"file_path" mapstructure:"file_path"`
	MaxSizeMB      int    `json:"max_size_mb" mapstructure:"max_size_mb"`
	MaxBackups     int    `json:"max_backups" mapstructure:"max_backups"`
	MaxAgeDays     int    `json:"max_age_days" mapstructure:"max_age_days"`
	Compress       bool   `json:"compress" mapstructure:"compress"`
	LogRequests    bool   `json:"log_requests" mapstructure:"log_requests"`
	LogHeaders     bool   `json:"log_headers" mapstructure:"log_headers"`
	LogBody        bool   `json:"log_body" mapstructure:"log_body"`
	ServiceName    string `json:"service_name" mapstructure:"service_name"`
	ServiceVersion string `json:"service_version" mapstructure:"service_version"`
	Environment    string `json:"environment" mapstructure:"environment"`
}

// EnhancedMonitoringConfig holds Prometheus monitoring configuration
type EnhancedMonitoringConfig struct {
	Enabled     bool   `json:"enabled" mapstructure:"enabled"`
	Namespace   string `json:"namespace" mapstructure:"namespace"`
	MetricsPath string `json:"metrics_path" mapstructure:"metrics_path"`
	ListenAddr  string `json:"listen_addr" mapstructure:"listen_addr"`
}

// ELKConfig holds Elasticsearch configuration for the ELK stack
type ELKConfig struct {
	Enabled     bool     `json:"enabled" mapstructure:"enabled"`
	URLs        []string `json:"urls" mapstructure:"urls"`
	Username    string   `json:"username" mapstructure:"username"`
	Password    string   `json:"password" mapstructure:"password"`
	IndexPrefix string   `json:"index_prefix" mapstructure:"index_prefix"`
	APIKey      string   `json:"api_key" mapstructure:"api_key"`
	MaxRetries  int      `json:"max_retries" mapstructure:"max_retries"`
	Compress    bool     `json:"compress" mapstructure:"compress"`
	BatchSize   int      `json:"batch_size" mapstructure:"batch_size"`
	BatchWait   string   `json:"batch_wait" mapstructure:"batch_wait"`
}

// GRPCConfig holds gRPC server configuration
type GRPCConfig struct {
	Enabled               bool              `json:"enabled" mapstructure:"enabled"`
	Port                  string            `json:"port" mapstructure:"port"`
	MaxReceiveSize        int               `json:"max_receive_size" mapstructure:"max_receive_size"`
	MaxSendSize           int               `json:"max_send_size" mapstructure:"max_send_size"`
	MaxConnectionIdle     time.Duration     `json:"max_connection_idle" mapstructure:"max_connection_idle"`
	MaxConnectionAge      time.Duration     `json:"max_connection_age" mapstructure:"max_connection_age"`
	MaxConnectionAgeGrace time.Duration     `json:"max_connection_age_grace" mapstructure:"max_connection_age_grace"`
	KeepAliveTime         time.Duration     `json:"keep_alive_time" mapstructure:"keep_alive_time"`
	KeepAliveTimeout      time.Duration     `json:"keep_alive_timeout" mapstructure:"keep_alive_timeout"`
	TLS                   *GRPCTLSConfig    `json:"tls,omitempty" mapstructure:"tls"`
	Reflection            bool              `json:"reflection" mapstructure:"reflection"`
	Gateway               GRPCGatewayConfig `json:"gateway" mapstructure:"gateway"`
}

// GRPCTLSConfig holds TLS configuration for gRPC
type GRPCTLSConfig struct {
	Enable   bool   `json:"enable" mapstructure:"enable"`
	CertFile string `json:"cert_file" mapstructure:"cert_file"`
	KeyFile  string `json:"key_file" mapstructure:"key_file"`
}

// GRPCGatewayConfig holds gRPC-Gateway configuration
type GRPCGatewayConfig struct {
	Enabled bool   `json:"enabled" mapstructure:"enabled"`
	Port    string `json:"port" mapstructure:"port"`
	Prefix  string `json:"prefix" mapstructure:"prefix"`
}

func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		fmt.Println("No .env file found, using environment variables")
	}

	config := &Config{
		App: AppConfig{
			Name:        getEnv("APP_NAME", "Go Template"),
			Version:     getEnv("APP_VERSION", "1.0.0"),
			Description: getEnv("APP_DESCRIPTION", "Professional Go application template"),
			Author:      getEnv("APP_AUTHOR", "Your Company"),
			License:     getEnv("APP_LICENSE", "MIT"),
		},
		Server: ServerConfig{
			Host:            getEnv("SERVER_HOST", "localhost"),
			Port:            getEnv("SERVER_PORT", "8080"),
			Mode:            getEnv("SERVER_MODE", "development"),
			ReadTimeout:     getEnvAsDuration("READ_TIMEOUT", 30*time.Second),
			WriteTimeout:    getEnvAsDuration("WRITE_TIMEOUT", 30*time.Second),
			IdleTimeout:     getEnvAsDuration("IDLE_TIMEOUT", 120*time.Second),
			ShutdownTimeout: getEnvAsDuration("SHUTDOWN_TIMEOUT", 30*time.Second),
			MaxBodySize:     getEnvAsInt64("MAX_BODY_SIZE", 10) * 1024 * 1024, // Convert MB to bytes
			EnablePprof:     getEnvAsBool("ENABLE_PPROF", true),
			EnableMetrics:   getEnvAsBool("ENABLE_METRICS", true),
			EnableSwagger:   getEnvAsBool("ENABLE_SWAGGER", true),
			EnableCORS:      getEnvAsBool("ENABLE_CORS", true),
		},
		Database: DatabaseConfig{
			Driver:          getEnv("DB_DRIVER", "postgres"),
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnv("DB_PORT", "5432"),
			User:            getEnv("DB_USER", "postgres"),
			Password:        getEnv("DB_PASSWORD", "password"),
			Database:        getEnv("DB_NAME", "go_template"),
			SSLMode:         getEnv("DB_SSLMODE", "disable"),
			MaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 5),
			MaxConnLifetime: getEnvAsDuration("DB_MAX_CONN_LIFETIME_HOURS", 1*time.Hour),
			QueryTimeout:    getEnvAsDuration("DB_QUERY_TIMEOUT", 30*time.Second),
			AutoMigrate:     getEnvAsBool("DB_AUTO_MIGRATE", false),
			MigrationPath:   getEnv("DB_MIGRATION_PATH", "./migrations/postgres"),
		},
		Redis: RedisConfig{
			Host:         getEnv("REDIS_HOST", "localhost"),
			Port:         getEnv("REDIS_PORT", "6379"),
			Password:     getEnv("REDIS_PASSWORD", ""),
			DB:           getEnvAsInt("REDIS_DB", 0),
			PoolSize:     getEnvAsInt("REDIS_POOL_SIZE", 10),
			MinIdleConns: getEnvAsInt("REDIS_MIN_IDLE_CONNS", 3),
			DialTimeout:  getEnvAsDuration("REDIS_DIAL_TIMEOUT", 5*time.Second),
			ReadTimeout:  getEnvAsDuration("REDIS_READ_TIMEOUT", 3*time.Second),
			WriteTimeout: getEnvAsDuration("REDIS_WRITE_TIMEOUT", 3*time.Second),
			UserTTL:      getEnvAsDuration("CACHE_USER_TTL", 3600*time.Second),
			SessionTTL:   getEnvAsDuration("CACHE_SESSION_TTL", 86400*time.Second),
			DefaultTTL:   getEnvAsDuration("CACHE_DEFAULT_TTL", 1800*time.Second),
		},
		MongoDB: MongoDBConfig{
			URI:                    getEnv("MONGODB_URI", "mongodb://localhost:27017"),
			Database:               getEnv("MONGODB_DATABASE", "go_template"),
			Username:               getEnv("MONGODB_USERNAME", ""),
			Password:               getEnv("MONGODB_PASSWORD", ""),
			AuthSource:             getEnv("MONGODB_AUTH_SOURCE", "admin"),
			MaxPoolSize:            getEnvAsInt("MONGODB_MAX_POOL_SIZE", 100),
			MinPoolSize:            getEnvAsInt("MONGODB_MIN_POOL_SIZE", 5),
			ConnectTimeout:         getEnvAsDuration("MONGODB_CONNECT_TIMEOUT", 10*time.Second),
			ServerSelectionTimeout: getEnvAsDuration("MONGODB_SERVER_SELECTION_TIMEOUT", 5*time.Second),
		},
		Elastic: ElasticConfig{
			URLs:          strings.Split(getEnv("ELASTICSEARCH_URLS", "http://localhost:9200"), ","),
			Username:      getEnv("ELASTICSEARCH_USERNAME", ""),
			Password:      getEnv("ELASTICSEARCH_PASSWORD", ""),
			IndexPrefix:   getEnv("ELASTICSEARCH_INDEX_PREFIX", "go_template"),
			APIKey:        getEnv("ELASTICSEARCH_API_KEY", ""),
			MaxRetries:    getEnvAsInt("ELASTICSEARCH_MAX_RETRIES", 3),
			Compress:      getEnvAsBool("ELASTICSEARCH_COMPRESS", true),
			DiscoverNodes: getEnvAsBool("ELASTICSEARCH_DISCOVER_NODES", false),
			Sniff:         getEnvAsBool("ELASTICSEARCH_SNIFF", false),
		},
	}

	// Load Auth configuration
	config.Auth = AuthConfig{
		JWT: JWTConfig{
			Secret:            getEnv("JWT_SECRET", "your-secret-key"),
			Expiration:        getEnvAsDuration("JWT_EXPIRATION_HOURS", 24*time.Hour),
			RefreshExpiration: getEnvAsDuration("JWT_REFRESH_EXPIRATION_HOURS", 168*time.Hour),
			Issuer:            getEnv("JWT_ISSUER", "go-template"),
			Algorithm:         getEnv("JWT_ALGORITHM", "HS256"),
		},
		Session: SessionConfig{
			Secret:   getEnv("SESSION_SECRET", "your-session-secret"),
			MaxAge:   getEnvAsDuration("SESSION_MAX_AGE_HOURS", 24*time.Hour),
			Secure:   getEnvAsBool("SESSION_SECURE", false),
			HTTPOnly: getEnvAsBool("SESSION_HTTP_ONLY", true),
			SameSite: getEnv("SESSION_SAME_SITE", "lax"),
		},
		Password: PasswordConfig{
			MinLength:        getEnvAsInt("PASSWORD_MIN_LENGTH", 8),
			RequireUppercase: getEnvAsBool("PASSWORD_REQUIRE_UPPERCASE", true),
			RequireLowercase: getEnvAsBool("PASSWORD_REQUIRE_LOWERCASE", true),
			RequireNumbers:   getEnvAsBool("PASSWORD_REQUIRE_NUMBERS", true),
			RequireSpecial:   getEnvAsBool("PASSWORD_REQUIRE_SPECIAL", true),
			MaxAge:           getEnvAsDuration("PASSWORD_MAX_AGE_DAYS", 90*24*time.Hour),
		},
		Account: AccountConfig{
			MaxLoginAttempts:          getEnvAsInt("MAX_LOGIN_ATTEMPTS", 5),
			LockoutDuration:           getEnvAsDuration("LOCKOUT_DURATION_MINUTES", 30*time.Minute),
			PasswordResetExpiry:       getEnvAsDuration("PASSWORD_RESET_EXPIRY_MINUTES", 30*time.Minute),
			EmailVerificationRequired: getEnvAsBool("EMAIL_VERIFICATION_REQUIRED", false),
		},
	}

	// Load Security configuration
	config.Security = SecurityConfig{
		RateLimit: RateLimitConfig{
			Global: getEnvAsInt("RATE_LIMIT_GLOBAL", 1000),
			Auth:   getEnvAsInt("RATE_LIMIT_AUTH", 10),
			API:    getEnvAsInt("RATE_LIMIT_API", 100),
			Public: getEnvAsInt("RATE_LIMIT_PUBLIC", 50),
		},
		IP: IPSecurityConfig{
			EnableWhitelist: getEnvAsBool("ENABLE_IP_WHITELIST", false),
			WhitelistedIPs:  getEnvAsStringSlice("WHITELISTED_IPS", "127.0.0.1,::1"),
			BlockedIPs:      getEnvAsStringSlice("BLOCKED_IPS", ""),
		},
		Headers: SecurityHeadersConfig{
			Enable:     getEnvAsBool("ENABLE_SECURITY_HEADERS", true),
			EnableHSTS: getEnvAsBool("ENABLE_HSTS", true),
			HSTSMaxAge: getEnvAsInt("HSTS_MAX_AGE", 31536000),
			CSPPolicy:  getEnv("CSP_POLICY", "default-src 'self'"),
		},
		CSRF: CSRFConfig{
			Key:      getEnv("CSRF_KEY", "32-character-key-for-csrf-protection"),
			Secure:   getEnvAsBool("CSRF_SECURE", false),
			HTTPOnly: getEnvAsBool("CSRF_HTTP_ONLY", true),
		},
	}

	// Load Storage configuration
	config.Storage = StorageConfig{
		Provider: getEnv("STORAGE_PROVIDER", "local"),
		Local: LocalStorageConfig{
			Path:      getEnv("LOCAL_STORAGE_PATH", "./uploads"),
			URLPrefix: getEnv("LOCAL_STORAGE_URL_PREFIX", "/uploads"),
		},
		S3: S3Config{
			Region:         getEnv("AWS_S3_REGION", "us-east-1"),
			Bucket:         getEnv("AWS_S3_BUCKET", ""),
			AccessKey:      getEnv("AWS_S3_ACCESS_KEY", ""),
			SecretKey:      getEnv("AWS_S3_SECRET_KEY", ""),
			UseSSL:         getEnvAsBool("AWS_S3_USE_SSL", true),
			ForcePathStyle: getEnvAsBool("AWS_S3_FORCE_PATH_STYLE", false),
		},
		MinIO: MinIOConfig{
			Endpoint:  getEnv("MINIO_ENDPOINT", "localhost:9000"),
			AccessKey: getEnv("MINIO_ACCESS_KEY", ""),
			SecretKey: getEnv("MINIO_SECRET_KEY", ""),
			Bucket:    getEnv("MINIO_BUCKET", "uploads"),
			UseSSL:    getEnvAsBool("MINIO_USE_SSL", false),
			PublicURL: getEnv("MINIO_PUBLIC_URL", ""),
		},
		CloudflareR2: CloudflareR2Config{
			AccountID: getEnv("CLOUDFLARE_R2_ACCOUNT_ID", ""),
			AccessKey: getEnv("CLOUDFLARE_R2_ACCESS_KEY", ""),
			SecretKey: getEnv("CLOUDFLARE_R2_SECRET_KEY", ""),
			Bucket:    getEnv("CLOUDFLARE_R2_BUCKET", ""),
			PublicURL: getEnv("CLOUDFLARE_R2_PUBLIC_URL", ""),
		},
		BackblazeB2: BackblazeB2Config{
			Region:    getEnv("BACKBLAZE_B2_REGION", "us-west-000"),
			KeyID:     getEnv("BACKBLAZE_B2_KEY_ID", ""),
			KeySecret: getEnv("BACKBLAZE_B2_KEY_SECRET", ""),
			Bucket:    getEnv("BACKBLAZE_B2_BUCKET", ""),
			PublicURL: getEnv("BACKBLAZE_B2_PUBLIC_URL", ""),
		},
		GCS: GCSConfig{
			Bucket:          getEnv("GCS_BUCKET", ""),
			ProjectID:       getEnv("GCS_PROJECT_ID", ""),
			CredentialsFile: getEnv("GCS_CREDENTIALS_FILE", "./credentials/gcs-service-account.json"),
		},
		Azure: AzureConfig{
			Account:   getEnv("AZURE_STORAGE_ACCOUNT", ""),
			Key:       getEnv("AZURE_STORAGE_KEY", ""),
			Container: getEnv("AZURE_STORAGE_CONTAINER", ""),
		},
		MaxUploadSizeMB:  getEnvAsInt("MAX_UPLOAD_SIZE_MB", 50),
		AllowedFileTypes: getEnvAsStringSlice("ALLOWED_FILE_TYPES", "jpg,jpeg,png,gif,pdf,doc,docx,txt"),
		UploadPath:       getEnv("UPLOAD_PATH", "uploads"),
	}

	// Load Feature flags
	config.Features = FeatureConfig{
		UserRegistration:  getEnvAsBool("FEATURE_USER_REGISTRATION", true),
		EmailVerification: getEnvAsBool("FEATURE_EMAIL_VERIFICATION", false),
		TwoFactorAuth:     getEnvAsBool("FEATURE_TWO_FACTOR_AUTH", false),
		SocialLogin:       getEnvAsBool("FEATURE_SOCIAL_LOGIN", false),
		APIRateLimiting:   getEnvAsBool("FEATURE_API_RATE_LIMITING", true),
		MaintenanceMode:   getEnvAsBool("FEATURE_MAINTENANCE_MODE", false),
		Payments:          getEnvAsBool("FEATURE_PAYMENTS", false),
		Subscriptions:     getEnvAsBool("FEATURE_SUBSCRIPTIONS", false),
		Invoicing:         getEnvAsBool("FEATURE_INVOICING", false),
		FileUpload:        getEnvAsBool("FEATURE_FILE_UPLOAD", true),
		ImageProcessing:   getEnvAsBool("FEATURE_IMAGE_PROCESSING", false),
		ContentModeration: getEnvAsBool("FEATURE_CONTENT_MODERATION", false),
	}

	// Load Message Broker configuration
	config.MessageBroker = MessageBrokerConfig{
		Enabled: getEnvAsBool("MESSAGE_BROKER_ENABLED", false),
		Driver:  getEnv("MESSAGE_BROKER_DRIVER", "redis"),
	}

	// RabbitMQ configuration
	if config.MessageBroker.Driver == "rabbitmq" && config.MessageBroker.Enabled {
		config.MessageBroker.RabbitMQ = &RabbitMQConfig{
			URL:               getEnv("RABBITMQ_URL", ""),
			Host:              getEnv("RABBITMQ_HOST", "localhost"),
			Port:              getEnvAsInt("RABBITMQ_PORT", 5672),
			Username:          getEnv("RABBITMQ_USERNAME", "guest"),
			Password:          getEnv("RABBITMQ_PASSWORD", "guest"),
			VHost:             getEnv("RABBITMQ_VHOST", "/"),
			Exchange:          getEnv("RABBITMQ_EXCHANGE", "go_template_exchange"),
			ExchangeType:      getEnv("RABBITMQ_EXCHANGE_TYPE", "topic"),
			ConnectionTimeout: getEnvAsDuration("RABBITMQ_CONNECTION_TIMEOUT", 30*time.Second),
			HeartbeatInterval: getEnvAsDuration("RABBITMQ_HEARTBEAT_INTERVAL", 60*time.Second),
			PrefetchCount:     getEnvAsInt("RABBITMQ_PREFETCH_COUNT", 10),
			Durable:           getEnvAsBool("RABBITMQ_DURABLE", true),
			AutoDelete:        getEnvAsBool("RABBITMQ_AUTO_DELETE", false),
		}
	}

	// Kafka configuration
	if config.MessageBroker.Driver == "kafka" && config.MessageBroker.Enabled {
		config.MessageBroker.Kafka = &KafkaConfig{
			Brokers:            getEnvAsStringSlice("KAFKA_BROKERS", "localhost:9092"),
			GroupID:            getEnv("KAFKA_GROUP_ID", "go-template-consumer-group"),
			ClientID:           getEnv("KAFKA_CLIENT_ID", "go-template-client"),
			Version:            getEnv("KAFKA_VERSION", "2.6.0"),
			ConnectTimeout:     getEnvAsDuration("KAFKA_CONNECT_TIMEOUT", 30*time.Second),
			SessionTimeout:     getEnvAsDuration("KAFKA_SESSION_TIMEOUT", 30*time.Second),
			HeartbeatInterval:  getEnvAsDuration("KAFKA_HEARTBEAT_INTERVAL", 3*time.Second),
			RebalanceTimeout:   getEnvAsDuration("KAFKA_REBALANCE_TIMEOUT", 60*time.Second),
			ReturnSuccesses:    getEnvAsBool("KAFKA_RETURN_SUCCESSES", true),
			RequiredAcks:       getEnvAsInt("KAFKA_REQUIRED_ACKS", 1),
			CompressionType:    getEnv("KAFKA_COMPRESSION", "snappy"),
			FlushFrequency:     getEnvAsDuration("KAFKA_FLUSH_FREQUENCY", 100*time.Millisecond),
			EnableAutoCommit:   getEnvAsBool("KAFKA_ENABLE_AUTO_COMMIT", true),
			AutoCommitInterval: getEnvAsDuration("KAFKA_AUTO_COMMIT_INTERVAL", 1*time.Second),
			InitialOffset:      getEnv("KAFKA_INITIAL_OFFSET", "newest"),
		}

		// SASL configuration for Kafka
		if getEnvAsBool("KAFKA_SASL_ENABLE", false) {
			config.MessageBroker.Kafka.SASL = &SASLConfig{
				Enable:    true,
				Mechanism: getEnv("KAFKA_SASL_MECHANISM", "PLAIN"),
				Username:  getEnv("KAFKA_SASL_USERNAME", ""),
				Password:  getEnv("KAFKA_SASL_PASSWORD", ""),
			}
		}

		// TLS configuration for Kafka
		if getEnvAsBool("KAFKA_TLS_ENABLE", false) {
			config.MessageBroker.Kafka.TLS = &TLSConfig{
				Enable:             true,
				CertFile:           getEnv("KAFKA_TLS_CERT_FILE", ""),
				KeyFile:            getEnv("KAFKA_TLS_KEY_FILE", ""),
				CAFile:             getEnv("KAFKA_TLS_CA_FILE", ""),
				InsecureSkipVerify: getEnvAsBool("KAFKA_TLS_INSECURE_SKIP_VERIFY", false),
			}
		}
	}

	// Redis Pub/Sub configuration
	if config.MessageBroker.Driver == "redis" && config.MessageBroker.Enabled {
		config.MessageBroker.Redis = &RedisPubSubConfig{
			Host:           getEnv("MESSAGE_BROKER_REDIS_HOST", config.Redis.Host),
			Port:           getEnvAsInt("MESSAGE_BROKER_REDIS_PORT", 6379),
			Password:       getEnv("MESSAGE_BROKER_REDIS_PASSWORD", config.Redis.Password),
			DB:             getEnvAsInt("MESSAGE_BROKER_REDIS_DB", 1), // Different DB than cache
			PoolSize:       getEnvAsInt("MESSAGE_BROKER_REDIS_POOL_SIZE", 10),
			MinIdleConns:   getEnvAsInt("MESSAGE_BROKER_REDIS_MIN_IDLE_CONNS", 3),
			MaxRetries:     getEnvAsInt("MESSAGE_BROKER_REDIS_MAX_RETRIES", 3),
			ConnectTimeout: getEnvAsDuration("MESSAGE_BROKER_REDIS_CONNECT_TIMEOUT", 5*time.Second),
			ReadTimeout:    getEnvAsDuration("MESSAGE_BROKER_REDIS_READ_TIMEOUT", 3*time.Second),
			WriteTimeout:   getEnvAsDuration("MESSAGE_BROKER_REDIS_WRITE_TIMEOUT", 3*time.Second),
			IdleTimeout:    getEnvAsDuration("MESSAGE_BROKER_REDIS_IDLE_TIMEOUT", 300*time.Second),
		}

		// TLS configuration for Redis
		if getEnvAsBool("MESSAGE_BROKER_REDIS_TLS_ENABLE", false) {
			config.MessageBroker.Redis.TLS = &TLSConfig{
				Enable:             true,
				CertFile:           getEnv("MESSAGE_BROKER_REDIS_TLS_CERT_FILE", ""),
				KeyFile:            getEnv("MESSAGE_BROKER_REDIS_TLS_KEY_FILE", ""),
				CAFile:             getEnv("MESSAGE_BROKER_REDIS_TLS_CA_FILE", ""),
				InsecureSkipVerify: getEnvAsBool("MESSAGE_BROKER_REDIS_TLS_INSECURE_SKIP_VERIFY", false),
			}
		}
	}

	// Retry configuration
	config.MessageBroker.Retry = &RetryConfig{
		MaxRetries:      getEnvAsInt("MESSAGE_BROKER_MAX_RETRIES", 3),
		InitialInterval: getEnvAsDuration("MESSAGE_BROKER_RETRY_INITIAL_INTERVAL", 1*time.Second),
		MaxInterval:     getEnvAsDuration("MESSAGE_BROKER_RETRY_MAX_INTERVAL", 30*time.Second),
		Multiplier:      getEnvAsFloat64("MESSAGE_BROKER_RETRY_MULTIPLIER", 2.0),
		RandomFactor:    getEnvAsFloat64("MESSAGE_BROKER_RETRY_RANDOM_FACTOR", 0.1),
	}

	// Load Logging configuration
	config.Logging = LoggingConfig{
		Level:       getEnv("LOG_LEVEL", "info"),
		Format:      getEnv("LOG_FORMAT", "json"),
		Output:      getEnv("LOG_OUTPUT", "stdout"),
		FilePath:    getEnv("LOG_FILE_PATH", "./logs/app.log"),
		MaxSizeMB:   getEnvAsInt("LOG_MAX_SIZE_MB", 100),
		MaxBackups:  getEnvAsInt("LOG_MAX_BACKUPS", 3),
		MaxAgeDays:  getEnvAsInt("LOG_MAX_AGE_DAYS", 28),
		Compress:    getEnvAsBool("LOG_COMPRESS", true),
		LogRequests: getEnvAsBool("LOG_REQUESTS", true),
		LogHeaders:  getEnvAsBool("LOG_HEADERS", false),
		LogBody:     getEnvAsBool("LOG_BODY", false),
	}

	// Load Monitoring configuration
	config.Monitoring = MonitoringConfig{
		Enable:   getEnvAsBool("MONITORING_ENABLED", true),
		Provider: getEnv("MONITORING_PROVIDER", "prometheus"),
		Prometheus: PrometheusConfig{
			Namespace:   getEnv("MONITORING_NAMESPACE", strings.ToLower(strings.ReplaceAll(config.App.Name, " ", "_"))),
			MetricsPath: getEnv("MONITORING_METRICS_PATH", "/metrics"),
		},
	}

	// Load ELK configuration
	config.ELK = ELKConfig{
		Enabled:     getEnvAsBool("ELK_ENABLED", false),
		URLs:        getEnvAsStringSlice("ELK_URLS", "http://localhost:9200"),
		Username:    getEnv("ELK_USERNAME", ""),
		Password:    getEnv("ELK_PASSWORD", ""),
		IndexPrefix: getEnv("ELK_INDEX_PREFIX", strings.ToLower(strings.ReplaceAll(config.App.Name, " ", "-"))),
		APIKey:      getEnv("ELK_API_KEY", ""),
		MaxRetries:  getEnvAsInt("ELK_MAX_RETRIES", 3),
		Compress:    getEnvAsBool("ELK_COMPRESS", true),
		BatchSize:   getEnvAsInt("ELK_BATCH_SIZE", 100),
		BatchWait:   getEnv("ELK_BATCH_WAIT", "5s"),
	}

	// Load gRPC configuration
	config.GRPC = GRPCConfig{
		Enabled:               getEnvAsBool("GRPC_ENABLED", true),
		Port:                  getEnv("GRPC_PORT", "9000"),
		MaxReceiveSize:        getEnvAsInt("GRPC_MAX_RECEIVE_SIZE", 4*1024*1024), // 4MB
		MaxSendSize:           getEnvAsInt("GRPC_MAX_SEND_SIZE", 4*1024*1024),    // 4MB
		MaxConnectionIdle:     getEnvAsDuration("GRPC_MAX_CONNECTION_IDLE", 15*time.Minute),
		MaxConnectionAge:      getEnvAsDuration("GRPC_MAX_CONNECTION_AGE", 30*time.Minute),
		MaxConnectionAgeGrace: getEnvAsDuration("GRPC_MAX_CONNECTION_AGE_GRACE", 5*time.Minute),
		KeepAliveTime:         getEnvAsDuration("GRPC_KEEP_ALIVE_TIME", 30*time.Second),
		KeepAliveTimeout:      getEnvAsDuration("GRPC_KEEP_ALIVE_TIMEOUT", 5*time.Second),
		Reflection:            getEnvAsBool("GRPC_REFLECTION", true),
		Gateway: GRPCGatewayConfig{
			Enabled: getEnvAsBool("GRPC_GATEWAY_ENABLED", true),
			Port:    getEnv("GRPC_GATEWAY_PORT", "8080"),
			Prefix:  getEnv("GRPC_GATEWAY_PREFIX", "/api"),
		},
	}

	// Load gRPC TLS configuration
	if getEnvAsBool("GRPC_TLS_ENABLED", false) {
		config.GRPC.TLS = &GRPCTLSConfig{
			Enable:   true,
			CertFile: getEnv("GRPC_TLS_CERT_FILE", "./certs/server.crt"),
			KeyFile:  getEnv("GRPC_TLS_KEY_FILE", "./certs/server.key"),
		}
	}

	// Validate configuration
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

func validateConfig(config *Config) error {
	// Validate required fields
	if config.Auth.JWT.Secret == "your-secret-key" {
		return fmt.Errorf("JWT_SECRET must be changed from default value")
	}

	if len(config.Auth.JWT.Secret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters long")
	}

	if config.Database.Password == "password" && config.Server.Mode == "production" {
		return fmt.Errorf("database password must be changed from default value in production")
	}

	if config.Server.Mode == "production" && !config.Security.Headers.Enable {
		return fmt.Errorf("security headers must be enabled in production")
	}

	return nil
}

// Helper functions for environment variable parsing
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvAsInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		// Try to parse as seconds first
		if seconds, err := strconv.Atoi(value); err == nil {
			return time.Duration(seconds) * time.Second
		}
		// Try to parse as duration string
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getEnvAsStringSlice(key, defaultValue string) []string {
	value := getEnv(key, defaultValue)
	if value == "" {
		return []string{}
	}
	return strings.Split(value, ",")
}

func getEnvAsFloat64(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
			return floatVal
		}
	}
	return defaultValue
}
