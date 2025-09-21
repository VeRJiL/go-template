package providers

import (
	"context"
)

// SMSProvider defines the interface for SMS providers
type SMSProvider interface {
	// Send sends an SMS message
	Send(ctx context.Context, message *SMSMessage) error

	// SendBatch sends multiple SMS messages
	SendBatch(ctx context.Context, messages []*SMSMessage) error

	// GetProviderName returns the provider name
	GetProviderName() string

	// Ping checks if the provider connection is alive
	Ping(ctx context.Context) error

	// Close closes the provider connection
	Close() error
}

// SMSMessage represents an SMS message
type SMSMessage struct {
	To         []string          `json:"to"`
	From       string            `json:"from,omitempty"`
	Body       string            `json:"body"`
	MediaURLs  []string          `json:"media_urls,omitempty"` // For MMS
	Tags       []string          `json:"tags,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	StatusCallback string         `json:"status_callback,omitempty"`
	ValidityPeriod int            `json:"validity_period,omitempty"` // in minutes
}

// SMSResponse represents the response from an SMS provider
type SMSResponse struct {
	MessageID    string                 `json:"message_id"`
	Status       string                 `json:"status"`
	To           string                 `json:"to"`
	From         string                 `json:"from"`
	Cost         float64                `json:"cost,omitempty"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// Factory functions for creating SMS providers

// NewTwilioSMSProvider creates a new Twilio SMS provider
func NewTwilioSMSProvider(config interface{}) (SMSProvider, error) {
	return NewTwilioProvider(config)
}

// NewAWSSNSProvider creates a new AWS SNS SMS provider
func NewAWSSNSProvider(config interface{}) (SMSProvider, error) {
	return NewAWSSNSSMSProvider(config)
}

// NewNexmoSMSProvider creates a new Nexmo SMS provider
func NewNexmoSMSProvider(config interface{}) (SMSProvider, error) {
	return NewNexmoProvider(config)
}

// NewTextMagicSMSProvider creates a new TextMagic SMS provider
func NewTextMagicSMSProvider(config interface{}) (SMSProvider, error) {
	return NewTextMagicProvider(config)
}

// These will be implemented by importing the SMS package implementations
var (
	NewTwilioProvider     func(interface{}) (SMSProvider, error)
	NewAWSSNSSMSProvider  func(interface{}) (SMSProvider, error)
	NewNexmoProvider      func(interface{}) (SMSProvider, error)
	NewTextMagicProvider  func(interface{}) (SMSProvider, error)
)