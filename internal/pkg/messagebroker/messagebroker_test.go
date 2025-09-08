package messagebroker

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessage(t *testing.T) {
	t.Run("should create message with correct values", func(t *testing.T) {
		now := time.Now()
		headers := map[string]string{"Content-Type": "application/json"}
		metadata := map[string]interface{}{"priority": "high"}

		message := &Message{
			ID:          "test-id",
			Topic:       "test-topic",
			Payload:     []byte(`{"test": "data"}`),
			Headers:     headers,
			Timestamp:   now,
			RetryCount:  0,
			MaxRetries:  3,
			Metadata:    metadata,
		}

		assert.Equal(t, "test-id", message.ID)
		assert.Equal(t, "test-topic", message.Topic)
		assert.Equal(t, []byte(`{"test": "data"}`), message.Payload)
		assert.Equal(t, headers, message.Headers)
		assert.Equal(t, now, message.Timestamp)
		assert.Equal(t, 0, message.RetryCount)
		assert.Equal(t, 3, message.MaxRetries)
		assert.Equal(t, metadata, message.Metadata)
	})
}

func TestJob(t *testing.T) {
	t.Run("should create job with correct values", func(t *testing.T) {
		now := time.Now()
		delay := 5 * time.Minute
		processedAt := now.Add(delay)
		metadata := map[string]interface{}{"user_id": 123}

		job := &Job{
			ID:          "job-123",
			Queue:       "email-queue",
			Handler:     "SendEmailHandler",
			Payload:     []byte(`{"email": "test@example.com"}`),
			Priority:    1,
			Delay:       delay,
			Attempts:    0,
			MaxAttempts: 3,
			CreatedAt:   now,
			ProcessedAt: &processedAt,
			Metadata:    metadata,
		}

		assert.Equal(t, "job-123", job.ID)
		assert.Equal(t, "email-queue", job.Queue)
		assert.Equal(t, "SendEmailHandler", job.Handler)
		assert.Equal(t, []byte(`{"email": "test@example.com"}`), job.Payload)
		assert.Equal(t, 1, job.Priority)
		assert.Equal(t, delay, job.Delay)
		assert.Equal(t, 0, job.Attempts)
		assert.Equal(t, 3, job.MaxAttempts)
		assert.Equal(t, now, job.CreatedAt)
		assert.Equal(t, &processedAt, job.ProcessedAt)
		assert.Equal(t, metadata, job.Metadata)
	})

	t.Run("should handle job without processed time", func(t *testing.T) {
		job := &Job{
			ID:          "job-456",
			Queue:       "default",
			Handler:     "TestHandler",
			Payload:     []byte("test"),
			ProcessedAt: nil,
		}

		assert.Nil(t, job.ProcessedAt)
	})
}

func TestTopicConfig(t *testing.T) {
	t.Run("should create topic config with correct values", func(t *testing.T) {
		retentionTime := 24 * time.Hour

		config := &TopicConfig{
			Partitions:        3,
			ReplicationFactor: 2,
			RetentionTime:     retentionTime,
			CleanupPolicy:     "delete",
		}

		assert.Equal(t, 3, config.Partitions)
		assert.Equal(t, 2, config.ReplicationFactor)
		assert.Equal(t, retentionTime, config.RetentionTime)
		assert.Equal(t, "delete", config.CleanupPolicy)
	})

	t.Run("should handle compact cleanup policy", func(t *testing.T) {
		config := &TopicConfig{
			Partitions:        1,
			ReplicationFactor: 1,
			CleanupPolicy:     "compact",
		}

		assert.Equal(t, "compact", config.CleanupPolicy)
	})
}

func TestTopicInfo(t *testing.T) {
	t.Run("should create topic info with correct values", func(t *testing.T) {
		createdAt := time.Now()

		info := &TopicInfo{
			Name:              "user-events",
			Partitions:        6,
			ReplicationFactor: 3,
			MessageCount:      12345,
			Size:              1024768,
			CreatedAt:         createdAt,
		}

		assert.Equal(t, "user-events", info.Name)
		assert.Equal(t, 6, info.Partitions)
		assert.Equal(t, 3, info.ReplicationFactor)
		assert.Equal(t, int64(12345), info.MessageCount)
		assert.Equal(t, int64(1024768), info.Size)
		assert.Equal(t, createdAt, info.CreatedAt)
	})
}

