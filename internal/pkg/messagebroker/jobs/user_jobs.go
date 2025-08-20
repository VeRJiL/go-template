package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/VeRJiL/go-template/internal/pkg/messagebroker"
)

// Job types for user operations
const (
	SendWelcomeEmailJob      = "send_welcome_email"
	SendPasswordResetJob     = "send_password_reset"
	SendEmailVerificationJob = "send_email_verification"
	UserDataExportJob        = "user_data_export"
	UserDataCleanupJob       = "user_data_cleanup"
	UserActivityReportJob    = "user_activity_report"
	UserNotificationJob      = "user_notification"
)

// Job queue names
const (
	EmailQueue        = "emails"
	ReportsQueue      = "reports"
	CleanupQueue      = "cleanup"
	NotificationQueue = "notifications"
	HighPriorityQueue = "high_priority"
)

// Email job data structures
type EmailJobData struct {
	UserID      uint                   `json:"user_id"`
	Email       string                 `json:"email"`
	TemplateID  string                 `json:"template_id"`
	Subject     string                 `json:"subject"`
	Variables   map[string]interface{} `json:"variables"`
	Priority    int                    `json:"priority"`
	ScheduledAt *time.Time             `json:"scheduled_at,omitempty"`
}

type WelcomeEmailJobData struct {
	EmailJobData
	Username        string `json:"username"`
	VerificationURL string `json:"verification_url,omitempty"`
}

type PasswordResetJobData struct {
	EmailJobData
	ResetToken string    `json:"reset_token"`
	ExpiresAt  time.Time `json:"expires_at"`
}

// Export job data
type UserDataExportJobData struct {
	UserID      uint      `json:"user_id"`
	Email       string    `json:"email"`
	RequestedBy uint      `json:"requested_by"`
	ExportType  string    `json:"export_type"` // full, partial
	Format      string    `json:"format"`      // json, csv, pdf
	RequestedAt time.Time `json:"requested_at"`
}

// Cleanup job data
type UserDataCleanupJobData struct {
	UserID     uint      `json:"user_id"`
	DeletedAt  time.Time `json:"deleted_at"`
	RetainDays int       `json:"retain_days"`
	DataTypes  []string  `json:"data_types"` // profiles, posts, comments, etc.
}

// Notification job data
type UserNotificationJobData struct {
	UserID      uint                   `json:"user_id"`
	Type        string                 `json:"type"`
	Title       string                 `json:"title"`
	Message     string                 `json:"message"`
	Data        map[string]interface{} `json:"data"`
	Channels    []string               `json:"channels"` // push, email, sms
	Priority    int                    `json:"priority"`
	ScheduledAt *time.Time             `json:"scheduled_at,omitempty"`
}

// UserJobEnqueuer handles enqueueing user-related jobs
type UserJobEnqueuer struct {
	messageBroker *messagebroker.Manager
}

// NewUserJobEnqueuer creates a new user job enqueuer
func NewUserJobEnqueuer(broker *messagebroker.Manager) *UserJobEnqueuer {
	return &UserJobEnqueuer{
		messageBroker: broker,
	}
}

// EnqueueWelcomeEmail enqueues a welcome email job
func (e *UserJobEnqueuer) EnqueueWelcomeEmail(ctx context.Context, data WelcomeEmailJobData) error {
	job, err := messagebroker.NewJob(EmailQueue, SendWelcomeEmailJob, data)
	if err != nil {
		return fmt.Errorf("failed to create welcome email job: %w", err)
	}

	job = job.WithPriority(data.Priority)
	if data.ScheduledAt != nil {
		delay := time.Until(*data.ScheduledAt)
		if delay > 0 {
			job = job.WithDelay(delay)
		}
	}

	return e.messageBroker.EnqueueJob(ctx, EmailQueue, job)
}

// EnqueuePasswordResetEmail enqueues a password reset email job
func (e *UserJobEnqueuer) EnqueuePasswordResetEmail(ctx context.Context, data PasswordResetJobData) error {
	job, err := messagebroker.NewJob(HighPriorityQueue, SendPasswordResetJob, data)
	if err != nil {
		return fmt.Errorf("failed to create password reset job: %w", err)
	}

	job = job.WithPriority(10) // High priority
	return e.messageBroker.EnqueueJob(ctx, HighPriorityQueue, job)
}

