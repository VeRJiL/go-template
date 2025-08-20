package messagebroker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/VeRJiL/go-template/internal/pkg/messagebroker/drivers"
)

// Manager manages message brokers with Laravel-style facade pattern
type Manager struct {
	drivers        map[string]MessageBroker
	defaultDriver  string
	config         *MessageBrokerConfig
	mu             sync.RWMutex
	healthCheckers map[string]*healthChecker
}

// healthChecker monitors driver health
type healthChecker struct {
	driver   MessageBroker
	interval time.Duration
	stop     chan bool
	healthy  bool
	mu       sync.RWMutex
}

// NewManager creates a new message broker manager
func NewManager(config *MessageBrokerConfig) (*Manager, error) {
	if config == nil {
		return nil, fmt.Errorf("message broker config cannot be nil")
	}

	manager := &Manager{
		drivers:        make(map[string]MessageBroker),
		defaultDriver:  config.Driver,
		config:         config,
		healthCheckers: make(map[string]*healthChecker),
	}

	// Initialize the configured driver
	if err := manager.initializeDriver(config.Driver); err != nil {
		return nil, fmt.Errorf("failed to initialize driver %s: %w", config.Driver, err)
	}

	return manager, nil
}

// initializeDriver initializes a specific driver
func (m *Manager) initializeDriver(driverName string) error {
	switch driverName {
	case "rabbitmq":
		if m.config.RabbitMQ == nil {
			return fmt.Errorf("RabbitMQ configuration is required")
		}
		driver, err := drivers.NewRabbitMQDriver(m.config.RabbitMQ)
		if err != nil {
			return err
		}
		m.drivers[driverName] = driver
		
	case "kafka":
		if m.config.Kafka == nil {
			return fmt.Errorf("Kafka configuration is required")
		}
		driver, err := drivers.NewKafkaDriver(m.config.Kafka)
		if err != nil {
			return err
		}
		m.drivers[driverName] = driver
		
	case "redis":
		if m.config.Redis == nil {
			return fmt.Errorf("Redis configuration is required")
		}
		driver, err := drivers.NewRedisPubSubDriver(m.config.Redis)
		if err != nil {
			return err
		}
		m.drivers[driverName] = driver
		
	default:
		return fmt.Errorf("unsupported message broker driver: %s", driverName)
	}

	// Start health checking for this driver
	m.startHealthCheck(driverName)
	
	return nil
}

// startHealthCheck starts health monitoring for a driver
func (m *Manager) startHealthCheck(driverName string) {
	driver := m.drivers[driverName]
	if driver == nil {
		return
	}

	checker := &healthChecker{
		driver:   driver,
		interval: 30 * time.Second, // Check every 30 seconds
		stop:     make(chan bool),
		healthy:  true,
	}

	m.healthCheckers[driverName] = checker

	go func() {
		ticker := time.NewTicker(checker.interval)
		defer ticker.Stop()

		for {
			select {
			case <-checker.stop:
				return
			case <-ticker.C:
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				err := driver.Ping(ctx)
				cancel()

				checker.mu.Lock()
				checker.healthy = (err == nil)
				checker.mu.Unlock()

				if err != nil {
					fmt.Printf("Health check failed for driver %s: %v\n", driverName, err)
				}
			}
		}
	}()
}

// Driver returns a specific message broker driver
func (m *Manager) Driver(name string) MessageBroker {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if driver, exists := m.drivers[name]; exists {
		return driver
	}
	
	return nil
}

// GetDefaultDriver returns the name of the default driver
func (m *Manager) GetDefaultDriver() string {
	return m.defaultDriver
}

// SetDefaultDriver changes the default driver
func (m *Manager) SetDefaultDriver(driver string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.drivers[driver]; !exists {
		// Try to initialize the driver if it doesn't exist
		if err := m.initializeDriver(driver); err != nil {
			return fmt.Errorf("failed to set default driver %s: %w", driver, err)
		}
	}

	m.defaultDriver = driver
	return nil
}

// GetAvailableDrivers returns list of available/initialized drivers
func (m *Manager) GetAvailableDrivers() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	drivers := make([]string, 0, len(m.drivers))
	for name := range m.drivers {
		drivers = append(drivers, name)
	}
	return drivers
}

// IsHealthy returns the health status of a driver
func (m *Manager) IsHealthy(driverName string) bool {
	m.mu.RLock()
	checker, exists := m.healthCheckers[driverName]
	m.mu.RUnlock()

	if !exists {
		return false
	}

	checker.mu.RLock()
	healthy := checker.healthy
	checker.mu.RUnlock()

	return healthy
}

