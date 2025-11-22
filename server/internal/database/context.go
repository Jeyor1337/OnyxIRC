package database

import (
    "context"
    "time"
)

// contextWithTimeout creates a context with timeout
func contextWithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
    return context.WithTimeout(context.Background(), timeout)
}

// defaultTimeout is the default timeout for database operations
const defaultTimeout = 10 * time.Second
