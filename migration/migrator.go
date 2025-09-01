package migration

import (
	"context"
	"time"
)

// Migrator defines the interface for database migration operations
type Migrator interface {
	// Core operations
	Up(ctx context.Context) error
	Down(ctx context.Context) error
	UpTo(ctx context.Context, version int64) error
	DownTo(ctx context.Context, version int64) error
	
	// Status and validation
	Status(ctx context.Context) ([]MigrationStatus, error)
	Version(ctx context.Context) (int64, error)
	Validate(ctx context.Context) error
	
	// Advanced operations
	DryRun(ctx context.Context) ([]MigrationPlan, error)
	Force(ctx context.Context, version int64) error
	Reset(ctx context.Context) error
	
	// Migration management
	AddMigration(migration Migration)
	GetMigrations() []Migration
}

// Migration represents a single database migration
type Migration struct {
	Version     int64                 // Timestamp-based version (YYYYMMDDHHMMSS)
	Name        string                // Descriptive name
	Description string                // Detailed description
	Checksum    string                // SHA256 hash of content
	Up          MigrationFunc         // Function to apply migration
	Down        MigrationFunc         // Function to rollback migration
	Tags        []string              // Optional tags for categorization
	Metadata    map[string]string     // Additional metadata
}

// MigrationFunc is the function signature for migration operations
type MigrationFunc func(ctx context.Context, tx Transaction) error

// Transaction defines the interface for database transactions
type Transaction interface {
	Execute(ctx context.Context, query string, args ...interface{}) (Result, error)
	Query(ctx context.Context, query string, args ...interface{}) (Rows, error)
	QueryRow(ctx context.Context, query string, args ...interface{}) Row
	Commit() error
	Rollback() error
}

// Result represents the result of an Execute operation
type Result interface {
	LastInsertId() (int64, error)
	RowsAffected() (int64, error)
}

// Rows represents a result set from a Query operation
type Rows interface {
	Next() bool
	Scan(dest ...interface{}) error
	Close() error
	Err() error
}

// Row represents a single row result
type Row interface {
	Scan(dest ...interface{}) error
}

// MigrationStatus represents the status of a migration
type MigrationStatus struct {
	Version     int64         `json:"version"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Applied     bool          `json:"applied"`
	AppliedAt   *time.Time    `json:"applied_at,omitempty"`
	Duration    time.Duration `json:"duration,omitempty"`
	Checksum    string        `json:"checksum"`
	Error       string        `json:"error,omitempty"`
}

// MigrationPlan represents a planned migration operation
type MigrationPlan struct {
	Version     int64    `json:"version"`
	Name        string   `json:"name"`
	Direction   string   `json:"direction"` // "up" or "down"
	SQL         []string `json:"sql"`        // SQL statements to be executed
	Destructive bool     `json:"destructive"` // Whether migration is destructive
}

// MigrationError represents an error during migration
type MigrationError struct {
	Version   int64
	Direction string
	Message   string
	Err       error
}

func (e *MigrationError) Error() string {
	return e.Message
}

func (e *MigrationError) Unwrap() error {
	return e.Err
}

// ValidationError represents a validation error
type ValidationError struct {
	Issues []ValidationIssue
}

type ValidationIssue struct {
	Type    string // "duplicate", "gap", "checksum", "syntax"
	Version int64
	Message string
}

func (e *ValidationError) Error() string {
	if len(e.Issues) == 0 {
		return "validation failed"
	}
	return e.Issues[0].Message
}