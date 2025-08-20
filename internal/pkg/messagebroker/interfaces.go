package messagebroker

import (
	"context"
	"time"
)

// MessageBroker defines the interface for message broker operations
type MessageBroker interface {
	// Producer methods
	Publish(ctx context.Context, topic string, message Message) error
	PublishBatch(ctx context.Context, topic string, messages []Message) error

	// Consumer methods
	Subscribe(ctx context.Context, topic string, handler MessageHandler) error
	Unsubscribe(topic string) error

	// Health and status
	Ping(ctx context.Context) error
	GetStats() *BrokerStats
	Close() error

	// Topic management
	CreateTopic(ctx context.Context, topic string, config TopicConfig) error
	DeleteTopic(ctx context.Context, topic string) error
	ListTopics(ctx context.Context) ([]string, error)
}

// MessageHandler is the function signature for message handlers
type MessageHandler func(ctx context.Context, message Message) error

// Message represents a message in the broker
type Message struct {
	ID        string                 `json:"id"`
	Topic     string                 `json:"topic"`
	Key       string                 `json:"key,omitempty"`
	Value     []byte                 `json:"value"`
	Headers   map[string]string      `json:"headers,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Partition int32                  `json:"partition,omitempty"`
	Offset    int64                  `json:"offset,omitempty"`
}

// BrokerStats contains statistics about the message broker
type BrokerStats struct {
	MessagesProduced    int64     `json:"messages_produced"`
	MessagesConsumed    int64     `json:"messages_consumed"`
	MessagesFailed      int64     `json:"messages_failed"`
	ActiveConsumers     int       `json:"active_consumers"`
	ActiveTopics        int       `json:"active_topics"`
	ConnectionStatus    string    `json:"connection_status"`
	LastActivity        time.Time `json:"last_activity"`
	Uptime             time.Duration `json:"uptime"`
	BytesProduced      int64     `json:"bytes_produced"`
	BytesConsumed      int64     `json:"bytes_consumed"`
}

// TopicConfig defines configuration for creating topics
type TopicConfig struct {
	Partitions        int               `json:"partitions"`
	ReplicationFactor int               `json:"replication_factor"`
	Config            map[string]string `json:"config,omitempty"`
}

// RetryPolicy defines retry behavior for failed messages
type RetryPolicy struct {
	MaxRetries      int           `json:"max_retries"`
	InitialInterval time.Duration `json:"initial_interval"`
	MaxInterval     time.Duration `json:"max_interval"`
	Multiplier      float64       `json:"multiplier"`
	RandomFactor    float64       `json:"random_factor"`
}