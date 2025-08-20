package drivers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/streadway/amqp"

	"github.com/VeRJiL/go-template/internal/pkg/messagebroker"
)

// RabbitMQDriver implements MessageBroker interface for RabbitMQ
type RabbitMQDriver struct {
	config    *messagebroker.RabbitMQConfig
	conn      *amqp.Connection
	channel   *amqp.Channel
	mu        sync.RWMutex
	closed    bool
	stats     *messagebroker.BrokerStats
	startTime time.Time
	exchanges map[string]bool
	queues    map[string]bool
}

// NewRabbitMQDriver creates a new RabbitMQ driver instance
func NewRabbitMQDriver(config *messagebroker.RabbitMQConfig) (*RabbitMQDriver, error) {
	if config == nil {
		return nil, fmt.Errorf("RabbitMQ config cannot be nil")
	}

	driver := &RabbitMQDriver{
		config:    config,
		startTime: time.Now(),
		exchanges: make(map[string]bool),
		queues:    make(map[string]bool),
		stats: &messagebroker.BrokerStats{
			DriverInfo: map[string]string{
				"driver":   "rabbitmq",
				"exchange": config.Exchange,
				"vhost":    config.VHost,
			},
		},
	}

	if err := driver.connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	return driver, nil
}

// connect establishes connection to RabbitMQ
func (r *RabbitMQDriver) connect() error {
	var connectionURL string

	if r.config.URL != "" {
		connectionURL = r.config.URL
	} else {
		connectionURL = fmt.Sprintf("amqp://%s:%s@%s:%d/%s",
			r.config.Username,
			r.config.Password,
			r.config.Host,
			r.config.Port,
			r.config.VHost)
	}

	conn, err := amqp.DialConfig(connectionURL, amqp.Config{
		Heartbeat: r.config.HeartbeatInterval,
		Dial:      amqp.DefaultDial(r.config.ConnectionTimeout),
	})
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to open channel: %w", err)
	}

	// Set QoS if configured
	if r.config.PrefetchCount > 0 {
		if err := channel.Qos(r.config.PrefetchCount, 0, false); err != nil {
			conn.Close()
			return fmt.Errorf("failed to set QoS: %w", err)
		}
	}

	r.conn = conn
	r.channel = channel

	// Declare the main exchange if it doesn't exist
	if r.config.Exchange != "" {
		if err := r.declareExchange(r.config.Exchange, r.config.ExchangeType); err != nil {
			return fmt.Errorf("failed to declare exchange: %w", err)
		}
	}

	// Setup connection loss handling
	go r.handleConnectionLoss()

	r.stats.ActiveConnections = 1
	return nil
}

// declareExchange declares an exchange if it doesn't exist
func (r *RabbitMQDriver) declareExchange(name, exchangeType string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.exchanges[name] {
		return nil // Already declared
	}

	if exchangeType == "" {
		exchangeType = "topic"
	}

	err := r.channel.ExchangeDeclare(
		name,
		exchangeType,
		r.config.Durable,
		r.config.AutoDelete,
		false, // internal
		false, // no-wait
		nil,   // arguments
	)

	if err != nil {
		return err
	}

	r.exchanges[name] = true
	return nil
}

// declareQueue declares a queue if it doesn't exist
func (r *RabbitMQDriver) declareQueue(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.queues[name] {
		return nil // Already declared
	}

	_, err := r.channel.QueueDeclare(
		name,
		r.config.Durable,
		r.config.AutoDelete,
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)

	if err != nil {
		return err
	}

	r.queues[name] = true
	return nil
}

// handleConnectionLoss monitors connection and attempts to reconnect
func (r *RabbitMQDriver) handleConnectionLoss() {
	connClosed := make(chan *amqp.Error)
	r.conn.NotifyClose(connClosed)

	for {
		select {
		case err := <-connClosed:
			if err != nil {
				log.Printf("RabbitMQ connection lost: %v", err)
				r.mu.Lock()
				r.stats.ActiveConnections = 0
				r.mu.Unlock()

				// Attempt to reconnect
				for {
					if err := r.connect(); err != nil {
						log.Printf("Failed to reconnect to RabbitMQ: %v", err)
						time.Sleep(5 * time.Second)
						continue
					}
					log.Println("Successfully reconnected to RabbitMQ")
					break
				}
			}
			return
		}
	}
}

