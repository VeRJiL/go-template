package messagebroker

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestInterfacesMessage(t *testing.T) {
	t.Run("should create message with correct structure from interfaces.go", func(t *testing.T) {
		now := time.Now()
		headers := map[string]string{"Content-Type": "application/json"}
		metadata := map[string]interface{}{"priority": "high"}

		// Using the Message struct defined in interfaces.go
		message := Message{
			ID:        "test-123",
			Topic:     "user-events",
			Key:       "user-123",
			Value:     []byte(`{"action": "login"}`),
			Headers:   headers,
			Metadata:  metadata,
			Timestamp: now,
			Partition: 1,
			Offset:    100,
		}

		assert.Equal(t, "test-123", message.ID)
		assert.Equal(t, "user-events", message.Topic)
		assert.Equal(t, "user-123", message.Key)
		assert.Equal(t, []byte(`{"action": "login"}`), message.Value)
		assert.Equal(t, headers, message.Headers)
		assert.Equal(t, metadata, message.Metadata)
		assert.Equal(t, now, message.Timestamp)
		assert.Equal(t, int32(1), message.Partition)
		assert.Equal(t, int64(100), message.Offset)
	})
}

func TestInterfacesBrokerStats(t *testing.T) {
	t.Run("should create broker stats with correct structure", func(t *testing.T) {
		lastActivity := time.Now()
		uptime := 2 * time.Hour

		stats := BrokerStats{
			MessagesProduced: 1000,
			MessagesConsumed: 950,
			MessagesFailed:   5,
			ActiveConsumers:  3,
			ActiveTopics:     10,
			ConnectionStatus: "connected",
			LastActivity:     lastActivity,
			Uptime:          uptime,
			BytesProduced:   1024000,
			BytesConsumed:   1020000,
		}

		assert.Equal(t, int64(1000), stats.MessagesProduced)
		assert.Equal(t, int64(950), stats.MessagesConsumed)
		assert.Equal(t, int64(5), stats.MessagesFailed)
		assert.Equal(t, 3, stats.ActiveConsumers)
		assert.Equal(t, 10, stats.ActiveTopics)
		assert.Equal(t, "connected", stats.ConnectionStatus)
		assert.Equal(t, lastActivity, stats.LastActivity)
		assert.Equal(t, uptime, stats.Uptime)
		assert.Equal(t, int64(1024000), stats.BytesProduced)
		assert.Equal(t, int64(1020000), stats.BytesConsumed)
	})
}

func TestInterfacesTopicConfig(t *testing.T) {
	t.Run("should create topic config with correct structure", func(t *testing.T) {
		config := map[string]string{
			"cleanup.policy":     "delete",
			"retention.ms":       "86400000",
			"compression.type":   "gzip",
		}

		topicConfig := TopicConfig{
			Partitions:        6,
			ReplicationFactor: 3,
			Config:            config,
		}

		assert.Equal(t, 6, topicConfig.Partitions)
		assert.Equal(t, 3, topicConfig.ReplicationFactor)
		assert.Equal(t, config, topicConfig.Config)
		assert.Equal(t, "delete", topicConfig.Config["cleanup.policy"])
		assert.Equal(t, "86400000", topicConfig.Config["retention.ms"])
		assert.Equal(t, "gzip", topicConfig.Config["compression.type"])
	})

	t.Run("should handle empty config", func(t *testing.T) {
		topicConfig := TopicConfig{
			Partitions:        1,
			ReplicationFactor: 1,
			Config:            nil,
		}

		assert.Equal(t, 1, topicConfig.Partitions)
		assert.Equal(t, 1, topicConfig.ReplicationFactor)
		assert.Nil(t, topicConfig.Config)
	})
}

func TestInterfacesRetryPolicy(t *testing.T) {
	t.Run("should create retry policy with correct structure", func(t *testing.T) {
		policy := RetryPolicy{
			MaxRetries:      5,
			InitialInterval: 500 * time.Millisecond,
			MaxInterval:     30 * time.Second,
			Multiplier:      2.0,
			RandomFactor:    0.1,
		}

		assert.Equal(t, 5, policy.MaxRetries)
		assert.Equal(t, 500*time.Millisecond, policy.InitialInterval)
		assert.Equal(t, 30*time.Second, policy.MaxInterval)
		assert.Equal(t, 2.0, policy.Multiplier)
		assert.Equal(t, 0.1, policy.RandomFactor)
	})

	t.Run("should handle zero values", func(t *testing.T) {
		policy := RetryPolicy{
			MaxRetries:      0,
			InitialInterval: 0,
			MaxInterval:     0,
			Multiplier:      0,
			RandomFactor:    0,
		}

		assert.Equal(t, 0, policy.MaxRetries)
		assert.Equal(t, time.Duration(0), policy.InitialInterval)
		assert.Equal(t, time.Duration(0), policy.MaxInterval)
		assert.Equal(t, 0.0, policy.Multiplier)
		assert.Equal(t, 0.0, policy.RandomFactor)
	})
}

