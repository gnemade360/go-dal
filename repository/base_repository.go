package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/gnemade360/go-dal/dal/errors"
	"github.com/gnemade360/go-dal/dal/interfaces"
	"github.com/gnemade360/go-dal/dal/types"
)

// BaseRepository provides a generic repository implementation
type BaseRepository[T Entity] struct {
	db     interfaces.Database
	mapper Mapper[T]
}

// NewBaseRepository creates a new base repository
func NewBaseRepository[T Entity](db interfaces.Database, mapper Mapper[T]) *BaseRepository[T] {
	if mapper == nil {
		mapper = NewBaseMapper[T]()
	}
	
	return &BaseRepository[T]{
		db:     db,
		mapper: mapper,
	}
}

// FindByID retrieves an entity by its ID
func (r *BaseRepository[T]) FindByID(ctx context.Context, id string) (*T, error) {
	query := fmt.Sprintf("SELECT %s FROM %s WHERE %s = %s",
		buildColumnList(r.mapper.Columns(), r.db.Dialect()),
		r.db.Dialect().Quote(r.mapper.TableName()),
		r.db.Dialect().Quote("id"),
		r.db.Dialect().Placeholder(1),
	)
	
	row := r.db.QueryRow(ctx, query, id)
	entity, err := r.mapper.FromRow(row)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, errors.ErrNotFound
		}
		return nil, err
	}
	
	return entity, nil
}

// FindAll retrieves all entities with optional query options
func (r *BaseRepository[T]) FindAll(ctx context.Context, opts ...QueryOption) ([]*T, error) {
	options := &types.QueryOptions{}
	for _, opt := range opts {
		opt(options)
	}
	
	query := r.buildSelectQuery(options)
	
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	entities := make([]*T, 0)
	for rows.Next() {
		entity, err := r.mapper.FromRow(rows)
		if err != nil {
			return nil, err
		}
		entities = append(entities, entity)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return entities, nil
}

// FindBySpec retrieves entities matching the specification
func (r *BaseRepository[T]) FindBySpec(ctx context.Context, spec Specification, opts ...QueryOption) ([]*T, error) {
	options := &types.QueryOptions{}
	for _, opt := range opts {
		opt(options)
	}

	query := r.buildSelectQuery(options)
	whereClause, args := r.buildWhereClause(spec)

	if whereClause != "" {
		query += " WHERE " + whereClause
	}

	// Add ORDER BY, LIMIT, OFFSET
	query += r.buildQuerySuffix(options)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entities := make([]*T, 0)
	for rows.Next() {
		entity, err := r.mapper.FromRow(rows)
		if err != nil {
			return nil, err
		}
		entities = append(entities, entity)
	}
	
	if err := rows.Err(); err != nil {
		return nil, err
	}
	
	return entities, nil
}

// Create inserts a new entity
func (r *BaseRepository[T]) Create(ctx context.Context, entity *T) error {
	columns := r.mapper.Columns()
	values, err := r.mapper.ToRow(entity)
	if err != nil {
		return err
	}
	
	// Build column and placeholder lists
	var colList []string
	var placeholders []string
	var args []interface{}
	
	for i, col := range columns {
		if val, ok := values[col]; ok {
			colList = append(colList, r.db.Dialect().Quote(col))
			placeholders = append(placeholders, r.db.Dialect().Placeholder(i+1))
			args = append(args, val)
		}
	}
	
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		r.db.Dialect().Quote(r.mapper.TableName()),
		strings.Join(colList, ", "),
		strings.Join(placeholders, ", "),
	)
	
	result, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return errors.WrapError(err)
	}
	
	// Set the ID if it's auto-generated and database supports LastInsertId
	if r.db.Dialect().SupportsLastInsertID() {
		if id, err := result.LastInsertId(); err == nil && id > 0 {
			// This would require reflection to set the ID field
			// For now, we assume the entity already has an ID
		}
	}
	
	return nil
}

// Update updates an existing entity
func (r *BaseRepository[T]) Update(ctx context.Context, entity *T) error {
	values, err := r.mapper.ToRow(entity)
	if err != nil {
		return err
	}
	
	id, ok := values["id"]
	if !ok {
		return errors.NewError(errors.CodeValidation, "entity missing ID", nil)
	}
	
	// Build SET clause
	var setClauses []string
	var args []interface{}
	argIndex := 1
	
	for col, val := range values {
		if col != "id" {
			setClauses = append(setClauses, fmt.Sprintf("%s = %s",
				r.db.Dialect().Quote(col),
				r.db.Dialect().Placeholder(argIndex),
			))
			args = append(args, val)
			argIndex++
		}
	}
	
	// Add ID as last argument
	args = append(args, id)
	
	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s = %s",
		r.db.Dialect().Quote(r.mapper.TableName()),
		strings.Join(setClauses, ", "),
		r.db.Dialect().Quote("id"),
		r.db.Dialect().Placeholder(argIndex),
	)
	
	result, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return errors.WrapError(err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	
	if rowsAffected == 0 {
		return errors.ErrNotFound
	}
	
	return nil
}

