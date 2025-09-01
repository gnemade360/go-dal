package unitofwork

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/airoles/go-dal/dal/interfaces"
)

// BaseUnitOfWork provides a base implementation of UnitOfWork
type BaseUnitOfWork struct {
	db           interfaces.Database
	tx           interfaces.Transaction
	repositories map[reflect.Type]interface{}
	factories    map[reflect.Type]RepositoryFactory
	mu           sync.RWMutex
}

// NewUnitOfWork creates a new unit of work
func NewUnitOfWork(db interfaces.Database) *BaseUnitOfWork {
	return &BaseUnitOfWork{
		db:           db,
		repositories: make(map[reflect.Type]interface{}),
		factories:    make(map[reflect.Type]RepositoryFactory),
	}
}

// Begin starts a new unit of work transaction
func (uow *BaseUnitOfWork) Begin(ctx context.Context) error {
	if uow.tx != nil {
		return fmt.Errorf("unit of work already started")
	}
	
	tx, err := uow.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	
	uow.tx = tx
	// Clear cached repositories for new transaction
	uow.repositories = make(map[reflect.Type]interface{})
	return nil
}

// Commit commits the unit of work
func (uow *BaseUnitOfWork) Commit() error {
	if uow.tx == nil {
		return fmt.Errorf("no active transaction")
	}
	
	err := uow.tx.Commit()
	uow.tx = nil
	uow.repositories = make(map[reflect.Type]interface{})
	return err
}

// Rollback rolls back the unit of work
func (uow *BaseUnitOfWork) Rollback() error {
	if uow.tx == nil {
		return fmt.Errorf("no active transaction")
	}
	
	err := uow.tx.Rollback()
	uow.tx = nil
	uow.repositories = make(map[reflect.Type]interface{})
	return err
}

// Repository returns a repository for the given entity type
func (uow *BaseUnitOfWork) Repository(entityType interface{}) interface{} {
	uow.mu.RLock()
	t := reflect.TypeOf(entityType)
	
	// Check if repository already exists
	if repo, exists := uow.repositories[t]; exists {
		uow.mu.RUnlock()
		return repo
	}
	uow.mu.RUnlock()
	
	// Create new repository
	uow.mu.Lock()
	defer uow.mu.Unlock()
	
	// Double-check after acquiring write lock
	if repo, exists := uow.repositories[t]; exists {
		return repo
	}
	
	// Get factory
	factory, exists := uow.factories[t]
	if !exists {
		panic(fmt.Sprintf("no repository factory registered for type %v", t))
	}
	
	// Create repository
	repo := factory(uow)
	uow.repositories[t] = repo
	return repo
}

// RegisterRepository registers a repository factory
func (uow *BaseUnitOfWork) RegisterRepository(entityType interface{}, factory RepositoryFactory) {
	uow.mu.Lock()
	defer uow.mu.Unlock()
	
	t := reflect.TypeOf(entityType)
	uow.factories[t] = factory
}

// InTransaction executes a function within a transaction
func (uow *BaseUnitOfWork) InTransaction(ctx context.Context, fn func(uow UnitOfWork) error) error {
	if err := uow.Begin(ctx); err != nil {
		return err
	}
	
	if err := fn(uow); err != nil {
		if rbErr := uow.Rollback(); rbErr != nil {
			return fmt.Errorf("transaction failed: %v, rollback failed: %v", err, rbErr)
		}
		return err
	}
	
	return uow.Commit()
}

// GetDatabase returns the underlying database or transaction
func (uow *BaseUnitOfWork) GetDatabase() interface{} {
	if uow.tx != nil {
		return uow.tx
	}
	return uow.db
}