// EnqueueEmailVerification enqueues an email verification job
func (e *UserJobEnqueuer) EnqueueEmailVerification(ctx context.Context, data EmailJobData) error {
	job, err := messagebroker.NewJob(EmailQueue, SendEmailVerificationJob, data)
	if err != nil {
		return fmt.Errorf("failed to create email verification job: %w", err)
	}

	job = job.WithPriority(8) // High priority
	return e.messageBroker.EnqueueJob(ctx, EmailQueue, job)
}

// EnqueueUserDataExport enqueues a user data export job
func (e *UserJobEnqueuer) EnqueueUserDataExport(ctx context.Context, data UserDataExportJobData) error {
	job, err := messagebroker.NewJob(ReportsQueue, UserDataExportJob, data)
	if err != nil {
		return fmt.Errorf("failed to create user data export job: %w", err)
	}

	job = job.WithPriority(5) // Medium priority
	return e.messageBroker.EnqueueJob(ctx, ReportsQueue, job)
}

// EnqueueUserDataCleanup enqueues a user data cleanup job
func (e *UserJobEnqueuer) EnqueueUserDataCleanup(ctx context.Context, data UserDataCleanupJobData, delay time.Duration) error {
	job, err := messagebroker.NewJob(CleanupQueue, UserDataCleanupJob, data)
	if err != nil {
		return fmt.Errorf("failed to create user data cleanup job: %w", err)
	}

	job = job.WithPriority(2).WithDelay(delay) // Low priority, delayed
	return e.messageBroker.EnqueueJob(ctx, CleanupQueue, job)
}

// EnqueueUserNotification enqueues a user notification job
func (e *UserJobEnqueuer) EnqueueUserNotification(ctx context.Context, data UserNotificationJobData) error {
	queueName := NotificationQueue
	if data.Priority >= 8 {
		queueName = HighPriorityQueue
	}

	job, err := messagebroker.NewJob(queueName, UserNotificationJob, data)
	if err != nil {
		return fmt.Errorf("failed to create user notification job: %w", err)
	}

	job = job.WithPriority(data.Priority)
	if data.ScheduledAt != nil {
		delay := time.Until(*data.ScheduledAt)
		if delay > 0 {
			job = job.WithDelay(delay)
		}
	}

	return e.messageBroker.EnqueueJob(ctx, queueName, job)
}

// EnqueueDelayedJob enqueues a delayed job
func (e *UserJobEnqueuer) EnqueueDelayedJob(ctx context.Context, queue, handler string, data interface{}, delay time.Duration, priority int) error {
	job, err := messagebroker.NewJob(queue, handler, data)
	if err != nil {
		return fmt.Errorf("failed to create delayed job: %w", err)
	}

	job = job.WithPriority(priority).WithDelay(delay)
	return e.messageBroker.EnqueueJob(ctx, queue, job)
}

// UserJobProcessor defines the interface for processing user jobs
type UserJobProcessor interface {
	ProcessWelcomeEmail(ctx context.Context, data WelcomeEmailJobData) error
	ProcessPasswordResetEmail(ctx context.Context, data PasswordResetJobData) error
	ProcessEmailVerification(ctx context.Context, data EmailJobData) error
	ProcessUserDataExport(ctx context.Context, data UserDataExportJobData) error
	ProcessUserDataCleanup(ctx context.Context, data UserDataCleanupJobData) error
	ProcessUserNotification(ctx context.Context, data UserNotificationJobData) error
}

// UserJobWorker handles processing user jobs
type UserJobWorker struct {
	messageBroker *messagebroker.Manager
	processor     UserJobProcessor
}

// NewUserJobWorker creates a new user job worker
func NewUserJobWorker(broker *messagebroker.Manager, processor UserJobProcessor) *UserJobWorker {
	return &UserJobWorker{
		messageBroker: broker,
		processor:     processor,
	}
}

// StartEmailWorker starts processing email jobs
func (w *UserJobWorker) StartEmailWorker(ctx context.Context) error {
	return w.messageBroker.ProcessJobs(ctx, EmailQueue, func(ctx context.Context, job *messagebroker.Job) error {
		switch job.Handler {
		case SendWelcomeEmailJob:
			var data WelcomeEmailJobData
			if err := json.Unmarshal(job.Payload, &data); err != nil {
				return fmt.Errorf("failed to unmarshal welcome email job: %w", err)
			}
			return w.processor.ProcessWelcomeEmail(ctx, data)

		case SendPasswordResetJob:
			var data PasswordResetJobData
			if err := json.Unmarshal(job.Payload, &data); err != nil {
				return fmt.Errorf("failed to unmarshal password reset job: %w", err)
			}
			return w.processor.ProcessPasswordResetEmail(ctx, data)

		case SendEmailVerificationJob:
			var data EmailJobData
			if err := json.Unmarshal(job.Payload, &data); err != nil {
				return fmt.Errorf("failed to unmarshal email verification job: %w", err)
			}
			return w.processor.ProcessEmailVerification(ctx, data)

		default:
			return fmt.Errorf("unknown email job handler: %s", job.Handler)
		}
	})
}

