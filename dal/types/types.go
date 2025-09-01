package types

import "time"

// LockMode represents the type of database lock
type LockMode string

const (
	LockModeNone        LockMode = ""
	LockModeForUpdate   LockMode = "FOR UPDATE"
	LockModeForShare    LockMode = "FOR SHARE"
	LockModeNoWait      LockMode = "NOWAIT"
	LockModeSkipLocked  LockMode = "SKIP LOCKED"
)

// OrderDirection represents the sort order
type OrderDirection string

const (
	OrderAsc  OrderDirection = "ASC"
	OrderDesc OrderDirection = "DESC"
)

// OrderClause represents a single order by clause
type OrderClause struct {
	Field     string
	Direction OrderDirection
}

// QueryOptions contains options for query execution
type QueryOptions struct {
	Limit      int
	Offset     int
	OrderBy    []OrderClause
	Preload    []string
	LockMode   LockMode
	ForUpdate  bool
	Distinct   bool
}

// Auditable represents entities with audit fields
type Auditable interface {
	GetCreatedAt() time.Time
	SetCreatedAt(time.Time)
	GetUpdatedAt() time.Time
	SetUpdatedAt(time.Time)
	GetCreatedBy() string
	SetCreatedBy(string)
	GetUpdatedBy() string
	SetUpdatedBy(string)
}

// SoftDeletable represents entities that support soft delete
type SoftDeletable interface {
	GetDeletedAt() *time.Time
	SetDeletedAt(*time.Time)
	IsDeleted() bool
}

// BaseEntity provides common fields for entities
type BaseEntity struct {
	ID        string     `db:"id" json:"id"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt time.Time  `db:"updated_at" json:"updated_at"`
	CreatedBy string     `db:"created_by" json:"created_by,omitempty"`
	UpdatedBy string     `db:"updated_by" json:"updated_by,omitempty"`
	DeletedAt *time.Time `db:"deleted_at" json:"deleted_at,omitempty"`
}

// GetID returns the entity ID
func (e BaseEntity) GetID() string { return e.ID }

// SetID sets the entity ID (value receiver for interface compliance)
func (e BaseEntity) SetID(id string) { 
	// Note: This has a value receiver to satisfy Entity interface constraints.
	// It won't actually modify the receiver. Repositories work with pointers.
}

// SetIDPtr sets the entity ID (pointer receiver for actual mutations)
func (e *BaseEntity) SetIDPtr(id string) { 
	e.ID = id
}

// GetCreatedAt returns the creation time
func (e *BaseEntity) GetCreatedAt() time.Time { return e.CreatedAt }

// SetCreatedAt sets the creation time
func (e *BaseEntity) SetCreatedAt(t time.Time) { e.CreatedAt = t }

// GetUpdatedAt returns the update time
func (e *BaseEntity) GetUpdatedAt() time.Time { return e.UpdatedAt }

// SetUpdatedAt sets the update time
func (e *BaseEntity) SetUpdatedAt(t time.Time) { e.UpdatedAt = t }

// GetCreatedBy returns the creator
func (e *BaseEntity) GetCreatedBy() string { return e.CreatedBy }

// SetCreatedBy sets the creator
func (e *BaseEntity) SetCreatedBy(by string) { e.CreatedBy = by }

// GetUpdatedBy returns the updater
func (e *BaseEntity) GetUpdatedBy() string { return e.UpdatedBy }

// SetUpdatedBy sets the updater
func (e *BaseEntity) SetUpdatedBy(by string) { e.UpdatedBy = by }

// GetDeletedAt returns the deletion time
func (e *BaseEntity) GetDeletedAt() *time.Time { return e.DeletedAt }

// SetDeletedAt sets the deletion time
func (e *BaseEntity) SetDeletedAt(t *time.Time) { e.DeletedAt = t }

// IsDeleted checks if the entity is soft deleted
func (e *BaseEntity) IsDeleted() bool { return e.DeletedAt != nil }