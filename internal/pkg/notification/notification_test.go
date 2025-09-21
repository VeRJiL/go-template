package notification

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/VeRJiL/go-template/internal/config"
)

func TestNotificationBuilder(t *testing.T) {
	t.Run("Basic Notification Building", func(t *testing.T) {
		notification := NewNotificationBuilder().
			To("test@example.com").
			Subject("Test Subject").
			Body("Test Body").
			Priority(PriorityHigh).
			Tags("test", "unit").
			Metadata("source", "test").
			Build()

		if len(notification.To) != 1 || notification.To[0] != "test@example.com" {
			t.Errorf("Expected recipient test@example.com, got %v", notification.To)
		}

		if notification.Subject != "Test Subject" {
			t.Errorf("Expected subject 'Test Subject', got %s", notification.Subject)
		}

		if notification.Body != "Test Body" {
			t.Errorf("Expected body 'Test Body', got %s", notification.Body)
		}

		if notification.Priority != PriorityHigh {
			t.Errorf("Expected priority High, got %v", notification.Priority)
		}

		if len(notification.Tags) != 2 {
			t.Errorf("Expected 2 tags, got %d", len(notification.Tags))
		}

		if notification.Metadata["source"] != "test" {
			t.Errorf("Expected metadata source 'test', got %s", notification.Metadata["source"])
		}

		if notification.ID == "" {
			t.Error("Expected notification ID to be generated")
		}
	})

	t.Run("Multiple Recipients", func(t *testing.T) {
		notification := NewNotificationBuilder().
			To("user1@example.com", "user2@example.com", "user3@example.com").
			Subject("Bulk Test").
			Body("Bulk notification test").
			Build()

		if len(notification.To) != 3 {
			t.Errorf("Expected 3 recipients, got %d", len(notification.To))
		}
	})

	t.Run("Template Notification", func(t *testing.T) {
		vars := map[string]interface{}{
			"name":  "John Doe",
			"token": "abc123",
		}

		notification := NewNotificationBuilder().
			To("user@example.com").
			Template("welcome", vars).
			Build()

		if notification.Template != "welcome" {
			t.Errorf("Expected template 'welcome', got %s", notification.Template)
		}

		if notification.TemplateVars["name"] != "John Doe" {
			t.Errorf("Expected template var name 'John Doe', got %v", notification.TemplateVars["name"])
		}
	})

	t.Run("Scheduled Notification", func(t *testing.T) {
		futureTime := time.Now().Add(1 * time.Hour)

		notification := NewNotificationBuilder().
			To("scheduled@example.com").
			Subject("Scheduled Test").
			Body("This is scheduled").
			ScheduleAt(futureTime).
			Build()

		if notification.ScheduledAt == nil {
			t.Error("Expected scheduled time to be set")
		}

		if !notification.ScheduledAt.Equal(futureTime) {
			t.Errorf("Expected scheduled time %v, got %v", futureTime, *notification.ScheduledAt)
		}
	})

	t.Run("Notification with Attachments", func(t *testing.T) {
		attachment := Attachment{
			Filename:    "test.pdf",
			Content:     []byte("test content"),
			ContentType: "application/pdf",
		}

		notification := NewNotificationBuilder().
			To("attachment@example.com").
			Subject("Attachment Test").
			Body("With attachment").
			Attachment(attachment).
			Build()

		if len(notification.Attachments) != 1 {
			t.Errorf("Expected 1 attachment, got %d", len(notification.Attachments))
		}

		if notification.Attachments[0].Filename != "test.pdf" {
			t.Errorf("Expected attachment filename 'test.pdf', got %s", notification.Attachments[0].Filename)
		}
	})
}

func TestConvenienceNotificationConstructors(t *testing.T) {
	t.Run("Email Notification", func(t *testing.T) {
		notification := NewEmailNotification(
			[]string{"email@example.com"},
			"Email Subject",
			"Email Body")

		if len(notification.To) != 1 || notification.To[0] != "email@example.com" {
			t.Errorf("Expected recipient email@example.com, got %v", notification.To)
		}

		if notification.Subject != "Email Subject" {
			t.Errorf("Expected subject 'Email Subject', got %s", notification.Subject)
		}

		if notification.Body != "Email Body" {
			t.Errorf("Expected body 'Email Body', got %s", notification.Body)
		}
	})

	t.Run("SMS Notification", func(t *testing.T) {
		notification := NewSMSNotification(
			[]string{"+1234567890"},
			"SMS Message")

		if len(notification.To) != 1 || notification.To[0] != "+1234567890" {
			t.Errorf("Expected recipient +1234567890, got %v", notification.To)
		}

		if notification.Body != "SMS Message" {
			t.Errorf("Expected body 'SMS Message', got %s", notification.Body)
		}

		if notification.Subject != "" {
			t.Error("SMS notification should not have a subject")
		}
	})

	t.Run("Push Notification", func(t *testing.T) {
		notification := NewPushNotification(
			[]string{"device_token_123"},
			"Push Title",
			"Push Body")

		if len(notification.To) != 1 || notification.To[0] != "device_token_123" {
			t.Errorf("Expected recipient device_token_123, got %v", notification.To)
		}

		if notification.Subject != "Push Title" {
			t.Errorf("Expected subject 'Push Title', got %s", notification.Subject)
		}

		if notification.Body != "Push Body" {
			t.Errorf("Expected body 'Push Body', got %s", notification.Body)
		}
	})
}

