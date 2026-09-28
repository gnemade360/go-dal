package interfaces

import "context"

type Store interface {
	FindByID(ctx context.Context, tableName string, id string, dest interface{}) error
	FindAll(ctx context.Context, tableName string, dest interface{}, opts QueryOptions) error
	FindByFilter(ctx context.Context, tableName string, dest interface{}, filter Filter, opts QueryOptions) error
	Count(ctx context.Context, tableName string, filter Filter) (int64, error)
	Create(ctx context.Context, tableName string, id string, data map[string]interface{}) error
	Update(ctx context.Context, tableName string, id string, data map[string]interface{}) error
	Delete(ctx context.Context, tableName string, id string) error
	BeginTx(ctx context.Context) (StoreTx, error)
}

type StoreTx interface {
	Store
	Commit() error
	Rollback() error
}

type QueryOptions struct {
	Limit   int
	Offset  int
	OrderBy []OrderClause
}

type OrderClause struct {
	Field     string
	Direction string
}

type Filter struct {
	Field string
	Op    string
	Value interface{}
	And   []Filter
	Or    []Filter
}