func TestBrokerStats(t *testing.T) {
	t.Run("should create broker stats with correct values", func(t *testing.T) {
		uptime := 2 * time.Hour
		driverInfo := map[string]string{
			"driver":  "redis",
			"version": "6.2.0",
		}

		stats := &BrokerStats{
			MessagesPublished: 1000,
			MessagesConsumed:  950,
			JobsEnqueued:      500,
			JobsProcessed:     480,
			ActiveConnections: 5,
			TopicCount:        10,
			QueueCount:        3,
			Uptime:            uptime,
			DriverInfo:        driverInfo,
		}

		assert.Equal(t, int64(1000), stats.MessagesPublished)
		assert.Equal(t, int64(950), stats.MessagesConsumed)
		assert.Equal(t, int64(500), stats.JobsEnqueued)
		assert.Equal(t, int64(480), stats.JobsProcessed)
		assert.Equal(t, 5, stats.ActiveConnections)
		assert.Equal(t, 10, stats.TopicCount)
		assert.Equal(t, 3, stats.QueueCount)
		assert.Equal(t, uptime, stats.Uptime)
		assert.Equal(t, driverInfo, stats.DriverInfo)
	})
}

func TestMessageBrokerConfig(t *testing.T) {
	t.Run("should create config with all drivers", func(t *testing.T) {
		config := &MessageBrokerConfig{
			Driver: "redis",
			RabbitMQ: &RabbitMQConfig{
				Host:     "localhost",
				Port:     5672,
				Username: "guest",
				Password: "guest",
			},
			Kafka: &KafkaConfig{
				Brokers: []string{"localhost:9092"},
				GroupID: "test-group",
			},
			Redis: &RedisPubSubConfig{
				Host: "localhost",
				Port: 6379,
				DB:   1,
			},
			RetryConfig: &RetryConfig{
				MaxRetries:      3,
				InitialInterval: time.Second,
				MaxInterval:     30 * time.Second,
				Multiplier:      2.0,
				RandomFactor:    0.1,
			},
		}

		assert.Equal(t, "redis", config.Driver)
		assert.NotNil(t, config.RabbitMQ)
		assert.NotNil(t, config.Kafka)
		assert.NotNil(t, config.Redis)
		assert.NotNil(t, config.RetryConfig)
	})
}

func TestRabbitMQConfig(t *testing.T) {
	t.Run("should create RabbitMQ config with correct values", func(t *testing.T) {
		config := &RabbitMQConfig{
			URL:               "amqp://guest:guest@localhost:5672/",
			Host:              "localhost",
			Port:              5672,
			Username:          "admin",
			Password:          "secret",
			VHost:             "/test",
			Exchange:          "events",
			ExchangeType:      "topic",
			ConnectionTimeout: 30 * time.Second,
			HeartbeatInterval: 60 * time.Second,
			PrefetchCount:     10,
			Durable:           true,
			AutoDelete:        false,
		}

		assert.Equal(t, "amqp://guest:guest@localhost:5672/", config.URL)
		assert.Equal(t, "localhost", config.Host)
		assert.Equal(t, 5672, config.Port)
		assert.Equal(t, "admin", config.Username)
		assert.Equal(t, "secret", config.Password)
		assert.Equal(t, "/test", config.VHost)
		assert.Equal(t, "events", config.Exchange)
		assert.Equal(t, "topic", config.ExchangeType)
		assert.Equal(t, 30*time.Second, config.ConnectionTimeout)
		assert.Equal(t, 60*time.Second, config.HeartbeatInterval)
		assert.Equal(t, 10, config.PrefetchCount)
		assert.True(t, config.Durable)
		assert.False(t, config.AutoDelete)
	})
}

