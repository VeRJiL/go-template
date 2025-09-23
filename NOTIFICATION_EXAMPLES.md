# Notification System Examples

This document provides comprehensive examples of using the Go Template notification system.

## Table of Contents

- [Basic Setup](#basic-setup)
- [Example 1: Basic Email Notification](#example-1-basic-email-notification)
- [Example 2: Using Notification Builder](#example-2-using-notification-builder)
- [Example 3: Driver Switching](#example-3-driver-switching)
- [Example 4: Async Notifications](#example-4-async-notifications)
- [Example 5: Scheduled Notifications](#example-5-scheduled-notifications)
- [Example 6: Batch Notifications](#example-6-batch-notifications)
- [Example 7: Broadcasting](#example-7-broadcasting)
- [Example 8: Statistics and Health Check](#example-8-statistics-and-health-check)
- [Message Broker Integration](#message-broker-integration)
- [Configuration](#configuration)
- [Tips](#tips)

## Basic Setup

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/VeRJiL/go-template/internal/config"
    "github.com/VeRJiL/go-template/internal/pkg/notification"
)

func main() {
    // Load configuration
    cfg, err := config.Load()
    if err != nil {
        log.Fatalf("Failed to load configuration: %v", err)
    }

    // Override config for demonstration (in real app, use environment variables)
    cfg.Notification.Enabled = true
    cfg.Notification.DefaultDriver = "email"
    cfg.Notification.Email.Enabled = true
    cfg.Notification.Email.Provider = "smtp"
    cfg.Notification.Email.SMTP = &config.SMTPConfig{
        Host:     "localhost",
        Port:     1025, // MailHog/MailCatcher port for testing
        Username: "",
        Password: "",
        From:     "noreply@go-template.com",
        FromName: "Go Template Notifications",
        UseTLS:   false,
        UseStartTLS: false,
        InsecureSkip: true,
        Timeout:  30,
    }

    // Initialize notification manager
    notificationManager, err := notification.NewManager(&cfg.Notification)
    if err != nil {
        log.Fatalf("Failed to create notification manager: %v", err)
    }
    defer notificationManager.Close()

    fmt.Printf("✅ Notification manager initialized with drivers: %v\n",
        notificationManager.GetAvailableDrivers())
}
```

## Example 1: Basic Email Notification

```go
func basicEmailExample(notificationManager *notification.Manager) {
    ctx := context.Background()
    err := notificationManager.SendEmail(ctx,
        []string{"user@example.com"},
        "Welcome to Go Template!",
        "Thank you for using our Go Template notification system!")

    if err != nil {
        fmt.Printf("❌ Failed to send email: %v\n", err)
    } else {
        fmt.Println("✅ Email sent successfully!")
    }
}
```

## Example 2: Using Notification Builder

```go
func notificationBuilderExample(notificationManager *notification.Manager) {
    ctx := context.Background()

    notification := notification.NewNotificationBuilder().
        To("admin@example.com", "support@example.com").
        Subject("System Alert").
        Body("<h1>Server Status</h1><p>All systems operational</p>").
        Priority(notification.PriorityHigh).
        Tags("system", "alert").
        Metadata("source", "monitoring").
        Build()

    err := notificationManager.Send(ctx, notification)
    if err != nil {
        fmt.Printf("❌ Failed to send notification: %v\n", err)
    } else {
        fmt.Println("✅ System alert sent successfully!")
    }
}
```

## Example 3: Driver Switching

```go
func driverSwitchingExample(notificationManager *notification.Manager, cfg *config.Config) {
    ctx := context.Background()

    // Enable SMS for demonstration
    cfg.Notification.SMS.Enabled = true
    cfg.Notification.SMS.Provider = "twilio"
    cfg.Notification.SMS.Twilio = &config.TwilioConfig{
        AccountSID: "demo_account_sid",
        AuthToken:  "demo_auth_token",
        FromNumber: "+1234567890",
        Timeout:    30,
    }

    // Re-initialize with SMS enabled
    notificationManager.Close()
    notificationManager, err := notification.NewManager(&cfg.Notification)
    if err != nil {
        log.Fatalf("Failed to create notification manager: %v", err)
    }
    defer notificationManager.Close()

    // Send via specific driver using method chaining
    smsNotification := notification.NewSMSNotification(
        []string{"+1987654321"},
        "Your verification code is: 123456")

    err = notificationManager.Via("sms").Send(ctx, smsNotification)
    if err != nil {
        fmt.Printf("⚠️  SMS send failed (demo mode): %v\n", err)
    } else {
        fmt.Println("✅ SMS sent via driver switching!")
    }
}
```

## Example 4: Async Notifications

```go
func asyncNotificationExample(notificationManager *notification.Manager) {
    ctx := context.Background()

    asyncNotification := notification.NewEmailNotification(
        []string{"async@example.com"},
        "Async Test",
        "This notification was sent asynchronously!")

    err := notificationManager.SendAsync(ctx, asyncNotification)
    if err != nil {
        fmt.Printf("❌ Failed to send async notification: %v\n", err)
    } else {
        fmt.Println("✅ Async notification queued!")
    }
}
```

## Example 5: Scheduled Notifications

```go
func scheduledNotificationExample(notificationManager *notification.Manager) {
    ctx := context.Background()

    scheduledNotification := notification.NewEmailNotification(
        []string{"future@example.com"},
        "Scheduled Message",
        "This message was scheduled for the future!")

    sendAt := time.Now().Add(5 * time.Second)
    err := notificationManager.SendScheduled(ctx, scheduledNotification, sendAt)
    if err != nil {
        fmt.Printf("❌ Failed to schedule notification: %v\n", err)
    } else {
        fmt.Printf("✅ Notification scheduled for %v\n", sendAt.Format("15:04:05"))
    }
}
```

## Example 6: Batch Notifications

```go
func batchNotificationExample(notificationManager *notification.Manager) {
    ctx := context.Background()

    batchNotifications := []*notification.Notification{
        notification.NewEmailNotification([]string{"user1@example.com"}, "Batch 1", "Message 1"),
        notification.NewEmailNotification([]string{"user2@example.com"}, "Batch 2", "Message 2"),
        notification.NewEmailNotification([]string{"user3@example.com"}, "Batch 3", "Message 3"),
    }

    err := notificationManager.SendBatch(ctx, batchNotifications)
    if err != nil {
        fmt.Printf("❌ Failed to send batch notifications: %v\n", err)
    } else {
        fmt.Println("✅ Batch notifications sent!")
    }
}
```

## Example 7: Broadcasting

```go
func broadcastExample(notificationManager *notification.Manager) {
    ctx := context.Background()

    broadcastNotification := notification.NewNotificationBuilder().
        To("broadcast@example.com").
        Subject("Broadcast Message").
        Body("This message is broadcasted to multiple drivers").
        Build()

    err := notificationManager.Broadcast(ctx, []string{"email"}, broadcastNotification)
    if err != nil {
        fmt.Printf("❌ Failed to broadcast: %v\n", err)
    } else {
        fmt.Println("✅ Message broadcasted!")
    }
}
```

## Example 8: Statistics and Health Check

```go
func statsAndHealthExample(notificationManager *notification.Manager) {
    ctx := context.Background()

    // Get health status
    health := notificationManager.HealthCheck(ctx)
    fmt.Println("Driver Health Status:")
    for driver, err := range health {
        status := "✅ Healthy"
        if err != nil {
            status = fmt.Sprintf("❌ Unhealthy: %v", err)
        }
        fmt.Printf("  %s: %s\n", driver, status)
    }

    // Get statistics
    managerStats := notificationManager.GetManagerStats()
    fmt.Printf("\nManager Statistics:\n")
    fmt.Printf("  Total Notifications: %d\n", managerStats.TotalNotifications)
    fmt.Printf("  Total Failed: %d\n", managerStats.TotalFailed)
    fmt.Printf("  By Driver: %v\n", managerStats.ByDriver)
    fmt.Printf("  By Type: %v\n", managerStats.ByType)

    // Get driver-specific stats
    allStats, err := notificationManager.GetAllStats()
    if err != nil {
        fmt.Printf("❌ Failed to get driver stats: %v\n", err)
    } else {
        fmt.Println("\nDriver-Specific Statistics:")
        for driverName, stats := range allStats {
            fmt.Printf("  %s:\n", driverName)
            fmt.Printf("    Sent: %d, Failed: %d, Error Rate: %.1f%%\n",
                stats.TotalSent, stats.TotalFailed, stats.ErrorRate)
            fmt.Printf("    Average Latency: %v, Uptime: %v\n",
                stats.AverageLatency, stats.Uptime)
        }
    }
}
```

## Message Broker Integration

You can integrate the notification system with the existing message broker for async processing:

```go
func messageBrokerIntegrationExample() {
    // Example: Publishing notification jobs to message broker
    // notificationJob := messagebroker.NewJob("notifications", "process_notification", map[string]interface{}{
    //     "type": "email",
    //     "to": []string{"user@example.com"},
    //     "subject": "Welcome!",
    //     "body": "Welcome to our platform!",
    // })

    // messageBrokerManager.EnqueueJob(ctx, "notifications", notificationJob)

    // Example: Processing notification jobs
    // messageBrokerManager.ProcessJobs(ctx, "notifications", func(job *messagebroker.Job) error {
    //     // Extract notification data from job
    //     // Create notification using builder
    //     // Send via notification manager
    //     return nil
    // })
}
```

## Configuration

### Environment Variables

Configure the notification system via environment variables:

```bash
# General notification settings
NOTIFICATION_ENABLED=true
NOTIFICATION_DEFAULT_DRIVER=email

# Email configuration
NOTIFICATION_EMAIL_ENABLED=true
NOTIFICATION_EMAIL_PROVIDER=smtp
EMAIL_SMTP_HOST=localhost
EMAIL_SMTP_PORT=1025
EMAIL_SMTP_USERNAME=
EMAIL_SMTP_PASSWORD=
EMAIL_FROM=noreply@go-template.com
EMAIL_FROM_NAME="Go Template Notifications"
EMAIL_USE_TLS=false
EMAIL_USE_STARTTLS=false
EMAIL_INSECURE_SKIP=true
EMAIL_TIMEOUT=30

# SMS configuration (Twilio)
NOTIFICATION_SMS_ENABLED=true
NOTIFICATION_SMS_PROVIDER=twilio
TWILIO_ACCOUNT_SID=your_account_sid
TWILIO_AUTH_TOKEN=your_auth_token
TWILIO_FROM_NUMBER=+1234567890
TWILIO_TIMEOUT=30
```

## Tips

- **Development Testing**: Use MailHog (port 1025) for local email testing
- **Environment Variables**: Configure environment variables in `.env` for production
- **Driver Configuration**: Enable different drivers via environment variables
- **Logging**: Check logs for detailed notification tracking
- **Health Monitoring**: Use health check endpoints to monitor driver status
- **Statistics**: Monitor notification statistics for performance insights
- **Async Processing**: Use async notifications for better performance
- **Batch Operations**: Use batch sending for multiple notifications
- **Middleware**: Implement custom middleware for notification processing
- **Error Handling**: Always handle errors and implement retry logic
- **Testing**: Use demo configurations for testing without real external services