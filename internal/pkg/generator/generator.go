package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/VeRJiL/go-template/internal/pkg/logger"
	"github.com/VeRJiL/go-template/internal/pkg/modules"
)

// Generator implements code generation for entities
type Generator struct {
	logger      *logger.Logger
	basePath    string
	packageName string
	templates   map[string]*template.Template
}

// NewGenerator creates a new code generator
func NewGenerator(logger *logger.Logger, basePath, packageName string) modules.Generator {
	g := &Generator{
		logger:      logger,
		basePath:    basePath,
		packageName: packageName,
		templates:   make(map[string]*template.Template),
	}

	g.loadTemplates()
	return g
}

// GenerateEntity generates entity struct and interfaces
func (g *Generator) GenerateEntity(config modules.EntityConfig) error {
	g.logger.Info("Generating entity", "name", config.Name)

	// Create entity directory
	entityDir := filepath.Join(g.basePath, "internal", "domain", "entities")
	if err := os.MkdirAll(entityDir, 0755); err != nil {
		return fmt.Errorf("failed to create entity directory: %w", err)
	}

	// Generate entity file
	entityFile := filepath.Join(entityDir, strings.ToLower(config.Name)+".go")
	if err := g.generateFromTemplate("entity", entityFile, config); err != nil {
		return fmt.Errorf("failed to generate entity file: %w", err)
	}

	g.logger.Info("Entity generated successfully", "file", entityFile)
	return nil
}

// GenerateRepository generates repository interface and implementation
func (g *Generator) GenerateRepository(config modules.EntityConfig) error {
	g.logger.Info("Generating repository", "name", config.Name)

	// Create repository directory
	repoDir := filepath.Join(g.basePath, "internal", "database", "repositories")
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		return fmt.Errorf("failed to create repository directory: %w", err)
	}

	// Generate repository interface
	interfaceFile := filepath.Join(repoDir, strings.ToLower(config.Name)+"_repository.go")
	if err := g.generateFromTemplate("repository_interface", interfaceFile, config); err != nil {
		return fmt.Errorf("failed to generate repository interface: %w", err)
	}

	// Generate repository implementation
	implFile := filepath.Join(repoDir, strings.ToLower(config.Name)+"_repository_impl.go")
	if err := g.generateFromTemplate("repository_impl", implFile, config); err != nil {
		return fmt.Errorf("failed to generate repository implementation: %w", err)
	}

	g.logger.Info("Repository generated successfully", "interface", interfaceFile, "implementation", implFile)
	return nil
}

// GenerateService generates service interface and implementation
func (g *Generator) GenerateService(config modules.EntityConfig) error {
	g.logger.Info("Generating service", "name", config.Name)

	// Create service directory
	serviceDir := filepath.Join(g.basePath, "internal", "domain", "services")
	if err := os.MkdirAll(serviceDir, 0755); err != nil {
		return fmt.Errorf("failed to create service directory: %w", err)
	}

	// Generate service interface
	interfaceFile := filepath.Join(serviceDir, strings.ToLower(config.Name)+"_service.go")
	if err := g.generateFromTemplate("service_interface", interfaceFile, config); err != nil {
		return fmt.Errorf("failed to generate service interface: %w", err)
	}

	// Generate service implementation
	implFile := filepath.Join(serviceDir, strings.ToLower(config.Name)+"_service_impl.go")
	if err := g.generateFromTemplate("service_impl", implFile, config); err != nil {
		return fmt.Errorf("failed to generate service implementation: %w", err)
	}

	g.logger.Info("Service generated successfully", "interface", interfaceFile, "implementation", implFile)
	return nil
}

// GenerateHandler generates HTTP handlers
func (g *Generator) GenerateHandler(config modules.EntityConfig) error {
	g.logger.Info("Generating handler", "name", config.Name)

	// Create handler directory
	handlerDir := filepath.Join(g.basePath, "internal", "api", "handlers")
	if err := os.MkdirAll(handlerDir, 0755); err != nil {
		return fmt.Errorf("failed to create handler directory: %w", err)
	}

	// Generate handler file
	handlerFile := filepath.Join(handlerDir, strings.ToLower(config.Name)+"_handler.go")
	if err := g.generateFromTemplate("handler", handlerFile, config); err != nil {
		return fmt.Errorf("failed to generate handler file: %w", err)
	}

	g.logger.Info("Handler generated successfully", "file", handlerFile)
	return nil
}

