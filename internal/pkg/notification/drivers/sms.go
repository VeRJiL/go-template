package drivers

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/VeRJiL/go-template/internal/pkg/notification"
	"github.com/VeRJiL/go-template/internal/pkg/notification/providers"
)

// SMSDriver handles SMS notifications with multiple provider support
type SMSDriver struct {
	provider     providers.SMSProvider
	driverName   string
	providerName string
	stats        *notification.DriverStats
	startTime    time.Time
}

// NewSMSDriver creates a new SMS driver
func NewSMSDriver(providerName string, config interface{}) (*SMSDriver, error) {
	var provider providers.SMSProvider
	var err error

	switch providerName {
	case "twilio":
		provider, err = providers.NewTwilioSMSProvider(config)
	case "aws_sns":
		provider, err = providers.NewAWSSNSProvider(config)
	case "nexmo":
		provider, err = providers.NewNexmoSMSProvider(config)
	case "textmagic":
		provider, err = providers.NewTextMagicSMSProvider(config)
	default:
		return nil, fmt.Errorf("unsupported SMS provider: %s", providerName)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create SMS provider %s: %w", providerName, err)
	}

	driver := &SMSDriver{
		provider:     provider,
		driverName:   "sms",
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

// Send sends an SMS notification
func (d *SMSDriver) Send(ctx context.Context, notif *notification.Notification) error {
	start := time.Now()

	// Validate notification
	if err := d.validateSMSNotification(notif); err != nil {
		d.updateStats(false, time.Since(start), err.Error())
		return fmt.Errorf("validation failed: %w", err)
	}

	// Convert to SMS message
	smsMsg, err := d.convertToSMSMessage(notif)
	if err != nil {
		d.updateStats(false, time.Since(start), err.Error())
		return fmt.Errorf("failed to convert notification: %w", err)
	}

	// Send via provider
	err = d.provider.Send(ctx, smsMsg)
	if err != nil {
		d.updateStats(false, time.Since(start), err.Error())
		return fmt.Errorf("failed to send SMS: %w", err)
	}

	d.updateStats(true, time.Since(start), "")
	return nil
}

// SendAsync sends an SMS notification asynchronously
func (d *SMSDriver) SendAsync(ctx context.Context, notif *notification.Notification) error {
	go func() {
		// Create a new context with timeout for the goroutine
		asyncCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := d.Send(asyncCtx, notif); err != nil {
			// In a real implementation, you might want to log this error
			// or send it to a monitoring service
			fmt.Printf("Async SMS send failed: %v\n", err)
		}
	}()

	return nil
}

// SendBatch sends multiple SMS notifications
func (d *SMSDriver) SendBatch(ctx context.Context, notifications []*notification.Notification) error {
	var lastErr error
	successCount := 0

	for i, notif := range notifications {
		if err := d.Send(ctx, notif); err != nil {
			lastErr = fmt.Errorf("failed to send SMS %d: %w", i, err)
		} else {
			successCount++
		}
	}

	if successCount == 0 && len(notifications) > 0 {
		return fmt.Errorf("all SMS notifications failed, last error: %w", lastErr)
	}

	return nil
}

// SendScheduled sends an SMS notification at a specific time
func (d *SMSDriver) SendScheduled(ctx context.Context, notif *notification.Notification, sendAt time.Time) error {
	// Calculate delay
	delay := time.Until(sendAt)
	if delay <= 0 {
		// Send immediately if time has already passed
		return d.Send(ctx, notif)
	}

	// Schedule the SMS
	go func() {
		timer := time.NewTimer(delay)
		defer timer.Stop()

		select {
		case <-timer.C:
			// Create a new context for the scheduled send
			schedCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			if err := d.Send(schedCtx, notif); err != nil {
				fmt.Printf("Scheduled SMS send failed: %v\n", err)
			}
		case <-ctx.Done():
			// Context was cancelled before send time
			return
		}
	}()

	return nil
}

// GetStats returns driver statistics
func (d *SMSDriver) GetStats() (*notification.DriverStats, error) {
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

// Ping checks if the SMS provider connection is alive
func (d *SMSDriver) Ping(ctx context.Context) error {
	return d.provider.Ping(ctx)
}

// GetDriverName returns the driver name
func (d *SMSDriver) GetDriverName() string {
	return d.driverName
}

// GetProviderName returns the provider name
func (d *SMSDriver) GetProviderName() string {
	return d.providerName
}

// Close closes the SMS driver
func (d *SMSDriver) Close() error {
	if d.provider != nil {
		return d.provider.Close()
	}
	return nil
}

// Helper methods

// validateSMSNotification validates an SMS notification
func (d *SMSDriver) validateSMSNotification(notif *notification.Notification) error {
	if len(notif.To) == 0 {
		return fmt.Errorf("no recipients specified")
	}

	for _, phone := range notif.To {
		if !isValidPhoneNumber(phone) {
			return fmt.Errorf("invalid phone number: %s", phone)
		}
	}

	if notif.Body == "" {
		return fmt.Errorf("message body is required")
	}

	// Check message length (typical SMS limit is 160 characters for plain text)
	if len(notif.Body) > 1600 { // Allow for longer messages that will be split
		return fmt.Errorf("message body too long: %d characters (max 1600)", len(notif.Body))
	}

	return nil
}

// convertToSMSMessage converts a notification to an SMS message
func (d *SMSDriver) convertToSMSMessage(notif *notification.Notification) (*providers.SMSMessage, error) {
	msg := &providers.SMSMessage{
		To:       notif.To,
		Body:     notif.Body,
		Tags:     notif.Tags,
		Metadata: notif.Metadata,
	}

	// Extract media URLs from attachments (for MMS)
	if len(notif.Attachments) > 0 {
		msg.MediaURLs = make([]string, 0)
		for _, att := range notif.Attachments {
			if att.URL != "" {
				msg.MediaURLs = append(msg.MediaURLs, att.URL)
			}
		}
	}

	// Set validity period from expiration
	if notif.ExpiresAt != nil {
		validityPeriod := int(time.Until(*notif.ExpiresAt).Minutes())
		if validityPeriod > 0 {
			msg.ValidityPeriod = validityPeriod
		}
	}

	return msg, nil
}

// updateStats updates driver statistics
func (d *SMSDriver) updateStats(success bool, latency time.Duration, errorMsg string) {
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
	if d.stats.TotalSent > 0 {
		d.stats.AverageLatency = (d.stats.AverageLatency*time.Duration(d.stats.TotalSent-1) + latency) / time.Duration(d.stats.TotalSent)
	}

	d.stats.ByType["sms"]++
}

// isValidPhoneNumber performs basic phone number validation
func isValidPhoneNumber(phone string) bool {
	// This is a basic validation - in production, use a proper phone validation library
	// Remove common formatting characters
	cleaned := regexp.MustCompile(`[^\d+]`).ReplaceAllString(phone, "")

	// Check if it's a valid format (simplified)
	// Should start with + and have 7-15 digits
	matched, _ := regexp.MatchString(`^\+\d{7,15}$`, cleaned)
	if matched {
		return true
	}

	// Also allow numbers without + if they have 10-15 digits
	matched, _ = regexp.MatchString(`^\d{10,15}$`, cleaned)
	return matched
}