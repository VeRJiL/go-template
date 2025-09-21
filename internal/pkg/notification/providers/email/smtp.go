package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"mime"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"github.com/VeRJiL/go-template/internal/pkg/notification/providers"
)

// SMTPProvider implements EmailProvider for SMTP
type SMTPProvider struct {
	config *SMTPConfig
	auth   smtp.Auth
}

// SMTPConfig holds SMTP configuration
type SMTPConfig struct {
	Host         string `json:"host" mapstructure:"host"`
	Port         int    `json:"port" mapstructure:"port"`
	Username     string `json:"username" mapstructure:"username"`
	Password     string `json:"password" mapstructure:"password"`
	From         string `json:"from" mapstructure:"from"`
	FromName     string `json:"from_name" mapstructure:"from_name"`
	UseTLS       bool   `json:"use_tls" mapstructure:"use_tls"`
	UseStartTLS  bool   `json:"use_starttls" mapstructure:"use_starttls"`
	InsecureSkip bool   `json:"insecure_skip" mapstructure:"insecure_skip"`
	KeepAlive    bool   `json:"keep_alive" mapstructure:"keep_alive"`
	Timeout      int    `json:"timeout" mapstructure:"timeout"` // seconds
}

// NewSMTPEmailProvider creates a new SMTP email provider
func NewSMTPEmailProvider(config interface{}) (*SMTPProvider, error) {
	smtpConfig, ok := config.(*SMTPConfig)
	if !ok {
		return nil, fmt.Errorf("invalid SMTP configuration type")
	}

	if err := validateSMTPConfig(smtpConfig); err != nil {
		return nil, fmt.Errorf("SMTP configuration validation failed: %w", err)
	}

	// Set defaults
	if smtpConfig.Port == 0 {
		if smtpConfig.UseTLS {
			smtpConfig.Port = 465
		} else {
			smtpConfig.Port = 587
		}
	}

	if smtpConfig.Timeout == 0 {
		smtpConfig.Timeout = 30
	}

	var auth smtp.Auth
	if smtpConfig.Username != "" && smtpConfig.Password != "" {
		auth = smtp.PlainAuth("", smtpConfig.Username, smtpConfig.Password, smtpConfig.Host)
	}

	provider := &SMTPProvider{
		config: smtpConfig,
		auth:   auth,
	}

	return provider, nil
}

// Send sends an email message via SMTP
func (p *SMTPProvider) Send(ctx context.Context, message *providers.EmailMessage) error {
	// Build the email message
	emailData, err := p.buildEmailMessage(message)
	if err != nil {
		return fmt.Errorf("failed to build email message: %w", err)
	}

	// Get server address
	addr := fmt.Sprintf("%s:%d", p.config.Host, p.config.Port)

	// Send the email
	if p.config.UseTLS {
		return p.sendWithTLS(addr, emailData, message.To)
	} else {
		return p.sendWithStartTLS(addr, emailData, message.To)
	}
}

// SendBatch sends multiple email messages
func (p *SMTPProvider) SendBatch(ctx context.Context, messages []*providers.EmailMessage) error {
	var lastErr error
	for i, message := range messages {
		if err := p.Send(ctx, message); err != nil {
			lastErr = fmt.Errorf("failed to send email %d: %w", i, err)
		}
	}
	return lastErr
}

// GetProviderName returns the provider name
func (p *SMTPProvider) GetProviderName() string {
	return "smtp"
}

// Ping checks if the SMTP server is reachable
func (p *SMTPProvider) Ping(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", p.config.Host, p.config.Port)

	timeout := time.Duration(p.config.Timeout) * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Try to establish a connection
	if p.config.UseTLS {
		tlsConfig := &tls.Config{
			ServerName:         p.config.Host,
			InsecureSkipVerify: p.config.InsecureSkip,
		}

		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("TLS connection failed: %w", err)
		}
		defer conn.Close()

		client, err := smtp.NewClient(conn, p.config.Host)
		if err != nil {
			return fmt.Errorf("SMTP client creation failed: %w", err)
		}
		defer client.Quit()
	} else {
		client, err := smtp.Dial(addr)
		if err != nil {
			return fmt.Errorf("SMTP connection failed: %w", err)
		}
		defer client.Quit()

		if p.config.UseStartTLS {
			tlsConfig := &tls.Config{
				ServerName:         p.config.Host,
				InsecureSkipVerify: p.config.InsecureSkip,
			}

			if err := client.StartTLS(tlsConfig); err != nil {
				return fmt.Errorf("StartTLS failed: %w", err)
			}
		}
	}

	return nil
}

