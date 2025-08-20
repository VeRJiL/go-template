package events

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/VeRJiL/go-template/internal/pkg/messagebroker"
)

// UserEvent types
const (
	UserCreatedEvent  = "user.created"
	UserUpdatedEvent  = "user.updated"
	UserDeletedEvent  = "user.deleted"
	UserLoginEvent    = "user.login"
	UserLogoutEvent   = "user.logout"
	UserPasswordReset = "user.password_reset"
	UserEmailVerified = "user.email_verified"
)

// UserEventData contains common user event information
type UserEventData struct {
	UserID    uint                   `json:"user_id"`
	Email     string                 `json:"email"`
	Username  string                 `json:"username,omitempty"`
	Action    string                 `json:"action"`
	Timestamp time.Time              `json:"timestamp"`
	IPAddress string                 `json:"ip_address,omitempty"`
	UserAgent string                 `json:"user_agent,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// UserCreatedData specific data for user creation
type UserCreatedData struct {
	UserEventData
	IsEmailVerified bool   `json:"is_email_verified"`
	Role            string `json:"role"`
}

// UserUpdatedData specific data for user updates
type UserUpdatedData struct {
	UserEventData
	ChangedFields []string               `json:"changed_fields"`
	OldValues     map[string]interface{} `json:"old_values,omitempty"`
}

// UserDeletedData specific data for user deletion
type UserDeletedData struct {
	UserEventData
	DeletedBy uint   `json:"deleted_by,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

// UserLoginData specific data for user login
type UserLoginData struct {
	UserEventData
	LoginMethod   string `json:"login_method"` // email, social, etc.
	Success       bool   `json:"success"`
	FailureReason string `json:"failure_reason,omitempty"`
}

// UserEventPublisher handles publishing user events
type UserEventPublisher struct {
	messageBroker *messagebroker.Manager
}

// NewUserEventPublisher creates a new user event publisher
func NewUserEventPublisher(broker *messagebroker.Manager) *UserEventPublisher {
	return &UserEventPublisher{
		messageBroker: broker,
	}
}

// PublishUserCreated publishes a user created event
func (p *UserEventPublisher) PublishUserCreated(ctx context.Context, data UserCreatedData) error {
	data.Action = "created"
	data.Timestamp = time.Now()

	return p.messageBroker.PublishJSON(ctx, UserCreatedEvent, data)
}

// PublishUserUpdated publishes a user updated event
func (p *UserEventPublisher) PublishUserUpdated(ctx context.Context, data UserUpdatedData) error {
	data.Action = "updated"
	data.Timestamp = time.Now()

	return p.messageBroker.PublishJSON(ctx, UserUpdatedEvent, data)
}

// PublishUserDeleted publishes a user deleted event
func (p *UserEventPublisher) PublishUserDeleted(ctx context.Context, data UserDeletedData) error {
	data.Action = "deleted"
	data.Timestamp = time.Now()

	return p.messageBroker.PublishJSON(ctx, UserDeletedEvent, data)
}

// PublishUserLogin publishes a user login event
func (p *UserEventPublisher) PublishUserLogin(ctx context.Context, data UserLoginData) error {
	data.Action = "login"
	data.Timestamp = time.Now()

	return p.messageBroker.PublishJSON(ctx, UserLoginEvent, data)
}

// PublishUserLogout publishes a user logout event
func (p *UserEventPublisher) PublishUserLogout(ctx context.Context, data UserEventData) error {
	data.Action = "logout"
	data.Timestamp = time.Now()

	return p.messageBroker.PublishJSON(ctx, UserLogoutEvent, data)
}

// PublishDelayedEvent publishes a delayed user event
func (p *UserEventPublisher) PublishDelayedEvent(ctx context.Context, eventType string, data interface{}, delay time.Duration) error {
	message, err := messagebroker.NewMessage(eventType, data)
	if err != nil {
		return fmt.Errorf("failed to create delayed message: %w", err)
	}

	return p.messageBroker.PublishWithDelay(ctx, eventType, message, delay)
}

// UserEventHandler defines the interface for handling user events
type UserEventHandler interface {
	HandleUserCreated(ctx context.Context, data UserCreatedData) error
	HandleUserUpdated(ctx context.Context, data UserUpdatedData) error
	HandleUserDeleted(ctx context.Context, data UserDeletedData) error
	HandleUserLogin(ctx context.Context, data UserLoginData) error
	HandleUserLogout(ctx context.Context, data UserEventData) error
}

// UserEventSubscriber handles subscribing to user events
type UserEventSubscriber struct {
	messageBroker *messagebroker.Manager
	handler       UserEventHandler
}

// NewUserEventSubscriber creates a new user event subscriber
func NewUserEventSubscriber(broker *messagebroker.Manager, handler UserEventHandler) *UserEventSubscriber {
	return &UserEventSubscriber{
		messageBroker: broker,
		handler:       handler,
	}
}

// SubscribeToUserEvents subscribes to all user events
func (s *UserEventSubscriber) SubscribeToUserEvents(ctx context.Context) error {
	// Subscribe to user created events
	err := s.messageBroker.Subscribe(ctx, UserCreatedEvent, func(ctx context.Context, msg *messagebroker.Message) error {
		var data UserCreatedData
		if err := json.Unmarshal(msg.Payload, &data); err != nil {
			return fmt.Errorf("failed to unmarshal user created event: %w", err)
		}
		return s.handler.HandleUserCreated(ctx, data)
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to user created events: %w", err)
	}

	// Subscribe to user updated events
	err = s.messageBroker.Subscribe(ctx, UserUpdatedEvent, func(ctx context.Context, msg *messagebroker.Message) error {
		var data UserUpdatedData
		if err := json.Unmarshal(msg.Payload, &data); err != nil {
			return fmt.Errorf("failed to unmarshal user updated event: %w", err)
		}
		return s.handler.HandleUserUpdated(ctx, data)
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to user updated events: %w", err)
	}

	// Subscribe to user deleted events
	err = s.messageBroker.Subscribe(ctx, UserDeletedEvent, func(ctx context.Context, msg *messagebroker.Message) error {
		var data UserDeletedData
		if err := json.Unmarshal(msg.Payload, &data); err != nil {
			return fmt.Errorf("failed to unmarshal user deleted event: %w", err)
		}
		return s.handler.HandleUserDeleted(ctx, data)
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to user deleted events: %w", err)
	}

	// Subscribe to user login events
	err = s.messageBroker.Subscribe(ctx, UserLoginEvent, func(ctx context.Context, msg *messagebroker.Message) error {
		var data UserLoginData
		if err := json.Unmarshal(msg.Payload, &data); err != nil {
			return fmt.Errorf("failed to unmarshal user login event: %w", err)
		}
		return s.handler.HandleUserLogin(ctx, data)
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to user login events: %w", err)
	}

	// Subscribe to user logout events
	err = s.messageBroker.Subscribe(ctx, UserLogoutEvent, func(ctx context.Context, msg *messagebroker.Message) error {
		var data UserEventData
		if err := json.Unmarshal(msg.Payload, &data); err != nil {
			return fmt.Errorf("failed to unmarshal user logout event: %w", err)
		}
		return s.handler.HandleUserLogout(ctx, data)
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to user logout events: %w", err)
	}

	return nil
}

// SubscribeToSpecificEvent subscribes to a specific user event with a custom handler
func (s *UserEventSubscriber) SubscribeToSpecificEvent(ctx context.Context, eventType string, handler messagebroker.MessageHandler) error {
	return s.messageBroker.Subscribe(ctx, eventType, handler)
}

// SubscribeWithGroup subscribes to user events with a specific consumer group
func (s *UserEventSubscriber) SubscribeWithGroup(ctx context.Context, eventType, group string, handler messagebroker.MessageHandler) error {
	return s.messageBroker.SubscribeWithGroup(ctx, eventType, group, handler)
}