// StartReportsWorker starts processing report jobs
func (w *UserJobWorker) StartReportsWorker(ctx context.Context) error {
	return w.messageBroker.ProcessJobs(ctx, ReportsQueue, func(ctx context.Context, job *messagebroker.Job) error {
		switch job.Handler {
		case UserDataExportJob:
			var data UserDataExportJobData
			if err := json.Unmarshal(job.Payload, &data); err != nil {
				return fmt.Errorf("failed to unmarshal user data export job: %w", err)
			}
			return w.processor.ProcessUserDataExport(ctx, data)

		case UserActivityReportJob:
			// Handle activity report job
			return fmt.Errorf("user activity report job not implemented yet")

		default:
			return fmt.Errorf("unknown reports job handler: %s", job.Handler)
		}
	})
}

// StartCleanupWorker starts processing cleanup jobs
func (w *UserJobWorker) StartCleanupWorker(ctx context.Context) error {
	return w.messageBroker.ProcessJobs(ctx, CleanupQueue, func(ctx context.Context, job *messagebroker.Job) error {
		switch job.Handler {
		case UserDataCleanupJob:
			var data UserDataCleanupJobData
			if err := json.Unmarshal(job.Payload, &data); err != nil {
				return fmt.Errorf("failed to unmarshal user data cleanup job: %w", err)
			}
			return w.processor.ProcessUserDataCleanup(ctx, data)

		default:
			return fmt.Errorf("unknown cleanup job handler: %s", job.Handler)
		}
	})
}

// StartNotificationWorker starts processing notification jobs
func (w *UserJobWorker) StartNotificationWorker(ctx context.Context) error {
	return w.messageBroker.ProcessJobs(ctx, NotificationQueue, func(ctx context.Context, job *messagebroker.Job) error {
		switch job.Handler {
		case UserNotificationJob:
			var data UserNotificationJobData
			if err := json.Unmarshal(job.Payload, &data); err != nil {
				return fmt.Errorf("failed to unmarshal user notification job: %w", err)
			}
			return w.processor.ProcessUserNotification(ctx, data)

		default:
			return fmt.Errorf("unknown notification job handler: %s", job.Handler)
		}
	})
}

// StartHighPriorityWorker starts processing high priority jobs
func (w *UserJobWorker) StartHighPriorityWorker(ctx context.Context) error {
	return w.messageBroker.ProcessJobs(ctx, HighPriorityQueue, func(ctx context.Context, job *messagebroker.Job) error {
		switch job.Handler {
		case SendPasswordResetJob:
			var data PasswordResetJobData
			if err := json.Unmarshal(job.Payload, &data); err != nil {
				return fmt.Errorf("failed to unmarshal password reset job: %w", err)
			}
			return w.processor.ProcessPasswordResetEmail(ctx, data)

		case UserNotificationJob:
			var data UserNotificationJobData
			if err := json.Unmarshal(job.Payload, &data); err != nil {
				return fmt.Errorf("failed to unmarshal high priority notification job: %w", err)
			}
			return w.processor.ProcessUserNotification(ctx, data)

		default:
			return fmt.Errorf("unknown high priority job handler: %s", job.Handler)
		}
	})
}

// StartAllWorkers starts all job workers concurrently
func (w *UserJobWorker) StartAllWorkers(ctx context.Context) {
	// Start workers in separate goroutines
	go func() {
		if err := w.StartEmailWorker(ctx); err != nil {
			fmt.Printf("Email worker error: %v\n", err)
		}
	}()

	go func() {
		if err := w.StartReportsWorker(ctx); err != nil {
			fmt.Printf("Reports worker error: %v\n", err)
		}
	}()

	go func() {
		if err := w.StartCleanupWorker(ctx); err != nil {
			fmt.Printf("Cleanup worker error: %v\n", err)
		}
	}()

	go func() {
		if err := w.StartNotificationWorker(ctx); err != nil {
			fmt.Printf("Notification worker error: %v\n", err)
		}
	}()

	go func() {
		if err := w.StartHighPriorityWorker(ctx); err != nil {
			fmt.Printf("High priority worker error: %v\n", err)
		}
	}()
}