func TestNotificationManager(t *testing.T) {
	// Create test configuration
	config := &config.NotificationConfig{
		Enabled:       true,
		DefaultDriver: "email",
		Email: config.NotificationEmailConfig{
			Enabled:  true,
			Provider: "smtp",
			SMTP: &config.SMTPConfig{
				Host:         "localhost",
				Port:         1025,
				Username:     "",
				Password:     "",
				From:         "test@example.com",
				FromName:     "Test Sender",
				UseTLS:       false,
				UseStartTLS:  false,
				InsecureSkip: true,
				Timeout:      30,
			},
		},
	}

	t.Run("Manager Creation", func(t *testing.T) {
		manager, err := NewManager(config)
		if err != nil {
			t.Fatalf("Failed to create notification manager: %v", err)
		}
		defer manager.Close()

		drivers := manager.GetAvailableDrivers()
		if len(drivers) == 0 {
			t.Error("Expected at least one driver to be available")
		}

		defaultDriver := manager.GetDefaultDriver()
		if defaultDriver != "email" {
			t.Errorf("Expected default driver 'email', got %s", defaultDriver)
		}
	})

	t.Run("Manager Statistics", func(t *testing.T) {
		manager, err := NewManager(config)
		if err != nil {
			t.Fatalf("Failed to create notification manager: %v", err)
		}
		defer manager.Close()

		stats := manager.GetManagerStats()
		if stats == nil {
			t.Error("Expected manager statistics to be returned")
		}

		if stats.TotalNotifications != 0 {
			t.Errorf("Expected 0 total notifications, got %d", stats.TotalNotifications)
		}
	})

	t.Run("Health Check", func(t *testing.T) {
		manager, err := NewManager(config)
		if err != nil {
			t.Fatalf("Failed to create notification manager: %v", err)
		}
		defer manager.Close()

		ctx := context.Background()
		health := manager.HealthCheck(ctx)

		if len(health) == 0 {
			t.Error("Expected health check results for at least one driver")
		}

		// Note: Health check might fail in test environment, but should not panic
		for driver, err := range health {
			t.Logf("Driver %s health: %v", driver, err)
		}
	})

	t.Run("Driver Switching", func(t *testing.T) {
		manager, err := NewManager(config)
		if err != nil {
			t.Fatalf("Failed to create notification manager: %v", err)
		}
		defer manager.Close()

		// Test changing default driver
		originalDefault := manager.GetDefaultDriver()

		// Since we only have email driver configured, this should fail
		err = manager.SetDefaultDriver("sms")
		if err == nil {
			t.Error("Expected error when setting non-existent driver as default")
		}

		// Verify original default is unchanged
		if manager.GetDefaultDriver() != originalDefault {
			t.Errorf("Default driver should remain %s after failed change", originalDefault)
		}
	})
}

func TestNotificationValidation(t *testing.T) {
	t.Run("Empty Recipients", func(t *testing.T) {
		notification := NewNotificationBuilder().
			Subject("Test").
			Body("Test body").
			Build()

		// Create a minimal email driver for testing
		config := &config.NotificationConfig{
			Enabled:       true,
			DefaultDriver: "email",
			Email: config.NotificationEmailConfig{
				Enabled:  true,
				Provider: "smtp",
				SMTP: &config.SMTPConfig{
					Host: "localhost",
					Port: 1025,
					From: "test@example.com",
				},
			},
		}

		manager, err := NewManager(config)
		if err != nil {
			t.Fatalf("Failed to create notification manager: %v", err)
		}
		defer manager.Close()

		ctx := context.Background()
		err = manager.Send(ctx, notification)

		// Should fail due to no recipients
		if err == nil {
			t.Error("Expected error for notification with no recipients")
		}
	})

	t.Run("Invalid Email Format", func(t *testing.T) {
		notification := NewEmailNotification(
			[]string{"invalid-email-format"},
			"Test Subject",
			"Test Body")

		config := &config.NotificationConfig{
			Enabled:       true,
			DefaultDriver: "email",
			Email: config.NotificationEmailConfig{
				Enabled:  true,
				Provider: "smtp",
				SMTP: &config.SMTPConfig{
					Host: "localhost",
					Port: 1025,
					From: "test@example.com",
				},
			},
		}

		manager, err := NewManager(config)
		if err != nil {
			t.Fatalf("Failed to create notification manager: %v", err)
		}
		defer manager.Close()

		ctx := context.Background()
		err = manager.Send(ctx, notification)

		// Should fail due to invalid email format
		if err == nil {
			t.Error("Expected error for notification with invalid email format")
		}
	})
}

