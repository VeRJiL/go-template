package drivers

import (
	"context"
	"fmt"
	"time"

	"github.com/VeRJiL/go-template/internal/pkg/notification"
	"github.com/VeRJiL/go-template/internal/pkg/notification/providers"
)

// EmailDriver handles email notifications with multiple provider support
type EmailDriver struct {
	provider    providers.EmailProvider
	driverName  string
	providerName string
	stats       *notification.DriverStats
	startTime   time.Time
}

// NewEmailDriver creates a new email driver
func NewEmailDriver(providerName string, config interface{}) (*EmailDriver, error) {
	var provider providers.EmailProvider
	var err error

	switch providerName {
	case "smtp":
		provider, err = providers.NewSMTPProvider(config)
	case "sendgrid":
		provider, err = providers.NewSendGridProvider(config)
	case "mailgun":
		provider, err = providers.NewMailgunProvider(config)
	case "aws_ses":
		provider, err = providers.NewAWSSESProvider(config)
	default:
		return nil, fmt.Errorf("unsupported email provider: %s", providerName)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create email provider %s: %w", providerName, err)
	}

	driver := &EmailDriver{
		provider:     provider,
		driverName:   "email",
		providerName: providerName,
		startTime:    time.Now(),
		stats: &notification.DriverStats{
			TotalSent:      0,
			TotalFailed:    0,
			TotalDelivered: 0,
			AverageLatency: 0,
			ErrorRate:      0,
			Uptime:         0,
			ByType:         make(map[string]int64),
			ByPriority:     make(map[notification.Priority]int64),
		},
	}

	return driver, nil
}

// Send sends an email notification
func (d *EmailDriver) Send(ctx context.Context, notif *notification.Notification) error {
	start := time.Now()

	// Validate notification
	if err := d.validateEmailNotification(notif); err != nil {
		d.updateStats(false, time.Since(start), err.Error())
		return fmt.Errorf("validation failed: %w", err)
	}

	// Convert to email message
	emailMsg, err := d.convertToEmailMessage(notif)
	if err != nil {
		d.updateStats(false, time.Since(start), err.Error())
		return fmt.Errorf("failed to convert notification: %w", err)
	}

	// Send via provider
	err = d.provider.Send(ctx, emailMsg)
	if err != nil {
		d.updateStats(false, time.Since(start), err.Error())
		return fmt.Errorf("failed to send email: %w", err)
	}

	d.updateStats(true, time.Since(start), "")
	return nil
}

// SendAsync sends an email notification asynchronously
func (d *EmailDriver) SendAsync(ctx context.Context, notif *notification.Notification) error {
	go func() {
		// Create a new context with timeout for the goroutine
		asyncCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := d.Send(asyncCtx, notif); err != nil {
			// In a real implementation, you might want to log this error
			// or send it to a monitoring service
			fmt.Printf("Async email send failed: %v\n", err)
		}
	}()

	return nil
}

// SendBatch sends multiple email notifications
func (d *EmailDriver) SendBatch(ctx context.Context, notifications []*notification.Notification) error {
	var lastErr error
	successCount := 0

	for i, notif := range notifications {
		if err := d.Send(ctx, notif); err != nil {
			lastErr = fmt.Errorf("failed to send email %d: %w", i, err)
		} else {
			successCount++
		}
	}

	if successCount == 0 && len(notifications) > 0 {
		return fmt.Errorf("all email notifications failed, last error: %w", lastErr)
	}

	return nil
}

// SendScheduled sends an email notification at a specific time
func (d *EmailDriver) SendScheduled(ctx context.Context, notif *notification.Notification, sendAt time.Time) error {
	// Calculate delay
	delay := time.Until(sendAt)
	if delay <= 0 {
		// Send immediately if time has already passed
		return d.Send(ctx, notif)
	}

	// Schedule the email
	go func() {
		timer := time.NewTimer(delay)
		defer timer.Stop()

		select {
		case <-timer.C:
			// Create a new context for the scheduled send
			schedCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			if err := d.Send(schedCtx, notif); err != nil {
				fmt.Printf("Scheduled email send failed: %v\n", err)
			}
		case <-ctx.Done():
			// Context was cancelled before send time
			return
		}
	}()

	return nil
}

