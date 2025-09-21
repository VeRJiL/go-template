# 🚀 Go Template Notification System

A comprehensive, Laravel-style notification system for Go applications with support for multiple drivers and providers.

## ✨ Features

- **🔧 Multiple Drivers**: Email, SMS, Push Notifications, Social Media (WhatsApp, Telegram, Slack, Discord)
- **🏭 Factory Pattern**: Easy to extend with new drivers and providers
- **🔄 Runtime Driver Switching**: Change drivers on-the-go with method chaining
- **⚡ Async Support**: Send notifications asynchronously
- **⏰ Scheduled Notifications**: Schedule notifications for future delivery
- **📦 Batch Processing**: Send multiple notifications at once
- **📊 Statistics & Health Monitoring**: Built-in metrics and health checks
- **🛡️ Middleware Support**: Process notifications through middleware pipeline
- **🎯 Template Support**: Use templates for dynamic content
- **📎 Attachment Support**: Send files with notifications
- **🌍 Broadcast**: Send same notification via multiple drivers

## 🏗️ Architecture

The notification system follows the **Strategy + Factory Pattern** and is inspired by Laravel's notification system:

```
NotificationManager
├── EmailDriver
│   ├── SMTPProvider
│   ├── SendGridProvider
│   ├── MailgunProvider
│   └── AWSSESProvider
├── SMSDriver
│   ├── TwilioProvider
│   ├── AWSSNSProvider
│   ├── NexmoProvider
│   └── TextMagicProvider
├── PushDriver
│   ├── FCMProvider
│   ├── APNSProvider
│   ├── PusherProvider
│   └── OneSignalProvider
└── SocialDriver
    ├── WhatsAppProvider
    ├── TelegramProvider
    ├── SlackProvider
    └── DiscordProvider
```

## 🚀 Quick Start

### 1. Configuration

Add notification configuration to your `.env` file:

```bash
# Enable notifications
NOTIFICATION_ENABLED=true
NOTIFICATION_DEFAULT_DRIVER=email

# Email Configuration
NOTIFICATION_EMAIL_ENABLED=true
NOTIFICATION_EMAIL_PROVIDER=smtp
NOTIFICATION_SMTP_HOST=localhost
NOTIFICATION_SMTP_PORT=587
NOTIFICATION_SMTP_USERNAME=your_username
NOTIFICATION_SMTP_PASSWORD=your_password
NOTIFICATION_SMTP_FROM=noreply@yourapp.com
NOTIFICATION_SMTP_FROM_NAME="Your App"
NOTIFICATION_SMTP_USE_STARTTLS=true

# SMS Configuration (Twilio)
NOTIFICATION_SMS_ENABLED=true
NOTIFICATION_SMS_PROVIDER=twilio
NOTIFICATION_TWILIO_ACCOUNT_SID=your_account_sid
NOTIFICATION_TWILIO_AUTH_TOKEN=your_auth_token
NOTIFICATION_TWILIO_FROM_NUMBER=+1234567890

# Push Notifications (FCM)
NOTIFICATION_PUSH_ENABLED=true
NOTIFICATION_PUSH_PROVIDER=fcm
NOTIFICATION_FCM_SERVER_KEY=your_server_key
NOTIFICATION_FCM_PROJECT_ID=your_project_id
```

### 2. Initialize the Manager

```go
package main

import (
    "context"
    "log"

    "github.com/VeRJiL/go-template/internal/config"
    "github.com/VeRJiL/go-template/internal/pkg/notification"
)

func main() {
    // Load configuration
    cfg, err := config.Load()
    if err != nil {
        log.Fatal(err)
    }

    // Create notification manager
    notificationManager, err := notification.NewManager(&cfg.Notification)
    if err != nil {
        log.Fatal(err)
    }
    defer notificationManager.Close()

    // Send a simple email
    ctx := context.Background()
    err = notificationManager.SendEmail(ctx,
        []string{"user@example.com"},
        "Welcome!",
        "Thanks for joining our platform!")

    if err != nil {
        log.Printf("Failed to send email: %v", err)
    }
}
```

## 📖 Usage Examples

### 📧 Basic Email Notification

