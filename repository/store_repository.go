package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gnemade360/go-dal/dal/errors"
	"github.com/gnemade360/go-dal/dal/interfaces"
	"github.com/gnemade360/go-dal/dal/types"
)

type StoreRepository[T Entity] struct {
	store interfaces.Store
}

func NewStoreRepository[T Entity](store interfaces.Store) *StoreRepository[T] {
	return &StoreRepository[T]{store: store}
}

func (r *StoreRepository[T]) FindByID(ctx context.Context, id string) (*T, error) {
	var entity T
	tableName := entity.TableName()

	if err := r.store.FindByID(ctx, tableName, id, &entity); err != nil {
		return nil, err
	}
	return &entity, nil
}

func (r *StoreRepository[T]) FindAll(ctx context.Context, opts ...QueryOption) ([]*T, error) {
	var entity T
	tableName := entity.TableName()

	qopts := convertQueryOptions(opts)

	var entities []T
	if err := r.store.FindAll(ctx, tableName, &entities, qopts); err != nil {
		return nil, err
	}

	result := make([]*T, len(entities))
	for i := range entities {
		result[i] = &entities[i]
	}
	return result, nil
}

func (r *StoreRepository[T]) FindOne(ctx context.Context, spec Specification) (*T, error) {
	entities, err := r.Find(ctx, spec)
	if err != nil {
		return nil, err
	}
	if len(entities) == 0 {
		return nil, errors.ErrNotFound
	}
	return entities[0], nil
}

func (r *StoreRepository[T]) Find(ctx context.Context, spec Specification) ([]*T, error) {
	return r.FindWithOptions(ctx, spec)
}

func (r *StoreRepository[T]) FindWithOptions(ctx context.Context, spec Specification, opts ...QueryOption) ([]*T, error) {
	var entity T
	tableName := entity.TableName()

	filter := specToFilter(spec)
	qopts := convertQueryOptions(opts)

	var entities []T
	if err := r.store.FindByFilter(ctx, tableName, &entities, filter, qopts); err != nil {
		return nil, err
	}

	result := make([]*T, len(entities))
	for i := range entities {
		result[i] = &entities[i]
	}
	return result, nil
}

func (r *StoreRepository[T]) Create(ctx context.Context, entity *T) error {
	e := *entity
	tableName := e.TableName()
	id := e.GetID()

	data, err := entityToMap(entity)
	if err != nil {
		return err
	}

	return r.store.Create(ctx, tableName, id, data)
}

func (r *StoreRepository[T]) Update(ctx context.Context, entity *T) error {
	e := *entity
	tableName := e.TableName()
	id := e.GetID()

	data, err := entityToMap(entity)
	if err != nil {
		return err
	}

	return r.store.Update(ctx, tableName, id, data)
}

func (r *StoreRepository[T]) Delete(ctx context.Context, id string) error {
	var entity T
	return r.store.Delete(ctx, entity.TableName(), id)
}

func (r *StoreRepository[T]) CreateBatch(ctx context.Context, entities []*T) error {
	for _, entity := range entities {
		if err := r.Create(ctx, entity); err != nil {
			return err
		}
	}
	return nil
}

func (r *StoreRepository[T]) UpdateBatch(ctx context.Context, entities []*T) error {
	for _, entity := range entities {
		if err := r.Update(ctx, entity); err != nil {
			return err
		}
	}
	return nil
}

func (r *StoreRepository[T]) DeleteBatch(ctx context.Context, ids []string) error {
	for _, id := range ids {
		if err := r.Delete(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

func (r *StoreRepository[T]) Count(ctx context.Context, spec Specification) (int64, error) {
	var entity T
	filter := specToFilter(spec)
	return r.store.Count(ctx, entity.TableName(), filter)
}

func (r *StoreRepository[T]) Exists(ctx context.Context, spec Specification) (bool, error) {
	count, err := r.Count(ctx, spec)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *StoreRepository[T]) WithTransaction(tx Transaction) Repository[T] {
	return r
}

func entityToMap(entity interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(entity)
	if err != nil {
		return nil, fmt.Errorf("marshal entity: %w", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("unmarshal to map: %w", err)
	}
	return m, nil
}

func specToFilter(spec Specification) interfaces.Filter {
	if spec == nil {
		return interfaces.Filter{}
	}

	switch s := spec.(type) {
	case *FieldSpecification:
		return interfaces.Filter{
			Field: s.Field,
			Op:    string(s.Operator),
			Value: s.Value,
		}
	case *IDSpecification:
		return interfaces.Filter{
			Field: "id",
			Op:    "=",
			Value: s.ID,
		}
	case *CompositeSpecification:
		left := specToFilter(s.Left)
		right := specToFilter(s.Right)
		switch s.Operator {
		case OperatorAnd:
			return interfaces.Filter{
				And: []interfaces.Filter{left, right},
			}
		case OperatorOr:
			return interfaces.Filter{
				Or: []interfaces.Filter{left, right},
			}
		}
	case *TrueSpecification:
		return interfaces.Filter{}
	case *FalseSpecification:
		return interfaces.Filter{Field: "1", Op: "=", Value: "0"}
	}

	return interfaces.Filter{}
}

func convertQueryOptions(opts []QueryOption) interfaces.QueryOptions {
	tOpts := &types.QueryOptions{}
	for _, opt := range opts {
		opt(tOpts)
	}

	result := interfaces.QueryOptions{
		Limit:  tOpts.Limit,
		Offset: tOpts.Offset,
	}

	for _, o := range tOpts.OrderBy {
		result.OrderBy = append(result.OrderBy, interfaces.OrderClause{
			Field:     o.Field,
			Direction: string(o.Direction),
		})
	}

	return result
}
