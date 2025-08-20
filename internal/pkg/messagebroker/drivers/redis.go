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

	"github.com/redis/go-redis/v9"

	"github.com/VeRJiL/go-template/internal/pkg/messagebroker"
)

// RedisPubSubDriver implements MessageBroker interface using Redis Pub/Sub
type RedisPubSubDriver struct {
	config      *messagebroker.RedisPubSubConfig
	client      *redis.Client
	pubsub      map[string]*redis.PubSub
	subscribers map[string]*redisSubscriber
	mu          sync.RWMutex
	closed      bool
	stats       *messagebroker.BrokerStats
	startTime   time.Time
	topics      map[string]bool
}

// redisSubscriber wraps Redis PubSub with our handler
type redisSubscriber struct {
	pubsub  *redis.PubSub
	handler messagebroker.MessageHandler
	topic   string
	group   string
	cancel  context.CancelFunc
}

// NewRedisPubSubDriver creates a new Redis Pub/Sub driver instance
func NewRedisPubSubDriver(config *messagebroker.RedisPubSubConfig) (*RedisPubSubDriver, error) {
	if config == nil {
		return nil, fmt.Errorf("Redis Pub/Sub config cannot be nil")
	}

	driver := &RedisPubSubDriver{
		config:      config,
		pubsub:      make(map[string]*redis.PubSub),
		subscribers: make(map[string]*redisSubscriber),
		startTime:   time.Now(),
		topics:      make(map[string]bool),
		stats: &messagebroker.BrokerStats{
			DriverInfo: map[string]string{
				"driver": "redis_pubsub",
				"host":   fmt.Sprintf("%s:%d", config.Host, config.Port),
				"db":     fmt.Sprintf("%d", config.DB),
			},
		},
	}

	if err := driver.connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return driver, nil
}

// connect establishes connection to Redis
func (r *RedisPubSubDriver) connect() error {
	options := &redis.Options{
		Addr:         fmt.Sprintf("%s:%d", r.config.Host, r.config.Port),
		Password:     r.config.Password,
		DB:           r.config.DB,
		PoolSize:     r.config.PoolSize,
		MinIdleConns: r.config.MinIdleConns,
		MaxRetries:   r.config.MaxRetries,
		DialTimeout:  r.config.ConnectTimeout,
		ReadTimeout:  r.config.ReadTimeout,
		WriteTimeout: r.config.WriteTimeout,
		IdleTimeout:  r.config.IdleTimeout,
	}

	// TLS configuration
	if r.config.TLS != nil && r.config.TLS.Enable {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: r.config.TLS.InsecureSkipVerify,
		}

		if r.config.TLS.CertFile != "" && r.config.TLS.KeyFile != "" {
			cert, err := tls.LoadX509KeyPair(r.config.TLS.CertFile, r.config.TLS.KeyFile)
			if err != nil {
				return fmt.Errorf("failed to load client certificate: %w", err)
			}
			tlsConfig.Certificates = []tls.Certificate{cert}
		}

		options.TLSConfig = tlsConfig
	}

	client := redis.NewClient(options)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to ping Redis: %w", err)
	}

	r.client = client
	r.stats.ActiveConnections = 1

	return nil
}

// Publish publishes a message to a topic
func (r *RedisPubSubDriver) Publish(ctx context.Context, topic string, message *messagebroker.Message) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.closed {
		return fmt.Errorf("Redis Pub/Sub driver is closed")
	}

	// Create Redis message with metadata
	redisMessage := map[string]interface{}{
		"id":          message.ID,
		"topic":       message.Topic,
		"payload":     string(message.Payload),
		"headers":     message.Headers,
		"timestamp":   message.Timestamp.Unix(),
		"retry_count": message.RetryCount,
		"max_retries": message.MaxRetries,
		"metadata":    message.Metadata,
	}

	// Serialize message
	data, err := json.Marshal(redisMessage)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Publish to Redis
	err = r.client.Publish(ctx, topic, data).Err()
	if err != nil {
		return &messagebroker.MessageBrokerError{
			Driver:  "redis_pubsub",
			Op:      "publish",
			Message: fmt.Sprintf("failed to publish message to topic %s", topic),
			Err:     err,
		}
	}

	r.mu.Lock()
	r.stats.MessagesPublished++
	r.topics[topic] = true
	r.mu.Unlock()

	return nil
}

