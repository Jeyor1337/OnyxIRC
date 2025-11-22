package database

import (
    "database/sql"
    "fmt"
    "time"

    _ "github.com/go-sql-driver/mysql"
    "github.com/onyxirc/server/internal/config"
)

type DB struct {
    *sql.DB
}

func NewConnection(cfg config.DatabaseConfig) (*DB, error) {
    
    dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
        cfg.User,
        cfg.Password,
        cfg.Host,
        cfg.Port,
        cfg.Name,
    )

    db, err := sql.Open("mysql", dsn)
    if err != nil {
        return nil, fmt.Errorf("failed to open database: %w", err)
    }

    db.SetMaxOpenConns(cfg.MaxOpenConns)
    db.SetMaxIdleConns(cfg.MaxIdleConns)
    db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

    if err := db.Ping(); err != nil {
        return nil, fmt.Errorf("failed to ping database: %w", err)
    }

    return &DB{db}, nil
}

func (db *DB) Close() error {
    return db.DB.Close()
}

func (db *DB) HealthCheck() error {
    ctx, cancel := contextWithTimeout(5 * time.Second)
    defer cancel()

    return db.PingContext(ctx)
}

func (db *DB) BeginTx() (*sql.Tx, error) {
    return db.DB.Begin()
}

func (db *DB) Stats() sql.DBStats {
    return db.DB.Stats()
}
