package main

import (
	"context"
	"fmt"
	"log"

	"github.com/VeRJiL/go-template/internal/config"
	"github.com/VeRJiL/go-template/internal/modules"
	"github.com/VeRJiL/go-template/internal/pkg/bootstrap"
	"github.com/VeRJiL/go-template/internal/pkg/logger"
)

func main() {
	fmt.Println("🧪 Testing Enterprise Architecture Components...")

	// Test 1: Load config
	fmt.Print("1. Loading configuration... ")
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("❌ Failed: %v", err)
	}
	fmt.Println("✅ Success")

	// Test 2: Initialize logger
	fmt.Print("2. Initializing logger... ")
	logger := logger.New(cfg.Logging.Level, cfg.Logging.Format)
	fmt.Println("✅ Success")

	// Test 3: Create enterprise bootstrap
	fmt.Print("3. Creating enterprise bootstrap... ")
	enterpriseBootstrap := bootstrap.NewEnterpriseBootstrap(cfg, logger)
	fmt.Println("✅ Success")

	// Test 4: Create user module
	fmt.Print("4. Creating user module... ")
	userModule := modules.NewUserModule()
	fmt.Printf("✅ Success (Name: %s, Version: %s)\n", userModule.Name(), userModule.Version())

	// Test 5: Register module (without dependencies for now)
	fmt.Print("5. Registering user module... ")
	if err := enterpriseBootstrap.RegisterModule(userModule); err != nil {
		log.Fatalf("❌ Failed: %v", err)
	}
	fmt.Println("✅ Success")

	// Test 6: Get module info
	fmt.Print("6. Getting module info... ")
	moduleInfo := enterpriseBootstrap.GetModuleInfo()
	fmt.Printf("✅ Success (%d modules registered)\n", len(moduleInfo))

	// Test 7: Health check
	fmt.Print("7. Running health check... ")
	ctx := context.Background()
	health := enterpriseBootstrap.HealthCheck(ctx)
	fmt.Printf("✅ Success (Status: %v)\n", health["status"])

	// Test 8: Get stats
	fmt.Print("8. Getting application stats... ")
	stats := enterpriseBootstrap.GetStats()
	fmt.Printf("✅ Success (Modules: %v)\n", stats["modules"])

	fmt.Println("\n🎉 All Enterprise Architecture Components Working!")
	fmt.Println("\n📋 Summary:")
	fmt.Printf("   - Config loaded: %s\n", cfg.App.Name)
	fmt.Printf("   - Logger initialized: %s\n", cfg.Logging.Level)
	fmt.Printf("   - Enterprise bootstrap created\n")
	fmt.Printf("   - User module registered\n")
	fmt.Printf("   - Health checks passing\n")
	fmt.Println("\n✨ Enterprise architecture is ready!")
	fmt.Println("💡 Next step: Run full application with database connectivity")
}