// PublishJSON publishes JSON data to a topic
func (r *RedisPubSubDriver) PublishJSON(ctx context.Context, topic string, data interface{}) error {
	message, err := messagebroker.NewMessage(topic, data)
	if err != nil {
		return fmt.Errorf("failed to create message: %w", err)
	}
	return r.Publish(ctx, topic, message)
}

// PublishWithDelay publishes a message with a delay using Redis sorted sets
func (r *RedisPubSubDriver) PublishWithDelay(ctx context.Context, topic string, message *messagebroker.Message, delay time.Duration) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.closed {
		return fmt.Errorf("Redis Pub/Sub driver is closed")
	}

	// Use Redis sorted sets for delayed messages
	delayedKey := fmt.Sprintf("delayed:%s", topic)
	executeAt := time.Now().Add(delay)

	// Create delayed message payload
	delayedMessage := map[string]interface{}{
		"id":          message.ID,
		"topic":       topic,
		"payload":     string(message.Payload),
		"headers":     message.Headers,
		"timestamp":   message.Timestamp.Unix(),
		"retry_count": message.RetryCount,
		"max_retries": message.MaxRetries,
		"metadata":    message.Metadata,
		"execute_at":  executeAt.Unix(),
		"delay":       delay.String(),
	}

	data, err := json.Marshal(delayedMessage)
	if err != nil {
		return fmt.Errorf("failed to marshal delayed message: %w", err)
	}

	// Add to sorted set with execution time as score
	err = r.client.ZAdd(ctx, delayedKey, &redis.Z{
		Score:  float64(executeAt.Unix()),
		Member: data,
	}).Err()

	if err != nil {
		return &messagebroker.MessageBrokerError{
			Driver:  "redis_pubsub",
			Op:      "publish_delayed",
			Message: fmt.Sprintf("failed to schedule delayed message for topic %s", topic),
			Err:     err,
		}
	}

	// Start delayed message processor if not already running
	go r.processDelayedMessages(ctx, topic)

	r.mu.Lock()
	r.stats.MessagesPublished++
	r.mu.Unlock()

	return nil
}

// processDelayedMessages processes delayed messages from sorted sets
func (r *RedisPubSubDriver) processDelayedMessages(ctx context.Context, topic string) {
	delayedKey := fmt.Sprintf("delayed:%s", topic)
	ticker := time.NewTicker(1 * time.Second) // Check every second
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := time.Now().Unix()

			// Get messages that are ready to be processed
			messages, err := r.client.ZRangeByScore(ctx, delayedKey, &redis.ZRangeBy{
				Min:    "0",
				Max:    fmt.Sprintf("%d", now),
				Offset: 0,
				Count:  100,
			}).Result()

			if err != nil {
				continue
			}

			for _, msgData := range messages {
				var delayedMsg map[string]interface{}
				if err := json.Unmarshal([]byte(msgData), &delayedMsg); err != nil {
					continue
				}

				// Remove from delayed set
				r.client.ZRem(ctx, delayedKey, msgData)

				// Publish the message
				originalTopic := delayedMsg["topic"].(string)
				err := r.client.Publish(ctx, originalTopic, msgData).Err()
				if err != nil {
					log.Printf("Failed to publish delayed message: %v", err)
					// Could implement retry logic here
				}
			}
		}
	}
}

// Subscribe subscribes to a topic with a message handler
func (r *RedisPubSubDriver) Subscribe(ctx context.Context, topic string, handler messagebroker.MessageHandler) error {
	return r.SubscribeWithGroup(ctx, topic, "", handler)
}

