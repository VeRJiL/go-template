package registry

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"sync"

	"github.com/VeRJiL/go-template/internal/pkg/container"
	"github.com/VeRJiL/go-template/internal/pkg/logger"
	"github.com/VeRJiL/go-template/internal/pkg/modules"
)

// ModuleRegistry implements module registration and auto-discovery
type ModuleRegistry struct {
	modules     map[string]modules.Module
	moduleOrder []string
	mu          sync.RWMutex
	logger      *logger.Logger
	container   *container.Container
	initialized bool
}

// NewModuleRegistry creates a new module registry
func NewModuleRegistry(logger *logger.Logger, container *container.Container) modules.ModuleRegistry {
	return &ModuleRegistry{
		modules:   make(map[string]modules.Module),
		logger:    logger,
		container: container,
	}
}

// Register registers a module with the registry
func (r *ModuleRegistry) Register(module modules.Module) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := module.Name()
	if name == "" {
		return fmt.Errorf("module name cannot be empty")
	}

	if _, exists := r.modules[name]; exists {
		return fmt.Errorf("module %s is already registered", name)
	}

	// Validate dependencies
	if err := r.validateDependencies(module); err != nil {
		return fmt.Errorf("dependency validation failed for module %s: %w", name, err)
	}

	r.modules[name] = module
	r.logger.Info("Module registered", "module", name, "version", module.Version())

	// Recalculate module order after registration
	r.calculateModuleOrder()

	return nil
}

// GetModule retrieves a module by name
func (r *ModuleRegistry) GetModule(name string) (modules.Module, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	module, exists := r.modules[name]
	if !exists {
		return nil, fmt.Errorf("module %s not found", name)
	}

	return module, nil
}

// GetModules returns all registered modules in dependency order
func (r *ModuleRegistry) GetModules() []modules.Module {
	r.mu.RLock()
	defer r.mu.RUnlock()

	orderedModules := make([]modules.Module, 0, len(r.moduleOrder))
	for _, name := range r.moduleOrder {
		if module, exists := r.modules[name]; exists {
			orderedModules = append(orderedModules, module)
		}
	}

	return orderedModules
}

// GetModuleInfo returns metadata for all registered modules
func (r *ModuleRegistry) GetModuleInfo() []modules.ModuleInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	infos := make([]modules.ModuleInfo, 0, len(r.modules))
	for _, module := range r.modules {
		info := modules.ModuleInfo{
			Name:         module.Name(),
			Version:      module.Version(),
			Dependencies: module.Dependencies(),
		}

		// Extract routes and entities using reflection
		info.Routes = r.extractRoutes(module)
		info.Entities = r.extractEntities(module)

		infos = append(infos, info)
	}

	// Sort by name for consistent output
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].Name < infos[j].Name
	})

	return infos
}

// LoadModules discovers and loads modules from registered sources
func (r *ModuleRegistry) LoadModules() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.logger.Info("Starting module discovery and loading")

	// Auto-discover modules from container
	if err := r.discoverModulesFromContainer(); err != nil {
		return fmt.Errorf("failed to discover modules from container: %w", err)
	}

	// Load external modules if configured
	if err := r.loadExternalModules(); err != nil {
		return fmt.Errorf("failed to load external modules: %w", err)
	}

	r.logger.Info("Module loading completed", "total_modules", len(r.modules))
	return nil
}

// Initialize initializes all modules in dependency order
func (r *ModuleRegistry) Initialize(ctx context.Context, deps *modules.Dependencies) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.initialized {
		return fmt.Errorf("modules already initialized")
	}

	r.logger.Info("Initializing modules", "count", len(r.modules))

	// Initialize modules in dependency order
	for _, name := range r.moduleOrder {
		module, exists := r.modules[name]
		if !exists {
			continue
		}

		r.logger.Debug("Initializing module", "module", name)

		// Register module services
		if err := module.RegisterServices(deps.Container); err != nil {
			return fmt.Errorf("failed to register services for module %s: %w", name, err)
		}

		// Initialize module
		if err := module.Initialize(ctx); err != nil {
			return fmt.Errorf("failed to initialize module %s: %w", name, err)
		}

		r.logger.Info("Module initialized successfully", "module", name)
	}

	r.initialized = true
	return nil
}

// Shutdown gracefully shuts down all modules in reverse order
func (r *ModuleRegistry) Shutdown(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.initialized {
		return nil
	}

	r.logger.Info("Shutting down modules")

	// Shutdown in reverse order
	for i := len(r.moduleOrder) - 1; i >= 0; i-- {
		name := r.moduleOrder[i]
		module, exists := r.modules[name]
		if !exists {
			continue
		}

		r.logger.Debug("Shutting down module", "module", name)
		if err := module.Shutdown(ctx); err != nil {
			r.logger.Error("Failed to shutdown module", "module", name, "error", err)
		}
	}

	r.initialized = false
	return nil
}