```go
// Simple email
err := notificationManager.SendEmail(ctx,
    []string{"user@example.com"},
    "Welcome!",
    "Thanks for joining!")

// Advanced email with builder
notification := notification.NewNotificationBuilder().
    To("user@example.com", "admin@example.com").
    Subject("System Alert").
    Body("<h1>Alert</h1><p>System status update</p>").
    Priority(notification.PriorityHigh).
    Tags("system", "alert").
    Metadata("source", "monitoring").
    Build()

err = notificationManager.Send(ctx, notification)
```

### 📱 SMS Notifications

```go
// Simple SMS
err := notificationManager.SendSMS(ctx,
    []string{"+1234567890"},
    "Your verification code: 123456")

// SMS with driver switching
notification := notification.NewSMSNotification(
    []string{"+1987654321"},
    "Welcome to our service!")

err = notificationManager.Via("sms").Send(ctx, notification)
```

### 🔔 Push Notifications

```go
// Push notification
err := notificationManager.SendPush(ctx,
    []string{"device_token_123"},
    "New Message",
    "You have a new message waiting")
```

### 🔄 Driver Switching (Method Chaining)

```go
// Send via specific driver
err = notificationManager.Via("email").Send(ctx, emailNotification)
err = notificationManager.Via("sms").Send(ctx, smsNotification)
err = notificationManager.Via("push").Send(ctx, pushNotification)

// Chain multiple operations
err = notificationManager.
    Via("email").
    SendAsync(ctx, notification)
```

### ⚡ Async Notifications

```go
// Send asynchronously (non-blocking)
err := notificationManager.SendAsync(ctx, notification)

// Send via specific driver asynchronously
err = notificationManager.Via("sms").SendAsync(ctx, smsNotification)
```

### ⏰ Scheduled Notifications

```go
// Schedule for future delivery
sendAt := time.Now().Add(24 * time.Hour)
err := notificationManager.SendScheduled(ctx, notification, sendAt)

// Using builder
notification := notification.NewNotificationBuilder().
    To("user@example.com").
    Subject("Reminder").
    Body("Don't forget your appointment tomorrow!").
    ScheduleAt(time.Now().Add(23 * time.Hour)).
    Build()

err = notificationManager.Send(ctx, notification)
```

### 📦 Batch Notifications

```go
notifications := []*notification.Notification{
    notification.NewEmailNotification([]string{"user1@example.com"}, "Welcome", "Hello user 1"),
    notification.NewEmailNotification([]string{"user2@example.com"}, "Welcome", "Hello user 2"),
    notification.NewEmailNotification([]string{"user3@example.com"}, "Welcome", "Hello user 3"),
}

err := notificationManager.SendBatch(ctx, notifications)
```

### 📢 Broadcasting

```go
// Send same notification via multiple drivers
notification := notification.NewNotificationBuilder().
    To("user@example.com").
    Subject("Important Update").
    Body("Please check your account").
    Build()

err := notificationManager.Broadcast(ctx, []string{"email", "sms", "push"}, notification)
```

### 📎 Attachments

```go
attachment := notification.Attachment{
    Filename:    "report.pdf",
    Content:     pdfContent, // []byte
    ContentType: "application/pdf",
}

notification := notification.NewNotificationBuilder().
    To("manager@example.com").
    Subject("Monthly Report").
    Body("Please find the monthly report attached.").
    Attachment(attachment).
    Build()

err = notificationManager.Send(ctx, notification)
```

### 🎯 Template-Based Notifications

```go
templateVars := map[string]interface{}{
    "user_name": "John Doe",
    "reset_link": "https://app.com/reset/abc123",
    "expires_in": "24 hours",
}

notification := notification.NewNotificationBuilder().
    To("user@example.com").
    Template("password_reset", templateVars).
    Build()

err = notificationManager.Send(ctx, notification)
```

### 📊 Statistics and Health Monitoring

```go
// Check driver health
health := notificationManager.HealthCheck(ctx)
for driver, err := range health {
    if err != nil {
        log.Printf("Driver %s is unhealthy: %v", driver, err)
    }
}

// Get manager statistics
managerStats := notificationManager.GetManagerStats()
log.Printf("Total sent: %d, Failed: %d",
    managerStats.TotalNotifications,
    managerStats.TotalFailed)

// Get driver-specific statistics
allStats, err := notificationManager.GetAllStats()
for driverName, stats := range allStats {
    log.Printf("%s: Sent=%d, Failed=%d, ErrorRate=%.1f%%",
        driverName, stats.TotalSent, stats.TotalFailed, stats.ErrorRate)
}
```

