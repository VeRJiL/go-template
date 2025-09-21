package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/VeRJiL/go-template/internal/pkg/notification/providers"
)

// SendGridProvider implements EmailProvider for SendGrid
type SendGridProvider struct {
	config     *SendGridConfig
	httpClient *http.Client
}

// SendGridConfig holds SendGrid configuration
type SendGridConfig struct {
	APIKey      string `json:"api_key" mapstructure:"api_key"`
	FromEmail   string `json:"from_email" mapstructure:"from_email"`
	FromName    string `json:"from_name" mapstructure:"from_name"`
	ReplyTo     string `json:"reply_to" mapstructure:"reply_to"`
	TemplateDir string `json:"template_dir" mapstructure:"template_dir"`
	Timeout     int    `json:"timeout" mapstructure:"timeout"` // seconds
}

// SendGridMessage represents a SendGrid API message
type SendGridMessage struct {
	Personalizations []SendGridPersonalization `json:"personalizations"`
	From             SendGridEmail             `json:"from"`
	ReplyTo          *SendGridEmail            `json:"reply_to,omitempty"`
	Subject          string                    `json:"subject"`
	Content          []SendGridContent         `json:"content"`
	Attachments      []SendGridAttachment      `json:"attachments,omitempty"`
	Categories       []string                  `json:"categories,omitempty"`
	CustomArgs       map[string]string         `json:"custom_args,omitempty"`
	Headers          map[string]string         `json:"headers,omitempty"`
	TemplateID       string                    `json:"template_id,omitempty"`
}

// SendGridPersonalization represents SendGrid personalization
type SendGridPersonalization struct {
	To                  []SendGridEmail           `json:"to"`
	CC                  []SendGridEmail           `json:"cc,omitempty"`
	BCC                 []SendGridEmail           `json:"bcc,omitempty"`
	Subject             string                    `json:"subject,omitempty"`
	Headers             map[string]string         `json:"headers,omitempty"`
	Substitutions       map[string]string         `json:"substitutions,omitempty"`
	DynamicTemplateData map[string]interface{}    `json:"dynamic_template_data,omitempty"`
}

// SendGridEmail represents a SendGrid email address
type SendGridEmail struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

// SendGridContent represents SendGrid email content
type SendGridContent struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// SendGridAttachment represents a SendGrid attachment
type SendGridAttachment struct {
	Content     string `json:"content"`
	Type        string `json:"type"`
	Filename    string `json:"filename"`
	Disposition string `json:"disposition,omitempty"`
	ContentID   string `json:"content_id,omitempty"`
}

// SendGridResponse represents SendGrid API response
type SendGridResponse struct {
	MessageID string `json:"message_id,omitempty"`
	Errors    []struct {
		Message string `json:"message"`
		Field   string `json:"field,omitempty"`
		Help    string `json:"help,omitempty"`
	} `json:"errors,omitempty"`
}

// NewSendGridEmailProvider creates a new SendGrid email provider
func NewSendGridEmailProvider(config interface{}) (*SendGridProvider, error) {
	sendGridConfig, ok := config.(*SendGridConfig)
	if !ok {
		return nil, fmt.Errorf("invalid SendGrid configuration type")
	}

	if err := validateSendGridConfig(sendGridConfig); err != nil {
		return nil, fmt.Errorf("SendGrid configuration validation failed: %w", err)
	}

	// Set defaults
	if sendGridConfig.Timeout == 0 {
		sendGridConfig.Timeout = 30
	}

	httpClient := &http.Client{
		Timeout: time.Duration(sendGridConfig.Timeout) * time.Second,
	}

	provider := &SendGridProvider{
		config:     sendGridConfig,
		httpClient: httpClient,
	}

	return provider, nil
}

// Send sends an email message via SendGrid
func (p *SendGridProvider) Send(ctx context.Context, message *providers.EmailMessage) error {
	// Convert to SendGrid message format
	sgMessage, err := p.convertToSendGridMessage(message)
	if err != nil {
		return fmt.Errorf("failed to convert message: %w", err)
	}

	// Send via SendGrid API
	return p.sendViaSendGridAPI(ctx, sgMessage)
}

// SendBatch sends multiple email messages
func (p *SendGridProvider) SendBatch(ctx context.Context, messages []*providers.EmailMessage) error {
	var lastErr error
	for i, message := range messages {
		if err := p.Send(ctx, message); err != nil {
			lastErr = fmt.Errorf("failed to send email %d: %w", i, err)
		}
	}
	return lastErr
}

// GetProviderName returns the provider name
func (p *SendGridProvider) GetProviderName() string {
	return "sendgrid"
}

// Ping checks if SendGrid API is reachable
func (p *SendGridProvider) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.sendgrid.com/v3", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("SendGrid API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("SendGrid API returned status: %d", resp.StatusCode)
	}

	return nil
}

// Close closes the SendGrid provider
func (p *SendGridProvider) Close() error {
	// No cleanup needed for HTTP client
	return nil
}

// Helper methods