// Close closes the SMTP provider (no-op for SMTP)
func (p *SMTPProvider) Close() error {
	return nil
}

// Helper methods

// sendWithTLS sends email using direct TLS connection
func (p *SMTPProvider) sendWithTLS(addr string, emailData []byte, to []string) error {
	tlsConfig := &tls.Config{
		ServerName:         p.config.Host,
		InsecureSkipVerify: p.config.InsecureSkip,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("TLS dial failed: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, p.config.Host)
	if err != nil {
		return fmt.Errorf("SMTP client creation failed: %w", err)
	}
	defer client.Quit()

	return p.sendWithClient(client, emailData, to)
}

// sendWithStartTLS sends email using STARTTLS
func (p *SMTPProvider) sendWithStartTLS(addr string, emailData []byte, to []string) error {
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("SMTP dial failed: %w", err)
	}
	defer client.Quit()

	if p.config.UseStartTLS {
		tlsConfig := &tls.Config{
			ServerName:         p.config.Host,
			InsecureSkipVerify: p.config.InsecureSkip,
		}

		if err := client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("StartTLS failed: %w", err)
		}
	}

	return p.sendWithClient(client, emailData, to)
}

// sendWithClient sends email using an established SMTP client
func (p *SMTPProvider) sendWithClient(client *smtp.Client, emailData []byte, to []string) error {
	// Authenticate if credentials are provided
	if p.auth != nil {
		if err := client.Auth(p.auth); err != nil {
			return fmt.Errorf("SMTP authentication failed: %w", err)
		}
	}

	// Set sender
	from := p.config.From
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// Set recipients
	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("failed to set recipient %s: %w", recipient, err)
		}
	}

	// Send data
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to open data writer: %w", err)
	}

	if _, err := writer.Write(emailData); err != nil {
		writer.Close()
		return fmt.Errorf("failed to write email data: %w", err)
	}

	return writer.Close()
}