func TestKafkaConfig(t *testing.T) {
	t.Run("should create Kafka config with correct values", func(t *testing.T) {
		brokers := []string{"kafka1:9092", "kafka2:9092", "kafka3:9092"}
		saslConfig := &SASLConfig{
			Enable:    true,
			Mechanism: "SCRAM-SHA-256",
			Username:  "kafka-user",
			Password:  "kafka-pass",
		}
		tlsConfig := &TLSConfig{
			Enable:             true,
			CertFile:           "/certs/client.crt",
			KeyFile:            "/certs/client.key",
			CAFile:             "/certs/ca.crt",
			InsecureSkipVerify: false,
		}

		config := &KafkaConfig{
			Brokers:               brokers,
			GroupID:               "consumer-group-1",
			ClientID:              "client-1",
			Version:               "2.8.0",
			ConnectTimeout:        10 * time.Second,
			SessionTimeout:        30 * time.Second,
			HeartbeatInterval:     3 * time.Second,
			RebalanceTimeout:      60 * time.Second,
			ReturnSuccesses:       true,
			RequiredAcks:          1,
			CompressionType:       "gzip",
			FlushFrequency:        100 * time.Millisecond,
			EnableAutoCommit:      true,
			AutoCommitInterval:    time.Second,
			InitialOffset:         "newest",
			SASL:                  saslConfig,
			TLS:                   tlsConfig,
		}

		assert.Equal(t, brokers, config.Brokers)
		assert.Equal(t, "consumer-group-1", config.GroupID)
		assert.Equal(t, "client-1", config.ClientID)
		assert.Equal(t, "2.8.0", config.Version)
		assert.Equal(t, 10*time.Second, config.ConnectTimeout)
		assert.Equal(t, 30*time.Second, config.SessionTimeout)
		assert.Equal(t, 3*time.Second, config.HeartbeatInterval)
		assert.Equal(t, 60*time.Second, config.RebalanceTimeout)
		assert.True(t, config.ReturnSuccesses)
		assert.Equal(t, 1, config.RequiredAcks)
		assert.Equal(t, "gzip", config.CompressionType)
		assert.Equal(t, 100*time.Millisecond, config.FlushFrequency)
		assert.True(t, config.EnableAutoCommit)
		assert.Equal(t, time.Second, config.AutoCommitInterval)
		assert.Equal(t, "newest", config.InitialOffset)
		assert.Equal(t, saslConfig, config.SASL)
		assert.Equal(t, tlsConfig, config.TLS)
	})

	t.Run("should handle oldest initial offset", func(t *testing.T) {
		config := &KafkaConfig{
			Brokers:       []string{"localhost:9092"},
			InitialOffset: "oldest",
		}

		assert.Equal(t, "oldest", config.InitialOffset)
	})
}

func TestRedisPubSubConfig(t *testing.T) {
	t.Run("should create Redis config with correct values", func(t *testing.T) {
		tlsConfig := &TLSConfig{
			Enable:             true,
			InsecureSkipVerify: true,
		}

		config := &RedisPubSubConfig{
			Host:           "redis-cluster.example.com",
			Port:           6380,
			Password:       "redis-secret",
			DB:             2,
			PoolSize:       20,
			MinIdleConns:   5,
			MaxRetries:     3,
			ConnectTimeout: 5 * time.Second,
			ReadTimeout:    3 * time.Second,
			WriteTimeout:   3 * time.Second,
			IdleTimeout:    300 * time.Second,
			TLS:            tlsConfig,
		}

		assert.Equal(t, "redis-cluster.example.com", config.Host)
		assert.Equal(t, 6380, config.Port)
		assert.Equal(t, "redis-secret", config.Password)
		assert.Equal(t, 2, config.DB)
		assert.Equal(t, 20, config.PoolSize)
		assert.Equal(t, 5, config.MinIdleConns)
		assert.Equal(t, 3, config.MaxRetries)
		assert.Equal(t, 5*time.Second, config.ConnectTimeout)
		assert.Equal(t, 3*time.Second, config.ReadTimeout)
		assert.Equal(t, 3*time.Second, config.WriteTimeout)
		assert.Equal(t, 300*time.Second, config.IdleTimeout)
		assert.Equal(t, tlsConfig, config.TLS)
	})
}

