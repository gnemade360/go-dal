package interfaces

import (
	"context"
	"database/sql"
)

// Database represents the main database interface
type Database interface {
	// Connection management
	Connect(ctx context.Context) error
	Close() error
	Ping(ctx context.Context) error

	// Query execution
	Query(ctx context.Context, query string, args ...interface{}) (Rows, error)
	QueryRow(ctx context.Context, query string, args ...interface{}) Row
	Exec(ctx context.Context, query string, args ...interface{}) (Result, error)

	// Transaction management
	BeginTx(ctx context.Context, opts *sql.TxOptions) (Transaction, error)

	// Database-specific features
	Driver() string
	Dialect() Dialect
}

// Transaction represents a database transaction
type Transaction interface {
	Commit() error
	Rollback() error

	Query(ctx context.Context, query string, args ...interface{}) (Rows, error)
	QueryRow(ctx context.Context, query string, args ...interface{}) Row
	Exec(ctx context.Context, query string, args ...interface{}) (Result, error)
}

// Dialect represents database-specific SQL dialect
type Dialect interface {
	// Query building
	Quote(identifier string) string
	Placeholder(index int) string

	// Type mapping
	DataType(goType string) string

	// Features
	SupportsReturning() bool
	SupportsLastInsertID() bool

	// Pagination
	LimitOffset(limit, offset int) string
}

// Scanner handles row scanning
type Scanner interface {
	Scan(dest ...interface{}) error
}

// Row represents a single database row
type Row interface {
	Scanner
	Err() error
}

// Rows represents multiple database rows
type Rows interface {
	Scanner
	Next() bool
	Close() error
	Err() error
	Columns() ([]string, error)
}

// Result represents an execution result
type Result interface {
	LastInsertId() (int64, error)
	RowsAffected() (int64, error)
}