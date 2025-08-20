package messagebroker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// MessageBroker defines the interface for message brokers (similar to Laravel's Queue interface)
type MessageBroker interface {
	// Publishing messages
	Publish(ctx context.Context, topic string, message *Message) error
	PublishJSON(ctx context.Context, topic string, data interface{}) error
	PublishWithDelay(ctx context.Context, topic string, message *Message, delay time.Duration) error
	
	// Subscribing and consuming
	Subscribe(ctx context.Context, topic string, handler MessageHandler) error
	SubscribeWithGroup(ctx context.Context, topic string, group string, handler MessageHandler) error
	
	// Queue operations (for job-like behavior)
	EnqueueJob(ctx context.Context, queue string, job *Job) error
	ProcessJobs(ctx context.Context, queue string, handler JobHandler) error
	
	// Management operations
	CreateTopic(ctx context.Context, topic string, config *TopicConfig) error
	DeleteTopic(ctx context.Context, topic string) error
	GetTopicInfo(ctx context.Context, topic string) (*TopicInfo, error)
	
	// Health and status
	Ping(ctx context.Context) error
	Close() error
	GetStats() (*BrokerStats, error)
}

// Message represents a message to be published/consumed
type Message struct {
	ID          string                 `json:"id"`
	Topic       string                 `json:"topic"`
	Payload     []byte                 `json:"payload"`
	Headers     map[string]string      `json:"headers,omitempty"`
	Timestamp   time.Time             `json:"timestamp"`
	RetryCount  int                   `json:"retry_count"`
	MaxRetries  int                   `json:"max_retries"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Job represents a job/task to be processed
type Job struct {
	ID          string                 `json:"id"`
	Queue       string                 `json:"queue"`
	Handler     string                 `json:"handler"`
	Payload     []byte                 `json:"payload"`
	Priority    int                   `json:"priority"`
	Delay       time.Duration         `json:"delay"`
	Attempts    int                   `json:"attempts"`
	MaxAttempts int                   `json:"max_attempts"`
	CreatedAt   time.Time             `json:"created_at"`
	ProcessedAt *time.Time            `json:"processed_at,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// MessageHandler is called when a message is received
type MessageHandler func(ctx context.Context, msg *Message) error

// JobHandler is called when a job is processed
type JobHandler func(ctx context.Context, job *Job) error

// TopicConfig holds configuration for creating topics
type TopicConfig struct {
	Partitions        int           `json:"partitions"`
	ReplicationFactor int           `json:"replication_factor"`
	RetentionTime     time.Duration `json:"retention_time"`
	CleanupPolicy     string        `json:"cleanup_policy"` // compact, delete
}

// TopicInfo contains information about a topic
type TopicInfo struct {
	Name              string    `json:"name"`
	Partitions        int       `json:"partitions"`
	ReplicationFactor int       `json:"replication_factor"`
	MessageCount      int64     `json:"message_count"`
	Size              int64     `json:"size"`
	CreatedAt         time.Time `json:"created_at"`
}

// BrokerStats contains statistics about the broker
type BrokerStats struct {
	MessagesPublished int64             `json:"messages_published"`
	MessagesConsumed  int64             `json:"messages_consumed"`
	JobsEnqueued      int64             `json:"jobs_enqueued"`
	JobsProcessed     int64             `json:"jobs_processed"`
	ActiveConnections int               `json:"active_connections"`
	TopicCount        int               `json:"topic_count"`
	QueueCount        int               `json:"queue_count"`
	Uptime            time.Duration     `json:"uptime"`
	DriverInfo        map[string]string `json:"driver_info"`
}

// MessageBrokerConfig holds configuration for different brokers
type MessageBrokerConfig struct {
	Driver      string              `json:"driver" mapstructure:"driver"`
	RabbitMQ    *RabbitMQConfig     `json:"rabbitmq,omitempty" mapstructure:"rabbitmq"`
	Kafka       *KafkaConfig        `json:"kafka,omitempty" mapstructure:"kafka"`
	Redis       *RedisPubSubConfig  `json:"redis,omitempty" mapstructure:"redis"`
	RetryConfig *RetryConfig        `json:"retry,omitempty" mapstructure:"retry"`
}

// RabbitMQConfig holds RabbitMQ-specific configuration
type RabbitMQConfig struct {
	URL                string        `json:"url" mapstructure:"url"`
	Host               string        `json:"host" mapstructure:"host"`
	Port               int           `json:"port" mapstructure:"port"`
	Username           string        `json:"username" mapstructure:"username"`
	Password           string        `json:"password" mapstructure:"password"`
	VHost              string        `json:"vhost" mapstructure:"vhost"`
	Exchange           string        `json:"exchange" mapstructure:"exchange"`
	ExchangeType       string        `json:"exchange_type" mapstructure:"exchange_type"`
	ConnectionTimeout  time.Duration `json:"connection_timeout" mapstructure:"connection_timeout"`
	HeartbeatInterval  time.Duration `json:"heartbeat_interval" mapstructure:"heartbeat_interval"`
	PrefetchCount      int           `json:"prefetch_count" mapstructure:"prefetch_count"`
	Durable            bool          `json:"durable" mapstructure:"durable"`
	AutoDelete         bool          `json:"auto_delete" mapstructure:"auto_delete"`
}

// KafkaConfig holds Kafka-specific configuration
type KafkaConfig struct {
	Brokers               []string      `json:"brokers" mapstructure:"brokers"`
	GroupID               string        `json:"group_id" mapstructure:"group_id"`
	ClientID              string        `json:"client_id" mapstructure:"client_id"`
	Version               string        `json:"version" mapstructure:"version"`
	ConnectTimeout        time.Duration `json:"connect_timeout" mapstructure:"connect_timeout"`
	SessionTimeout        time.Duration `json:"session_timeout" mapstructure:"session_timeout"`
	HeartbeatInterval     time.Duration `json:"heartbeat_interval" mapstructure:"heartbeat_interval"`
	RebalanceTimeout      time.Duration `json:"rebalance_timeout" mapstructure:"rebalance_timeout"`
	ReturnSuccesses       bool          `json:"return_successes" mapstructure:"return_successes"`
	RequiredAcks          int           `json:"required_acks" mapstructure:"required_acks"`
	CompressionType       string        `json:"compression" mapstructure:"compression"`
	FlushFrequency        time.Duration `json:"flush_frequency" mapstructure:"flush_frequency"`
	EnableAutoCommit      bool          `json:"enable_auto_commit" mapstructure:"enable_auto_commit"`
	AutoCommitInterval    time.Duration `json:"auto_commit_interval" mapstructure:"auto_commit_interval"`
	InitialOffset         string        `json:"initial_offset" mapstructure:"initial_offset"` // oldest, newest
	SASL                  *SASLConfig   `json:"sasl,omitempty" mapstructure:"sasl"`
	TLS                   *TLSConfig    `json:"tls,omitempty" mapstructure:"tls"`
}

// RedisPubSubConfig holds Redis Pub/Sub configuration
type RedisPubSubConfig struct {
	Host               string        `json:"host" mapstructure:"host"`
	Port               int           `json:"port" mapstructure:"port"`
	Password           string        `json:"password" mapstructure:"password"`
	DB                 int           `json:"db" mapstructure:"db"`
	PoolSize           int           `json:"pool_size" mapstructure:"pool_size"`
	MinIdleConns       int           `json:"min_idle_conns" mapstructure:"min_idle_conns"`
	MaxRetries         int           `json:"max_retries" mapstructure:"max_retries"`
	ConnectTimeout     time.Duration `json:"connect_timeout" mapstructure:"connect_timeout"`
	ReadTimeout        time.Duration `json:"read_timeout" mapstructure:"read_timeout"`
	WriteTimeout       time.Duration `json:"write_timeout" mapstructure:"write_timeout"`
	IdleTimeout        time.Duration `json:"idle_timeout" mapstructure:"idle_timeout"`
	TLS                *TLSConfig    `json:"tls,omitempty" mapstructure:"tls"`
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
	Mechanism string `json:"mechanism" mapstructure:"mechanism"` // PLAIN, SCRAM-SHA-256, SCRAM-SHA-512
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

// Helper functions for creating messages and jobs
func NewMessage(topic string, payload interface{}) (*Message, error) {
	var data []byte
	var err error

	switch v := payload.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		data, err = json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal payload: %w", err)
		}
	}

	return &Message{
		ID:         uuid.New().String(),
		Topic:      topic,
		Payload:    data,
		Headers:    make(map[string]string),
		Timestamp:  time.Now(),
		RetryCount: 0,
		MaxRetries: 3,
		Metadata:   make(map[string]interface{}),
	}, nil
}