func TestRetryConfig(t *testing.T) {
	t.Run("should create retry config with correct values", func(t *testing.T) {
		config := &RetryConfig{
			MaxRetries:      5,
			InitialInterval: 500 * time.Millisecond,
			MaxInterval:     60 * time.Second,
			Multiplier:      2.5,
			RandomFactor:    0.2,
		}

		assert.Equal(t, 5, config.MaxRetries)
		assert.Equal(t, 500*time.Millisecond, config.InitialInterval)
		assert.Equal(t, 60*time.Second, config.MaxInterval)
		assert.Equal(t, 2.5, config.Multiplier)
		assert.Equal(t, 0.2, config.RandomFactor)
	})
}

func TestSASLConfig(t *testing.T) {
	t.Run("should create SASL config for PLAIN mechanism", func(t *testing.T) {
		config := &SASLConfig{
			Enable:    true,
			Mechanism: "PLAIN",
			Username:  "user",
			Password:  "pass",
		}

		assert.True(t, config.Enable)
		assert.Equal(t, "PLAIN", config.Mechanism)
		assert.Equal(t, "user", config.Username)
		assert.Equal(t, "pass", config.Password)
	})

	t.Run("should create SASL config for SCRAM-SHA-512 mechanism", func(t *testing.T) {
		config := &SASLConfig{
			Enable:    true,
			Mechanism: "SCRAM-SHA-512",
			Username:  "secure-user",
			Password:  "secure-pass",
		}

		assert.True(t, config.Enable)
		assert.Equal(t, "SCRAM-SHA-512", config.Mechanism)
		assert.Equal(t, "secure-user", config.Username)
		assert.Equal(t, "secure-pass", config.Password)
	})

	t.Run("should handle disabled SASL", func(t *testing.T) {
		config := &SASLConfig{
			Enable: false,
		}

		assert.False(t, config.Enable)
	})
}

func TestTLSConfig(t *testing.T) {
	t.Run("should create TLS config with all certificates", func(t *testing.T) {
		config := &TLSConfig{
			Enable:             true,
			CertFile:           "/etc/ssl/certs/client.crt",
			KeyFile:            "/etc/ssl/private/client.key",
			CAFile:             "/etc/ssl/certs/ca.crt",
			InsecureSkipVerify: false,
		}

		assert.True(t, config.Enable)
		assert.Equal(t, "/etc/ssl/certs/client.crt", config.CertFile)
		assert.Equal(t, "/etc/ssl/private/client.key", config.KeyFile)
		assert.Equal(t, "/etc/ssl/certs/ca.crt", config.CAFile)
		assert.False(t, config.InsecureSkipVerify)
	})

	t.Run("should handle disabled TLS", func(t *testing.T) {
		config := &TLSConfig{
			Enable: false,
		}

		assert.False(t, config.Enable)
	})

	t.Run("should handle insecure skip verify", func(t *testing.T) {
		config := &TLSConfig{
			Enable:             true,
			InsecureSkipVerify: true,
		}

		assert.True(t, config.Enable)
		assert.True(t, config.InsecureSkipVerify)
	})
}

