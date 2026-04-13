package repository

import (
	"context"
	"github.com/gnemade360/go-dal/dal/interfaces"
)

// Repository defines the generic repository interface
type Repository interface {
	Create(ctx context.Context, entity interface{}) error
	FindByID(ctx context.Context, id string, dest interface{}) error
	FindAll(ctx context.Context, dest interface{}) error
	Update(ctx context.Context, entity interface{}) error
	Delete(ctx context.Context, id string) error
}

// GenericRepository provides a base repository with common functionality
type GenericRepository struct {
	db        interfaces.Database
	tableName string
}

// NewGenericRepository creates a new generic repository
func NewGenericRepository(db interfaces.Database, tableName string) *GenericRepository {
	return &GenericRepository{
		db:        db,
		tableName: tableName,
	}
}

// GetDB returns the database connection for custom queries
func (r *GenericRepository) GetDB() interfaces.Database {
	return r.db
}

// GetTableName returns the table name
func (r *GenericRepository) GetTableName() string {
	return r.tableName
}

// Execute runs a custom query
func (r *GenericRepository) Execute(ctx context.Context, query string, args ...interface{}) (interfaces.Result, error) {
	return r.db.Exec(ctx, query, args...)
}

// Query runs a custom query that returns rows
func (r *GenericRepository) Query(ctx context.Context, query string, args ...interface{}) (interfaces.Rows, error) {
	return r.db.Query(ctx, query, args...)
}

// QueryRow runs a custom query that returns a single row
func (r *GenericRepository) QueryRow(ctx context.Context, query string, args ...interface{}) interfaces.Row {
	return r.db.QueryRow(ctx, query, args...)
}

// BeginTx starts a new transaction
func (r *GenericRepository) BeginTx(ctx context.Context) (interfaces.Transaction, error) {
	return r.db.BeginTx(ctx, nil)
}