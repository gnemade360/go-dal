package sqlite

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gnemade360/go-dal/dal/errors"
	"github.com/gnemade360/go-dal/dal/interfaces"
)

type StoreAdapter struct {
	provider *SQLiteProvider
}

func NewStoreAdapter(provider *SQLiteProvider) *StoreAdapter {
	return &StoreAdapter{provider: provider}
}

func (s *StoreAdapter) FindByID(ctx context.Context, tableName string, id string, dest interface{}) error {
	d := s.provider.Dialect()
	query := fmt.Sprintf("SELECT * FROM %s WHERE %s = ?", d.Quote(tableName), d.Quote("id"))

	row := s.provider.QueryRow(ctx, query, id)
	columns, err := s.getTableColumns(ctx, tableName)
	if err != nil {
		return err
	}

	values := make([]interface{}, len(columns))
	ptrs := make([]interface{}, len(columns))
	for i := range values {
		ptrs[i] = &values[i]
	}

	if err := row.Scan(ptrs...); err != nil {
		if err.Error() == "sql: no rows in result set" {
			return errors.NewNotFoundError(tableName, id)
		}
		return err
	}

	record := make(map[string]interface{})
	for i, col := range columns {
		record[col] = values[i]
	}

	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

func (s *StoreAdapter) FindAll(ctx context.Context, tableName string, dest interface{}, opts interfaces.QueryOptions) error {
	return s.FindByFilter(ctx, tableName, dest, interfaces.Filter{}, opts)
}

func (s *StoreAdapter) FindByFilter(ctx context.Context, tableName string, dest interface{}, filter interfaces.Filter, opts interfaces.QueryOptions) error {
	d := s.provider.Dialect()
	query := fmt.Sprintf("SELECT * FROM %s", d.Quote(tableName))

	var args []interface{}
	if filter.Field != "" || len(filter.And) > 0 || len(filter.Or) > 0 {
		where, whereArgs := buildSQLFilter(filter, d)
		if where != "" {
			query += " WHERE " + where
			args = append(args, whereArgs...)
		}
	}

	if len(opts.OrderBy) > 0 {
		var orderClauses []string
		for _, o := range opts.OrderBy {
			dir := "ASC"
			if strings.ToUpper(o.Direction) == "DESC" {
				dir = "DESC"
			}
			orderClauses = append(orderClauses, d.Quote(o.Field)+" "+dir)
		}
		query += " ORDER BY " + strings.Join(orderClauses, ", ")
	}

	if limitOffset := d.LimitOffset(opts.Limit, opts.Offset); limitOffset != "" {
		query += " " + limitOffset
	}

	rows, err := s.provider.Query(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	var records []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		ptrs := make([]interface{}, len(columns))
		for i := range values {
			ptrs[i] = &values[i]
		}

		if err := rows.Scan(ptrs...); err != nil {
			return err
		}

		record := make(map[string]interface{})
		for i, col := range columns {
			record[col] = values[i]
		}
		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return err
	}

	if records == nil {
		records = []map[string]interface{}{}
	}

	data, err := json.Marshal(records)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

func (s *StoreAdapter) Count(ctx context.Context, tableName string, filter interfaces.Filter) (int64, error) {
	d := s.provider.Dialect()
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", d.Quote(tableName))

	var args []interface{}
	if filter.Field != "" || len(filter.And) > 0 || len(filter.Or) > 0 {
		where, whereArgs := buildSQLFilter(filter, d)
		if where != "" {
			query += " WHERE " + where
			args = append(args, whereArgs...)
		}
	}

	var count int64
	row := s.provider.QueryRow(ctx, query, args...)
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (s *StoreAdapter) Create(ctx context.Context, tableName string, id string, data map[string]interface{}) error {
	d := s.provider.Dialect()

	var columns []string
	var placeholders []string
	var args []interface{}

	for col, val := range data {
		columns = append(columns, d.Quote(col))
		placeholders = append(placeholders, "?")
		args = append(args, val)
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		d.Quote(tableName),
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)

	_, err := s.provider.Exec(ctx, query, args...)
	return err
}

func (s *StoreAdapter) Update(ctx context.Context, tableName string, id string, data map[string]interface{}) error {
	d := s.provider.Dialect()

	var setClauses []string
	var args []interface{}

	for col, val := range data {
		if col == "id" {
			continue
		}
		setClauses = append(setClauses, fmt.Sprintf("%s = ?", d.Quote(col)))
		args = append(args, val)
	}

	args = append(args, id)

	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s = ?",
		d.Quote(tableName),
		strings.Join(setClauses, ", "),
		d.Quote("id"),
	)

	result, err := s.provider.Exec(ctx, query, args...)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return errors.NewNotFoundError(tableName, id)
	}
	return nil
}

func (s *StoreAdapter) Delete(ctx context.Context, tableName string, id string) error {
	d := s.provider.Dialect()
	query := fmt.Sprintf("DELETE FROM %s WHERE %s = ?", d.Quote(tableName), d.Quote("id"))

	result, err := s.provider.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return errors.NewNotFoundError(tableName, id)
	}
	return nil
}