func TestNotificationTypes(t *testing.T) {
	tests := []struct {
		name         string
		notification *Notification
		expectedType NotificationType
	}{
		{
			name:         "Email Type Detection",
			notification: NewEmailNotification([]string{"test@example.com"}, "Subject", "Body"),
			expectedType: TypeEmail,
		},
		{
			name:         "SMS Type Detection",
			notification: NewSMSNotification([]string{"+1234567890"}, "Message"),
			expectedType: TypeSMS,
		},
		{
			name:         "Push Type Detection",
			notification: NewPushNotification([]string{"device_token"}, "Title", "Body"),
			expectedType: TypePush,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: Our current implementation doesn't auto-detect type
			// This test is here for future enhancement
			t.Logf("Notification: %+v", tt.notification)
			// Add type detection logic in the future
		})
	}
}

func TestNotificationPriorities(t *testing.T) {
	priorities := []Priority{
		PriorityLow,
		PriorityNormal,
		PriorityHigh,
		PriorityCritical,
	}

	for i, priority := range priorities {
		t.Run(fmt.Sprintf("Priority_%d", int(priority)), func(t *testing.T) {
			notification := NewNotificationBuilder().
				To("test@example.com").
				Subject("Priority Test").
				Body("Testing priority").
				Priority(priority).
				Build()

			if notification.Priority != priority {
				t.Errorf("Expected priority %v, got %v", priority, notification.Priority)
			}

			// Verify priority ordering
			if int(priority) != i {
				t.Errorf("Expected priority value %d, got %d", i, int(priority))
			}
		})
	}
}

// Benchmark tests
func BenchmarkNotificationBuilder(b *testing.B) {
	for i := 0; i < b.N; i++ {
		notification := NewNotificationBuilder().
			To("bench@example.com").
			Subject("Benchmark Test").
			Body("Benchmark notification").
			Priority(PriorityNormal).
			Tags("benchmark").
			Metadata("run", fmt.Sprintf("%d", i)).
			Build()

		_ = notification
	}
}

func BenchmarkManagerCreation(b *testing.B) {
	config := &config.NotificationConfig{
		Enabled:       true,
		DefaultDriver: "email",
		Email: config.NotificationEmailConfig{
			Enabled:  true,
			Provider: "smtp",
			SMTP: &config.SMTPConfig{
				Host: "localhost",
				Port: 1025,
				From: "bench@example.com",
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager, err := NewManager(config)
		if err != nil {
			b.Fatalf("Failed to create manager: %v", err)
		}
		manager.Close()
	}
}

// Mock driver for testing
type MockDriver struct {
	sendCalled     bool
	sendAsyncCalled bool
	sendBatchCalled bool
	lastNotification *Notification
}

func (m *MockDriver) Send(ctx context.Context, notification *Notification) error {
	m.sendCalled = true
	m.lastNotification = notification
	return nil
}

func (m *MockDriver) SendAsync(ctx context.Context, notification *Notification) error {
	m.sendAsyncCalled = true
	m.lastNotification = notification
	return nil
}

func (m *MockDriver) SendBatch(ctx context.Context, notifications []*Notification) error {
	m.sendBatchCalled = true
	if len(notifications) > 0 {
		m.lastNotification = notifications[0]
	}
	return nil
}

func (m *MockDriver) SendScheduled(ctx context.Context, notification *Notification, sendAt time.Time) error {
	m.lastNotification = notification
	return nil
}

func (m *MockDriver) GetStats() (*DriverStats, error) {
	return &DriverStats{}, nil
}

func (m *MockDriver) Ping(ctx context.Context) error {
	return nil
}

func (m *MockDriver) GetDriverName() string {
	return "mock"
}

func (m *MockDriver) GetProviderName() string {
	return "mock"
}

func (m *MockDriver) Close() error {
	return nil
}

func TestMockDriver(t *testing.T) {
	mock := &MockDriver{}

	notification := NewEmailNotification([]string{"test@example.com"}, "Test", "Body")
	ctx := context.Background()

	err := mock.Send(ctx, notification)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !mock.sendCalled {
		t.Error("Expected Send to be called")
	}

	if mock.lastNotification != notification {
		t.Error("Expected last notification to be set")
	}
}