func NewJob(queue, handler string, payload interface{}) (*Job, error) {
	var data []byte
	var err error

	switch v := payload.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		data, err = json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal payload: %w", err)
		}
	}

	return &Job{
		ID:          uuid.New().String(),
		Queue:       queue,
		Handler:     handler,
		Payload:     data,
		Priority:    0,
		Delay:       0,
		Attempts:    0,
		MaxAttempts: 3,
		CreatedAt:   time.Now(),
		Metadata:    make(map[string]interface{}),
	}, nil
}

// WithHeaders adds headers to a message
func (m *Message) WithHeaders(headers map[string]string) *Message {
	if m.Headers == nil {
		m.Headers = make(map[string]string)
	}
	for k, v := range headers {
		m.Headers[k] = v
	}
	return m
}

// WithMetadata adds metadata to a message
func (m *Message) WithMetadata(metadata map[string]interface{}) *Message {
	if m.Metadata == nil {
		m.Metadata = make(map[string]interface{})
	}
	for k, v := range metadata {
		m.Metadata[k] = v
	}
	return m
}

// WithPriority sets job priority
func (j *Job) WithPriority(priority int) *Job {
	j.Priority = priority
	return j
}

// WithDelay adds delay to job execution
func (j *Job) WithDelay(delay time.Duration) *Job {
	j.Delay = delay
	return j
}

