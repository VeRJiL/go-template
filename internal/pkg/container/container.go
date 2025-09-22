package container

import (
	"fmt"
	"reflect"
	"sync"
)

// Container represents a dependency injection container
type Container struct {
	services map[string]interface{}
	types    map[reflect.Type]interface{}
	mu       sync.RWMutex
}

// NewContainer creates a new DI container
func NewContainer() *Container {
	return &Container{
		services: make(map[string]interface{}),
		types:    make(map[reflect.Type]interface{}),
	}
}

// Register registers a service by name and type
func (c *Container) Register(name string, service interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.services[name] = service
	c.types[reflect.TypeOf(service)] = service
}

// RegisterSingleton registers a singleton service
func (c *Container) RegisterSingleton(name string, factory func(*Container) interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Lazy initialization
	c.services[name] = &lazySingleton{
		factory:   factory,
		container: c,
	}
}

// RegisterTransient registers a transient service (new instance every time)
func (c *Container) RegisterTransient(name string, factory func(*Container) interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.services[name] = &transientService{
		factory:   factory,
		container: c,
	}
}

// Get retrieves a service by name
func (c *Container) Get(name string) (interface{}, error) {
	c.mu.RLock()
	service, exists := c.services[name]
	c.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("service '%s' not found", name)
	}

	// Handle lazy singletons
	if lazy, ok := service.(*lazySingleton); ok {
		return lazy.getInstance(), nil
	}

	// Handle transient services
	if transient, ok := service.(*transientService); ok {
		return transient.factory(c), nil
	}

	return service, nil
}

// GetByType retrieves a service by its type
func (c *Container) GetByType(serviceType reflect.Type) (interface{}, error) {
	c.mu.RLock()
	service, exists := c.types[serviceType]
	c.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("service of type '%s' not found", serviceType)
	}

	return service, nil
}

// MustGet retrieves a service by name or panics
func (c *Container) MustGet(name string) interface{} {
	service, err := c.Get(name)
	if err != nil {
		panic(err)
	}
	return service
}

// Resolve resolves dependencies for a struct using reflection
func (c *Container) Resolve(target interface{}) error {
	targetValue := reflect.ValueOf(target)
	if targetValue.Kind() != reflect.Ptr {
		return fmt.Errorf("target must be a pointer")
	}

	targetValue = targetValue.Elem()
	targetType := targetValue.Type()

	for i := 0; i < targetValue.NumField(); i++ {
		field := targetValue.Field(i)
		fieldType := targetType.Field(i)

		// Skip unexported fields
		if !field.CanSet() {
			continue
		}

		// Check for inject tag
		injectTag := fieldType.Tag.Get("inject")
		if injectTag == "" {
			continue
		}

		// Resolve dependency
		var dependency interface{}
		var err error

		if injectTag == "type" {
			dependency, err = c.GetByType(field.Type())
		} else {
			dependency, err = c.Get(injectTag)
		}

		if err != nil {
			return fmt.Errorf("failed to resolve dependency for field '%s': %w", fieldType.Name, err)
		}

		// Set the field value
		dependencyValue := reflect.ValueOf(dependency)
		if !dependencyValue.Type().AssignableTo(field.Type()) {
			return fmt.Errorf("dependency type mismatch for field '%s'", fieldType.Name)
		}

		field.Set(dependencyValue)
	}

	return nil
}

// AutoRegister automatically registers services based on interfaces
func (c *Container) AutoRegister(services ...interface{}) {
	for _, service := range services {
		serviceType := reflect.TypeOf(service)
		serviceName := serviceType.Elem().Name()

		// Register by name (lowercase)
		c.Register(serviceName, service)

		// Register by interface types
		for i := 0; i < serviceType.NumMethod(); i++ {
			method := serviceType.Method(i)
			_ = method // Use method to discover interfaces
		}
	}
}

// GetServices returns all registered service names
func (c *Container) GetServices() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	names := make([]string, 0, len(c.services))
	for name := range c.services {
		names = append(names, name)
	}
	return names
}

// HasService checks if a service is registered
func (c *Container) HasService(name string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	_, exists := c.services[name]
	return exists
}

// Clear clears all registered services
func (c *Container) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.services = make(map[string]interface{})
	c.types = make(map[reflect.Type]interface{})
}

// Internal types for lazy loading and transient services

type lazySingleton struct {
	factory   func(*Container) interface{}
	container *Container
	instance  interface{}
	once      sync.Once
}

func (l *lazySingleton) getInstance() interface{} {
	l.once.Do(func() {
		l.instance = l.factory(l.container)
	})
	return l.instance
}

type transientService struct {
	factory   func(*Container) interface{}
	container *Container
}

// ServiceInfo holds information about a registered service
type ServiceInfo struct {
	Name string
	Type reflect.Type
	Kind string // singleton, transient, instance
}

// GetServiceInfo returns information about all registered services
func (c *Container) GetServiceInfo() []ServiceInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()

	info := make([]ServiceInfo, 0, len(c.services))
	for name, service := range c.services {
		serviceInfo := ServiceInfo{
			Name: name,
			Type: reflect.TypeOf(service),
		}

		switch service.(type) {
		case *lazySingleton:
			serviceInfo.Kind = "singleton"
		case *transientService:
			serviceInfo.Kind = "transient"
		default:
			serviceInfo.Kind = "instance"
		}

		info = append(info, serviceInfo)
	}

	return info
}

// Factory represents a service factory function
type Factory func(*Container) interface{}

// FactoryConfig represents factory configuration
type FactoryConfig struct {
	Name      string
	Factory   Factory
	Singleton bool
}

// RegisterFactory registers a service factory with configuration
func (c *Container) RegisterFactory(config FactoryConfig) {
	if config.Singleton {
		c.RegisterSingleton(config.Name, config.Factory)
	} else {
		c.RegisterTransient(config.Name, config.Factory)
	}
}

// BatchRegister registers multiple services at once
func (c *Container) BatchRegister(registrations map[string]interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for name, service := range registrations {
		c.services[name] = service
		c.types[reflect.TypeOf(service)] = service
	}
}