package logger

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Run("should create logger with default settings", func(t *testing.T) {
		logger := New("", "")

		assert.NotNil(t, logger)
		assert.NotNil(t, logger.Logger)
		assert.Equal(t, logrus.InfoLevel, logger.Logger.Level)
	})

	t.Run("should set debug log level", func(t *testing.T) {
		logger := New("debug", "text")

		assert.Equal(t, logrus.DebugLevel, logger.Logger.Level)
	})

	t.Run("should set info log level", func(t *testing.T) {
		logger := New("info", "text")

		assert.Equal(t, logrus.InfoLevel, logger.Logger.Level)
	})

	t.Run("should set warn log level", func(t *testing.T) {
		logger := New("warn", "text")

		assert.Equal(t, logrus.WarnLevel, logger.Logger.Level)
	})

	t.Run("should set error log level", func(t *testing.T) {
		logger := New("error", "text")

		assert.Equal(t, logrus.ErrorLevel, logger.Logger.Level)
	})

	t.Run("should default to info level for unknown level", func(t *testing.T) {
		logger := New("unknown", "text")

		assert.Equal(t, logrus.InfoLevel, logger.Logger.Level)
	})

	t.Run("should set JSON formatter", func(t *testing.T) {
		logger := New("info", "json")

		formatter := logger.Logger.Formatter
		_, ok := formatter.(*logrus.JSONFormatter)
		assert.True(t, ok, "Should use JSON formatter")
	})

	t.Run("should set text formatter", func(t *testing.T) {
		logger := New("info", "text")

		formatter := logger.Logger.Formatter
		textFormatter, ok := formatter.(*logrus.TextFormatter)
		assert.True(t, ok, "Should use text formatter")
		assert.True(t, textFormatter.FullTimestamp)
		assert.True(t, textFormatter.ForceColors)
	})

	t.Run("should default to text formatter for unknown format", func(t *testing.T) {
		logger := New("info", "unknown")

		formatter := logger.Logger.Formatter
		_, ok := formatter.(*logrus.TextFormatter)
		assert.True(t, ok, "Should default to text formatter")
	})
}

func TestLogger_LoggingMethods(t *testing.T) {
	tests := []struct {
		name     string
		logLevel string
		method   func(*Logger, string, ...interface{})
		message  string
		expected bool
	}{
		{"debug logs at debug level", "debug", (*Logger).Debug, "debug message", true},
		{"debug doesn't log at info level", "info", (*Logger).Debug, "debug message", false},
		{"info logs at debug level", "debug", (*Logger).Info, "info message", true},
		{"info logs at info level", "info", (*Logger).Info, "info message", true},
		{"info doesn't log at warn level", "warn", (*Logger).Info, "info message", false},
		{"warn logs at debug level", "debug", (*Logger).Warn, "warn message", true},
		{"warn logs at warn level", "warn", (*Logger).Warn, "warn message", true},
		{"warn doesn't log at error level", "error", (*Logger).Warn, "warn message", false},
		{"error logs at all levels", "debug", (*Logger).Error, "error message", true},
		{"error logs at error level", "error", (*Logger).Error, "error message", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := New(tt.logLevel, "text")
			logger.Logger.SetOutput(&buf)

			tt.method(logger, tt.message)

			output := buf.String()
			if tt.expected {
				assert.Contains(t, output, tt.message)
			} else {
				assert.Empty(t, output)
			}
		})
	}
}

