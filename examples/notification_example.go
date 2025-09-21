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
	fmt.Println("🚀 Go Template Notification System Example")
	fmt.Println("==========================================")

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

	// Example 1: Basic Email Notification
	fmt.Println("\n📧 Example 1: Basic Email Notification")
	fmt.Println("--------------------------------------")

	ctx := context.Background()
	err = notificationManager.SendEmail(ctx,
		[]string{"user@example.com"},
		"Welcome to Go Template!",
		"Thank you for using our Go Template notification system!")

	if err != nil {
		fmt.Printf("❌ Failed to send email: %v\n", err)
	} else {
		fmt.Println("✅ Email sent successfully!")
	}

	// Example 2: Using Notification Builder
	fmt.Println("\n🏗️  Example 2: Using Notification Builder")
	fmt.Println("----------------------------------------")

	notification := notification.NewNotificationBuilder().
		To("admin@example.com", "support@example.com").
		Subject("System Alert").
		Body("<h1>Server Status</h1><p>All systems operational</p>").
		Priority(notification.PriorityHigh).
		Tags("system", "alert").
		Metadata("source", "monitoring").
		Build()

	err = notificationManager.Send(ctx, notification)
	if err != nil {
		fmt.Printf("❌ Failed to send notification: %v\n", err)
	} else {
		fmt.Println("✅ System alert sent successfully!")
	}

	// Example 3: Driver Switching (Method Chaining)
	fmt.Println("\n🔄 Example 3: Driver Switching")
	fmt.Println("------------------------------")

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
	notificationManager, err = notification.NewManager(&cfg.Notification)
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

	// Example 4: Async Notifications
	fmt.Println("\n⚡ Example 4: Async Notifications")
	fmt.Println("----------------------------------")

	asyncNotification := notification.NewEmailNotification(
		[]string{"async@example.com"},
		"Async Test",
		"This notification was sent asynchronously!")

	err = notificationManager.SendAsync(ctx, asyncNotification)
	if err != nil {
		fmt.Printf("❌ Failed to send async notification: %v\n", err)
	} else {
		fmt.Println("✅ Async notification queued!")
	}

	// Example 5: Scheduled Notifications
	fmt.Println("\n⏰ Example 5: Scheduled Notifications")
	fmt.Println("------------------------------------")

	scheduledNotification := notification.NewEmailNotification(
		[]string{"future@example.com"},
		"Scheduled Message",
		"This message was scheduled for the future!")

	sendAt := time.Now().Add(5 * time.Second)
	err = notificationManager.SendScheduled(ctx, scheduledNotification, sendAt)
	if err != nil {
		fmt.Printf("❌ Failed to schedule notification: %v\n", err)
	} else {
		fmt.Printf("✅ Notification scheduled for %v\n", sendAt.Format("15:04:05"))
	}

	// Example 6: Batch Notifications
	fmt.Println("\n📦 Example 6: Batch Notifications")
	fmt.Println("----------------------------------")

	batchNotifications := []*notification.Notification{
		notification.NewEmailNotification([]string{"user1@example.com"}, "Batch 1", "Message 1"),
		notification.NewEmailNotification([]string{"user2@example.com"}, "Batch 2", "Message 2"),
		notification.NewEmailNotification([]string{"user3@example.com"}, "Batch 3", "Message 3"),
	}

	err = notificationManager.SendBatch(ctx, batchNotifications)
	if err != nil {
		fmt.Printf("❌ Failed to send batch notifications: %v\n", err)
	} else {
		fmt.Println("✅ Batch notifications sent!")
	}

	// Example 7: Broadcasting to Multiple Drivers
	fmt.Println("\n📢 Example 7: Broadcasting")
	fmt.Println("-------------------------")

	broadcastNotification := notification.NewNotificationBuilder().
		To("broadcast@example.com").
		Subject("Broadcast Message").
		Body("This message is broadcasted to multiple drivers").
		Build()

	err = notificationManager.Broadcast(ctx, []string{"email"}, broadcastNotification)
	if err != nil {
		fmt.Printf("❌ Failed to broadcast: %v\n", err)
	} else {
		fmt.Println("✅ Message broadcasted!")
	}

	// Example 8: Statistics and Health Check
	fmt.Println("\n📊 Example 8: Statistics and Health")
	fmt.Println("-----------------------------------")

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

	// Wait a bit for scheduled notification
	fmt.Println("\n⏳ Waiting for scheduled notification...")
	time.Sleep(6 * time.Second)

	fmt.Println("\n🎉 Notification system example completed!")
	fmt.Println("=========================================")
	fmt.Println("💡 Tips:")
	fmt.Println("   - Configure environment variables in .env for production")
	fmt.Println("   - Use MailHog (port 1025) for local email testing")
	fmt.Println("   - Enable different drivers via environment variables")
	fmt.Println("   - Check logs for detailed notification tracking")
}

// Example middleware for notification processing
type LoggingMiddleware struct{}

func (m *LoggingMiddleware) Process(ctx context.Context, notif *notification.Notification) (*notification.Notification, error) {
	fmt.Printf("🔍 Processing notification: %s to %v\n", notif.Subject, notif.To)
	return notif, nil
}

// Example of integrating with the existing message broker system
func MessageBrokerIntegrationExample() {
	fmt.Println("\n🔗 Message Broker Integration Example")
	fmt.Println("------------------------------------")

	// This shows how you can integrate the notification system
	// with the existing message broker for async processing

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

	fmt.Println("💡 This example shows how to integrate with the existing message broker")
	fmt.Println("   for async notification processing and job queuing.")
}