### 🛡️ Middleware

```go
// Custom middleware
type LoggingMiddleware struct{}

func (m *LoggingMiddleware) Process(ctx context.Context, notif *notification.Notification) (*notification.Notification, error) {
    log.Printf("Processing notification: %s to %v", notif.Subject, notif.To)
    return notif, nil
}

// Add middleware
notificationManager.AddMiddleware(&LoggingMiddleware{})
```

## 🔧 Configuration Options

### Email Providers

#### SMTP
```bash
NOTIFICATION_EMAIL_PROVIDER=smtp
NOTIFICATION_SMTP_HOST=smtp.gmail.com
NOTIFICATION_SMTP_PORT=587
NOTIFICATION_SMTP_USERNAME=your_email@gmail.com
NOTIFICATION_SMTP_PASSWORD=your_app_password
NOTIFICATION_SMTP_FROM=noreply@yourapp.com
NOTIFICATION_SMTP_FROM_NAME="Your App"
NOTIFICATION_SMTP_USE_STARTTLS=true
```

#### SendGrid
```bash
NOTIFICATION_EMAIL_PROVIDER=sendgrid
NOTIFICATION_SENDGRID_API_KEY=your_sendgrid_api_key
NOTIFICATION_SENDGRID_FROM_EMAIL=noreply@yourapp.com
NOTIFICATION_SENDGRID_FROM_NAME="Your App"
```

### SMS Providers

#### Twilio
```bash
NOTIFICATION_SMS_PROVIDER=twilio
NOTIFICATION_TWILIO_ACCOUNT_SID=your_account_sid
NOTIFICATION_TWILIO_AUTH_TOKEN=your_auth_token
NOTIFICATION_TWILIO_FROM_NUMBER=+1234567890
```

#### AWS SNS
```bash
NOTIFICATION_SMS_PROVIDER=aws_sns
NOTIFICATION_AWS_SNS_REGION=us-east-1
NOTIFICATION_AWS_SNS_ACCESS_KEY=your_access_key
NOTIFICATION_AWS_SNS_SECRET_KEY=your_secret_key
```

### Push Notification Providers

#### Firebase Cloud Messaging (FCM)
```bash
NOTIFICATION_PUSH_PROVIDER=fcm
NOTIFICATION_FCM_SERVER_KEY=your_server_key
NOTIFICATION_FCM_PROJECT_ID=your_project_id
```

#### Apple Push Notification Service (APNS)
```bash
NOTIFICATION_PUSH_PROVIDER=apns
NOTIFICATION_APNS_KEY_ID=your_key_id
NOTIFICATION_APNS_TEAM_ID=your_team_id
NOTIFICATION_APNS_BUNDLE_ID=com.yourapp.bundle
NOTIFICATION_APNS_KEY_FILE=./certs/apns.p8
NOTIFICATION_APNS_PRODUCTION=false
```

### Social Media Providers

#### Telegram
```bash
NOTIFICATION_TELEGRAM_ENABLED=true
NOTIFICATION_TELEGRAM_BOT_TOKEN=your_bot_token
```

#### Slack
```bash
NOTIFICATION_SLACK_ENABLED=true
NOTIFICATION_SLACK_PROVIDER=webhook
NOTIFICATION_SLACK_WEBHOOK_URL=https://hooks.slack.com/...
```

## 🧪 Testing

### Local Email Testing

Use MailHog for local email testing:

```bash
# Start MailHog
docker run -d -p 1025:1025 -p 8025:8025 mailhog/mailhog

# Configure for testing
NOTIFICATION_SMTP_HOST=localhost
NOTIFICATION_SMTP_PORT=1025
NOTIFICATION_SMTP_USE_TLS=false
NOTIFICATION_SMTP_USE_STARTTLS=false
```

Visit http://localhost:8025 to see captured emails.

### Running Tests

