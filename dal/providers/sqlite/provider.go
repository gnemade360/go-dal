package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/gnemade360/go-dal/dal/interfaces"
)

// Default SQL driver name. Callers using mattn/go-sqlite3 should blank-import
// it themselves; callers using modernc.org/sqlite (pure-Go, no CGO) should
// blank-import that and call WithDriver("sqlite") on the provider.
const defaultDriverName = "sqlite3"

type SQLiteProvider struct {
	db         *sql.DB
	dsn        string
	driverName string
}

// New returns a SQLiteProvider that opens "sqlite3" by default. Callers must
// register the underlying driver themselves with a blank import (e.g.
// `_ "github.com/mattn/go-sqlite3"` or `_ "modernc.org/sqlite"` plus
// WithDriver("sqlite")). This avoids forcing a CGO dependency on every
// consumer of go-dal.
func New() *SQLiteProvider {
	return &SQLiteProvider{driverName: defaultDriverName}
}

// WithDriver overrides the database/sql driver name used in sql.Open. Use
// "sqlite" for modernc.org/sqlite (pure-Go), "sqlite3" for mattn (CGO).
func (p *SQLiteProvider) WithDriver(name string) *SQLiteProvider {
	if name != "" {
		p.driverName = name
	}
	return p
}

func (p *SQLiteProvider) SetDSN(dsn string) {
	p.dsn = dsn
}

func (p *SQLiteProvider) Connect(ctx context.Context) error {
	driver := p.driverName
	if driver == "" {
		driver = defaultDriverName
	}
	db, err := sql.Open(driver, p.dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// SQLite is embedded — single writer, no connection pool needed
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	p.db = db
	return nil
}

func (p *SQLiteProvider) Close() error {
	if p.db != nil {
		return p.db.Close()
	}
	return nil
}

func (p *SQLiteProvider) Ping(ctx context.Context) error {
	if p.db == nil {
		return fmt.Errorf("database not connected")
	}
	return p.db.PingContext(ctx)
}

func (p *SQLiteProvider) Exec(ctx context.Context, query string, args ...interface{}) (interfaces.Result, error) {
	if p.db == nil {
		return nil, fmt.Errorf("database not connected")
	}
	result, err := p.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &sqliteResult{result: result}, nil
}

func (p *SQLiteProvider) QueryRow(ctx context.Context, query string, args ...interface{}) interfaces.Row {
	if p.db == nil {
		return &sqliteRow{err: fmt.Errorf("database not connected")}
	}
	return &sqliteRow{row: p.db.QueryRowContext(ctx, query, args...)}
}

func (p *SQLiteProvider) Query(ctx context.Context, query string, args ...interface{}) (interfaces.Rows, error) {
	if p.db == nil {
		return nil, fmt.Errorf("database not connected")
	}
	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &sqliteRows{rows: rows}, nil
}

func (p *SQLiteProvider) BeginTx(ctx context.Context, opts *sql.TxOptions) (interfaces.Transaction, error) {
	if p.db == nil {
		return nil, fmt.Errorf("database not connected")
	}
	tx, err := p.db.BeginTx(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	return &sqliteTransaction{tx: tx}, nil
}

func (p *SQLiteProvider) Driver() string {
	if p.driverName == "" {
		return defaultDriverName
	}
	return p.driverName
}

func (p *SQLiteProvider) Dialect() interfaces.Dialect {
	return &sqliteDialect{}
}

type sqliteTransaction struct {
	tx *sql.Tx
}

func (t *sqliteTransaction) Commit() error {
	return t.tx.Commit()
}

func (t *sqliteTransaction) Rollback() error {
	return t.tx.Rollback()
}

func (t *sqliteTransaction) Exec(ctx context.Context, query string, args ...interface{}) (interfaces.Result, error) {
	result, err := t.tx.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &sqliteResult{result: result}, nil
}

func (t *sqliteTransaction) QueryRow(ctx context.Context, query string, args ...interface{}) interfaces.Row {
	return &sqliteRow{row: t.tx.QueryRowContext(ctx, query, args...)}
}

func (t *sqliteTransaction) Query(ctx context.Context, query string, args ...interface{}) (interfaces.Rows, error) {
	rows, err := t.tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &sqliteRows{rows: rows}, nil
}

type sqliteDialect struct{}

func (d *sqliteDialect) Quote(identifier string) string {
	return fmt.Sprintf("`%s`", identifier)
}

func (d *sqliteDialect) Placeholder(_ int) string {
	return "?"
}

func (d *sqliteDialect) DataType(goType string) string {
	switch goType {
	case "string":
		return "TEXT"
	case "int", "int64":
		return "INTEGER"
	case "bool":
		return "INTEGER"
	case "time.Time":
		return "DATETIME"
	default:
		return "TEXT"
	}
}

func (d *sqliteDialect) SupportsReturning() bool {
	return false
}

func (d *sqliteDialect) SupportsLastInsertID() bool {
	return true
}

func (d *sqliteDialect) LimitOffset(limit, offset int) string {
	if limit > 0 && offset > 0 {
		return fmt.Sprintf("LIMIT %d OFFSET %d", limit, offset)
	} else if limit > 0 {
		return fmt.Sprintf("LIMIT %d", limit)
	} else if offset > 0 {
		return fmt.Sprintf("OFFSET %d", offset)
	}
	return ""
}

type sqliteResult struct {
	result sql.Result
}

func (r *sqliteResult) LastInsertId() (int64, error) {
	return r.result.LastInsertId()
}

func (r *sqliteResult) RowsAffected() (int64, error) {
	return r.result.RowsAffected()
}

type sqliteRow struct {
	row *sql.Row
	err error
}

func (r *sqliteRow) Scan(dest ...interface{}) error {
	if r.err != nil {
		return r.err
	}
	if r.row == nil {
		return fmt.Errorf("row is nil")
	}
	return r.row.Scan(dest...)
}

func (r *sqliteRow) Err() error {
	if r.err != nil {
		return r.err
	}
	if r.row == nil {
		return nil
	}
	return r.row.Err()
}

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

func (r *sqliteRows) Columns() ([]string, error) {
	return r.rows.Columns()
}