// SubscribeWithGroup subscribes to a topic with a group (simulated using key prefixing)
func (r *RedisPubSubDriver) SubscribeWithGroup(ctx context.Context, topic string, group string, handler messagebroker.MessageHandler) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return fmt.Errorf("Redis Pub/Sub driver is closed")
	}

	// Create subscription key
	subscriptionKey := topic
	if group != "" {
		subscriptionKey = fmt.Sprintf("%s:group:%s", topic, group)
	}

	if _, exists := r.subscribers[subscriptionKey]; exists {
		return fmt.Errorf("subscription already exists for topic %s and group %s", topic, group)
	}

	// Create PubSub instance
	pubsub := r.client.Subscribe(ctx, topic)

	// Create subscriber context
	subCtx, cancel := context.WithCancel(ctx)
	subscriber := &redisSubscriber{
		pubsub:  pubsub,
		handler: handler,
		topic:   topic,
		group:   group,
		cancel:  cancel,
	}

	r.subscribers[subscriptionKey] = subscriber
	r.pubsub[subscriptionKey] = pubsub

	// Start message processing
	go r.processMessages(subCtx, subscriber)

	return nil
}

// processMessages processes incoming messages for a subscriber
func (r *RedisPubSubDriver) processMessages(ctx context.Context, subscriber *redisSubscriber) {
	ch := subscriber.pubsub.Channel()
	defer func() {
		subscriber.pubsub.Close()

		r.mu.Lock()
		subscriptionKey := subscriber.topic
		if subscriber.group != "" {
			subscriptionKey = fmt.Sprintf("%s:group:%s", subscriber.topic, subscriber.group)
		}
		delete(r.subscribers, subscriptionKey)
		delete(r.pubsub, subscriptionKey)
		r.mu.Unlock()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case redisMsg := <-ch:
			if redisMsg == nil {
				continue
			}

			// Parse Redis message
			var msgData map[string]interface{}
			if err := json.Unmarshal([]byte(redisMsg.Payload), &msgData); err != nil {
				log.Printf("Failed to unmarshal message: %v", err)
				continue
			}

			// Convert to our message format
			message := &messagebroker.Message{
				Topic:     subscriber.topic,
				Timestamp: time.Now(),
				Headers:   make(map[string]string),
				Metadata:  make(map[string]interface{}),
			}

			// Extract message fields
			if id, ok := msgData["id"].(string); ok {
				message.ID = id
			}
			if payload, ok := msgData["payload"].(string); ok {
				message.Payload = []byte(payload)
			}
			if timestamp, ok := msgData["timestamp"].(float64); ok {
				message.Timestamp = time.Unix(int64(timestamp), 0)
			}
			if retryCount, ok := msgData["retry_count"].(float64); ok {
				message.RetryCount = int(retryCount)
			}
			if maxRetries, ok := msgData["max_retries"].(float64); ok {
				message.MaxRetries = int(maxRetries)
			}

			// Extract headers
			if headers, ok := msgData["headers"].(map[string]interface{}); ok {
				for k, v := range headers {
					if strVal, ok := v.(string); ok {
						message.Headers[k] = strVal
					}
				}
			}

			// Extract metadata
			if metadata, ok := msgData["metadata"].(map[string]interface{}); ok {
				message.Metadata = metadata
			}

			// Check if this is a delayed message that's being executed
			if _, isDelayed := msgData["execute_at"]; isDelayed {
				// This is a delayed message being executed, clean up the delay metadata
				delete(message.Metadata, "execute_at")
				delete(message.Metadata, "delay")
			}

			// Handle the message
			if err := subscriber.handler(ctx, message); err != nil {
				// Handle retry logic
				if message.RetryCount < message.MaxRetries {
					message.RetryCount++
					if retryErr := r.Publish(ctx, subscriber.topic, message); retryErr != nil {
						log.Printf("Failed to retry message %s: %v", message.ID, retryErr)
					}
				} else {
					log.Printf("Message %s exceeded max retries", message.ID)
				}
			} else {
				r.mu.Lock()
				r.stats.MessagesConsumed++
				r.mu.Unlock()
			}
		}
	}
}