func TestNewMessage(t *testing.T) {
	t.Run("should create message from byte slice", func(t *testing.T) {
		topic := "test-topic"
		payload := []byte(`{"key": "value"}`)

		message, err := NewMessage(topic, payload)

		require.NoError(t, err)
		assert.Equal(t, topic, message.Topic)
		assert.Equal(t, payload, message.Payload)
		assert.NotEmpty(t, message.ID)
		assert.False(t, message.Timestamp.IsZero())
	})

	t.Run("should create message from string", func(t *testing.T) {
		topic := "test-topic"
		payload := "hello world"

		message, err := NewMessage(topic, payload)

		require.NoError(t, err)
		assert.Equal(t, topic, message.Topic)
		assert.Equal(t, []byte(payload), message.Payload)
		assert.NotEmpty(t, message.ID)
		assert.False(t, message.Timestamp.IsZero())
	})

	t.Run("should create message from struct", func(t *testing.T) {
		topic := "test-topic"
		payload := map[string]interface{}{
			"user_id": 123,
			"action":  "login",
		}

		message, err := NewMessage(topic, payload)

		require.NoError(t, err)
		assert.Equal(t, topic, message.Topic)
		assert.NotEmpty(t, message.Payload)
		assert.NotEmpty(t, message.ID)
		assert.False(t, message.Timestamp.IsZero())

		// Verify JSON structure
		assert.Contains(t, string(message.Payload), "user_id")
		assert.Contains(t, string(message.Payload), "action")
	})

	t.Run("should handle invalid JSON marshaling", func(t *testing.T) {
		topic := "test-topic"
		// Function cannot be marshaled to JSON
		payload := func() {}

		message, err := NewMessage(topic, payload)

		assert.Error(t, err)
		assert.Nil(t, message)
	})
}

func TestNewJob(t *testing.T) {
	t.Run("should create job with correct defaults", func(t *testing.T) {
		queue := "email-queue"
		handler := "SendEmailHandler"
		payload := map[string]string{"email": "test@example.com"}

		job, err := NewJob(queue, handler, payload)

		require.NoError(t, err)
		assert.Equal(t, queue, job.Queue)
		assert.Equal(t, handler, job.Handler)
		assert.NotEmpty(t, job.ID)
		assert.NotEmpty(t, job.Payload)
		assert.Equal(t, 0, job.Priority)
		assert.Equal(t, time.Duration(0), job.Delay)
		assert.Equal(t, 0, job.Attempts)
		assert.Equal(t, 3, job.MaxAttempts) // Default max attempts
		assert.False(t, job.CreatedAt.IsZero())
		assert.Nil(t, job.ProcessedAt)
	})

	t.Run("should create job from byte slice", func(t *testing.T) {
		queue := "processing-queue"
		handler := "DataProcessor"
		payload := []byte(`{"data": "raw"}`)

		job, err := NewJob(queue, handler, payload)

		require.NoError(t, err)
		assert.Equal(t, payload, job.Payload)
	})

	t.Run("should handle invalid JSON marshaling for job", func(t *testing.T) {
		queue := "test-queue"
		handler := "TestHandler"
		// Function cannot be marshaled to JSON
		payload := func() {}

		job, err := NewJob(queue, handler, payload)

		assert.Error(t, err)
		assert.Nil(t, job)
	})
}

func TestMessageHandlerType(t *testing.T) {
	t.Run("should define correct handler function signature", func(t *testing.T) {
		var handler MessageHandler = func(ctx context.Context, msg *Message) error {
			// Test that the handler can access message properties
			assert.NotNil(t, msg)
			assert.NotEmpty(t, msg.Topic)
			return nil
		}

		// Test calling the handler
		ctx := context.Background()
		msg := &Message{
			ID:    "test",
			Topic: "test-topic",
		}

		err := handler(ctx, msg)
		assert.NoError(t, err)
	})
}

func TestJobHandlerType(t *testing.T) {
	t.Run("should define correct job handler function signature", func(t *testing.T) {
		var handler JobHandler = func(ctx context.Context, job *Job) error {
			// Test that the handler can access job properties
			assert.NotNil(t, job)
			assert.NotEmpty(t, job.Queue)
			return nil
		}

		// Test calling the handler
		ctx := context.Background()
		job := &Job{
			ID:      "test",
			Queue:   "test-queue",
			Handler: "TestHandler",
		}

		err := handler(ctx, job)
		assert.NoError(t, err)
	})
}

