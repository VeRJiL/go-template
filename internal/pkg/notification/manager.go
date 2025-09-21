package notification

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/VeRJiL/go-template/internal/config"
)

// Manager manages notification drivers with Laravel-style facade pattern
type Manager struct {
	drivers       map[string]NotificationDriver
	defaultDriver string
	config        *config.NotificationConfig
	mu            sync.RWMutex
	stats         *ManagerStats
	middleware    []Middleware
}

// ManagerStats represents overall manager statistics
type ManagerStats struct {
	TotalNotifications int64              `json:"total_notifications"`
	TotalFailed        int64              `json:"total_failed"`
	ByDriver           map[string]int64   `json:"by_driver"`
	ByType             map[string]int64   `json:"by_type"`
	LastUsed           map[string]time.Time `json:"last_used"`
	mu                 sync.RWMutex
}

// Middleware interface for notification processing
type Middleware interface {
	Process(ctx context.Context, notification *Notification) (*Notification, error)
}

// NewManager creates a new notification manager
func NewManager(config *config.NotificationConfig) (*Manager, error) {
	if config == nil {
		return nil, fmt.Errorf("notification config cannot be nil")
	}

	manager := &Manager{
		drivers:       make(map[string]NotificationDriver),
		defaultDriver: config.DefaultDriver,
		config:        config,
		stats: &ManagerStats{
			ByDriver: make(map[string]int64),
			ByType:   make(map[string]int64),
			LastUsed: make(map[string]time.Time),
		},
		middleware: make([]Middleware, 0),
	}

	// Initialize configured drivers
	if err := manager.initializeDrivers(); err != nil {
		return nil, fmt.Errorf("failed to initialize drivers: %w", err)
	}

	return manager, nil
}

// initializeDrivers initializes all configured drivers
func (m *Manager) initializeDrivers() error {
	// Initialize email drivers
	if m.config.Email.Enabled {
		if err := m.initializeEmailDrivers(); err != nil {
			return fmt.Errorf("failed to initialize email drivers: %w", err)
		}
	}

	// Initialize SMS drivers
	if m.config.SMS.Enabled {
		if err := m.initializeSMSDrivers(); err != nil {
			return fmt.Errorf("failed to initialize SMS drivers: %w", err)
		}
	}

	// Initialize push notification drivers
	if m.config.Push.Enabled {
		if err := m.initializePushDrivers(); err != nil {
			return fmt.Errorf("failed to initialize push drivers: %w", err)
		}
	}

	// Initialize social media drivers
	if m.config.Social.Enabled {
		if err := m.initializeSocialDrivers(); err != nil {
			return fmt.Errorf("failed to initialize social drivers: %w", err)
		}
	}

	// Validate default driver exists
	if _, exists := m.drivers[m.defaultDriver]; !exists && len(m.drivers) > 0 {
		// Set first available driver as default if configured default doesn't exist
		for name := range m.drivers {
			m.defaultDriver = name
			break
		}
	}

	if len(m.drivers) == 0 {
		return fmt.Errorf("no notification drivers configured")
	}

	return nil
}

// Driver returns a specific notification driver
func (m *Manager) Driver(name string) NotificationDriver {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if driver, exists := m.drivers[name]; exists {
		return driver
	}
	return nil
}

// SetDefaultDriver changes the default driver
func (m *Manager) SetDefaultDriver(driverName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.drivers[driverName]; !exists {
		return fmt.Errorf("driver %s not found", driverName)
	}

	m.defaultDriver = driverName
	return nil
}

// GetDefaultDriver returns the name of the default driver
func (m *Manager) GetDefaultDriver() string {
	return m.defaultDriver
}

// GetAvailableDrivers returns list of available drivers
func (m *Manager) GetAvailableDrivers() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	drivers := make([]string, 0, len(m.drivers))
	for name := range m.drivers {
		drivers = append(drivers, name)
	}
	return drivers
}

// AddMiddleware adds middleware to the notification pipeline
func (m *Manager) AddMiddleware(middleware Middleware) {
	m.middleware = append(m.middleware, middleware)
}

// Default driver facade methods - these delegate to the default driver

// Send sends a notification using the default driver
func (m *Manager) Send(ctx context.Context, notification *Notification) error {
	driver := m.Driver(m.defaultDriver)
	if driver == nil {
		return fmt.Errorf("default driver %s not available", m.defaultDriver)
	}

	// Apply middleware
	processedNotification, err := m.applyMiddleware(ctx, notification)
	if err != nil {
		return fmt.Errorf("middleware processing failed: %w", err)
	}

	// Update stats
	m.updateStats(m.defaultDriver, string(notification.Type))

	return driver.Send(ctx, processedNotification)
}