// GetHealthStatus returns health status of all drivers
func (m *Manager) GetHealthStatus() map[string]bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := make(map[string]bool)
	for name, checker := range m.healthCheckers {
		checker.mu.RLock()
		status[name] = checker.healthy
		checker.mu.RUnlock()
	}
	return status
}

// Close closes all drivers and stops health checking
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Stop health checkers
	for _, checker := range m.healthCheckers {
		close(checker.stop)
	}

	// Close all drivers
	var lastErr error
	for name, driver := range m.drivers {
		if err := driver.Close(); err != nil {
			fmt.Printf("Error closing driver %s: %v\n", name, err)
			lastErr = err
		}
	}

	return lastErr
}

// Default driver facade methods - these delegate to the default driver
// This provides Laravel-style static method access pattern

// Publish publishes a message using the default driver
func (m *Manager) Publish(ctx context.Context, topic string, message *Message) error {
	driver := m.Driver(m.defaultDriver)
	if driver == nil {
		return fmt.Errorf("default driver %s not available", m.defaultDriver)
	}
	return driver.Publish(ctx, topic, message)
}

// PublishJSON publishes JSON data using the default driver
func (m *Manager) PublishJSON(ctx context.Context, topic string, data interface{}) error {
	driver := m.Driver(m.defaultDriver)
	if driver == nil {
		return fmt.Errorf("default driver %s not available", m.defaultDriver)
	}
	return driver.PublishJSON(ctx, topic, data)
}

// PublishWithDelay publishes a delayed message using the default driver
func (m *Manager) PublishWithDelay(ctx context.Context, topic string, message *Message, delay time.Duration) error {
	driver := m.Driver(m.defaultDriver)
	if driver == nil {
		return fmt.Errorf("default driver %s not available", m.defaultDriver)
	}
	return driver.PublishWithDelay(ctx, topic, message, delay)
}

// Subscribe subscribes to a topic using the default driver
func (m *Manager) Subscribe(ctx context.Context, topic string, handler MessageHandler) error {
	driver := m.Driver(m.defaultDriver)
	if driver == nil {
		return fmt.Errorf("default driver %s not available", m.defaultDriver)
	}
	return driver.Subscribe(ctx, topic, handler)
}

// SubscribeWithGroup subscribes to a topic with a group using the default driver
func (m *Manager) SubscribeWithGroup(ctx context.Context, topic string, group string, handler MessageHandler) error {
	driver := m.Driver(m.defaultDriver)
	if driver == nil {
		return fmt.Errorf("default driver %s not available", m.defaultDriver)
	}
	return driver.SubscribeWithGroup(ctx, topic, group, handler)
}

// EnqueueJob enqueues a job using the default driver
func (m *Manager) EnqueueJob(ctx context.Context, queue string, job *Job) error {
	driver := m.Driver(m.defaultDriver)
	if driver == nil {
		return fmt.Errorf("default driver %s not available", m.defaultDriver)
	}
	return driver.EnqueueJob(ctx, queue, job)
}

// ProcessJobs processes jobs from a queue using the default driver
func (m *Manager) ProcessJobs(ctx context.Context, queue string, handler JobHandler) error {
	driver := m.Driver(m.defaultDriver)
	if driver == nil {
		return fmt.Errorf("default driver %s not available", m.defaultDriver)
	}
	return driver.ProcessJobs(ctx, queue, handler)
}

// CreateTopic creates a topic using the default driver
func (m *Manager) CreateTopic(ctx context.Context, topic string, config *TopicConfig) error {
	driver := m.Driver(m.defaultDriver)
	if driver == nil {
		return fmt.Errorf("default driver %s not available", m.defaultDriver)
	}
	return driver.CreateTopic(ctx, topic, config)
}

// DeleteTopic deletes a topic using the default driver
func (m *Manager) DeleteTopic(ctx context.Context, topic string) error {
	driver := m.Driver(m.defaultDriver)
	if driver == nil {
		return fmt.Errorf("default driver %s not available", m.defaultDriver)
	}
	return driver.DeleteTopic(ctx, topic)
}

// GetTopicInfo returns topic information using the default driver
func (m *Manager) GetTopicInfo(ctx context.Context, topic string) (*TopicInfo, error) {
	driver := m.Driver(m.defaultDriver)
	if driver == nil {
		return nil, fmt.Errorf("default driver %s not available", m.defaultDriver)
	}
	return driver.GetTopicInfo(ctx, topic)
}

// Ping checks if the default driver connection is alive
func (m *Manager) Ping(ctx context.Context) error {
	driver := m.Driver(m.defaultDriver)
	if driver == nil {
		return fmt.Errorf("default driver %s not available", m.defaultDriver)
	}
	return driver.Ping(ctx)
}

