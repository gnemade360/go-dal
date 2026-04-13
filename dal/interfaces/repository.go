package interfaces

import "context"

// Repository provides generic CRUD operations for entities
type Repository[T Entity] interface {
	// Create inserts a new entity
	Create(ctx context.Context, entity *T) error

	// FindByID retrieves an entity by its ID
	FindByID(ctx context.Context, id string) (*T, error)

	// FindAll retrieves all entities
	FindAll(ctx context.Context) ([]*T, error)

	// Update updates an existing entity
	Update(ctx context.Context, entity *T) error

	// Delete removes an entity by ID
	Delete(ctx context.Context, id string) error

	// Count returns the number of entities
	Count(ctx context.Context) (int64, error)

	// Exists checks if an entity with the given ID exists
	Exists(ctx context.Context, id string) (bool, error)
}