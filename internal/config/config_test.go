package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	// Set required environment variables for test (must be at least 32 characters)
	os.Setenv("JWT_SECRET", "test-secret-key-for-testing-123456789")
	defer os.Unsetenv("JWT_SECRET")

	// Test loading configuration with default values
	config, err := Load()

	// Test that config loads without error
	require.NoError(t, err)
	require.NotNil(t, config)

	// Test default values
	assert.Equal(t, "Go Template", config.App.Name)
	assert.Equal(t, "1.0.0", config.App.Version)
	assert.Equal(t, "localhost", config.Server.Host)
	assert.Equal(t, 8080, config.Server.Port)
}

func TestLoadWithEnvVars(t *testing.T) {
	// Set required and test environment variables (must be at least 32 characters)
	os.Setenv("JWT_SECRET", "test-secret-key-for-testing-123456789")
	os.Setenv("APP_NAME", "Test App")
	os.Setenv("SERVER_PORT", "9000")
	defer func() {
		os.Unsetenv("JWT_SECRET")
		os.Unsetenv("APP_NAME")
		os.Unsetenv("SERVER_PORT")
	}()

	config, err := Load()

	// Test that config loads without error
	require.NoError(t, err)
	require.NotNil(t, config)

	// Test that environment variables are used
	assert.Equal(t, "Test App", config.App.Name)
	assert.Equal(t, 9000, config.Server.Port)
}

func TestGetEnvAsInt(t *testing.T) {
	tests := []struct {
		name         string
		envValue     string
		defaultValue int
		expected     int
	}{
		{"Valid integer", "8080", 3000, 8080},
		{"Invalid integer", "invalid", 3000, 3000},
		{"Empty value", "", 3000, 3000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv("TEST_PORT", tt.envValue)
				defer os.Unsetenv("TEST_PORT")
			}

			result := getEnvAsInt("TEST_PORT", tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetEnvAsBool(t *testing.T) {
	tests := []struct {
		name         string
		envValue     string
		defaultValue bool
		expected     bool
	}{
		{"True value", "true", false, true},
		{"False value", "false", true, false},
		{"1 value", "1", false, true},
		{"0 value", "0", true, false},
		{"Invalid value", "invalid", true, true},
		{"Empty value", "", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv("TEST_BOOL", tt.envValue)
				defer os.Unsetenv("TEST_BOOL")
			}

			result := getEnvAsBool("TEST_BOOL", tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConfigValidation(t *testing.T) {
	// Set required environment variables (must be at least 32 characters)
	os.Setenv("JWT_SECRET", "test-secret-key-for-testing-123456789")
	defer os.Unsetenv("JWT_SECRET")

	config, err := Load()

	// Test that config loads without error
	require.NoError(t, err)
	require.NotNil(t, config)

	// Test that required fields are not empty
	assert.NotEmpty(t, config.App.Name)
	assert.NotEmpty(t, config.App.Version)
	assert.NotZero(t, config.Server.Port)
	assert.NotEmpty(t, config.Database.Driver)
}