// EnqueueJob enqueues a job using Redis lists
func (r *RedisPubSubDriver) EnqueueJob(ctx context.Context, queue string, job *messagebroker.Job) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.closed {
		return fmt.Errorf("Redis Pub/Sub driver is closed")
	}

	// Serialize job
	jobData, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	// Use Redis lists for job queues
	queueKey := fmt.Sprintf("queue:%s", queue)

	if job.Delay > 0 {
		// Use delayed processing for jobs with delay
		return r.enqueueDelayedJob(ctx, queue, job)
	}

	// Push job to queue (priority queues could use sorted sets)
	if job.Priority > 0 {
		// Use sorted set for priority queue
		priorityKey := fmt.Sprintf("priority:%s", queue)
		err = r.client.ZAdd(ctx, priorityKey, &redis.Z{
			Score:  float64(-job.Priority), // Negative for high priority first
			Member: jobData,
		}).Err()
	} else {
		// Use regular list for FIFO
		err = r.client.LPush(ctx, queueKey, jobData).Err()
	}

	if err != nil {
		return &messagebroker.MessageBrokerError{
			Driver:  "redis_pubsub",
			Op:      "enqueue_job",
			Message: fmt.Sprintf("failed to enqueue job to queue %s", queue),
			Err:     err,
		}
	}

	r.mu.Lock()
	r.stats.JobsEnqueued++
	r.mu.Unlock()

	return nil
}

// enqueueDelayedJob enqueues a job with delay
func (r *RedisPubSubDriver) enqueueDelayedJob(ctx context.Context, queue string, job *messagebroker.Job) error {
	delayedKey := fmt.Sprintf("delayed_jobs:%s", queue)
	executeAt := time.Now().Add(job.Delay)

	jobData, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal delayed job: %w", err)
	}

	err = r.client.ZAdd(ctx, delayedKey, &redis.Z{
		Score:  float64(executeAt.Unix()),
		Member: jobData,
	}).Err()

	if err != nil {
		return err
	}

	// Start delayed job processor
	go r.processDelayedJobs(ctx, queue)

	return nil
}

// processDelayedJobs processes delayed jobs
func (r *RedisPubSubDriver) processDelayedJobs(ctx context.Context, queue string) {
	delayedKey := fmt.Sprintf("delayed_jobs:%s", queue)
	queueKey := fmt.Sprintf("queue:%s", queue)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := time.Now().Unix()

			jobs, err := r.client.ZRangeByScore(ctx, delayedKey, &redis.ZRangeBy{
				Min:    "0",
				Max:    fmt.Sprintf("%d", now),
				Offset: 0,
				Count:  100,
			}).Result()

			if err != nil {
				continue
			}

			for _, jobData := range jobs {
				// Move job from delayed set to regular queue
				pipe := r.client.Pipeline()
				pipe.ZRem(ctx, delayedKey, jobData)
				pipe.LPush(ctx, queueKey, jobData)
				pipe.Exec(ctx)
			}
		}
	}
}

// ProcessJobs processes jobs from a queue
func (r *RedisPubSubDriver) ProcessJobs(ctx context.Context, queue string, handler messagebroker.JobHandler) error {
	queueKey := fmt.Sprintf("queue:%s", queue)
	priorityKey := fmt.Sprintf("priority:%s", queue)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				// Check priority queue first
				priorityJobs, err := r.client.ZPopMax(ctx, priorityKey, 1).Result()
				if err == nil && len(priorityJobs) > 0 {
					r.processJob(ctx, priorityJobs[0].Member.(string), handler)
					continue
				}

				// Then check regular queue
				jobData, err := r.client.BRPop(ctx, 1*time.Second, queueKey).Result()
				if err != nil || len(jobData) < 2 {
					continue
				}

				r.processJob(ctx, jobData[1], handler)
			}
		}
	}()

	return nil
}

