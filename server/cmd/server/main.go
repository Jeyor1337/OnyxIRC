package main

import (
    "flag"
    "fmt"
    "log"
    "os"
    "os/signal"
    "syscall"

    "github.com/onyxirc/server/internal/config"
    "github.com/onyxirc/server/internal/database"
    "github.com/onyxirc/server/internal/server"
)

func main() {
    // Parse command-line flags
    configPath := flag.String("config", "configs/server.yaml", "Path to configuration file")
    flag.Parse()

    // Load configuration
    cfg, err := config.Load(*configPath)
    if err != nil {
        log.Fatalf("Failed to load configuration: %v", err)
    }

    // Initialize database connection
    db, err := database.NewConnection(cfg.Database)
    if err != nil {
        log.Fatalf("Failed to connect to database: %v", err)
    }
    defer db.Close()

    // Run database migrations if needed
    if err := database.RunMigrations(db); err != nil {
        log.Fatalf("Failed to run migrations: %v", err)
    }

    // Create and start IRC server
    ircServer, err := server.New(cfg, db)
    if err != nil {
        log.Fatalf("Failed to create server: %v", err)
    }

    // Start server in a goroutine
    go func() {
        log.Printf("Starting OnyxIRC server on %s:%d", cfg.Server.Host, cfg.Server.Port)
        if err := ircServer.Start(); err != nil {
            log.Fatalf("Server error: %v", err)
        }
    }()

    // Wait for interrupt signal to gracefully shutdown
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    log.Println("Shutting down server...")
    if err := ircServer.Shutdown(); err != nil {
        log.Printf("Error during shutdown: %v", err)
    }

    fmt.Println("Server stopped successfully")
}
