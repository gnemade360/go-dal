package repository

import "github.com/gnemade360/go-dal/dal/types"

// QueryOption configures query behavior
type QueryOption func(*types.QueryOptions)

// WithLimit sets the query limit
func WithLimit(limit int) QueryOption {
	return func(opts *types.QueryOptions) {
		opts.Limit = limit
	}
}

// WithOffset sets the query offset
func WithOffset(offset int) QueryOption {
	return func(opts *types.QueryOptions) {
		opts.Offset = offset
	}
}

// WithOrderBy adds an order by clause
func WithOrderBy(field string, direction types.OrderDirection) QueryOption {
	return func(opts *types.QueryOptions) {
		opts.OrderBy = append(opts.OrderBy, types.OrderClause{
			Field:     field,
			Direction: direction,
		})
	}
}

// WithPreload adds a relation to preload
func WithPreload(relation string) QueryOption {
	return func(opts *types.QueryOptions) {
		opts.Preload = append(opts.Preload, relation)
	}
}

// WithLock sets the lock mode
func WithLock(mode types.LockMode) QueryOption {
	return func(opts *types.QueryOptions) {
		opts.LockMode = mode
	}
}

// WithForUpdate sets FOR UPDATE lock
func WithForUpdate() QueryOption {
	return func(opts *types.QueryOptions) {
		opts.ForUpdate = true
		opts.LockMode = types.LockModeForUpdate
	}
}

// WithDistinct enables distinct results
func WithDistinct() QueryOption {
	return func(opts *types.QueryOptions) {
		opts.Distinct = true
	}
}

// WithPagination is a convenience function for limit and offset
func WithPagination(page, pageSize int) QueryOption {
	return func(opts *types.QueryOptions) {
		if page > 0 && pageSize > 0 {
			opts.Limit = pageSize
			opts.Offset = (page - 1) * pageSize
		}
	}
}