func TestInterfacesConnectionStatus(t *testing.T) {
	t.Run("should handle different connection statuses", func(t *testing.T) {
		statuses := []string{
			"connected",
			"disconnected",
			"connecting",
			"reconnecting",
			"error",
		}

		for _, status := range statuses {
			stats := BrokerStats{
				ConnectionStatus: status,
			}

			assert.Equal(t, status, stats.ConnectionStatus)
		}
	})
}

func TestInterfacesMessageMetadata(t *testing.T) {
	t.Run("should handle various metadata types", func(t *testing.T) {
		metadata := map[string]interface{}{
			"string_value":  "test",
			"int_value":     123,
			"float_value":   45.67,
			"bool_value":    true,
			"array_value":   []string{"a", "b", "c"},
			"object_value":  map[string]string{"key": "value"},
		}

		message := Message{
			ID:       "test",
			Topic:    "test-topic",
			Metadata: metadata,
		}

		assert.Equal(t, "test", message.Metadata["string_value"])
		assert.Equal(t, 123, message.Metadata["int_value"])
		assert.Equal(t, 45.67, message.Metadata["float_value"])
		assert.Equal(t, true, message.Metadata["bool_value"])
		assert.Equal(t, []string{"a", "b", "c"}, message.Metadata["array_value"])
		assert.Equal(t, map[string]string{"key": "value"}, message.Metadata["object_value"])
	})
}

func TestInterfacesKafkaSpecificFields(t *testing.T) {
	t.Run("should handle Kafka-specific message fields", func(t *testing.T) {
		message := Message{
			ID:        "kafka-msg-1",
			Topic:     "user-events",
			Key:       "user-123",
			Value:     []byte(`{"user_id": 123, "action": "login"}`),
			Partition: 2,
			Offset:    1500,
		}

		assert.Equal(t, "kafka-msg-1", message.ID)
		assert.Equal(t, "user-events", message.Topic)
		assert.Equal(t, "user-123", message.Key)
		assert.NotEmpty(t, message.Value)
		assert.Equal(t, int32(2), message.Partition)
		assert.Equal(t, int64(1500), message.Offset)
	})

	t.Run("should handle messages without Kafka fields", func(t *testing.T) {
		message := Message{
			ID:    "simple-msg",
			Topic: "notifications",
			Value: []byte("Hello World"),
		}

		assert.Equal(t, "simple-msg", message.ID)
		assert.Equal(t, "notifications", message.Topic)
		assert.Equal(t, []byte("Hello World"), message.Value)
		assert.Empty(t, message.Key)
		assert.Equal(t, int32(0), message.Partition)
		assert.Equal(t, int64(0), message.Offset)
	})
}

func TestInterfacesMessageHeaders(t *testing.T) {
	t.Run("should handle message headers", func(t *testing.T) {
		headers := map[string]string{
			"Content-Type":     "application/json",
			"Content-Encoding": "gzip",
			"User-Agent":       "MyApp/1.0",
			"Correlation-ID":   "abc-123-def",
			"Retry-Count":      "3",
		}

		message := Message{
			ID:      "header-test",
			Topic:   "api-events",
			Headers: headers,
		}

		assert.Equal(t, "application/json", message.Headers["Content-Type"])
		assert.Equal(t, "gzip", message.Headers["Content-Encoding"])
		assert.Equal(t, "MyApp/1.0", message.Headers["User-Agent"])
		assert.Equal(t, "abc-123-def", message.Headers["Correlation-ID"])
		assert.Equal(t, "3", message.Headers["Retry-Count"])
		assert.Len(t, message.Headers, 5)
	})
}

func TestInterfacesBrokerStatsCalculations(t *testing.T) {
	t.Run("should handle statistical calculations", func(t *testing.T) {
		stats := BrokerStats{
			MessagesProduced: 1000,
			MessagesConsumed: 950,
			MessagesFailed:   50,
			BytesProduced:   1048576, // 1MB
			BytesConsumed:   1000000, // ~1MB
		}

		// Calculate success rate
		totalMessages := stats.MessagesProduced
		successfulMessages := stats.MessagesConsumed
		failureRate := float64(stats.MessagesFailed) / float64(totalMessages) * 100
		successRate := float64(successfulMessages) / float64(totalMessages) * 100

		assert.Equal(t, int64(1000), totalMessages)
		assert.Equal(t, int64(950), successfulMessages)
		assert.Equal(t, 5.0, failureRate)  // 5% failure rate
		assert.Equal(t, 95.0, successRate) // 95% success rate

		// Calculate average message size
		avgProducedSize := float64(stats.BytesProduced) / float64(stats.MessagesProduced)
		avgConsumedSize := float64(stats.BytesConsumed) / float64(stats.MessagesConsumed)

		assert.InDelta(t, 1048.576, avgProducedSize, 0.001) // ~1KB per message
		assert.InDelta(t, 1052.631, avgConsumedSize, 0.001) // ~1KB per message
	})
}