package database

import (
    "database/sql"
    "fmt"
    "time"

    _ "github.com/go-sql-driver/mysql"
    "github.com/onyxirc/server/internal/config"
)

// DB wraps the database connection
type DB struct {
    *sql.DB
}

// NewConnection creates a new database connection with connection pooling
func NewConnection(cfg config.DatabaseConfig) (*DB, error) {
    // Build DSN (Data Source Name)
    dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
        cfg.User,
        cfg.Password,
        cfg.Host,
        cfg.Port,
        cfg.Name,
    )

    // Open database connection
    db, err := sql.Open("mysql", dsn)
    if err != nil {
        return nil, fmt.Errorf("failed to open database: %w", err)
    }

    // Configure connection pool
    db.SetMaxOpenConns(cfg.MaxOpenConns)
    db.SetMaxIdleConns(cfg.MaxIdleConns)
    db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

    // Test the connection
    if err := db.Ping(); err != nil {
        return nil, fmt.Errorf("failed to ping database: %w", err)
    }

    return &DB{db}, nil
}

// Close closes the database connection
func (db *DB) Close() error {
    return db.DB.Close()
}

// HealthCheck performs a health check on the database
func (db *DB) HealthCheck() error {
    ctx, cancel := contextWithTimeout(5 * time.Second)
    defer cancel()

    return db.PingContext(ctx)
}

// BeginTx starts a new transaction
func (db *DB) BeginTx() (*sql.Tx, error) {
    return db.DB.Begin()
}

// Stats returns database statistics
func (db *DB) Stats() sql.DBStats {
    return db.DB.Stats()
}
