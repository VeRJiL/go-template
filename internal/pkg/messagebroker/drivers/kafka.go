package drivers

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/IBM/sarama"

	"github.com/VeRJiL/go-template/internal/config"
)

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
	MessagesProduced int64         `json:"messages_produced"`
	MessagesConsumed int64         `json:"messages_consumed"`
	MessagesFailed   int64         `json:"messages_failed"`
	ActiveConsumers  int           `json:"active_consumers"`
	ActiveTopics     int           `json:"active_topics"`
	ConnectionStatus string        `json:"connection_status"`
	LastActivity     time.Time     `json:"last_activity"`
	Uptime           time.Duration `json:"uptime"`
	BytesProduced    int64         `json:"bytes_produced"`
	BytesConsumed    int64         `json:"bytes_consumed"`
}

// TopicConfig defines configuration for creating topics
type TopicConfig struct {
	Partitions        int               `json:"partitions"`
	ReplicationFactor int               `json:"replication_factor"`
	Config            map[string]string `json:"config,omitempty"`
}

// MessageHandler is the function signature for message handlers
type MessageHandler func(ctx context.Context, message Message) error

// KafkaDriver implements MessageBroker interface for Apache Kafka
type KafkaDriver struct {
	config        *config.KafkaConfig
	client        sarama.Client
	producer      sarama.SyncProducer
	consumerGroup sarama.ConsumerGroup
	consumers     map[string]*kafkaConsumer
	mu            sync.RWMutex
	closed        bool
	stats         *BrokerStats
	startTime     time.Time
	topics        map[string]bool
}

// kafkaConsumer wraps Sarama consumer with our handler
type kafkaConsumer struct {
	handler messagebroker.MessageHandler
	ready   chan bool
}

// NewKafkaDriver creates a new Kafka driver instance
func NewKafkaDriver(config *messagebroker.KafkaConfig) (*KafkaDriver, error) {
	if config == nil {
		return nil, fmt.Errorf("Kafka config cannot be nil")
	}

	// Set default values
	if config.ClientID == "" {
		config.ClientID = "go-template-kafka-client"
	}
	if config.GroupID == "" {
		config.GroupID = "go-template-consumer-group"
	}
	if config.Version == "" {
		config.Version = "2.6.0"
	}

	driver := &KafkaDriver{
		config:    config,
		startTime: time.Now(),
		consumers: make(map[string]*kafkaConsumer),
		topics:    make(map[string]bool),
		stats: &messagebroker.BrokerStats{
			DriverInfo: map[string]string{
				"driver":   "kafka",
				"brokers":  strings.Join(config.Brokers, ","),
				"group_id": config.GroupID,
				"version":  config.Version,
			},
		},
	}

	if err := driver.connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to Kafka: %w", err)
	}

	return driver, nil
}