// UnmarshalPayload unmarshals the message payload into the provided interface
func (m *Message) UnmarshalPayload(v interface{}) error {
	return json.Unmarshal(m.Payload, v)
}

// UnmarshalPayload unmarshals the job payload into the provided interface
func (j *Job) UnmarshalPayload(v interface{}) error {
	return json.Unmarshal(j.Payload, v)
}

// GetPayloadString returns the payload as a string
func (m *Message) GetPayloadString() string {
	return string(m.Payload)
}

// GetPayloadString returns the payload as a string
func (j *Job) GetPayloadString() string {
	return string(j.Payload)
}

// Error types
type MessageBrokerError struct {
	Driver  string
	Op      string
	Message string
	Err     error
}

func (e *MessageBrokerError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("messagebroker: %s driver failed on %s: %s (%v)", e.Driver, e.Op, e.Message, e.Err)
	}
	return fmt.Sprintf("messagebroker: %s driver failed on %s: %s", e.Driver, e.Op, e.Message)
}

func (e *MessageBrokerError) Unwrap() error {
	return e.Err
}

// Common error variables
var (
	ErrDriverNotSupported   = fmt.Errorf("message broker driver not supported")
	ErrConnectionFailed     = fmt.Errorf("failed to connect to message broker")
	ErrTopicNotFound        = fmt.Errorf("topic not found")
	ErrQueueNotFound        = fmt.Errorf("queue not found")
	ErrInvalidConfiguration = fmt.Errorf("invalid configuration")
	ErrMessageTooLarge      = fmt.Errorf("message too large")
	ErrMaxRetriesExceeded   = fmt.Errorf("maximum retries exceeded")
)