package postgres

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
	"github.com/gnemade360/go-dal/dal/interfaces"
)

// PostgresProvider implements the Provider interface for PostgreSQL
type PostgresProvider struct {
	db  *sql.DB
	dsn string
}

// New creates a new PostgreSQL provider
func New() *PostgresProvider {
	return &PostgresProvider{}
}

// SetDSN sets the connection string
func (p *PostgresProvider) SetDSN(dsn string) {
	p.dsn = dsn
}

// Connect establishes a connection to the PostgreSQL database
func (p *PostgresProvider) Connect(ctx context.Context) error {
	db, err := sql.Open("postgres", p.dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	p.db = db
	return nil
}

// Close closes the database connection
func (p *PostgresProvider) Close() error {
	if p.db != nil {
		return p.db.Close()
	}
	return nil
}

// Ping verifies the database connection is still alive
func (p *PostgresProvider) Ping(ctx context.Context) error {
	if p.db == nil {
		return fmt.Errorf("database not connected")
	}
	return p.db.PingContext(ctx)
}

// Exec executes a query that doesn't return rows
func (p *PostgresProvider) Exec(ctx context.Context, query string, args ...interface{}) (interfaces.Result, error) {
	if p.db == nil {
		return nil, fmt.Errorf("database not connected")
	}
	result, err := p.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &postgresResult{result: result}, nil
}

// QueryRow executes a query that returns at most one row
func (p *PostgresProvider) QueryRow(ctx context.Context, query string, args ...interface{}) interfaces.Row {
	if p.db == nil {
		return &postgresRow{err: fmt.Errorf("database not connected")}
	}
	return &postgresRow{row: p.db.QueryRowContext(ctx, query, args...)}
}

// Query executes a query that returns rows
func (p *PostgresProvider) Query(ctx context.Context, query string, args ...interface{}) (interfaces.Rows, error) {
	if p.db == nil {
		return nil, fmt.Errorf("database not connected")
	}
	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &postgresRows{rows: rows}, nil
}

// BeginTx starts a new transaction
func (p *PostgresProvider) BeginTx(ctx context.Context, opts *sql.TxOptions) (interfaces.Transaction, error) {
	if p.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	tx, err := p.db.BeginTx(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	return &postgresTransaction{tx: tx}, nil
}

// Driver returns the database driver name
func (p *PostgresProvider) Driver() string {
	return "postgres"
}

// Dialect returns the PostgreSQL dialect
func (p *PostgresProvider) Dialect() interfaces.Dialect {
	return &postgresDialect{}
}

// postgresTransaction wraps sql.Tx to implement the Transaction interface
type postgresTransaction struct {
	tx *sql.Tx
}

// Commit commits the transaction
func (t *postgresTransaction) Commit() error {
	return t.tx.Commit()
}

// Rollback rolls back the transaction
func (t *postgresTransaction) Rollback() error {
	return t.tx.Rollback()
}

// Exec executes a query within the transaction
func (t *postgresTransaction) Exec(ctx context.Context, query string, args ...interface{}) (interfaces.Result, error) {
	result, err := t.tx.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &postgresResult{result: result}, nil
}

// QueryRow executes a query within the transaction that returns at most one row
func (t *postgresTransaction) QueryRow(ctx context.Context, query string, args ...interface{}) interfaces.Row {
	return &postgresRow{row: t.tx.QueryRowContext(ctx, query, args...)}
}

// Query executes a query within the transaction that returns rows
func (t *postgresTransaction) Query(ctx context.Context, query string, args ...interface{}) (interfaces.Rows, error) {
	rows, err := t.tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &postgresRows{rows: rows}, nil
}

// postgresDialect implements the Dialect interface for PostgreSQL
type postgresDialect struct{}

func (d *postgresDialect) Quote(identifier string) string {
	return fmt.Sprintf(`"%s"`, identifier)
}

func (d *postgresDialect) Placeholder(index int) string {
	return fmt.Sprintf("$%d", index)
}

func (d *postgresDialect) DataType(goType string) string {
	switch goType {
	case "string":
		return "VARCHAR(255)"
	case "int", "int64":
		return "BIGINT"
	case "bool":
		return "BOOLEAN"
	case "time.Time":
		return "TIMESTAMP"
	default:
		return "TEXT"
	}
}

func (d *postgresDialect) SupportsReturning() bool {
	return true
}

func (d *postgresDialect) SupportsLastInsertID() bool {
	return false
}

func (d *postgresDialect) LimitOffset(limit, offset int) string {
	if limit > 0 && offset > 0 {
		return fmt.Sprintf("LIMIT %d OFFSET %d", limit, offset)
	} else if limit > 0 {
		return fmt.Sprintf("LIMIT %d", limit)
	} else if offset > 0 {
		return fmt.Sprintf("OFFSET %d", offset)
	}
	return ""
}

// postgresResult wraps sql.Result
type postgresResult struct {
	result sql.Result
}

func (r *postgresResult) LastInsertId() (int64, error) {
	return r.result.LastInsertId()
}

func (r *postgresResult) RowsAffected() (int64, error) {
	return r.result.RowsAffected()
}

// postgresRow wraps sql.Row
type postgresRow struct {
	row *sql.Row
	err error
}

func (r *postgresRow) Scan(dest ...interface{}) error {
	if r.err != nil {
		return r.err
	}
	if r.row == nil {
		return fmt.Errorf("row is nil")
	}
	return r.row.Scan(dest...)
}

func (r *postgresRow) Err() error {
	if r.err != nil {
		return r.err
	}
	if r.row == nil {
		return nil
	}
	return r.row.Err()
}

// postgresRows wraps sql.Rows
type postgresRows struct {
	rows *sql.Rows
}

func (r *postgresRows) Next() bool {
	return r.rows.Next()
}

func (r *postgresRows) Scan(dest ...interface{}) error {
	return r.rows.Scan(dest...)
}

func (r *postgresRows) Close() error {
	return r.rows.Close()
}

func (r *postgresRows) Err() error {
	return r.rows.Err()
}

func (r *postgresRows) Columns() ([]string, error) {
	return r.rows.Columns()
}