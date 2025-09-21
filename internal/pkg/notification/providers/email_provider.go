package providers

import (
	"context"
)

// EmailProvider defines the interface for email providers
type EmailProvider interface {
	// Send sends an email message
	Send(ctx context.Context, message *EmailMessage) error

	// SendBatch sends multiple email messages
	SendBatch(ctx context.Context, messages []*EmailMessage) error

	// GetProviderName returns the provider name
	GetProviderName() string

	// Ping checks if the provider connection is alive
	Ping(ctx context.Context) error

	// Close closes the provider connection
	Close() error
}

// EmailMessage represents an email message
type EmailMessage struct {
	To          []string                   `json:"to"`
	CC          []string                   `json:"cc,omitempty"`
	BCC         []string                   `json:"bcc,omitempty"`
	From        string                     `json:"from,omitempty"`
	FromName    string                     `json:"from_name,omitempty"`
	ReplyTo     string                     `json:"reply_to,omitempty"`
	Subject     string                     `json:"subject"`
	Body        string                     `json:"body"`
	HTMLBody    string                     `json:"html_body,omitempty"`
	TextBody    string                     `json:"text_body,omitempty"`
	Attachments []EmailAttachment          `json:"attachments,omitempty"`
	Headers     map[string]string          `json:"headers,omitempty"`
	Tags        []string                   `json:"tags,omitempty"`
	Metadata    map[string]string          `json:"metadata,omitempty"`
	TrackOpens  bool                       `json:"track_opens,omitempty"`
	TrackClicks bool                       `json:"track_clicks,omitempty"`
	Template    string                     `json:"template,omitempty"`
	TemplateVars map[string]interface{}    `json:"template_vars,omitempty"`
}

// EmailAttachment represents an email attachment
type EmailAttachment struct {
	Filename    string `json:"filename"`
	Content     []byte `json:"content"`
	ContentType string `json:"content_type"`
	ContentID   string `json:"content_id,omitempty"`
	Disposition string `json:"disposition,omitempty"` // attachment or inline
}

// EmailResponse represents the response from an email provider
type EmailResponse struct {
	MessageID    string                 `json:"message_id"`
	Status       string                 `json:"status"`
	Accepted     []string               `json:"accepted,omitempty"`
	Rejected     []string               `json:"rejected,omitempty"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// Factory functions for creating email providers

// NewSMTPProvider creates a new SMTP email provider
func NewSMTPProvider(config interface{}) (EmailProvider, error) {
	// Import the email package providers
	return NewSMTPEmailProvider(config)
}

// NewSendGridProvider creates a new SendGrid email provider
func NewSendGridProvider(config interface{}) (EmailProvider, error) {
	return NewSendGridEmailProvider(config)
}

// NewMailgunProvider creates a new Mailgun email provider
func NewMailgunProvider(config interface{}) (EmailProvider, error) {
	return NewMailgunEmailProvider(config)
}

// NewAWSSESProvider creates a new AWS SES email provider
func NewAWSSESProvider(config interface{}) (EmailProvider, error) {
	return NewAWSSESEmailProvider(config)
}

// These will be implemented by importing the email package implementations
var (
	NewSMTPEmailProvider     func(interface{}) (EmailProvider, error)
	NewSendGridEmailProvider func(interface{}) (EmailProvider, error)
	NewMailgunEmailProvider  func(interface{}) (EmailProvider, error)
	NewAWSSESEmailProvider   func(interface{}) (EmailProvider, error)
)