// buildEmailMessage builds the email message with headers and body
func (p *SMTPProvider) buildEmailMessage(message *providers.EmailMessage) ([]byte, error) {
	var email strings.Builder

	// From header
	from := message.From
	if from == "" {
		from = p.config.From
	}
	if p.config.FromName != "" || message.FromName != "" {
		fromName := message.FromName
		if fromName == "" {
			fromName = p.config.FromName
		}
		email.WriteString(fmt.Sprintf("From: %s <%s>\r\n", fromName, from))
	} else {
		email.WriteString(fmt.Sprintf("From: %s\r\n", from))
	}

	// To header
	email.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(message.To, ", ")))

	// CC header
	if len(message.CC) > 0 {
		email.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(message.CC, ", ")))
	}

	// Subject header
	email.WriteString(fmt.Sprintf("Subject: %s\r\n", message.Subject))

	// Reply-To header
	if message.ReplyTo != "" {
		email.WriteString(fmt.Sprintf("Reply-To: %s\r\n", message.ReplyTo))
	}

	// Date header
	email.WriteString(fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123Z)))

	// Message-ID header
	email.WriteString(fmt.Sprintf("Message-ID: <%d.%s@%s>\r\n",
		time.Now().UnixNano(), generateRandomString(10), p.config.Host))

	// Custom headers
	for key, value := range message.Headers {
		email.WriteString(fmt.Sprintf("%s: %s\r\n", key, value))
	}

	// MIME headers
	email.WriteString("MIME-Version: 1.0\r\n")

	// Handle attachments and content type
	if len(message.Attachments) > 0 {
		boundary := generateBoundary()
		email.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=%s\r\n", boundary))
		email.WriteString("\r\n")

		// Message body part
		email.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		if message.HTMLBody != "" && message.TextBody != "" {
			// Both HTML and text
			textBoundary := generateBoundary()
			email.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=%s\r\n", textBoundary))
			email.WriteString("\r\n")

			// Text part
			email.WriteString(fmt.Sprintf("--%s\r\n", textBoundary))
			email.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
			email.WriteString("\r\n")
			email.WriteString(message.TextBody)
			email.WriteString("\r\n")

			// HTML part
			email.WriteString(fmt.Sprintf("--%s\r\n", textBoundary))
			email.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
			email.WriteString("\r\n")
			email.WriteString(message.HTMLBody)
			email.WriteString("\r\n")

			email.WriteString(fmt.Sprintf("--%s--\r\n", textBoundary))
		} else if message.HTMLBody != "" {
			// HTML only
			email.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
			email.WriteString("\r\n")
			email.WriteString(message.HTMLBody)
			email.WriteString("\r\n")
		} else {
			// Text only
			email.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
			email.WriteString("\r\n")
			email.WriteString(message.Body)
			email.WriteString("\r\n")
		}

		// Attachments
		for _, attachment := range message.Attachments {
			email.WriteString(fmt.Sprintf("--%s\r\n", boundary))
			email.WriteString(fmt.Sprintf("Content-Type: %s\r\n", attachment.ContentType))
			email.WriteString("Content-Transfer-Encoding: base64\r\n")
			if attachment.ContentID != "" {
				email.WriteString(fmt.Sprintf("Content-ID: <%s>\r\n", attachment.ContentID))
			}
			disposition := attachment.Disposition
			if disposition == "" {
				disposition = "attachment"
			}
			email.WriteString(fmt.Sprintf("Content-Disposition: %s; filename=%s\r\n",
				disposition, mime.QEncoding.Encode("UTF-8", attachment.Filename)))
			email.WriteString("\r\n")
			email.WriteString(encodeBase64(attachment.Content))
			email.WriteString("\r\n")
		}

		email.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
	} else {
		// No attachments
		if message.HTMLBody != "" && message.TextBody != "" {
			// Both HTML and text
			boundary := generateBoundary()
			email.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=%s\r\n", boundary))
			email.WriteString("\r\n")

			// Text part
			email.WriteString(fmt.Sprintf("--%s\r\n", boundary))
			email.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
			email.WriteString("\r\n")
			email.WriteString(message.TextBody)
			email.WriteString("\r\n")

			// HTML part
			email.WriteString(fmt.Sprintf("--%s\r\n", boundary))
			email.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
			email.WriteString("\r\n")
			email.WriteString(message.HTMLBody)
			email.WriteString("\r\n")

			email.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
		} else if message.HTMLBody != "" {
			// HTML only
			email.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
			email.WriteString("\r\n")
			email.WriteString(message.HTMLBody)
		} else {
			// Text only
			email.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
			email.WriteString("\r\n")
			email.WriteString(message.Body)
		}
	}

	return []byte(email.String()), nil
}

// validateSMTPConfig validates SMTP configuration
func validateSMTPConfig(config *SMTPConfig) error {
	if config.Host == "" {
		return fmt.Errorf("SMTP host is required")
	}

	if config.From == "" {
		return fmt.Errorf("from address is required")
	}

	return nil
}

// Helper functions

func generateBoundary() string {
	return fmt.Sprintf("boundary_%d_%s", time.Now().Unix(), generateRandomString(10))
}

func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(result)
}

func encodeBase64(data []byte) string {
	// Simple base64 encoding in chunks of 76 characters per line
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

	var result strings.Builder
	for i := 0; i < len(data); i += 3 {
		var b1, b2, b3 byte
		b1 = data[i]
		if i+1 < len(data) {
			b2 = data[i+1]
		}
		if i+2 < len(data) {
			b3 = data[i+2]
		}

		// Convert 3 bytes to 4 base64 characters
		c1 := b1 >> 2
		c2 := ((b1 & 0x03) << 4) | (b2 >> 4)
		c3 := ((b2 & 0x0f) << 2) | (b3 >> 6)
		c4 := b3 & 0x3f

		result.WriteByte(charset[c1])
		result.WriteByte(charset[c2])
		if i+1 < len(data) {
			result.WriteByte(charset[c3])
		} else {
			result.WriteByte('=')
		}
		if i+2 < len(data) {
			result.WriteByte(charset[c4])
		} else {
			result.WriteByte('=')
		}

		// Add line break every 76 characters
		if (result.Len())%76 == 0 {
			result.WriteString("\r\n")
		}
	}

	return result.String()
}