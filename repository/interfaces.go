package repository

import (
	"context"
)

// Entity represents a database entity
// All methods should have value receivers to work with Go's type system constraints
// Actual mutations happen through pointer types in repository methods
type Entity interface {
	GetID() string
	SetID(id string) // Should have value receiver (won't mutate in practice)
	TableName() string
}

// Repository provides generic CRUD operations
type Repository[T Entity] interface {
	// Basic CRUD
	FindByID(ctx context.Context, id string) (*T, error)
	FindAll(ctx context.Context, opts ...QueryOption) ([]*T, error)
	FindOne(ctx context.Context, spec Specification) (*T, error)
	Create(ctx context.Context, entity *T) error
	Update(ctx context.Context, entity *T) error
	Delete(ctx context.Context, id string) error

	// Batch operations
	CreateBatch(ctx context.Context, entities []*T) error
	UpdateBatch(ctx context.Context, entities []*T) error
	DeleteBatch(ctx context.Context, ids []string) error

	// Aggregations
	Count(ctx context.Context, spec Specification) (int64, error)
	Exists(ctx context.Context, spec Specification) (bool, error)

	// Advanced queries
	Find(ctx context.Context, spec Specification) ([]*T, error)
	FindWithOptions(ctx context.Context, spec Specification, opts ...QueryOption) ([]*T, error)

	// Transaction support
	WithTransaction(tx Transaction) Repository[T]
}

// UnitOfWork manages transactions across repositories
type UnitOfWork interface {
	Begin(ctx context.Context) (UnitOfWorkTransaction, error)
	RegisterRepository(name string, factory RepositoryFactory)
	Repository(name string) interface{}
}

// UnitOfWorkTransaction represents an active transaction
type UnitOfWorkTransaction interface {
	Commit() error
	Rollback() error
	Repository(name string) interface{}
}

// RepositoryFactory creates repository instances
type RepositoryFactory func(tx Transaction) interface{}

// Transaction represents a database transaction for repositories
type Transaction interface {
	context.Context
}