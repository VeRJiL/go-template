package notification

import (
	"context"
	"time"
)

// NotificationDriver defines the interface for all notification drivers
type NotificationDriver interface {
	// Send sends a notification
	Send(ctx context.Context, notification *Notification) error

	// SendAsync sends a notification asynchronously
	SendAsync(ctx context.Context, notification *Notification) error

	// SendBatch sends multiple notifications
	SendBatch(ctx context.Context, notifications []*Notification) error

	// SendScheduled sends a notification at a specific time
	SendScheduled(ctx context.Context, notification *Notification, sendAt time.Time) error

	// GetStats returns driver statistics
	GetStats() (*DriverStats, error)

	// Health check for the driver
	Ping(ctx context.Context) error

	// GetDriverName returns the driver name
	GetDriverName() string

	// GetProviderName returns the provider name
	GetProviderName() string

	// Close closes the driver
	Close() error
}

// Notification represents a notification message
type Notification struct {
	ID          string                 `json:"id"`
	Type        NotificationType       `json:"type"`
	To          []string               `json:"to"`
	Subject     string                 `json:"subject,omitempty"`
	Body        string                 `json:"body"`
	Data        map[string]interface{} `json:"data,omitempty"`
	Template    string                 `json:"template,omitempty"`
	TemplateVars map[string]interface{} `json:"template_vars,omitempty"`
	Priority    Priority               `json:"priority"`
	Tags        []string               `json:"tags,omitempty"`
	Metadata    map[string]string      `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	ScheduledAt *time.Time             `json:"scheduled_at,omitempty"`
	ExpiresAt   *time.Time             `json:"expires_at,omitempty"`
	Attachments []Attachment           `json:"attachments,omitempty"`
}

// NotificationType represents the type of notification
type NotificationType string

const (
	TypeEmail       NotificationType = "email"
	TypeSMS         NotificationType = "sms"
	TypePush        NotificationType = "push"
	TypeWhatsApp    NotificationType = "whatsapp"
	TypeTelegram    NotificationType = "telegram"
	TypeSlack       NotificationType = "slack"
	TypeDiscord     NotificationType = "discord"
	TypeWebhook     NotificationType = "webhook"
)

// Priority represents notification priority
type Priority int

const (
	PriorityLow Priority = iota
	PriorityNormal
	PriorityHigh
	PriorityCritical
)

// Attachment represents a file attachment
type Attachment struct {
	Filename    string `json:"filename"`
	Content     []byte `json:"content"`
	ContentType string `json:"content_type"`
	URL         string `json:"url,omitempty"`
}

// NotificationResponse represents the response after sending a notification
type NotificationResponse struct {
	ID           string                 `json:"id"`
	Status       DeliveryStatus         `json:"status"`
	MessageID    string                 `json:"message_id,omitempty"`
	ExternalID   string                 `json:"external_id,omitempty"`
	DeliveredAt  *time.Time             `json:"delivered_at,omitempty"`
	FailedAt     *time.Time             `json:"failed_at,omitempty"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	Cost         float64                `json:"cost,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// DeliveryStatus represents the delivery status of a notification
type DeliveryStatus string

const (
	StatusPending   DeliveryStatus = "pending"
	StatusSent      DeliveryStatus = "sent"
	StatusDelivered DeliveryStatus = "delivered"
	StatusFailed    DeliveryStatus = "failed"
	StatusExpired   DeliveryStatus = "expired"
)

// DriverStats represents statistics for a notification driver
type DriverStats struct {
	TotalSent      int64              `json:"total_sent"`
	TotalFailed    int64              `json:"total_failed"`
	TotalDelivered int64              `json:"total_delivered"`
	AverageLatency time.Duration      `json:"average_latency"`
	ErrorRate      float64            `json:"error_rate"`
	LastError      string             `json:"last_error,omitempty"`
	LastErrorAt    *time.Time         `json:"last_error_at,omitempty"`
	Uptime         time.Duration      `json:"uptime"`
	ByType         map[string]int64   `json:"by_type"`
	ByPriority     map[Priority]int64 `json:"by_priority"`
}

// NotificationBuilder provides a fluent interface for building notifications
type NotificationBuilder struct {
	notification *Notification
}

// NewNotificationBuilder creates a new notification builder
func NewNotificationBuilder() *NotificationBuilder {
	return &NotificationBuilder{
		notification: &Notification{
			Data:         make(map[string]interface{}),
			TemplateVars: make(map[string]interface{}),
			Metadata:     make(map[string]string),
			Priority:     PriorityNormal,
			CreatedAt:    time.Now(),
		},
	}
}

// To sets the recipients
func (nb *NotificationBuilder) To(recipients ...string) *NotificationBuilder {
	nb.notification.To = recipients
	return nb
}

// Subject sets the subject
func (nb *NotificationBuilder) Subject(subject string) *NotificationBuilder {
	nb.notification.Subject = subject
	return nb
}

// Body sets the body content
func (nb *NotificationBuilder) Body(body string) *NotificationBuilder {
	nb.notification.Body = body
	return nb
}

// Template sets the template and variables
func (nb *NotificationBuilder) Template(template string, vars map[string]interface{}) *NotificationBuilder {
	nb.notification.Template = template
	nb.notification.TemplateVars = vars
	return nb
}

// Priority sets the priority
func (nb *NotificationBuilder) Priority(priority Priority) *NotificationBuilder {
	nb.notification.Priority = priority
	return nb
}

// Data sets custom data
func (nb *NotificationBuilder) Data(key string, value interface{}) *NotificationBuilder {
	nb.notification.Data[key] = value
	return nb
}

// Metadata sets metadata
func (nb *NotificationBuilder) Metadata(key, value string) *NotificationBuilder {
	nb.notification.Metadata[key] = value
	return nb
}

// Tags sets tags
func (nb *NotificationBuilder) Tags(tags ...string) *NotificationBuilder {
	nb.notification.Tags = tags
	return nb
}

// Attachment adds an attachment
func (nb *NotificationBuilder) Attachment(attachment Attachment) *NotificationBuilder {
	nb.notification.Attachments = append(nb.notification.Attachments, attachment)
	return nb
}

// ExpiresAt sets expiration time
func (nb *NotificationBuilder) ExpiresAt(expiresAt time.Time) *NotificationBuilder {
	nb.notification.ExpiresAt = &expiresAt
	return nb
}

// ScheduleAt sets scheduled time
func (nb *NotificationBuilder) ScheduleAt(scheduledAt time.Time) *NotificationBuilder {
	nb.notification.ScheduledAt = &scheduledAt
	return nb
}

// Build creates the notification
func (nb *NotificationBuilder) Build() *Notification {
	// Generate ID if not set
	if nb.notification.ID == "" {
		nb.notification.ID = generateNotificationID()
	}
	return nb.notification
}

// Notification convenience methods

// NewEmailNotification creates a new email notification
func NewEmailNotification(to []string, subject, body string) *Notification {
	return NewNotificationBuilder().
		To(to...).
		Subject(subject).
		Body(body).
		Build()
}

// NewSMSNotification creates a new SMS notification
func NewSMSNotification(to []string, message string) *Notification {
	return NewNotificationBuilder().
		To(to...).
		Body(message).
		Build()
}

// NewPushNotification creates a new push notification
func NewPushNotification(to []string, title, body string) *Notification {
	return NewNotificationBuilder().
		To(to...).
		Subject(title).
		Body(body).
		Build()
}

// Helper functions

func generateNotificationID() string {
	// Simple UUID generation - in production, use proper UUID library
	return "notification_" + time.Now().Format("20060102150405") + "_" + randomString(8)
}

func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(result)
}