// connect establishes connection to Kafka
func (k *KafkaDriver) connect() error {
	saramaConfig := sarama.NewConfig()

	// Set version
	version, err := sarama.ParseKafkaVersion(k.config.Version)
	if err != nil {
		return fmt.Errorf("invalid Kafka version %s: %w", k.config.Version, err)
	}
	saramaConfig.Version = version

	// Client ID
	saramaConfig.ClientID = k.config.ClientID

	// Producer configuration
	saramaConfig.Producer.Return.Successes = k.config.ReturnSuccesses
	saramaConfig.Producer.RequiredAcks = sarama.RequiredAcks(k.config.RequiredAcks)
	saramaConfig.Producer.Retry.Max = 3
	saramaConfig.Producer.Flush.Frequency = k.config.FlushFrequency

	// Compression
	switch strings.ToLower(k.config.CompressionType) {
	case "gzip":
		saramaConfig.Producer.Compression = sarama.CompressionGZIP
	case "snappy":
		saramaConfig.Producer.Compression = sarama.CompressionSnappy
	case "lz4":
		saramaConfig.Producer.Compression = sarama.CompressionLZ4
	case "zstd":
		saramaConfig.Producer.Compression = sarama.CompressionZSTD
	default:
		saramaConfig.Producer.Compression = sarama.CompressionNone
	}

	// Consumer configuration
	if k.config.InitialOffset == "oldest" {
		saramaConfig.Consumer.Offsets.Initial = sarama.OffsetOldest
	} else {
		saramaConfig.Consumer.Offsets.Initial = sarama.OffsetNewest
	}

	saramaConfig.Consumer.Group.Session.Timeout = k.config.SessionTimeout
	saramaConfig.Consumer.Group.Heartbeat.Interval = k.config.HeartbeatInterval
	saramaConfig.Consumer.Group.Rebalance.Timeout = k.config.RebalanceTimeout

	// Auto-commit configuration
	if k.config.EnableAutoCommit {
		saramaConfig.Consumer.Offsets.AutoCommit.Enable = true
		saramaConfig.Consumer.Offsets.AutoCommit.Interval = k.config.AutoCommitInterval
	} else {
		saramaConfig.Consumer.Offsets.AutoCommit.Enable = false
	}

	// Timeouts
	saramaConfig.Net.DialTimeout = k.config.ConnectTimeout
	saramaConfig.Net.ReadTimeout = 10 * time.Second
	saramaConfig.Net.WriteTimeout = 10 * time.Second

	// SASL configuration
	if k.config.SASL != nil && k.config.SASL.Enable {
		saramaConfig.Net.SASL.Enable = true
		saramaConfig.Net.SASL.User = k.config.SASL.Username
		saramaConfig.Net.SASL.Password = k.config.SASL.Password

		switch k.config.SASL.Mechanism {
		case "SCRAM-SHA-256":
			saramaConfig.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA256
		case "SCRAM-SHA-512":
			saramaConfig.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA512
		default:
			saramaConfig.Net.SASL.Mechanism = sarama.SASLTypePlaintext
		}
	}

	// TLS configuration
	if k.config.TLS != nil && k.config.TLS.Enable {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: k.config.TLS.InsecureSkipVerify,
		}

		if k.config.TLS.CertFile != "" && k.config.TLS.KeyFile != "" {
			cert, err := tls.LoadX509KeyPair(k.config.TLS.CertFile, k.config.TLS.KeyFile)
			if err != nil {
				return fmt.Errorf("failed to load client certificate: %w", err)
			}
			tlsConfig.Certificates = []tls.Certificate{cert}
		}

		saramaConfig.Net.TLS.Enable = true
		saramaConfig.Net.TLS.Config = tlsConfig
	}

	// Create client
	client, err := sarama.NewClient(k.config.Brokers, saramaConfig)
	if err != nil {
		return fmt.Errorf("failed to create Kafka client: %w", err)
	}

	// Create producer
	producer, err := sarama.NewSyncProducerFromClient(client)
	if err != nil {
		client.Close()
		return fmt.Errorf("failed to create Kafka producer: %w", err)
	}

	// Create consumer group
	consumerGroup, err := sarama.NewConsumerGroupFromClient(k.config.GroupID, client)
	if err != nil {
		producer.Close()
		client.Close()
		return fmt.Errorf("failed to create Kafka consumer group: %w", err)
	}

	k.client = client
	k.producer = producer
	k.consumerGroup = consumerGroup

	k.stats.ActiveConnections = 1
	return nil
}

// Publish publishes a message to a topic
func (k *KafkaDriver) Publish(ctx context.Context, topic string, message *messagebroker.Message) error {
	k.mu.RLock()
	defer k.mu.RUnlock()

	if k.closed {
		return fmt.Errorf("Kafka driver is closed")
	}

	// Create Kafka headers
	headers := make([]sarama.RecordHeader, 0)
	for key, value := range message.Headers {
		headers = append(headers, sarama.RecordHeader{
			Key:   []byte(key),
			Value: []byte(value),
		})
	}

	// Add metadata to headers
	for key, value := range message.Metadata {
		if strVal, ok := value.(string); ok {
			headers = append(headers, sarama.RecordHeader{
				Key:   []byte(fmt.Sprintf("meta_%s", key)),
				Value: []byte(strVal),
			})
		}
	}

	// Add message info to headers
	headers = append(headers,
		sarama.RecordHeader{Key: []byte("message_id"), Value: []byte(message.ID)},
		sarama.RecordHeader{Key: []byte("retry_count"), Value: []byte(fmt.Sprintf("%d", message.RetryCount))},
		sarama.RecordHeader{Key: []byte("max_retries"), Value: []byte(fmt.Sprintf("%d", message.MaxRetries))},
		sarama.RecordHeader{Key: []byte("timestamp"), Value: []byte(fmt.Sprintf("%d", message.Timestamp.Unix()))},
	)

	kafkaMessage := &sarama.ProducerMessage{
		Topic:     topic,
		Key:       sarama.StringEncoder(message.ID),
		Value:     sarama.ByteEncoder(message.Payload),
		Headers:   headers,
		Timestamp: message.Timestamp,
	}

	partition, offset, err := k.producer.SendMessage(kafkaMessage)
	if err != nil {
		return &messagebroker.MessageBrokerError{
			Driver:  "kafka",
			Op:      "publish",
			Message: fmt.Sprintf("failed to publish message to topic %s", topic),
			Err:     err,
		}
	}

	log.Printf("Message published to topic %s, partition %d, offset %d", topic, partition, offset)

	k.mu.Lock()
	k.stats.MessagesPublished++
	k.mu.Unlock()

	return nil
}

