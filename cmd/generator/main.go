package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/VeRJiL/go-template/internal/pkg/generator"
	"github.com/VeRJiL/go-template/internal/pkg/logger"
	"github.com/VeRJiL/go-template/internal/pkg/modules"
)

func main() {
	var (
		entityName  = flag.String("entity", "", "Entity name (required)")
		tableName   = flag.String("table", "", "Table name (defaults to snake_case of entity name)")
		softDelete  = flag.Bool("soft-delete", false, "Enable soft delete")
		timestamps  = flag.Bool("timestamps", true, "Enable timestamps")
		cache       = flag.Bool("cache", true, "Enable caching")
		generateAll = flag.Bool("all", false, "Generate entity, repository, service, handler, module, and tests")
		genEntity   = flag.Bool("gen-entity", false, "Generate entity")
		genRepo     = flag.Bool("gen-repo", false, "Generate repository")
		genService  = flag.Bool("gen-service", false, "Generate service")
		genHandler  = flag.Bool("gen-handler", false, "Generate handler")
		genModule   = flag.Bool("gen-module", false, "Generate module")
		genTests    = flag.Bool("gen-tests", false, "Generate tests")
		packageName = flag.String("package", "github.com/VeRJiL/go-template", "Package name")
		basePath    = flag.String("base-path", ".", "Base path for generation")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Enterprise Code Generator for Go Template\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  # Generate complete module for Product entity\n")
		fmt.Fprintf(os.Stderr, "  %s -entity=Product -all\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Generate only entity and repository\n")
		fmt.Fprintf(os.Stderr, "  %s -entity=Product -gen-entity -gen-repo\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Generate with soft delete and custom table name\n")
		fmt.Fprintf(os.Stderr, "  %s -entity=Product -table=products -soft-delete -all\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	// Validate required parameters
	if *entityName == "" {
		fmt.Fprintf(os.Stderr, "Error: -entity is required\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Set default table name if not provided
	if *tableName == "" {
		*tableName = toSnakeCase(*entityName)
	}

	// Determine what to generate
	if !*generateAll && !*genEntity && !*genRepo && !*genService && !*genHandler && !*genModule && !*genTests {
		fmt.Fprintf(os.Stderr, "Error: Must specify what to generate. Use -all or specific -gen-* flags\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Initialize logger
	loggerInstance := logger.New("info", "text")

	// Initialize generator
	gen := generator.NewGenerator(loggerInstance, *basePath, *packageName)

	// Create entity config
	config := modules.EntityConfig{
		Name:       *entityName,
		TableName:  *tableName,
		SoftDelete: *softDelete,
		Timestamps: *timestamps,
		Cache: modules.CacheConfig{
			Enabled: *cache,
			TTL:     "1h",
			Prefix:  strings.ToLower(*entityName),
		},
		Validation: modules.ValidationConfig{
			Required: []string{"name"},
			Rules:    map[string]string{"name": "required,min=2,max=100"},
		},
		Permissions: modules.PermissionConfig{
			Create: []string{"admin", "user"},
			Read:   []string{"admin", "user", "guest"},
			Update: []string{"admin", "user"},
			Delete: []string{"admin"},
			List:   []string{"admin", "user", "guest"},
		},
	}

	fmt.Printf("🚀 Starting code generation for entity '%s'\n", *entityName)
	fmt.Printf("📋 Configuration:\n")
	fmt.Printf("   - Entity: %s\n", config.Name)
	fmt.Printf("   - Table: %s\n", config.TableName)
	fmt.Printf("   - Soft Delete: %v\n", config.SoftDelete)
	fmt.Printf("   - Timestamps: %v\n", config.Timestamps)
	fmt.Printf("   - Cache: %v\n", config.Cache.Enabled)
	fmt.Printf("   - Package: %s\n", *packageName)
	fmt.Printf("   - Base Path: %s\n", *basePath)
	fmt.Println()

	// Generate components
	var errors []error

	if *generateAll || *genEntity {
		fmt.Print("📝 Generating entity... ")
		if err := gen.GenerateEntity(config); err != nil {
			fmt.Printf("❌ Failed: %v\n", err)
			errors = append(errors, err)
		} else {
			fmt.Println("✅ Success")
		}
	}

	if *generateAll || *genRepo {
		fmt.Print("🗄️  Generating repository... ")
		if err := gen.GenerateRepository(config); err != nil {
			fmt.Printf("❌ Failed: %v\n", err)
			errors = append(errors, err)
		} else {
			fmt.Println("✅ Success")
		}
	}

	if *generateAll || *genService {
		fmt.Print("⚙️  Generating service... ")
		if err := gen.GenerateService(config); err != nil {
			fmt.Printf("❌ Failed: %v\n", err)
			errors = append(errors, err)
		} else {
			fmt.Println("✅ Success")
		}
	}

	if *generateAll || *genHandler {
		fmt.Print("🌐 Generating handler... ")
		if err := gen.GenerateHandler(config); err != nil {
			fmt.Printf("❌ Failed: %v\n", err)
			errors = append(errors, err)
		} else {
			fmt.Println("✅ Success")
		}
	}

	if *generateAll || *genModule {
		fmt.Print("📦 Generating module... ")
		if err := gen.GenerateModule(config); err != nil {
			fmt.Printf("❌ Failed: %v\n", err)
			errors = append(errors, err)
		} else {
			fmt.Println("✅ Success")
		}
	}

	if *generateAll || *genTests {
		fmt.Print("🧪 Generating tests... ")
		if err := gen.GenerateTests(config); err != nil {
			fmt.Printf("❌ Failed: %v\n", err)
			errors = append(errors, err)
		} else {
			fmt.Println("✅ Success")
		}
	}

	fmt.Println()

	if len(errors) > 0 {
		fmt.Printf("❌ Generation completed with %d errors:\n", len(errors))
		for i, err := range errors {
			fmt.Printf("   %d. %v\n", i+1, err)
		}
		os.Exit(1)
	}

	fmt.Printf("🎉 Code generation completed successfully for entity '%s'!\n", *entityName)
	fmt.Println()
	fmt.Println("📋 Next steps:")
	fmt.Println("   1. Review generated files and customize as needed")
	fmt.Println("   2. Run database migrations")
	fmt.Println("   3. Register the module in your application")
	fmt.Println("   4. Run tests to verify functionality")
	fmt.Println()
	fmt.Println("💡 Example module registration:")
	fmt.Printf("   registry.Register(modules.New%sModule())\n", *entityName)
}

// toSnakeCase converts CamelCase to snake_case
func toSnakeCase(str string) string {
	var result strings.Builder
	for i, r := range str {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

// Helper function to get absolute path
func getAbsolutePath(path string) string {
	absPath, err := filepath.Abs(path)
	if err != nil {
		log.Fatalf("Failed to get absolute path for %s: %v", path, err)
	}
	return absPath
}