// Helper methods

func (r *ModuleRegistry) validateDependencies(module modules.Module) error {
	dependencies := module.Dependencies()
	for _, depName := range dependencies {
		if _, exists := r.modules[depName]; !exists {
			return fmt.Errorf("dependency %s not found", depName)
		}
	}
	return nil
}

func (r *ModuleRegistry) calculateModuleOrder() {
	// Topological sort to determine module initialization order
	visited := make(map[string]bool)
	tempMark := make(map[string]bool)
	var order []string

	var visit func(string) error
	visit = func(name string) error {
		if tempMark[name] {
			return fmt.Errorf("circular dependency detected involving module %s", name)
		}
		if visited[name] {
			return nil
		}

		tempMark[name] = true

		module, exists := r.modules[name]
		if exists {
			for _, dep := range module.Dependencies() {
				if err := visit(dep); err != nil {
					return err
				}
			}
		}

		tempMark[name] = false
		visited[name] = true
		order = append(order, name)
		return nil
	}

	// Visit all modules
	for name := range r.modules {
		if !visited[name] {
			if err := visit(name); err != nil {
				r.logger.Error("Failed to calculate module order", "error", err)
				// Fallback to alphabetical order
				order = make([]string, 0, len(r.modules))
				for name := range r.modules {
					order = append(order, name)
				}
				sort.Strings(order)
				break
			}
		}
	}

	r.moduleOrder = order
}

func (r *ModuleRegistry) discoverModulesFromContainer() error {
	// Get all services from container and check if they implement Module interface
	services := r.container.GetServices()

	for _, serviceName := range services {
		service, err := r.container.Get(serviceName)
		if err != nil {
			continue
		}

		// Check if service implements Module interface
		if module, ok := service.(modules.Module); ok {
			if err := r.Register(module); err != nil {
				r.logger.Warn("Failed to register discovered module", "service", serviceName, "error", err)
			}
		}
	}

	return nil
}

func (r *ModuleRegistry) loadExternalModules() error {
	// Placeholder for loading external modules from plugins/libraries
	// In a real implementation, this could load from:
	// - Plugin directories
	// - Shared libraries (.so files)
	// - Remote module repositories
	// - Configuration-defined module sources

	r.logger.Debug("External module loading not yet implemented")
	return nil
}

func (r *ModuleRegistry) extractRoutes(module modules.Module) []modules.Route {
	// For now, return empty routes to avoid reflection issues
	// In a real implementation, you could:
	// 1. Have modules implement a GetRouteInfo() method
	// 2. Use struct tags to define routes
	// 3. Parse the RegisterRoutes method body (complex)

	var routes []modules.Route

	// Add some basic route information based on module name
	moduleName := module.Name()
	routes = append(routes, modules.Route{
		Method:  "GET",
		Path:    "/" + moduleName + "s",
		Handler: "List",
		Auth:    true,
	})
	routes = append(routes, modules.Route{
		Method:  "POST",
		Path:    "/" + moduleName + "s",
		Handler: "Create",
		Auth:    true,
	})

	return routes
}

func (r *ModuleRegistry) extractEntities(module modules.Module) []string {
	// Use reflection to extract entity information
	var entities []string

	// This is a simplified implementation - in a real system,
	// you might look for struct types that implement Entity interface
	moduleType := reflect.TypeOf(module)

	// Look for embedded types or fields that might be entities
	if moduleType.Kind() == reflect.Ptr {
		moduleType = moduleType.Elem()
	}

	if moduleType.Kind() == reflect.Struct {
		for i := 0; i < moduleType.NumField(); i++ {
			field := moduleType.Field(i)
			fieldType := field.Type

			// Check if field type implements Entity interface
			entityInterface := reflect.TypeOf((*modules.Entity)(nil)).Elem()
			if fieldType.Implements(entityInterface) {
				entities = append(entities, fieldType.Name())
			}
		}
	}

	return entities
}

// HasModule checks if a module is registered
func (r *ModuleRegistry) HasModule(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.modules[name]
	return exists
}

// GetModuleCount returns the number of registered modules
func (r *ModuleRegistry) GetModuleCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.modules)
}

// GetDependencyGraph returns the module dependency graph
func (r *ModuleRegistry) GetDependencyGraph() map[string][]string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	graph := make(map[string][]string)
	for name, module := range r.modules {
		graph[name] = module.Dependencies()
	}

	return graph
}

// IsInitialized returns whether modules have been initialized
func (r *ModuleRegistry) IsInitialized() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.initialized
}