// PublishJSON publishes JSON data to a topic
func (k *KafkaDriver) PublishJSON(ctx context.Context, topic string, data interface{}) error {
	message, err := messagebroker.NewMessage(topic, data)
	if err != nil {
		return fmt.Errorf("failed to create message: %w", err)
	}
	return k.Publish(ctx, topic, message)
}

// PublishWithDelay publishes a message with a delay (simulated with metadata)
func (k *KafkaDriver) PublishWithDelay(ctx context.Context, topic string, message *messagebroker.Message, delay time.Duration) error {
	// Kafka doesn't have native delayed message support
	// We'll add delay info to metadata and let the consumer handle it
	if message.Metadata == nil {
		message.Metadata = make(map[string]interface{})
	}

	message.Metadata["delayed_until"] = time.Now().Add(delay).Unix()
	message.Metadata["original_delay"] = delay.String()

	// Use a delayed topic pattern
	delayedTopic := fmt.Sprintf("%s.delayed", topic)
	return k.Publish(ctx, delayedTopic, message)
}

// Subscribe subscribes to a topic with a message handler
func (k *KafkaDriver) Subscribe(ctx context.Context, topic string, handler messagebroker.MessageHandler) error {
	return k.SubscribeWithGroup(ctx, topic, k.config.GroupID, handler)
}

// SubscribeWithGroup subscribes to a topic with a specific consumer group
func (k *KafkaDriver) SubscribeWithGroup(ctx context.Context, topic string, group string, handler messagebroker.MessageHandler) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	if k.closed {
		return fmt.Errorf("Kafka driver is closed")
	}

	consumerKey := fmt.Sprintf("%s:%s", group, topic)
	if _, exists := k.consumers[consumerKey]; exists {
		return fmt.Errorf("consumer already exists for topic %s and group %s", topic, group)
	}

	consumer := &kafkaConsumer{
		handler: handler,
		ready:   make(chan bool),
	}

	k.consumers[consumerKey] = consumer

	// Start consuming in a goroutine
	go func() {
		defer func() {
			k.mu.Lock()
			delete(k.consumers, consumerKey)
			k.mu.Unlock()
		}()

		for {
			select {
			case <-ctx.Done():
				return
			default:
				// Create a new consumer group for this subscription if group is different
				var cg sarama.ConsumerGroup
				var err error

				if group != k.config.GroupID {
					cg, err = sarama.NewConsumerGroupFromClient(group, k.client)
					if err != nil {
						log.Printf("Failed to create consumer group %s: %v", group, err)
						return
					}
					defer cg.Close()
				} else {
					cg = k.consumerGroup
				}

				err = cg.Consume(ctx, []string{topic}, consumer)
				if err != nil {
					log.Printf("Error consuming from topic %s: %v", topic, err)
					time.Sleep(time.Second)
					continue
				}
			}
		}
	}()

	// Wait for consumer to be ready
	<-consumer.ready

	return nil
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (c *kafkaConsumer) Setup(sarama.ConsumerGroupSession) error {
	close(c.ready)
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (c *kafkaConsumer) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages()
func (c *kafkaConsumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case message := <-claim.Messages():
			if message == nil {
				return nil
			}

			// Convert Kafka message to our message format
			msg := &messagebroker.Message{
				Topic:     message.Topic,
				Payload:   message.Value,
				Headers:   make(map[string]string),
				Timestamp: message.Timestamp,
				Metadata:  make(map[string]interface{}),
			}

			// Extract headers
			for _, header := range message.Headers {
				key := string(header.Key)
				value := string(header.Value)

				if strings.HasPrefix(key, "meta_") {
					msg.Metadata[strings.TrimPrefix(key, "meta_")] = value
				} else {
					switch key {
					case "message_id":
						msg.ID = value
					case "retry_count":
						if count := parseInt(value); count >= 0 {
							msg.RetryCount = count
						}
					case "max_retries":
						if max := parseInt(value); max >= 0 {
							msg.MaxRetries = max
						}
					default:
						msg.Headers[key] = value
					}
				}
			}

			// Check if message is delayed
			if delayedUntil, exists := msg.Metadata["delayed_until"]; exists {
				if timestamp, ok := delayedUntil.(string); ok {
					if delayTime := parseInt64(timestamp); delayTime > 0 {
						if time.Now().Unix() < delayTime {
							// Message is still delayed, skip for now
							// In a production system, you'd want to requeue this properly
							continue
						}
					}
				}
			}

			// Handle the message
			ctx := context.Background()
			if err := c.handler(ctx, msg); err != nil {
				log.Printf("Error handling message: %v", err)
				// Handle retry logic here if needed
			}

			// Mark message as processed
			session.MarkMessage(message, "")

		case <-session.Context().Done():
			return nil
		}
	}
}