// GetStats returns statistics from the default driver
func (m *Manager) GetStats() (*BrokerStats, error) {
	driver := m.Driver(m.defaultDriver)
	if driver == nil {
		return nil, fmt.Errorf("default driver %s not available", m.defaultDriver)
	}
	return driver.GetStats()
}

// Utility methods for easier usage

// SendMessage is a convenience method to send a message quickly
func (m *Manager) SendMessage(topic string, payload interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return m.PublishJSON(ctx, topic, payload)
}

// SendDelayedMessage is a convenience method to send a delayed message
func (m *Manager) SendDelayedMessage(topic string, payload interface{}, delay time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	message, err := NewMessage(topic, payload)
	if err != nil {
		return err
	}

	return m.PublishWithDelay(ctx, topic, message, delay)
}

// SendJob is a convenience method to enqueue a job
func (m *Manager) SendJob(queue, handler string, payload interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	job, err := NewJob(queue, handler, payload)
	if err != nil {
		return err
	}

	return m.EnqueueJob(ctx, queue, job)
}

// SendDelayedJob is a convenience method to enqueue a delayed job
func (m *Manager) SendDelayedJob(queue, handler string, payload interface{}, delay time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	job, err := NewJob(queue, handler, payload)
	if err != nil {
		return err
	}

	job = job.WithDelay(delay)
	return m.EnqueueJob(ctx, queue, job)
}

// SendPriorityJob is a convenience method to enqueue a priority job
func (m *Manager) SendPriorityJob(queue, handler string, payload interface{}, priority int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	job, err := NewJob(queue, handler, payload)
	if err != nil {
		return err
	}

	job = job.WithPriority(priority)
	return m.EnqueueJob(ctx, queue, job)
}

// StartWorker starts a worker for processing jobs from a queue
func (m *Manager) StartWorker(queue string, handler JobHandler) error {
	ctx := context.Background() // Long-running context
	return m.ProcessJobs(ctx, queue, handler)
}

// ListenToTopic starts listening to a topic with a message handler
func (m *Manager) ListenToTopic(topic string, handler MessageHandler) error {
	ctx := context.Background() // Long-running context
	return m.Subscribe(ctx, topic, handler)
}

// ListenToTopicWithGroup starts listening to a topic with a group
func (m *Manager) ListenToTopicWithGroup(topic, group string, handler MessageHandler) error {
	ctx := context.Background() // Long-running context
	return m.SubscribeWithGroup(ctx, topic, group, handler)
}

// Broadcast sends a message to multiple topics
func (m *Manager) Broadcast(topics []string, payload interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, topic := range topics {
		if err := m.PublishJSON(ctx, topic, payload); err != nil {
			return fmt.Errorf("failed to broadcast to topic %s: %w", topic, err)
		}
	}

	return nil
}

// GetAllStats returns statistics from all drivers
func (m *Manager) GetAllStats() (map[string]*BrokerStats, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]*BrokerStats)
	for name, driver := range m.drivers {
		driverStats, err := driver.GetStats()
		if err != nil {
			return nil, fmt.Errorf("failed to get stats from driver %s: %w", name, err)
		}
		stats[name] = driverStats
	}

	return stats, nil
}

// SwitchDriver temporarily switches the default driver for a single operation
type DriverSwitcher struct {
	manager       *Manager
	originalDriver string
}

// Using returns a driver switcher for one-time operations
func (m *Manager) Using(driver string) *DriverSwitcher {
	return &DriverSwitcher{
		manager:       m,
		originalDriver: m.defaultDriver,
	}
}

// PublishJSON publishes using the specified driver
func (ds *DriverSwitcher) PublishJSON(ctx context.Context, topic string, data interface{}) error {
	driver := ds.manager.Driver(ds.originalDriver)
	if driver == nil {
		return fmt.Errorf("driver not available")
	}
	return driver.PublishJSON(ctx, topic, data)
}

// EnqueueJob enqueues a job using the specified driver
func (ds *DriverSwitcher) EnqueueJob(ctx context.Context, queue string, job *Job) error {
	driver := ds.manager.Driver(ds.originalDriver)
	if driver == nil {
		return fmt.Errorf("driver not available")
	}
	return driver.EnqueueJob(ctx, queue, job)
}

// Cross-driver operations

// Mirror sends the same message to multiple drivers
func (m *Manager) Mirror(drivers []string, topic string, payload interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, driverName := range drivers {
		driver := m.Driver(driverName)
		if driver == nil {
			return fmt.Errorf("driver %s not available", driverName)
		}
		
		if err := driver.PublishJSON(ctx, topic, payload); err != nil {
			return fmt.Errorf("failed to mirror to driver %s: %w", driverName, err)
		}
	}

	return nil
}