func TestMessageWithHeaders(t *testing.T) {
	t.Run("should add headers to message", func(t *testing.T) {
		message := &Message{
			ID:    "test",
			Topic: "test-topic",
		}

		headers := map[string]string{
			"Content-Type": "application/json",
			"User-Agent":   "test-agent",
		}

		result := message.WithHeaders(headers)

		assert.Equal(t, message, result) // Should return the same instance
		assert.Equal(t, "application/json", message.Headers["Content-Type"])
		assert.Equal(t, "test-agent", message.Headers["User-Agent"])
	})

	t.Run("should initialize headers if nil", func(t *testing.T) {
		message := &Message{
			ID:      "test",
			Topic:   "test-topic",
			Headers: nil,
		}

		headers := map[string]string{"Key": "Value"}
		message.WithHeaders(headers)

		assert.NotNil(t, message.Headers)
		assert.Equal(t, "Value", message.Headers["Key"])
	})

	t.Run("should merge with existing headers", func(t *testing.T) {
		message := &Message{
			ID:    "test",
			Topic: "test-topic",
			Headers: map[string]string{
				"Existing": "value",
			},
		}

		newHeaders := map[string]string{
			"New":      "header",
			"Existing": "updated", // Should overwrite
		}

		message.WithHeaders(newHeaders)

		assert.Equal(t, "updated", message.Headers["Existing"])
		assert.Equal(t, "header", message.Headers["New"])
	})
}

func TestMessageWithMetadata(t *testing.T) {
	t.Run("should add metadata to message", func(t *testing.T) {
		message := &Message{
			ID:    "test",
			Topic: "test-topic",
		}

		metadata := map[string]interface{}{
			"priority": "high",
			"user_id":  123,
		}

		result := message.WithMetadata(metadata)

		assert.Equal(t, message, result) // Should return the same instance
		assert.Equal(t, "high", message.Metadata["priority"])
		assert.Equal(t, 123, message.Metadata["user_id"])
	})

	t.Run("should initialize metadata if nil", func(t *testing.T) {
		message := &Message{
			ID:       "test",
			Topic:    "test-topic",
			Metadata: nil,
		}

		metadata := map[string]interface{}{"key": "value"}
		message.WithMetadata(metadata)

		assert.NotNil(t, message.Metadata)
		assert.Equal(t, "value", message.Metadata["key"])
	})
}

func TestJobWithPriority(t *testing.T) {
	t.Run("should set job priority", func(t *testing.T) {
		job := &Job{
			ID:       "test",
			Queue:    "test-queue",
			Priority: 0,
		}

		result := job.WithPriority(5)

		assert.Equal(t, job, result) // Should return the same instance
		assert.Equal(t, 5, job.Priority)
	})
}

func TestJobWithDelay(t *testing.T) {
	t.Run("should set job delay", func(t *testing.T) {
		job := &Job{
			ID:    "test",
			Queue: "test-queue",
			Delay: 0,
		}

		delay := 10 * time.Minute
		result := job.WithDelay(delay)

		assert.Equal(t, job, result) // Should return the same instance
		assert.Equal(t, delay, job.Delay)
	})
}

func TestMessageUnmarshalPayload(t *testing.T) {
	t.Run("should unmarshal JSON payload", func(t *testing.T) {
		data := map[string]interface{}{
			"name": "John",
			"age":  30,
		}
		payload, _ := NewMessage("test", data)

		var result map[string]interface{}
		err := payload.UnmarshalPayload(&result)

		require.NoError(t, err)
		assert.Equal(t, "John", result["name"])
		assert.Equal(t, float64(30), result["age"]) // JSON numbers are float64
	})

	t.Run("should handle invalid JSON", func(t *testing.T) {
		message := &Message{
			Payload: []byte(`invalid json`),
		}

		var result map[string]interface{}
		err := message.UnmarshalPayload(&result)

		assert.Error(t, err)
	})

	t.Run("should unmarshal into struct", func(t *testing.T) {
		type User struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}

		data := User{Name: "Alice", Age: 25}
		message, _ := NewMessage("test", data)

		var result User
		err := message.UnmarshalPayload(&result)

		require.NoError(t, err)
		assert.Equal(t, "Alice", result.Name)
		assert.Equal(t, 25, result.Age)
	})
}