// GetStats returns driver statistics
func (d *EmailDriver) GetStats() (*notification.DriverStats, error) {
	d.stats.Uptime = time.Since(d.startTime)

	// Calculate error rate
	total := d.stats.TotalSent + d.stats.TotalFailed
	if total > 0 {
		d.stats.ErrorRate = float64(d.stats.TotalFailed) / float64(total) * 100
	}

	// Create a copy to avoid race conditions
	statsCopy := &notification.DriverStats{
		TotalSent:      d.stats.TotalSent,
		TotalFailed:    d.stats.TotalFailed,
		TotalDelivered: d.stats.TotalDelivered,
		AverageLatency: d.stats.AverageLatency,
		ErrorRate:      d.stats.ErrorRate,
		LastError:      d.stats.LastError,
		LastErrorAt:    d.stats.LastErrorAt,
		Uptime:         d.stats.Uptime,
		ByType:         make(map[string]int64),
		ByPriority:     make(map[notification.Priority]int64),
	}

	for k, v := range d.stats.ByType {
		statsCopy.ByType[k] = v
	}
	for k, v := range d.stats.ByPriority {
		statsCopy.ByPriority[k] = v
	}

	return statsCopy, nil
}

// Ping checks if the email provider connection is alive
func (d *EmailDriver) Ping(ctx context.Context) error {
	return d.provider.Ping(ctx)
}

// GetDriverName returns the driver name
func (d *EmailDriver) GetDriverName() string {
	return d.driverName
}

// GetProviderName returns the provider name
func (d *EmailDriver) GetProviderName() string {
	return d.providerName
}

// Close closes the email driver
func (d *EmailDriver) Close() error {
	if d.provider != nil {
		return d.provider.Close()
	}
	return nil
}

// Helper methods

// validateEmailNotification validates an email notification
func (d *EmailDriver) validateEmailNotification(notif *notification.Notification) error {
	if len(notif.To) == 0 {
		return fmt.Errorf("no recipients specified")
	}

	for _, email := range notif.To {
		if !isValidEmail(email) {
			return fmt.Errorf("invalid email address: %s", email)
		}
	}

	if notif.Subject == "" && notif.Template == "" {
		return fmt.Errorf("either subject or template must be specified")
	}

	if notif.Body == "" && notif.Template == "" {
		return fmt.Errorf("either body or template must be specified")
	}

	return nil
}

// convertToEmailMessage converts a notification to an email message
func (d *EmailDriver) convertToEmailMessage(notif *notification.Notification) (*providers.EmailMessage, error) {
	msg := &providers.EmailMessage{
		To:          notif.To,
		Subject:     notif.Subject,
		Body:        notif.Body,
		HTMLBody:    notif.Body, // For now, assume body is HTML
		Attachments: make([]providers.EmailAttachment, len(notif.Attachments)),
		Headers:     make(map[string]string),
		Tags:        notif.Tags,
		Metadata:    notif.Metadata,
	}

	// Convert attachments
	for i, att := range notif.Attachments {
		msg.Attachments[i] = providers.EmailAttachment{
			Filename:    att.Filename,
			Content:     att.Content,
			ContentType: att.ContentType,
		}
	}

	// Add custom headers from metadata
	for key, value := range notif.Metadata {
		if key[:2] == "X-" { // Custom headers typically start with X-
			msg.Headers[key] = value
		}
	}

	// Handle template rendering if template is specified
	if notif.Template != "" {
		renderedSubject, renderedBody, err := d.renderTemplate(notif.Template, notif.TemplateVars)
		if err != nil {
			return nil, fmt.Errorf("template rendering failed: %w", err)
		}
		if msg.Subject == "" {
			msg.Subject = renderedSubject
		}
		msg.Body = renderedBody
		msg.HTMLBody = renderedBody
	}

	return msg, nil
}

// renderTemplate renders an email template
func (d *EmailDriver) renderTemplate(template string, vars map[string]interface{}) (string, string, error) {
	// This is a simplified template rendering
	// In a real implementation, you would use a proper template engine like html/template

	subject := template + " Subject" // Placeholder
	body := template + " Body"       // Placeholder

	// Apply variables (simplified)
	for key, value := range vars {
		// In a real implementation, you would properly replace template variables
		_ = key
		_ = value
	}

	return subject, body, nil
}

// updateStats updates driver statistics
func (d *EmailDriver) updateStats(success bool, latency time.Duration, errorMsg string) {
	if success {
		d.stats.TotalSent++
		d.stats.TotalDelivered++
	} else {
		d.stats.TotalFailed++
		d.stats.LastError = errorMsg
		now := time.Now()
		d.stats.LastErrorAt = &now
	}

	// Update average latency
	// This is a simplified calculation - in production, you might want to use a more sophisticated approach
	if d.stats.TotalSent > 0 {
		d.stats.AverageLatency = (d.stats.AverageLatency*time.Duration(d.stats.TotalSent-1) + latency) / time.Duration(d.stats.TotalSent)
	}

	d.stats.ByType["email"]++
}

// isValidEmail performs basic email validation
func isValidEmail(email string) bool {
	// This is a very basic validation - in production, use a proper email validation library
	return len(email) > 0 &&
		   len(email) <= 254 &&
		   contains(email, "@") &&
		   contains(email, ".") &&
		   email[0] != '@' &&
		   email[len(email)-1] != '@'
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}