// Publish publishes a message to a topic
func (r *RabbitMQDriver) Publish(ctx context.Context, topic string, message *messagebroker.Message) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.closed {
		return fmt.Errorf("RabbitMQ driver is closed")
	}

	// Ensure exchange exists
	if err := r.declareExchange(r.config.Exchange, r.config.ExchangeType); err != nil {
		return fmt.Errorf("failed to declare exchange: %w", err)
	}

	// Convert message to AMQP format
	headers := make(amqp.Table)
	for k, v := range message.Headers {
		headers[k] = v
	}

	// Add metadata to headers
	for k, v := range message.Metadata {
		headers[fmt.Sprintf("meta_%s", k)] = v
	}

	// Add message info to headers
	headers["message_id"] = message.ID
	headers["retry_count"] = message.RetryCount
	headers["max_retries"] = message.MaxRetries
	headers["timestamp"] = message.Timestamp.Unix()

	publishing := amqp.Publishing{
		DeliveryMode: amqp.Persistent, // Make message persistent
		ContentType:  "application/json",
		Body:         message.Payload,
		MessageId:    message.ID,
		Timestamp:    message.Timestamp,
		Headers:      headers,
	}

	err := r.channel.Publish(
		r.config.Exchange, // exchange
		topic,             // routing key
		false,             // mandatory
		false,             // immediate
		publishing,
	)

	if err != nil {
		return &messagebroker.MessageBrokerError{
			Driver:  "rabbitmq",
			Op:      "publish",
			Message: fmt.Sprintf("failed to publish message to topic %s", topic),
			Err:     err,
		}
	}

	r.mu.Lock()
	r.stats.MessagesPublished++
	r.mu.Unlock()

	return nil
}

// PublishJSON publishes JSON data to a topic
func (r *RabbitMQDriver) PublishJSON(ctx context.Context, topic string, data interface{}) error {
	message, err := messagebroker.NewMessage(topic, data)
	if err != nil {
		return fmt.Errorf("failed to create message: %w", err)
	}
	return r.Publish(ctx, topic, message)
}

// PublishWithDelay publishes a message with a delay (uses RabbitMQ delayed message plugin)
func (r *RabbitMQDriver) PublishWithDelay(ctx context.Context, topic string, message *messagebroker.Message, delay time.Duration) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.closed {
		return fmt.Errorf("RabbitMQ driver is closed")
	}

	// For delayed messages, we need a delayed exchange or use TTL + DLX pattern
	delayedExchange := r.config.Exchange + ".delayed"
	if err := r.declareExchange(delayedExchange, "x-delayed-message"); err != nil {
		// Fallback to TTL + Dead Letter Exchange pattern
		return r.publishWithTTLDelay(ctx, topic, message, delay)
	}

	headers := make(amqp.Table)
	for k, v := range message.Headers {
		headers[k] = v
	}

	// Add delay header for x-delayed-message exchange
	headers["x-delay"] = int64(delay.Milliseconds())
	headers["message_id"] = message.ID
	headers["timestamp"] = message.Timestamp.Unix()

	publishing := amqp.Publishing{
		DeliveryMode: amqp.Persistent,
		ContentType:  "application/json",
		Body:         message.Payload,
		MessageId:    message.ID,
		Timestamp:    message.Timestamp,
		Headers:      headers,
	}

	err := r.channel.Publish(
		delayedExchange,
		topic,
		false,
		false,
		publishing,
	)

	if err != nil {
		return &messagebroker.MessageBrokerError{
			Driver:  "rabbitmq",
			Op:      "publish_delayed",
			Message: fmt.Sprintf("failed to publish delayed message to topic %s", topic),
			Err:     err,
		}
	}

	r.mu.Lock()
	r.stats.MessagesPublished++
	r.mu.Unlock()

	return nil
}

