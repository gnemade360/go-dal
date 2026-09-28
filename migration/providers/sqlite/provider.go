package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/gnemade360/go-dal/migration"
)

// defaultDriverName matches the dal/providers/sqlite default. Callers must
// blank-import the driver themselves (mattn/go-sqlite3 or modernc.org/sqlite)
// and call WithDriver if using a non-default name.
const defaultDriverName = "sqlite3"

// Provider implements the migration.Provider interface for SQLite
type Provider struct {
	db         *sql.DB
	dsn        string
	tableName  string
	driverName string
}

// NewProvider creates a new SQLite migration provider
func NewProvider(dsn string, tableName string) *Provider {
	if tableName == "" {
		tableName = "schema_migrations"
	}

	return &Provider{
		dsn:        dsn,
		tableName:  tableName,
		driverName: defaultDriverName,
	}
}

// WithDriver overrides the database/sql driver name. Use "sqlite" for
// modernc.org/sqlite (pure-Go), "sqlite3" for mattn (CGO).
func (p *Provider) WithDriver(name string) *Provider {
	if name != "" {
		p.driverName = name
	}
	return p
}

// Connect establishes a database connection
func (p *Provider) Connect(ctx context.Context) error {
	driver := p.driverName
	if driver == "" {
		driver = defaultDriverName
	}
	db, err := sql.Open(driver, p.dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return fmt.Errorf("failed to ping database: %w", err)
	}

	p.db = db
	return nil
}

// Disconnect closes the database connection
func (p *Provider) Disconnect(ctx context.Context) error {
	if p.db != nil {
		return p.db.Close()
	}
	return nil
}

// BeginTransaction starts a new transaction
func (p *Provider) BeginTransaction(ctx context.Context) (migration.Transaction, error) {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	
	return &sqliteTransaction{tx: tx}, nil
}

// CreateMigrationsTable creates the migrations tracking table
func (p *Provider) CreateMigrationsTable(ctx context.Context) error {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			version BIGINT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT,
			checksum TEXT,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			duration_ms INTEGER
		)
	`, p.tableName)
	
	_, err := p.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}
	
	// Create index on applied_at
	indexQuery := fmt.Sprintf(
		"CREATE INDEX IF NOT EXISTS idx_%s_applied_at ON %s(applied_at)",
		p.tableName, p.tableName,
	)
	_, err = p.db.ExecContext(ctx, indexQuery)
	
	return err
}

// GetAppliedMigrations returns all applied migrations
func (p *Provider) GetAppliedMigrations(ctx context.Context) ([]migration.MigrationStatus, error) {
	query := fmt.Sprintf(
		"SELECT version, name, description, checksum, applied_at, duration_ms FROM %s ORDER BY version",
		p.tableName,
	)
	
	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query migrations: %w", err)
	}
	defer rows.Close()
	
	var migrations []migration.MigrationStatus
	for rows.Next() {
		var status migration.MigrationStatus
		var appliedAt sql.NullTime
		var durationMs sql.NullInt64
		
		err := rows.Scan(
			&status.Version,
			&status.Name,
			&status.Description,
			&status.Checksum,
			&appliedAt,
			&durationMs,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan migration row: %w", err)
		}
		
		status.Applied = true
		if appliedAt.Valid {
			status.AppliedAt = &appliedAt.Time
		}
		if durationMs.Valid {
			status.Duration = time.Duration(durationMs.Int64) * time.Millisecond
		}
		
		migrations = append(migrations, status)
	}
	
	return migrations, rows.Err()
}

// RecordMigration records a completed migration
func (p *Provider) RecordMigration(ctx context.Context, status migration.MigrationStatus) error {
	query := fmt.Sprintf(
		"INSERT INTO %s (version, name, description, checksum, applied_at, duration_ms) VALUES (?, ?, ?, ?, ?, ?)",
		p.tableName,
	)
	
	durationMs := status.Duration.Milliseconds()
	
	_, err := p.db.ExecContext(ctx, query,
		status.Version,
		status.Name,
		status.Description,
		status.Checksum,
		status.AppliedAt,
		durationMs,
	)
	
	if err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}
	
	return nil
}

// RemoveMigration removes a migration record
func (p *Provider) RemoveMigration(ctx context.Context, version int64) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE version = ?", p.tableName)
	
	result, err := p.db.ExecContext(ctx, query, version)
	if err != nil {
		return fmt.Errorf("failed to remove migration: %w", err)
	}
	
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	
	if affected == 0 {
		return fmt.Errorf("migration %d not found", version)
	}
	
	return nil
}

// AcquireLock acquires a migration lock (no-op for SQLite in single-connection mode)
func (p *Provider) AcquireLock(ctx context.Context, timeout time.Duration) error {
	// SQLite handles locking at the database level
	// For a more robust solution, we could implement application-level locking
	// using a separate table, but for now, we rely on SQLite's built-in locking
	return nil
}

// ReleaseLock releases the migration lock (no-op for SQLite)
func (p *Provider) ReleaseLock(ctx context.Context) error {
	return nil
}

// SupportsTransactionalDDL returns whether the database supports transactional DDL
func (p *Provider) SupportsTransactionalDDL() bool {
	// SQLite supports transactional DDL
	return true
}

// GetDatabaseName returns the database name
func (p *Provider) GetDatabaseName() string {
	return p.dsn
}

// sqliteTransaction wraps sql.Tx to implement migration.Transaction
type sqliteTransaction struct {
	tx *sql.Tx
}

func (t *sqliteTransaction) Execute(ctx context.Context, query string, args ...interface{}) (migration.Result, error) {
	result, err := t.tx.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &sqliteResult{result: result}, nil
}

func (t *sqliteTransaction) Query(ctx context.Context, query string, args ...interface{}) (migration.Rows, error) {
	rows, err := t.tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &sqliteRows{rows: rows}, nil
}

func (t *sqliteTransaction) QueryRow(ctx context.Context, query string, args ...interface{}) migration.Row {
	row := t.tx.QueryRowContext(ctx, query, args...)
	return &sqliteRow{row: row}
}

func (t *sqliteTransaction) Commit() error {
	return t.tx.Commit()
}

func (t *sqliteTransaction) Rollback() error {
	return t.tx.Rollback()
}

// sqliteResult wraps sql.Result
type sqliteResult struct {
	result sql.Result
}

func (r *sqliteResult) LastInsertId() (int64, error) {
	return r.result.LastInsertId()
}

func (r *sqliteResult) RowsAffected() (int64, error) {
	return r.result.RowsAffected()
}

// sqliteRows wraps sql.Rows
type sqliteRows struct {
	rows *sql.Rows
}

func (r *sqliteRows) Next() bool {
	return r.rows.Next()
}

func (r *sqliteRows) Scan(dest ...interface{}) error {
	return r.rows.Scan(dest...)
}

func (r *sqliteRows) Close() error {
	return r.rows.Close()
}

func (r *sqliteRows) Err() error {
	return r.rows.Err()
}

// sqliteRow wraps sql.Row
type sqliteRow struct {
	row *sql.Row
}

func (r *sqliteRow) Scan(dest ...interface{}) error {
	return r.row.Scan(dest...)
}