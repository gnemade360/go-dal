package interfaces

// Entity represents a database entity
type Entity interface {
	// GetID returns the entity's unique identifier
	GetID() string

	// SetID sets the entity's unique identifier
	SetID(id string)

	// TableName returns the database table name for this entity
	TableName() string
}