// processJob processes a single job
func (r *RedisPubSubDriver) processJob(ctx context.Context, jobData string, handler messagebroker.JobHandler) {
	var job messagebroker.Job
	if err := json.Unmarshal([]byte(jobData), &job); err != nil {
		log.Printf("Failed to unmarshal job: %v", err)
		return
	}

	job.Attempts++
	now := time.Now()
	job.ProcessedAt = &now

	if err := handler(ctx, &job); err != nil {
		log.Printf("Job processing failed: %v", err)
		// Could implement retry logic here
	} else {
		r.mu.Lock()
		r.stats.JobsProcessed++
		r.mu.Unlock()
	}
}

// CreateTopic creates a topic (no-op for Redis Pub/Sub as topics are created dynamically)
func (r *RedisPubSubDriver) CreateTopic(ctx context.Context, topic string, config *messagebroker.TopicConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.topics[topic] = true
	r.stats.TopicCount++

	return nil
}

// DeleteTopic deletes a topic (cleanup any related keys)
func (r *RedisPubSubDriver) DeleteTopic(ctx context.Context, topic string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Clean up any delayed message keys
	delayedKey := fmt.Sprintf("delayed:%s", topic)
	r.client.Del(ctx, delayedKey)

	// Cancel any active subscriptions
	for key, subscriber := range r.subscribers {
		if strings.HasPrefix(key, topic) {
			subscriber.cancel()
			delete(r.subscribers, key)
		}
	}

	delete(r.topics, topic)
	r.stats.TopicCount--

	return nil
}

// GetTopicInfo returns information about a topic
func (r *RedisPubSubDriver) GetTopicInfo(ctx context.Context, topic string) (*messagebroker.TopicInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if !r.topics[topic] {
		return nil, messagebroker.ErrTopicNotFound
	}

	// Get number of subscribers
	pubsubChannels, err := r.client.PubSubNumSub(ctx, topic).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get topic info: %w", err)
	}

	subscribers := int64(0)
	if count, exists := pubsubChannels[topic]; exists {
		subscribers = count
	}

	return &messagebroker.TopicInfo{
		Name:              topic,
		Partitions:        1,           // Redis Pub/Sub doesn't have partitions
		ReplicationFactor: 1,           // Depends on Redis setup
		MessageCount:      subscribers, // Use subscriber count as a proxy
		Size:              0,           // Not tracked in Redis Pub/Sub
		CreatedAt:         time.Now(),  // Not tracked
	}, nil
}

// Ping checks if the Redis connection is alive
func (r *RedisPubSubDriver) Ping(ctx context.Context) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.closed || r.client == nil {
		return fmt.Errorf("Redis connection is not available")
	}

	return r.client.Ping(ctx).Err()
}

// Close closes the Redis connection
func (r *RedisPubSubDriver) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return nil
	}

	r.closed = true

	// Cancel all subscribers
	for _, subscriber := range r.subscribers {
		subscriber.cancel()
	}

	// Close all PubSub instances
	for _, pubsub := range r.pubsub {
		pubsub.Close()
	}

	// Close client
	if r.client != nil {
		r.client.Close()
	}

	r.stats.ActiveConnections = 0
	return nil
}

// GetStats returns broker statistics
func (r *RedisPubSubDriver) GetStats() (*messagebroker.BrokerStats, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Update uptime
	r.stats.Uptime = time.Since(r.startTime)
	r.stats.TopicCount = len(r.topics)
	r.stats.QueueCount = len(r.subscribers)

	// Create a copy to avoid race conditions
	statsCopy := *r.stats
	statsCopy.DriverInfo = make(map[string]string)
	for k, v := range r.stats.DriverInfo {
		statsCopy.DriverInfo[k] = v
	}

	return &statsCopy, nil
}
