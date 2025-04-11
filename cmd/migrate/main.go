package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gsbingo17/es-to-mongodb/pkg/config"
	"github.com/gsbingo17/es-to-mongodb/pkg/logger"
	"github.com/gsbingo17/es-to-mongodb/pkg/migration"
)

func main() {
	// Parse command-line flags
	configPath := flag.String("config", "elasticsearch_to_mongodb_config.json", "Path to configuration file")
	logLevel := flag.String("log-level", "info", "Log level: debug, info, warn, error")
	help := flag.Bool("help", false, "Display help information")
	flag.Parse()

	// Display help if requested
	if *help {
		displayUsage()
		os.Exit(0)
	}

	// Create logger
	log := logger.New()
	log.SetLevel(*logLevel)

	// Load configuration
	log.Info("Loading configuration...")
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-signalChan
		log.Info("Received interrupt signal. Shutting down...")
		cancel()
		// Give some time for graceful shutdown
		time.Sleep(2 * time.Second)
		os.Exit(0)
	}()

	// Create migrator
	migrator := migration.NewESMigrator(cfg, log)

	// Start migration
	startTime := time.Now()
	log.Info("Starting Elasticsearch to MongoDB migration process")

	if err := migrator.Start(ctx); err != nil {
		// Check if the error is due to context cancellation (Ctrl+C)
		if err == context.Canceled {
			log.Info("Process stopped due to user interrupt (Ctrl+C)")
		} else {
			log.Fatalf("Error during migration process: %v", err)
		}
	}

	// Log completion
	duration := time.Since(startTime)
	log.Infof("Migration completed in %.2f seconds", duration.Seconds())
}

// displayUsage displays usage information
func displayUsage() {
	fmt.Println("\nElasticsearch to MongoDB Migration Tool")
	fmt.Println("======================================")
	fmt.Println("Usage: migrate [options]")
	fmt.Println("Options:")
	fmt.Println("  -config string")
	fmt.Println("        Path to configuration file (default \"elasticsearch_to_mongodb_config.json\")")
	fmt.Println("  -log-level string")
	fmt.Println("        Log level: debug, info, warn, error (default \"info\")")
	fmt.Println("  -help")
	fmt.Println("        Display this help information")
	fmt.Println("Examples:")
	fmt.Println("  migrate")
	fmt.Println("  migrate -config=custom_config.json -log-level=debug")
}