func TestLogger_StructuredLogging(t *testing.T) {
	t.Run("should log with structured fields", func(t *testing.T) {
		var buf bytes.Buffer
		logger := New("info", "json")
		logger.Logger.SetOutput(&buf)

		logger.Info("test message", "key1", "value1", "key2", 42)

		output := buf.String()
		assert.Contains(t, output, "test message")

		// Parse JSON to verify structured fields
		var logData map[string]interface{}
		err := json.Unmarshal([]byte(output), &logData)
		require.NoError(t, err)

		assert.Equal(t, "test message", logData["msg"])
		assert.Equal(t, "value1", logData["key1"])
		assert.Equal(t, float64(42), logData["key2"]) // JSON unmarshals numbers as float64
		assert.Equal(t, "info", logData["level"])
	})

	t.Run("should log with multiple structured fields", func(t *testing.T) {
		var buf bytes.Buffer
		logger := New("debug", "json")
		logger.Logger.SetOutput(&buf)

		logger.Debug("debug message",
			"user_id", "123",
			"action", "login",
			"ip", "192.168.1.1",
			"success", true,
			"duration", 150.5)

		output := buf.String()
		var logData map[string]interface{}
		err := json.Unmarshal([]byte(output), &logData)
		require.NoError(t, err)

		assert.Equal(t, "debug message", logData["msg"])
		assert.Equal(t, "123", logData["user_id"])
		assert.Equal(t, "login", logData["action"])
		assert.Equal(t, "192.168.1.1", logData["ip"])
		assert.Equal(t, true, logData["success"])
		assert.Equal(t, 150.5, logData["duration"])
	})

	t.Run("should handle empty fields", func(t *testing.T) {
		var buf bytes.Buffer
		logger := New("info", "json")
		logger.Logger.SetOutput(&buf)

		logger.Info("message without fields")

		output := buf.String()
		var logData map[string]interface{}
		err := json.Unmarshal([]byte(output), &logData)
		require.NoError(t, err)

		assert.Equal(t, "message without fields", logData["msg"])
		assert.Equal(t, "info", logData["level"])
	})

	t.Run("should log with text formatter and fields", func(t *testing.T) {
		var buf bytes.Buffer
		logger := New("info", "text")
		// Disable colors for predictable output
		if textFormatter, ok := logger.Logger.Formatter.(*logrus.TextFormatter); ok {
			textFormatter.DisableColors = true
		}
		logger.Logger.SetOutput(&buf)

		logger.Info("text message", "key", "value", "number", 123)

		output := buf.String()
		assert.Contains(t, output, "text message")
		assert.Contains(t, output, "key=value")
		assert.Contains(t, output, "number=123")
	})
}

func TestParseFields(t *testing.T) {
	t.Run("should parse even number of arguments", func(t *testing.T) {
		fields := parseFields("key1", "value1", "key2", 42, "key3", true)

		expected := logrus.Fields{
			"key1": "value1",
			"key2": 42,
			"key3": true,
		}

		assert.Equal(t, expected, fields)
	})

	t.Run("should handle odd number of arguments", func(t *testing.T) {
		fields := parseFields("key1", "value1", "key2")

		expected := logrus.Fields{
			"key1": "value1",
		}

		assert.Equal(t, expected, fields)
	})

	t.Run("should handle empty arguments", func(t *testing.T) {
		fields := parseFields()

		assert.Empty(t, fields)
	})

	t.Run("should handle non-string keys", func(t *testing.T) {
		fields := parseFields(123, "value1", "key2", "value2")

		expected := logrus.Fields{
			"key2": "value2",
		}

		assert.Equal(t, expected, fields)
	})

	t.Run("should handle mixed types", func(t *testing.T) {
		fields := parseFields(
			"string_key", "string_value",
			"int_key", 42,
			"float_key", 3.14,
			"bool_key", true,
			"nil_key", nil,
		)

		expected := logrus.Fields{
			"string_key": "string_value",
			"int_key":    42,
			"float_key":  3.14,
			"bool_key":   true,
			"nil_key":    nil,
		}

		assert.Equal(t, expected, fields)
	})

	t.Run("should handle single argument", func(t *testing.T) {
		fields := parseFields("lonely_key")

		assert.Empty(t, fields)
	})
}