func (s *StoreAdapter) BeginTx(ctx context.Context) (interfaces.StoreTx, error) {
	tx, err := s.provider.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &sqlStoreTx{tx: tx, adapter: s}, nil
}

func (s *StoreAdapter) getTableColumns(ctx context.Context, tableName string) ([]string, error) {
	query := fmt.Sprintf("PRAGMA table_info(%s)", s.provider.Dialect().Quote(tableName))
	rows, err := s.provider.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var cid int
		var name, colType string
		var notNull int
		var dfltValue interface{}
		var pk int
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			return nil, err
		}
		columns = append(columns, name)
	}
	return columns, rows.Err()
}

func buildSQLFilter(filter interfaces.Filter, d interfaces.Dialect) (string, []interface{}) {
	var parts []string
	var args []interface{}

	if filter.Field != "" {
		clause, a := buildFieldClause(filter, d)
		parts = append(parts, clause)
		args = append(args, a...)
	}

	for _, f := range filter.And {
		clause, a := buildSQLFilter(f, d)
		if clause != "" {
			parts = append(parts, clause)
			args = append(args, a...)
		}
	}

	where := strings.Join(parts, " AND ")

	if len(filter.Or) > 0 {
		var orParts []string
		for _, f := range filter.Or {
			clause, a := buildSQLFilter(f, d)
			if clause != "" {
				orParts = append(orParts, clause)
				args = append(args, a...)
			}
		}
		if len(orParts) > 0 {
			orClause := "(" + strings.Join(orParts, " OR ") + ")"
			if where != "" {
				where += " AND " + orClause
			} else {
				where = orClause
			}
		}
	}

	return where, args
}

func buildFieldClause(filter interfaces.Filter, d interfaces.Dialect) (string, []interface{}) {
	col := d.Quote(filter.Field)
	switch strings.ToUpper(filter.Op) {
	case "IS NULL":
		return col + " IS NULL", nil
	case "IS NOT NULL":
		return col + " IS NOT NULL", nil
	case "IN":
		if vals, ok := filter.Value.([]interface{}); ok {
			placeholders := make([]string, len(vals))
			for i := range vals {
				placeholders[i] = "?"
			}
			return fmt.Sprintf("%s IN (%s)", col, strings.Join(placeholders, ", ")), vals
		}
		return col + " IN (?)", []interface{}{filter.Value}
	default:
		op := filter.Op
		if op == "" || op == "EQ" {
			op = "="
		}
		return fmt.Sprintf("%s %s ?", col, op), []interface{}{filter.Value}
	}
}

type sqlStoreTx struct {
	tx      interfaces.Transaction
	adapter *StoreAdapter
}

func (t *sqlStoreTx) FindByID(ctx context.Context, tableName string, id string, dest interface{}) error {
	return t.adapter.FindByID(ctx, tableName, id, dest)
}

func (t *sqlStoreTx) FindAll(ctx context.Context, tableName string, dest interface{}, opts interfaces.QueryOptions) error {
	return t.adapter.FindAll(ctx, tableName, dest, opts)
}

func (t *sqlStoreTx) FindByFilter(ctx context.Context, tableName string, dest interface{}, filter interfaces.Filter, opts interfaces.QueryOptions) error {
	return t.adapter.FindByFilter(ctx, tableName, dest, filter, opts)
}

func (t *sqlStoreTx) Count(ctx context.Context, tableName string, filter interfaces.Filter) (int64, error) {
	return t.adapter.Count(ctx, tableName, filter)
}

func (t *sqlStoreTx) Create(ctx context.Context, tableName string, id string, data map[string]interface{}) error {
	return t.adapter.Create(ctx, tableName, id, data)
}

func (t *sqlStoreTx) Update(ctx context.Context, tableName string, id string, data map[string]interface{}) error {
	return t.adapter.Update(ctx, tableName, id, data)
}

func (t *sqlStoreTx) Delete(ctx context.Context, tableName string, id string) error {
	return t.adapter.Delete(ctx, tableName, id)
}

func (t *sqlStoreTx) BeginTx(ctx context.Context) (interfaces.StoreTx, error) {
	return t, nil
}

func (t *sqlStoreTx) Commit() error {
	return t.tx.Commit()
}

func (t *sqlStoreTx) Rollback() error {
	return t.tx.Rollback()
}