// SendAsync sends a notification asynchronously using the default driver
func (m *Manager) SendAsync(ctx context.Context, notification *Notification) error {
	driver := m.Driver(m.defaultDriver)
	if driver == nil {
		return fmt.Errorf("default driver %s not available", m.defaultDriver)
	}

	// Apply middleware
	processedNotification, err := m.applyMiddleware(ctx, notification)
	if err != nil {
		return fmt.Errorf("middleware processing failed: %w", err)
	}

	// Update stats
	m.updateStats(m.defaultDriver, string(notification.Type))

	return driver.SendAsync(ctx, processedNotification)
}

// SendBatch sends multiple notifications using the default driver
func (m *Manager) SendBatch(ctx context.Context, notifications []*Notification) error {
	driver := m.Driver(m.defaultDriver)
	if driver == nil {
		return fmt.Errorf("default driver %s not available", m.defaultDriver)
	}

	// Apply middleware to each notification
	processedNotifications := make([]*Notification, len(notifications))
	for i, notification := range notifications {
		processed, err := m.applyMiddleware(ctx, notification)
		if err != nil {
			return fmt.Errorf("middleware processing failed for notification %d: %w", i, err)
		}
		processedNotifications[i] = processed

		// Update stats
		m.updateStats(m.defaultDriver, string(notification.Type))
	}

	return driver.SendBatch(ctx, processedNotifications)
}

// SendScheduled sends a notification at a specific time using the default driver
func (m *Manager) SendScheduled(ctx context.Context, notification *Notification, sendAt time.Time) error {
	driver := m.Driver(m.defaultDriver)
	if driver == nil {
		return fmt.Errorf("default driver %s not available", m.defaultDriver)
	}

	// Apply middleware
	processedNotification, err := m.applyMiddleware(ctx, notification)
	if err != nil {
		return fmt.Errorf("middleware processing failed: %w", err)
	}

	// Update stats
	m.updateStats(m.defaultDriver, string(notification.Type))

	return driver.SendScheduled(ctx, processedNotification, sendAt)
}

// GetStats returns statistics from the default driver
func (m *Manager) GetStats() (*DriverStats, error) {
	driver := m.Driver(m.defaultDriver)
	if driver == nil {
		return nil, fmt.Errorf("default driver %s not available", m.defaultDriver)
	}
	return driver.GetStats()
}

// GetAllStats returns statistics from all drivers
func (m *Manager) GetAllStats() (map[string]*DriverStats, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]*DriverStats)
	for name, driver := range m.drivers {
		driverStats, err := driver.GetStats()
		if err != nil {
			return nil, fmt.Errorf("failed to get stats from driver %s: %w", name, err)
		}
		stats[name] = driverStats
	}
	return stats, nil
}

// GetManagerStats returns overall manager statistics
func (m *Manager) GetManagerStats() *ManagerStats {
	m.stats.mu.RLock()
	defer m.stats.mu.RUnlock()

	// Create a copy to avoid race conditions
	stats := &ManagerStats{
		TotalNotifications: m.stats.TotalNotifications,
		TotalFailed:        m.stats.TotalFailed,
		ByDriver:           make(map[string]int64),
		ByType:             make(map[string]int64),
		LastUsed:           make(map[string]time.Time),
	}

	for k, v := range m.stats.ByDriver {
		stats.ByDriver[k] = v
	}
	for k, v := range m.stats.ByType {
		stats.ByType[k] = v
	}
	for k, v := range m.stats.LastUsed {
		stats.LastUsed[k] = v
	}

	return stats
}

// Ping checks if the default driver connection is alive
func (m *Manager) Ping(ctx context.Context) error {
	driver := m.Driver(m.defaultDriver)
	if driver == nil {
		return fmt.Errorf("default driver %s not available", m.defaultDriver)
	}
	return driver.Ping(ctx)
}

// Close closes all drivers
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var lastErr error
	for name, driver := range m.drivers {
		if err := driver.Close(); err != nil {
			fmt.Printf("Error closing notification driver %s: %v\n", name, err)
			lastErr = err
		}
	}
	return lastErr
}

// Method chaining for driver switching (Laravel-style)

// DriverSwitcher allows temporary driver switching for single operations
type DriverSwitcher struct {
	manager    *Manager
	driverName string
}

// Via returns a driver switcher for one-time operations
func (m *Manager) Via(driverName string) *DriverSwitcher {
	return &DriverSwitcher{
		manager:    m,
		driverName: driverName,
	}
}