func TestLogger_IntegrationWithDifferentLevels(t *testing.T) {
	t.Run("should only log appropriate levels", func(t *testing.T) {
		testCases := []struct {
			level    string
			expected []string
			notFound []string
		}{
			{
				level:    "debug",
				expected: []string{"debug", "info", "warning", "error"},
				notFound: []string{},
			},
			{
				level:    "info",
				expected: []string{"info", "warning", "error"},
				notFound: []string{"debug"},
			},
			{
				level:    "warn",
				expected: []string{"warning", "error"},
				notFound: []string{"debug", "info"},
			},
			{
				level:    "error",
				expected: []string{"error"},
				notFound: []string{"debug", "info", "warning"},
			},
		}

		for _, tc := range testCases {
			t.Run("level_"+tc.level, func(t *testing.T) {
				var buf bytes.Buffer
				logger := New(tc.level, "text")
				// Disable colors for predictable output
				if textFormatter, ok := logger.Logger.Formatter.(*logrus.TextFormatter); ok {
					textFormatter.DisableColors = true
				}
				logger.Logger.SetOutput(&buf)

				logger.Debug("debug message")
				logger.Info("info message")
				logger.Warn("warn message")
				logger.Error("error message")

				output := buf.String()

				for _, expectedLevel := range tc.expected {
					assert.Contains(t, output, expectedLevel, "Should contain %s level", expectedLevel)
				}

				for _, notFoundLevel := range tc.notFound {
					assert.NotContains(t, output, notFoundLevel, "Should not contain %s level", notFoundLevel)
				}
			})
		}
	})
}

func TestLogger_JSONFormatOutput(t *testing.T) {
	t.Run("should produce valid JSON for all log levels", func(t *testing.T) {
		var buf bytes.Buffer
		logger := New("debug", "json")
		logger.Logger.SetOutput(&buf)

		logger.Debug("debug message", "debug_field", "debug_value")
		logger.Info("info message", "info_field", "info_value")
		logger.Warn("warn message", "warn_field", "warn_value")
		logger.Error("error message", "error_field", "error_value")

		output := buf.String()
		lines := strings.Split(strings.TrimSpace(output), "\n")

		assert.Len(t, lines, 4, "Should have 4 log lines")

		// Verify each line is valid JSON
		for i, line := range lines {
			var logData map[string]interface{}
			err := json.Unmarshal([]byte(line), &logData)
			require.NoError(t, err, "Line %d should be valid JSON", i+1)

			// Verify required fields
			assert.Contains(t, logData, "msg")
			assert.Contains(t, logData, "level")
			assert.Contains(t, logData, "time")
		}
	})
}

func TestLogger_ErrorHandling(t *testing.T) {
	t.Run("should handle nil values in structured logging", func(t *testing.T) {
		var buf bytes.Buffer
		logger := New("info", "json")
		logger.Logger.SetOutput(&buf)

		logger.Info("test message", "nil_key", nil, "valid_key", "valid_value")

		output := buf.String()
		var logData map[string]interface{}
		err := json.Unmarshal([]byte(output), &logData)
		require.NoError(t, err)

		assert.Equal(t, "test message", logData["msg"])
		assert.Nil(t, logData["nil_key"])
		assert.Equal(t, "valid_value", logData["valid_key"])
	})

	t.Run("should handle complex data structures", func(t *testing.T) {
		var buf bytes.Buffer
		logger := New("info", "json")
		logger.Logger.SetOutput(&buf)

		complexStruct := map[string]interface{}{
			"nested": map[string]string{
				"inner": "value",
			},
			"array": []int{1, 2, 3},
		}

		logger.Info("complex message", "complex_data", complexStruct)

		output := buf.String()
		var logData map[string]interface{}
		err := json.Unmarshal([]byte(output), &logData)
		require.NoError(t, err)

		assert.Equal(t, "complex message", logData["msg"])
		assert.NotNil(t, logData["complex_data"])
	})
}

func TestLogger_ConcurrencySafety(t *testing.T) {
	t.Run("should handle concurrent logging", func(t *testing.T) {
		var buf bytes.Buffer
		logger := New("info", "json")
		logger.Logger.SetOutput(&buf)

		done := make(chan bool, 10)

		// Launch multiple goroutines
		for i := 0; i < 10; i++ {
			go func(id int) {
				logger.Info("concurrent message", "goroutine_id", id, "message", "from goroutine")
				done <- true
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < 10; i++ {
			<-done
		}

		output := buf.String()
		lines := strings.Split(strings.TrimSpace(output), "\n")

		// Should have 10 log entries
		assert.Len(t, lines, 10)

		// Each line should be valid JSON
		for _, line := range lines {
			var logData map[string]interface{}
			err := json.Unmarshal([]byte(line), &logData)
			assert.NoError(t, err, "Each log line should be valid JSON")
		}
	})
}