```bash
# Run notification system tests
cd internal/pkg/notification
go test -v

# Run with coverage
go test -v -cover

# Run benchmarks
go test -bench=.
```

### Example Test

```go
func TestNotificationSystem(t *testing.T) {
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

    manager, err := notification.NewManager(config)
    if err != nil {
        t.Fatal(err)
    }
    defer manager.Close()

    notification := notification.NewEmailNotification(
        []string{"test@example.com"},
        "Test Subject",
        "Test Body")

    ctx := context.Background()
    err = manager.Send(ctx, notification)

    // In test environment with MailHog, this should not error
    if err != nil {
        t.Logf("Send failed (expected in test env): %v", err)
    }
}
```

## 🔌 Integration with Message Broker

Integrate with the existing message broker system for async processing:

```go
// Example: Queue notification jobs
notificationJob := messagebroker.NewJob("notifications", "send_email", map[string]interface{}{
    "to":      []string{"user@example.com"},
    "subject": "Welcome!",
    "body":    "Thanks for joining!",
})

messageBrokerManager.EnqueueJob(ctx, "notifications", notificationJob)

// Example: Process notification jobs
messageBrokerManager.ProcessJobs(ctx, "notifications", func(job *messagebroker.Job) error {
    // Extract notification data and send
    notification := extractNotificationFromJob(job)
    return notificationManager.Send(ctx, notification)
})
```

## 🚀 Extending the System

### Adding New Providers

1. **Create Provider Interface Implementation**:

```go
// internal/pkg/notification/providers/email/new_provider.go
type NewEmailProvider struct {
    config *NewProviderConfig
}

func (p *NewEmailProvider) Send(ctx context.Context, message *providers.EmailMessage) error {
    // Implementation
    return nil
}

func (p *NewEmailProvider) GetProviderName() string {
    return "new_provider"
}

// ... implement other required methods
```

2. **Update Factory**:

```go
// internal/pkg/notification/providers/providers.go
func init() {
    // Add new provider factory
    NewNewProviderEmailProvider = func(config interface{}) (EmailProvider, error) {
        return email.NewNewProviderEmailProvider(config)
    }
}
```

3. **Update Configuration**:

```go
// internal/config/config.go
type NotificationEmailConfig struct {
    // ... existing providers
    NewProvider *NewProviderConfig `json:"new_provider,omitempty" mapstructure:"new_provider"`
}
```

### Adding New Drivers

1. **Create Driver**:

```go
// internal/pkg/notification/drivers/new_driver.go
type NewDriver struct {
    // Implementation following NotificationDriver interface
}
```

2. **Update Manager**:

```go
// internal/pkg/notification/drivers_init.go
func (m *Manager) initializeNewDrivers() error {
    // Add initialization logic
}
```

## 🎯 Best Practices

1. **Configuration**: Always use environment variables for sensitive data
2. **Error Handling**: Implement proper error handling and logging
3. **Rate Limiting**: Be aware of provider rate limits
4. **Testing**: Use MailHog or similar tools for local testing
5. **Monitoring**: Monitor notification delivery rates and errors
6. **Templates**: Use templates for consistent messaging
7. **Async Processing**: Use async methods for better performance
8. **Graceful Degradation**: Handle provider failures gracefully

## 🐛 Troubleshooting

### Common Issues

1. **SMTP Authentication Failed**
   - Check username/password
   - Verify SMTP settings
   - Enable "Less Secure Apps" for Gmail

2. **Twilio SMS Failed**
   - Verify Account SID and Auth Token
   - Check phone number format (+1234567890)
   - Ensure sufficient account balance

3. **Push Notifications Not Delivered**
   - Verify server key/certificates
   - Check device token validity
   - Ensure proper app configuration

### Debug Mode

Enable debug logging:

```bash
LOG_LEVEL=debug
```

Check notification statistics:

```go
stats := notificationManager.GetManagerStats()
health := notificationManager.HealthCheck(ctx)
```

## 📝 License

This notification system is part of the Go Template project and follows the same license terms.

## 🤝 Contributing

1. Follow existing code patterns
2. Add comprehensive tests
3. Update documentation
4. Follow the Strategy + Factory pattern for new providers/drivers

---

**Happy Notifying! 🎉**