// Send sends using the specified driver
func (ds *DriverSwitcher) Send(ctx context.Context, notification *Notification) error {
	driver := ds.manager.Driver(ds.driverName)
	if driver == nil {
		return fmt.Errorf("driver %s not available", ds.driverName)
	}

	// Apply middleware
	processedNotification, err := ds.manager.applyMiddleware(ctx, notification)
	if err != nil {
		return fmt.Errorf("middleware processing failed: %w", err)
	}

	// Update stats
	ds.manager.updateStats(ds.driverName, string(notification.Type))

	return driver.Send(ctx, processedNotification)
}

// SendAsync sends asynchronously using the specified driver
func (ds *DriverSwitcher) SendAsync(ctx context.Context, notification *Notification) error {
	driver := ds.manager.Driver(ds.driverName)
	if driver == nil {
		return fmt.Errorf("driver %s not available", ds.driverName)
	}

	// Apply middleware
	processedNotification, err := ds.manager.applyMiddleware(ctx, notification)
	if err != nil {
		return fmt.Errorf("middleware processing failed: %w", err)
	}

	// Update stats
	ds.manager.updateStats(ds.driverName, string(notification.Type))

	return driver.SendAsync(ctx, processedNotification)
}

// SendBatch sends multiple notifications using the specified driver
func (ds *DriverSwitcher) SendBatch(ctx context.Context, notifications []*Notification) error {
	driver := ds.manager.Driver(ds.driverName)
	if driver == nil {
		return fmt.Errorf("driver %s not available", ds.driverName)
	}

	// Apply middleware to each notification
	processedNotifications := make([]*Notification, len(notifications))
	for i, notification := range notifications {
		processed, err := ds.manager.applyMiddleware(ctx, notification)
		if err != nil {
			return fmt.Errorf("middleware processing failed for notification %d: %w", i, err)
		}
		processedNotifications[i] = processed

		// Update stats
		ds.manager.updateStats(ds.driverName, string(notification.Type))
	}

	return driver.SendBatch(ctx, processedNotifications)
}

// Convenience methods for quick sending

// SendEmail sends an email notification quickly
func (m *Manager) SendEmail(ctx context.Context, to []string, subject, body string) error {
	notification := NewEmailNotification(to, subject, body)
	return m.Send(ctx, notification)
}

// SendSMS sends an SMS notification quickly
func (m *Manager) SendSMS(ctx context.Context, to []string, message string) error {
	notification := NewSMSNotification(to, message)
	return m.Send(ctx, notification)
}

// SendPush sends a push notification quickly
func (m *Manager) SendPush(ctx context.Context, to []string, title, body string) error {
	notification := NewPushNotification(to, title, body)
	return m.Send(ctx, notification)
}

// Template-based convenience methods

// SendEmailFromTemplate sends an email using a template
func (m *Manager) SendEmailFromTemplate(ctx context.Context, to []string, template string, vars map[string]interface{}) error {
	notification := NewNotificationBuilder().
		To(to...).
		Template(template, vars).
		Build()
	return m.Send(ctx, notification)
}

// Broadcast sends the same notification via multiple drivers
func (m *Manager) Broadcast(ctx context.Context, drivers []string, notification *Notification) error {
	for _, driverName := range drivers {
		if err := m.Via(driverName).Send(ctx, notification); err != nil {
			return fmt.Errorf("failed to send via driver %s: %w", driverName, err)
		}
	}
	return nil
}

// Helper methods

// applyMiddleware applies all middleware to a notification
func (m *Manager) applyMiddleware(ctx context.Context, notification *Notification) (*Notification, error) {
	result := notification
	for _, middleware := range m.middleware {
		processed, err := middleware.Process(ctx, result)
		if err != nil {
			return nil, err
		}
		result = processed
	}
	return result, nil
}

// updateStats updates manager statistics
func (m *Manager) updateStats(driverName, notificationType string) {
	m.stats.mu.Lock()
	defer m.stats.mu.Unlock()

	m.stats.TotalNotifications++
	m.stats.ByDriver[driverName]++
	m.stats.ByType[notificationType]++
	m.stats.LastUsed[driverName] = time.Now()
}

// Health checking methods

// HealthCheck checks the health of all drivers
func (m *Manager) HealthCheck(ctx context.Context) map[string]error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	health := make(map[string]error)
	for name, driver := range m.drivers {
		health[name] = driver.Ping(ctx)
	}
	return health
}

// GetHealthyDrivers returns list of healthy drivers
func (m *Manager) GetHealthyDrivers(ctx context.Context) []string {
	health := m.HealthCheck(ctx)
	healthy := make([]string, 0)
	for name, err := range health {
		if err == nil {
			healthy = append(healthy, name)
		}
	}
	return healthy
}