// publishWithTTLDelay implements delay using TTL + Dead Letter Exchange pattern
func (r *RabbitMQDriver) publishWithTTLDelay(ctx context.Context, topic string, message *messagebroker.Message, delay time.Duration) error {
	// Create temporary queue with TTL that routes to main exchange after expiry
	tempQueueName := fmt.Sprintf("delay_%s_%d", message.ID, delay.Milliseconds())

	_, err := r.channel.QueueDeclare(
		tempQueueName,
		false, // durable
		true,  // auto-delete
		false, // exclusive
		false, // no-wait
		amqp.Table{
			"x-message-ttl":             int64(delay.Milliseconds()),
			"x-dead-letter-exchange":    r.config.Exchange,
			"x-dead-letter-routing-key": topic,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to declare delay queue: %w", err)
	}

	headers := make(amqp.Table)
	for k, v := range message.Headers {
		headers[k] = v
	}

	publishing := amqp.Publishing{
		ContentType: "application/json",
		Body:        message.Payload,
		MessageId:   message.ID,
		Timestamp:   message.Timestamp,
		Headers:     headers,
	}

	return r.channel.Publish("", tempQueueName, false, false, publishing)
}

// Subscribe subscribes to a topic with a message handler
func (r *RabbitMQDriver) Subscribe(ctx context.Context, topic string, handler messagebroker.MessageHandler) error {
	return r.SubscribeWithGroup(ctx, topic, "", handler)
}

// SubscribeWithGroup subscribes to a topic with a consumer group (queue)
func (r *RabbitMQDriver) SubscribeWithGroup(ctx context.Context, topic string, group string, handler messagebroker.MessageHandler) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.closed {
		return fmt.Errorf("RabbitMQ driver is closed")
	}

	// Create queue name based on group or generate one
	queueName := topic
	if group != "" {
		queueName = fmt.Sprintf("%s.%s", topic, group)
	}

	// Declare queue
	if err := r.declareQueue(queueName); err != nil {
		return fmt.Errorf("failed to declare queue %s: %w", queueName, err)
	}

	// Bind queue to exchange
	if r.config.Exchange != "" {
		err := r.channel.QueueBind(
			queueName,
			topic, // routing key
			r.config.Exchange,
			false, // no-wait
			nil,   // arguments
		)
		if err != nil {
			return fmt.Errorf("failed to bind queue %s to exchange: %w", queueName, err)
		}
	}

	// Start consuming
	msgs, err := r.channel.Consume(
		queueName,
		"",    // consumer tag
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to start consuming from queue %s: %w", queueName, err)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-msgs:
				if !ok {
					return
				}

				// Convert AMQP message to our message format
				message := &messagebroker.Message{
					ID:        msg.MessageId,
					Topic:     topic,
					Payload:   msg.Body,
					Headers:   make(map[string]string),
					Timestamp: msg.Timestamp,
					Metadata:  make(map[string]interface{}),
				}

				// Extract headers
				for k, v := range msg.Headers {
					if strVal, ok := v.(string); ok {
						if strings.HasPrefix(k, "meta_") {
							message.Metadata[strings.TrimPrefix(k, "meta_")] = strVal
						} else {
							message.Headers[k] = strVal
						}
					}
				}

				// Extract retry information
				if retryCount, exists := msg.Headers["retry_count"]; exists {
					if count, ok := retryCount.(int); ok {
						message.RetryCount = count
					}
				}
				if maxRetries, exists := msg.Headers["max_retries"]; exists {
					if max, ok := maxRetries.(int); ok {
						message.MaxRetries = max
					}
				}

				// Handle message
				if err := handler(ctx, message); err != nil {
					// Handle retry logic
					if message.RetryCount < message.MaxRetries {
						message.RetryCount++
						if retryErr := r.Publish(ctx, topic, message); retryErr != nil {
							log.Printf("Failed to retry message %s: %v", message.ID, retryErr)
						}
					}
					msg.Nack(false, false) // Don't requeue, we handle retry ourselves
				} else {
					msg.Ack(false)
					r.mu.Lock()
					r.stats.MessagesConsumed++
					r.mu.Unlock()
				}
			}
		}
	}()

	return nil
}

// EnqueueJob enqueues a job for processing
func (r *RabbitMQDriver) EnqueueJob(ctx context.Context, queue string, job *messagebroker.Job) error {
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
		Metadata:  map[string]interface{}{"job_priority": job.Priority},
	}

	if job.Delay > 0 {
		return r.PublishWithDelay(ctx, queue, message, job.Delay)
	}

	err = r.Publish(ctx, queue, message)
	if err == nil {
		r.mu.Lock()
		r.stats.JobsEnqueued++
		r.mu.Unlock()
	}
	return err
}

