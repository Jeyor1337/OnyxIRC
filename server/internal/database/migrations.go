package database

import (
    "fmt"
    "log"
)

// Migration represents a database migration
type Migration struct {
    Version     int
    Description string
    SQL         string
}

// RunMigrations runs all pending database migrations
func RunMigrations(db *DB) error {
    // Create migrations table if it doesn't exist
    if err := createMigrationsTable(db); err != nil {
        return fmt.Errorf("failed to create migrations table: %w", err)
    }

    // Get current version
    currentVersion, err := getCurrentVersion(db)
    if err != nil {
        return fmt.Errorf("failed to get current version: %w", err)
    }

    // Define migrations
    migrations := []Migration{
        {
            Version:     1,
            Description: "Initial schema setup",
            SQL:         "", // Schema is created manually via schema.sql
        },
    }

    // Run pending migrations
    for _, migration := range migrations {
        if migration.Version <= currentVersion {
            continue
        }

        log.Printf("Running migration %d: %s", migration.Version, migration.Description)

        if migration.SQL != "" {
            if _, err := db.Exec(migration.SQL); err != nil {
                return fmt.Errorf("migration %d failed: %w", migration.Version, err)
            }
        }

        // Update version
        if err := updateVersion(db, migration.Version); err != nil {
            return fmt.Errorf("failed to update version: %w", err)
        }

        log.Printf("Migration %d completed", migration.Version)
    }

    return nil
}

// createMigrationsTable creates the schema_migrations table
func createMigrationsTable(db *DB) error {
    query := `
        CREATE TABLE IF NOT EXISTS schema_migrations (
            version INT PRIMARY KEY,
            applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )
    `
    _, err := db.Exec(query)
    return err
}

// getCurrentVersion gets the current schema version
func getCurrentVersion(db *DB) (int, error) {
    var version int
    err := db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&version)
    if err != nil {
        return 0, err
    }
    return version, nil
}

// updateVersion updates the schema version
func updateVersion(db *DB, version int) error {
    _, err := db.Exec("INSERT INTO schema_migrations (version) VALUES (?)", version)
    return err
}
