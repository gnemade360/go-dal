package unitofwork

import (
	"context"

	"github.com/gnemade360/go-dal/repository"
)

// UnitOfWork represents a unit of work pattern
type UnitOfWork interface {
	// Begin starts a new unit of work
	Begin(ctx context.Context) error
	
	// Commit commits all changes
	Commit() error
	
	// Rollback rolls back all changes
	Rollback() error
	
	// Repository returns a repository for the given entity type
	Repository(entityType interface{}) interface{}
	
	// RegisterRepository registers a repository factory
	RegisterRepository(entityType interface{}, factory RepositoryFactory)
}

// RepositoryFactory creates repository instances for a unit of work
type RepositoryFactory func(uow UnitOfWork) interface{}

// TransactionalUnitOfWork extends UnitOfWork with transaction support
type TransactionalUnitOfWork interface {
	UnitOfWork
	
	// InTransaction executes a function within a transaction
	InTransaction(ctx context.Context, fn func(uow UnitOfWork) error) error
}

// EntityRepository provides generic repository access
type EntityRepository[T repository.Entity] interface {
	Get() repository.Repository[T]
}