// FindOne retrieves a single entity matching the specification
func (r *BaseRepository[T]) FindOne(ctx context.Context, spec Specification) (*T, error) {
	entities, err := r.Find(ctx, spec)
	if err != nil {
		return nil, err
	}
	if len(entities) == 0 {
		return nil, errors.ErrNotFound
	}
	return entities[0], nil
}

// Find retrieves entities matching the specification
func (r *BaseRepository[T]) Find(ctx context.Context, spec Specification) ([]*T, error) {
	return r.FindWithOptions(ctx, spec)
}

// FindWithOptions retrieves entities matching the specification with query options
func (r *BaseRepository[T]) FindWithOptions(ctx context.Context, spec Specification, opts ...QueryOption) ([]*T, error) {
	return r.FindBySpec(ctx, spec, opts...)
}

// CreateBatch inserts multiple entities
func (r *BaseRepository[T]) CreateBatch(ctx context.Context, entities []*T) error {
	if len(entities) == 0 {
		return nil
	}
	
	// For now, implement as a loop - can be optimized with bulk insert later
	for _, entity := range entities {
		if err := r.Create(ctx, entity); err != nil {
			return err
		}
	}
	return nil
}

// UpdateBatch updates multiple entities
func (r *BaseRepository[T]) UpdateBatch(ctx context.Context, entities []*T) error {
	if len(entities) == 0 {
		return nil
	}
	
	// For now, implement as a loop - can be optimized with bulk update later
	for _, entity := range entities {
		if err := r.Update(ctx, entity); err != nil {
			return err
		}
	}
	return nil
}

// DeleteBatch removes multiple entities by IDs
func (r *BaseRepository[T]) DeleteBatch(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	
	// For now, implement as a loop - can be optimized with bulk delete later
	for _, id := range ids {
		if err := r.Delete(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

// WithTransaction returns a repository that uses the given transaction
func (r *BaseRepository[T]) WithTransaction(tx Transaction) Repository[T] {
	if dbTx, ok := tx.(interfaces.Transaction); ok {
		return &transactionalRepository[T]{
			BaseRepository: r,
			tx:            dbTx,
		}
	}
	return r
}

// Delete removes an entity by ID (renamed to match interface)
func (r *BaseRepository[T]) Delete(ctx context.Context, id string) error {
	return r.DeleteByID(ctx, id)
}

// DeleteByID removes an entity by ID
func (r *BaseRepository[T]) DeleteByID(ctx context.Context, id string) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE %s = %s",
		r.db.Dialect().Quote(r.mapper.TableName()),
		r.db.Dialect().Quote("id"),
		r.db.Dialect().Placeholder(1),
	)
	
	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return errors.WrapError(err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	
	if rowsAffected == 0 {
		return errors.ErrNotFound
	}
	
	return nil
}

// DeleteBySpec removes entities matching the specification
func (r *BaseRepository[T]) DeleteBySpec(ctx context.Context, spec Specification) (int64, error) {
	whereClause, args := r.buildWhereClause(spec)
	
	query := fmt.Sprintf("DELETE FROM %s", r.db.Dialect().Quote(r.mapper.TableName()))
	if whereClause != "" {
		query += " WHERE " + whereClause
	}
	
	result, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return 0, errors.WrapError(err)
	}
	
	return result.RowsAffected()
}

// Count returns the number of entities matching the specification
func (r *BaseRepository[T]) Count(ctx context.Context, spec Specification) (int64, error) {
	whereClause, args := r.buildWhereClause(spec)
	
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s",
		r.db.Dialect().Quote(r.mapper.TableName()),
	)
	
	if whereClause != "" {
		query += " WHERE " + whereClause
	}
	
	var count int64
	row := r.db.QueryRow(ctx, query, args...)
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	
	return count, nil
}

// Exists checks if any entity matches the specification
func (r *BaseRepository[T]) Exists(ctx context.Context, spec Specification) (bool, error) {
	count, err := r.Count(ctx, spec)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// Transaction executes a function within a transaction
func (r *BaseRepository[T]) Transaction(ctx context.Context, fn func(ctx context.Context, repo Repository[T]) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	
	// Create a new repository instance with the transaction
	txRepo := &transactionalRepository[T]{
		BaseRepository: r,
		tx:            tx,
	}
	
	if err := fn(ctx, txRepo); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return errors.NewError(errors.CodeTransaction, "failed to rollback transaction", rbErr)
		}
		return err
	}
	
	if err := tx.Commit(); err != nil {
		return errors.NewError(errors.CodeTransaction, "failed to commit transaction", err)
	}
	
	return nil
}