// convertToSendGridMessage converts EmailMessage to SendGrid format
func (p *SendGridProvider) convertToSendGridMessage(message *providers.EmailMessage) (*SendGridMessage, error) {
	sgMessage := &SendGridMessage{
		From: SendGridEmail{
			Email: p.getFromEmail(message),
			Name:  p.getFromName(message),
		},
		Subject: message.Subject,
		Content: make([]SendGridContent, 0),
		Headers: message.Headers,
	}

	// Set reply-to
	replyTo := message.ReplyTo
	if replyTo == "" {
		replyTo = p.config.ReplyTo
	}
	if replyTo != "" {
		sgMessage.ReplyTo = &SendGridEmail{Email: replyTo}
	}

	// Personalization
	personalization := SendGridPersonalization{
		To: make([]SendGridEmail, len(message.To)),
	}

	for i, email := range message.To {
		personalization.To[i] = SendGridEmail{Email: email}
	}

	// CC recipients
	if len(message.CC) > 0 {
		personalization.CC = make([]SendGridEmail, len(message.CC))
		for i, email := range message.CC {
			personalization.CC[i] = SendGridEmail{Email: email}
		}
	}

	// BCC recipients
	if len(message.BCC) > 0 {
		personalization.BCC = make([]SendGridEmail, len(message.BCC))
		for i, email := range message.BCC {
			personalization.BCC[i] = SendGridEmail{Email: email}
		}
	}

	// Template variables
	if len(message.TemplateVars) > 0 {
		personalization.DynamicTemplateData = message.TemplateVars
	}

	sgMessage.Personalizations = []SendGridPersonalization{personalization}

	// Content
	if message.TextBody != "" {
		sgMessage.Content = append(sgMessage.Content, SendGridContent{
			Type:  "text/plain",
			Value: message.TextBody,
		})
	}

	if message.HTMLBody != "" {
		sgMessage.Content = append(sgMessage.Content, SendGridContent{
			Type:  "text/html",
			Value: message.HTMLBody,
		})
	} else if message.Body != "" {
		// Default to HTML if no explicit text/HTML distinction
		sgMessage.Content = append(sgMessage.Content, SendGridContent{
			Type:  "text/html",
			Value: message.Body,
		})
	}

	// Attachments
	if len(message.Attachments) > 0 {
		sgMessage.Attachments = make([]SendGridAttachment, len(message.Attachments))
		for i, att := range message.Attachments {
			sgMessage.Attachments[i] = SendGridAttachment{
				Content:     encodeBase64Simple(att.Content),
				Type:        att.ContentType,
				Filename:    att.Filename,
				Disposition: att.Disposition,
				ContentID:   att.ContentID,
			}
		}
	}

	// Categories (tags)
	if len(message.Tags) > 0 {
		sgMessage.Categories = message.Tags
	}

	// Custom args (metadata)
	if len(message.Metadata) > 0 {
		sgMessage.CustomArgs = message.Metadata
	}

	// Template ID
	if message.Template != "" {
		sgMessage.TemplateID = message.Template
	}

	return sgMessage, nil
}

// sendViaSendGridAPI sends message via SendGrid API
func (p *SendGridProvider) sendViaSendGridAPI(ctx context.Context, message *SendGridMessage) error {
	// Serialize message to JSON
	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.sendgrid.com/v3/mail/send", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("SendGrid API request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode >= 400 {
		var sgResp SendGridResponse
		if err := json.Unmarshal(respBody, &sgResp); err == nil && len(sgResp.Errors) > 0 {
			return fmt.Errorf("SendGrid API error: %s", sgResp.Errors[0].Message)
		}
		return fmt.Errorf("SendGrid API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// getFromEmail returns the from email address
func (p *SendGridProvider) getFromEmail(message *providers.EmailMessage) string {
	if message.From != "" {
		return message.From
	}
	return p.config.FromEmail
}

// getFromName returns the from name
func (p *SendGridProvider) getFromName(message *providers.EmailMessage) string {
	if message.FromName != "" {
		return message.FromName
	}
	return p.config.FromName
}

// validateSendGridConfig validates SendGrid configuration
func validateSendGridConfig(config *SendGridConfig) error {
	if config.APIKey == "" {
		return fmt.Errorf("SendGrid API key is required")
	}

	if config.FromEmail == "" {
		return fmt.Errorf("from email address is required")
	}

	return nil
}

// encodeBase64Simple provides a simple base64 encoding
func encodeBase64Simple(data []byte) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

	var result []byte
	for i := 0; i < len(data); i += 3 {
		var b1, b2, b3 byte
		b1 = data[i]
		if i+1 < len(data) {
			b2 = data[i+1]
		}
		if i+2 < len(data) {
			b3 = data[i+2]
		}

		c1 := b1 >> 2
		c2 := ((b1 & 0x03) << 4) | (b2 >> 4)
		c3 := ((b2 & 0x0f) << 2) | (b3 >> 6)
		c4 := b3 & 0x3f

		result = append(result, charset[c1])
		result = append(result, charset[c2])
		if i+1 < len(data) {
			result = append(result, charset[c3])
		} else {
			result = append(result, '=')
		}
		if i+2 < len(data) {
			result = append(result, charset[c4])
		} else {
			result = append(result, '=')
		}
	}

	return string(result)
}