func TestJobUnmarshalPayload(t *testing.T) {
	t.Run("should unmarshal JSON payload", func(t *testing.T) {
		data := map[string]string{
			"email": "test@example.com",
			"name":  "Test User",
		}
		job, _ := NewJob("email", "SendEmail", data)

		var result map[string]string
		err := job.UnmarshalPayload(&result)

		require.NoError(t, err)
		assert.Equal(t, "test@example.com", result["email"])
		assert.Equal(t, "Test User", result["name"])
	})

	t.Run("should handle invalid JSON", func(t *testing.T) {
		job := &Job{
			Payload: []byte(`invalid json`),
		}

		var result map[string]interface{}
		err := job.UnmarshalPayload(&result)

		assert.Error(t, err)
	})
}

func TestGetPayloadString(t *testing.T) {
	t.Run("should return message payload as string", func(t *testing.T) {
		payload := "hello world"
		message, _ := NewMessage("test", payload)

		result := message.GetPayloadString()

		assert.Equal(t, payload, result)
	})

	t.Run("should return job payload as string", func(t *testing.T) {
		payload := "job data"
		job, _ := NewJob("test", "handler", payload)

		result := job.GetPayloadString()

		assert.Equal(t, payload, result)
	})

	t.Run("should return JSON payload as string", func(t *testing.T) {
		data := map[string]string{"key": "value"}
		message, _ := NewMessage("test", data)

		result := message.GetPayloadString()

		assert.Contains(t, result, "key")
		assert.Contains(t, result, "value")
	})
}

func TestMessageBrokerError(t *testing.T) {
	t.Run("should create error with underlying error", func(t *testing.T) {
		underlyingErr := assert.AnError
		err := &MessageBrokerError{
			Driver:  "redis",
			Op:      "publish",
			Message: "connection failed",
			Err:     underlyingErr,
		}

		expectedMsg := "messagebroker: redis driver failed on publish: connection failed (assert.AnError general error for testing)"
		assert.Equal(t, expectedMsg, err.Error())
	})

	t.Run("should create error without underlying error", func(t *testing.T) {
		err := &MessageBrokerError{
			Driver:  "kafka",
			Op:      "subscribe",
			Message: "topic not found",
			Err:     nil,
		}

		expectedMsg := "messagebroker: kafka driver failed on subscribe: topic not found"
		assert.Equal(t, expectedMsg, err.Error())
	})

	t.Run("should unwrap underlying error", func(t *testing.T) {
		underlyingErr := assert.AnError
		err := &MessageBrokerError{
			Driver:  "rabbitmq",
			Op:      "connect",
			Message: "auth failed",
			Err:     underlyingErr,
		}

		assert.Equal(t, underlyingErr, err.Unwrap())
	})

	t.Run("should return nil when no underlying error", func(t *testing.T) {
		err := &MessageBrokerError{
			Driver:  "redis",
			Op:      "ping",
			Message: "timeout",
			Err:     nil,
		}

		assert.Nil(t, err.Unwrap())
	})
}

func TestCommonErrors(t *testing.T) {
	t.Run("should define common error variables", func(t *testing.T) {
		assert.NotNil(t, ErrDriverNotSupported)
		assert.NotNil(t, ErrConnectionFailed)
		assert.NotNil(t, ErrTopicNotFound)
		assert.NotNil(t, ErrQueueNotFound)
		assert.NotNil(t, ErrInvalidConfiguration)
		assert.NotNil(t, ErrMessageTooLarge)
		assert.NotNil(t, ErrMaxRetriesExceeded)

		// Check error messages
		assert.Contains(t, ErrDriverNotSupported.Error(), "not supported")
		assert.Contains(t, ErrConnectionFailed.Error(), "connect")
		assert.Contains(t, ErrTopicNotFound.Error(), "topic")
		assert.Contains(t, ErrQueueNotFound.Error(), "queue")
		assert.Contains(t, ErrInvalidConfiguration.Error(), "configuration")
		assert.Contains(t, ErrMessageTooLarge.Error(), "large")
		assert.Contains(t, ErrMaxRetriesExceeded.Error(), "retries")
	})
}