// EnqueueJob enqueues a job for processing
func (k *KafkaDriver) EnqueueJob(ctx context.Context, queue string, job *messagebroker.Job) error {
	// Convert job to message
	jobData, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	message := &messagebroker.Message{
		ID:        job.ID,
		Topic:     queue,
		Payload:   jobData,
		Headers:   map[string]string{"job_handler": job.Handler},
		Timestamp: time.Now(),
		Metadata:  map[string]interface{}{"job_priority": fmt.Sprintf("%d", job.Priority)},
	}

	if job.Delay > 0 {
		return k.PublishWithDelay(ctx, queue, message, job.Delay)
	}

	err = k.Publish(ctx, queue, message)
	if err == nil {
		k.mu.Lock()
		k.stats.JobsEnqueued++
		k.mu.Unlock()
	}
	return err
}

// ProcessJobs processes jobs from a queue
func (k *KafkaDriver) ProcessJobs(ctx context.Context, queue string, handler messagebroker.JobHandler) error {
	return k.SubscribeWithGroup(ctx, queue, k.config.GroupID, func(ctx context.Context, msg *messagebroker.Message) error {
		var job messagebroker.Job
		if err := json.Unmarshal(msg.Payload, &job); err != nil {
			return fmt.Errorf("failed to unmarshal job: %w", err)
		}

		job.Attempts++
		now := time.Now()
		job.ProcessedAt = &now

		err := handler(ctx, &job)
		if err == nil {
			k.mu.Lock()
			k.stats.JobsProcessed++
			k.mu.Unlock()
		}
		return err
	})
}

// CreateTopic creates a topic
func (k *KafkaDriver) CreateTopic(ctx context.Context, topic string, config *messagebroker.TopicConfig) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	if k.closed {
		return fmt.Errorf("Kafka driver is closed")
	}

	// Check if topic already exists
	topics, err := k.client.Topics()
	if err != nil {
		return fmt.Errorf("failed to list topics: %w", err)
	}

	for _, existingTopic := range topics {
		if existingTopic == topic {
			return nil // Topic already exists
		}
	}

	// Create topic using cluster admin
	clusterAdmin, err := sarama.NewClusterAdminFromClient(k.client)
	if err != nil {
		return fmt.Errorf("failed to create cluster admin: %w", err)
	}
	defer clusterAdmin.Close()

	topicDetail := sarama.TopicDetail{
		NumPartitions:     int32(config.Partitions),
		ReplicationFactor: int16(config.ReplicationFactor),
		ConfigEntries:     make(map[string]*string),
	}

	// Add retention configuration
	if config.RetentionTime > 0 {
		retentionMs := fmt.Sprintf("%d", config.RetentionTime.Milliseconds())
		topicDetail.ConfigEntries["retention.ms"] = &retentionMs
	}

	if config.CleanupPolicy != "" {
		topicDetail.ConfigEntries["cleanup.policy"] = &config.CleanupPolicy
	}

	err = clusterAdmin.CreateTopic(topic, &topicDetail, false)
	if err != nil {
		return fmt.Errorf("failed to create topic %s: %w", topic, err)
	}

	k.topics[topic] = true
	k.stats.TopicCount++

	return nil
}