// GenerateModule generates complete module with all components
func (g *Generator) GenerateModule(config modules.EntityConfig) error {
	g.logger.Info("Generating complete module", "name", config.Name)

	// Generate all components
	if err := g.GenerateEntity(config); err != nil {
		return err
	}

	if err := g.GenerateRepository(config); err != nil {
		return err
	}

	if err := g.GenerateService(config); err != nil {
		return err
	}

	if err := g.GenerateHandler(config); err != nil {
		return err
	}

	// Generate module file
	moduleDir := filepath.Join(g.basePath, "internal", "modules")
	if err := os.MkdirAll(moduleDir, 0755); err != nil {
		return fmt.Errorf("failed to create module directory: %w", err)
	}

	moduleFile := filepath.Join(moduleDir, strings.ToLower(config.Name)+"_module.go")
	if err := g.generateFromTemplate("module", moduleFile, config); err != nil {
		return fmt.Errorf("failed to generate module file: %w", err)
	}

	g.logger.Info("Complete module generated successfully", "name", config.Name)
	return nil
}

// GenerateTests generates test files for all components
func (g *Generator) GenerateTests(config modules.EntityConfig) error {
	g.logger.Info("Generating tests", "name", config.Name)

	// Generate entity tests
	entityTestDir := filepath.Join(g.basePath, "internal", "domain", "entities")
	entityTestFile := filepath.Join(entityTestDir, strings.ToLower(config.Name)+"_test.go")
	if err := g.generateFromTemplate("entity_test", entityTestFile, config); err != nil {
		return fmt.Errorf("failed to generate entity tests: %w", err)
	}

	// Generate repository tests
	repoTestDir := filepath.Join(g.basePath, "internal", "database", "repositories")
	repoTestFile := filepath.Join(repoTestDir, strings.ToLower(config.Name)+"_repository_test.go")
	if err := g.generateFromTemplate("repository_test", repoTestFile, config); err != nil {
		return fmt.Errorf("failed to generate repository tests: %w", err)
	}

	// Generate service tests
	serviceTestDir := filepath.Join(g.basePath, "internal", "domain", "services")
	serviceTestFile := filepath.Join(serviceTestDir, strings.ToLower(config.Name)+"_service_test.go")
	if err := g.generateFromTemplate("service_test", serviceTestFile, config); err != nil {
		return fmt.Errorf("failed to generate service tests: %w", err)
	}

	// Generate handler tests
	handlerTestDir := filepath.Join(g.basePath, "internal", "api", "handlers")
	handlerTestFile := filepath.Join(handlerTestDir, strings.ToLower(config.Name)+"_handler_test.go")
	if err := g.generateFromTemplate("handler_test", handlerTestFile, config); err != nil {
		return fmt.Errorf("failed to generate handler tests: %w", err)
	}

	g.logger.Info("Tests generated successfully", "name", config.Name)
	return nil
}

// Helper methods

func (g *Generator) generateFromTemplate(templateName, outputFile string, config modules.EntityConfig) error {
	tmpl, exists := g.templates[templateName]
	if !exists {
		return fmt.Errorf("template %s not found", templateName)
	}

	// Create output file
	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", outputFile, err)
	}
	defer file.Close()

	// Prepare template data
	data := g.prepareTemplateData(config)

	// Execute template
	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}

func (g *Generator) prepareTemplateData(config modules.EntityConfig) map[string]interface{} {
	return map[string]interface{}{
		"PackageName":   g.packageName,
		"EntityName":    config.Name,
		"EntityLower":   strings.ToLower(config.Name),
		"TableName":     config.TableName,
		"SoftDelete":    config.SoftDelete,
		"Timestamps":    config.Timestamps,
		"Cache":         config.Cache,
		"Validation":    config.Validation,
		"Permissions":   config.Permissions,
		"Routes":        config.Routes,
		"GeneratedAt":   time.Now().Format(time.RFC3339),
		"Generator":     "go-template enterprise generator",
	}
}

func (g *Generator) loadTemplates() {
	g.templates["entity"] = template.Must(template.New("entity").Parse(entityTemplate))
	g.templates["repository_interface"] = template.Must(template.New("repository_interface").Parse(repositoryInterfaceTemplate))
	g.templates["repository_impl"] = template.Must(template.New("repository_impl").Parse(repositoryImplTemplate))
	g.templates["service_interface"] = template.Must(template.New("service_interface").Parse(serviceInterfaceTemplate))
	g.templates["service_impl"] = template.Must(template.New("service_impl").Parse(serviceImplTemplate))
	g.templates["handler"] = template.Must(template.New("handler").Parse(handlerTemplate))
	g.templates["module"] = template.Must(template.New("module").Parse(moduleTemplate))
	g.templates["entity_test"] = template.Must(template.New("entity_test").Parse(entityTestTemplate))
	g.templates["repository_test"] = template.Must(template.New("repository_test").Parse(repositoryTestTemplate))
	g.templates["service_test"] = template.Must(template.New("service_test").Parse(serviceTestTemplate))
	g.templates["handler_test"] = template.Must(template.New("handler_test").Parse(handlerTestTemplate))
}