// buildSelectQuery builds the SELECT part of the query
func (r *BaseRepository[T]) buildSelectQuery(options *types.QueryOptions) string {
	columns := buildColumnList(r.mapper.Columns(), r.db.Dialect())
	
	query := "SELECT "
	if options.Distinct {
		query += "DISTINCT "
	}
	
	query += columns + " FROM " + r.db.Dialect().Quote(r.mapper.TableName())
	
	return query
}

// buildWhereClause builds the WHERE clause from a specification
func (r *BaseRepository[T]) buildWhereClause(spec Specification) (string, []interface{}) {
	if spec == nil {
		return "", nil
	}
	
	sql, args, err := spec.ToSQL(r.db.Dialect())
	if err != nil {
		// For now, return empty clause on error - could log this
		return "", nil
	}
	return sql, args
}

// buildQuerySuffix builds ORDER BY, LIMIT, OFFSET clauses
func (r *BaseRepository[T]) buildQuerySuffix(options *types.QueryOptions) string {
	var parts []string
	
	// ORDER BY
	if len(options.OrderBy) > 0 {
		var orderClauses []string
		for _, order := range options.OrderBy {
			clause := r.db.Dialect().Quote(order.Field)
			if order.Direction == types.OrderDesc {
				clause += " DESC"
			} else {
				clause += " ASC"
			}
			orderClauses = append(orderClauses, clause)
		}
		parts = append(parts, "ORDER BY "+strings.Join(orderClauses, ", "))
	}
	
	// LIMIT/OFFSET
	if limitOffset := r.db.Dialect().LimitOffset(options.Limit, options.Offset); limitOffset != "" {
		parts = append(parts, limitOffset)
	}
	
	// FOR UPDATE
	if options.ForUpdate {
		parts = append(parts, "FOR UPDATE")
	}
	
	if len(parts) > 0 {
		return " " + strings.Join(parts, " ")
	}
	
	return ""
}

// transactionalRepository wraps BaseRepository for transaction support
type transactionalRepository[T Entity] struct {
	*BaseRepository[T]
	tx interfaces.Transaction
}

// Override database operations to use transaction
func (r *transactionalRepository[T]) FindByID(ctx context.Context, id string) (*T, error) {
	query := fmt.Sprintf("SELECT %s FROM %s WHERE %s = %s",
		buildColumnList(r.mapper.Columns(), r.db.Dialect()),
		r.db.Dialect().Quote(r.mapper.TableName()),
		r.db.Dialect().Quote("id"),
		r.db.Dialect().Placeholder(1),
	)
	
	row := r.tx.QueryRow(ctx, query, id)
	entity, err := r.mapper.FromRow(row)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, errors.ErrNotFound
		}
		return nil, err
	}
	
	return entity, nil
}

// FindOne retrieves a single entity matching the specification  
func (r *transactionalRepository[T]) FindOne(ctx context.Context, spec Specification) (*T, error) {
	entities, err := r.Find(ctx, spec)
	if err != nil {
		return nil, err
	}
	if len(entities) == 0 {
		return nil, errors.ErrNotFound
	}
	return entities[0], nil
}

// Find retrieves entities matching the specification
func (r *transactionalRepository[T]) Find(ctx context.Context, spec Specification) ([]*T, error) {
	return r.FindWithOptions(ctx, spec)
}

// FindWithOptions retrieves entities matching the specification with query options
func (r *transactionalRepository[T]) FindWithOptions(ctx context.Context, spec Specification, opts ...QueryOption) ([]*T, error) {
	return r.FindBySpec(ctx, spec, opts...)
}

// CreateBatch inserts multiple entities in transaction
func (r *transactionalRepository[T]) CreateBatch(ctx context.Context, entities []*T) error {
	if len(entities) == 0 {
		return nil
	}
	
	for _, entity := range entities {
		if err := r.Create(ctx, entity); err != nil {
			return err
		}
	}
	return nil
}

// UpdateBatch updates multiple entities in transaction
func (r *transactionalRepository[T]) UpdateBatch(ctx context.Context, entities []*T) error {
	if len(entities) == 0 {
		return nil
	}
	
	for _, entity := range entities {
		if err := r.Update(ctx, entity); err != nil {
			return err
		}
	}
	return nil
}

// DeleteBatch removes multiple entities by IDs in transaction
func (r *transactionalRepository[T]) DeleteBatch(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	
	for _, id := range ids {
		if err := r.Delete(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

// WithTransaction returns self as it's already a transactional repository
func (r *transactionalRepository[T]) WithTransaction(tx Transaction) Repository[T] {
	return r
}