// DeleteTopic deletes a topic
func (k *KafkaDriver) DeleteTopic(ctx context.Context, topic string) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	if k.closed {
		return fmt.Errorf("Kafka driver is closed")
	}

	clusterAdmin, err := sarama.NewClusterAdminFromClient(k.client)
	if err != nil {
		return fmt.Errorf("failed to create cluster admin: %w", err)
	}
	defer clusterAdmin.Close()

	err = clusterAdmin.DeleteTopic(topic)
	if err != nil {
		return fmt.Errorf("failed to delete topic %s: %w", topic, err)
	}

	delete(k.topics, topic)
	k.stats.TopicCount--

	return nil
}

// GetTopicInfo returns information about a topic
func (k *KafkaDriver) GetTopicInfo(ctx context.Context, topic string) (*messagebroker.TopicInfo, error) {
	k.mu.RLock()
	defer k.mu.RUnlock()

	if k.closed {
		return nil, fmt.Errorf("Kafka driver is closed")
	}

	clusterAdmin, err := sarama.NewClusterAdminFromClient(k.client)
	if err != nil {
		return nil, fmt.Errorf("failed to create cluster admin: %w", err)
	}
	defer clusterAdmin.Close()

	metadata, err := clusterAdmin.DescribeTopics([]string{topic})
	if err != nil {
		return nil, fmt.Errorf("failed to describe topic %s: %w", topic, err)
	}

	topicMetadata, exists := metadata[topic]
	if !exists {
		return nil, messagebroker.ErrTopicNotFound
	}

	return &messagebroker.TopicInfo{
		Name:              topic,
		Partitions:        len(topicMetadata.Partitions),
		ReplicationFactor: 1,          // Would need to inspect partition metadata for exact value
		MessageCount:      0,          // Not easily available without consuming
		Size:              0,          // Not easily available
		CreatedAt:         time.Now(), // Not tracked by Kafka metadata
	}, nil
}

// Ping checks if the connection is alive
func (k *KafkaDriver) Ping(ctx context.Context) error {
	k.mu.RLock()
	defer k.mu.RUnlock()

	if k.closed || k.client == nil || k.client.Closed() {
		return fmt.Errorf("Kafka connection is not available")
	}

	// Try to get broker list as a health check
	brokers := k.client.Brokers()
	if len(brokers) == 0 {
		return fmt.Errorf("no Kafka brokers available")
	}

	return nil
}

// Close closes the Kafka connection
func (k *KafkaDriver) Close() error {
	k.mu.Lock()
	defer k.mu.Unlock()

	if k.closed {
		return nil
	}

	k.closed = true

	// Close all consumers
	for _, consumer := range k.consumers {
		close(consumer.ready)
	}

	// Close consumer group
	if k.consumerGroup != nil {
		k.consumerGroup.Close()
	}

	// Close producer
	if k.producer != nil {
		k.producer.Close()
	}

	// Close client
	if k.client != nil {
		k.client.Close()
	}

	k.stats.ActiveConnections = 0
	return nil
}

// GetStats returns broker statistics
func (k *KafkaDriver) GetStats() (*messagebroker.BrokerStats, error) {
	k.mu.RLock()
	defer k.mu.RUnlock()

	// Update uptime
	k.stats.Uptime = time.Since(k.startTime)
	k.stats.TopicCount = len(k.topics)

	// Create a copy to avoid race conditions
	statsCopy := *k.stats
	statsCopy.DriverInfo = make(map[string]string)
	for key, value := range k.stats.DriverInfo {
		statsCopy.DriverInfo[key] = value
	}

	return &statsCopy, nil
}

// Helper functions
func parseInt(s string) int {
	var result int
	fmt.Sscanf(s, "%d", &result)
	return result
}

func parseInt64(s string) int64 {
	var result int64
	fmt.Sscanf(s, "%d", &result)
	return result
}