// ProcessJobs processes jobs from a queue
func (r *RabbitMQDriver) ProcessJobs(ctx context.Context, queue string, handler messagebroker.JobHandler) error {
	return r.SubscribeWithGroup(ctx, queue, "", func(ctx context.Context, msg *messagebroker.Message) error {
		var job messagebroker.Job
		if err := json.Unmarshal(msg.Payload, &job); err != nil {
			return fmt.Errorf("failed to unmarshal job: %w", err)
		}

		job.Attempts++
		job.ProcessedAt = &msg.Timestamp

		err := handler(ctx, &job)
		if err == nil {
			r.mu.Lock()
			r.stats.JobsProcessed++
			r.mu.Unlock()
		}
		return err
	})
}

// CreateTopic creates a topic (exchange + queue binding)
func (r *RabbitMQDriver) CreateTopic(ctx context.Context, topic string, config *messagebroker.TopicConfig) error {
	// In RabbitMQ, "creating a topic" means declaring an exchange and optionally a queue
	if err := r.declareExchange(r.config.Exchange, r.config.ExchangeType); err != nil {
		return fmt.Errorf("failed to declare exchange for topic %s: %w", topic, err)
	}

	// Create a default queue for this topic if needed
	if err := r.declareQueue(topic); err != nil {
		return fmt.Errorf("failed to declare queue for topic %s: %w", topic, err)
	}

	// Bind the queue to the exchange
	if r.config.Exchange != "" {
		err := r.channel.QueueBind(
			topic,             // queue name
			topic,             // routing key
			r.config.Exchange, // exchange
			false,             // no-wait
			nil,               // arguments
		)
		if err != nil {
			return fmt.Errorf("failed to bind queue %s to exchange: %w", topic, err)
		}
	}

	r.mu.Lock()
	r.stats.TopicCount++
	r.mu.Unlock()

	return nil
}

// DeleteTopic deletes a topic (queue)
func (r *RabbitMQDriver) DeleteTopic(ctx context.Context, topic string) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.closed {
		return fmt.Errorf("RabbitMQ driver is closed")
	}

	_, err := r.channel.QueueDelete(topic, false, false, false)
	if err != nil {
		return fmt.Errorf("failed to delete topic %s: %w", topic, err)
	}

	delete(r.queues, topic)
	r.stats.TopicCount--

	return nil
}

// GetTopicInfo returns information about a topic
func (r *RabbitMQDriver) GetTopicInfo(ctx context.Context, topic string) (*messagebroker.TopicInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.closed {
		return nil, fmt.Errorf("RabbitMQ driver is closed")
	}

	queue, err := r.channel.QueueInspect(topic)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect topic %s: %w", topic, err)
	}

	return &messagebroker.TopicInfo{
		Name:              topic,
		Partitions:        1, // RabbitMQ doesn't have partitions like Kafka
		ReplicationFactor: 1, // Depends on cluster setup
		MessageCount:      int64(queue.Messages),
		Size:              0,          // Not easily available in RabbitMQ
		CreatedAt:         time.Now(), // Not tracked by RabbitMQ
	}, nil
}

// Ping checks if the connection is alive
func (r *RabbitMQDriver) Ping(ctx context.Context) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.closed || r.conn == nil || r.conn.IsClosed() {
		return fmt.Errorf("RabbitMQ connection is not available")
	}

	return nil
}

// Close closes the RabbitMQ connection
func (r *RabbitMQDriver) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return nil
	}

	r.closed = true

	if r.channel != nil {
		r.channel.Close()
	}

	if r.conn != nil {
		r.conn.Close()
	}

	r.stats.ActiveConnections = 0
	return nil
}

// GetStats returns broker statistics
func (r *RabbitMQDriver) GetStats() (*messagebroker.BrokerStats, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Update uptime
	r.stats.Uptime = time.Since(r.startTime)
	r.stats.QueueCount = len(r.queues)

	// Create a copy to avoid race conditions
	statsCopy := *r.stats
	statsCopy.DriverInfo = make(map[string]string)
	for k, v := range r.stats.DriverInfo {
		statsCopy.DriverInfo[k] = v
	}

	return &statsCopy, nil
}
