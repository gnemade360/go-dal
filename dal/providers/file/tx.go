package file

import (
	"context"

	"github.com/gnemade360/go-dal/dal/interfaces"
)

type txOp struct {
	action    string
	tableName string
	id        string
	data      map[string]interface{}
}

type fileTx struct {
	provider  *FileProvider
	ops       []txOp
	committed bool
}

func (t *fileTx) FindByID(ctx context.Context, tableName string, id string, dest interface{}) error {
	return t.provider.FindByID(ctx, tableName, id, dest)
}

func (t *fileTx) FindAll(ctx context.Context, tableName string, dest interface{}, opts interfaces.QueryOptions) error {
	return t.provider.FindAll(ctx, tableName, dest, opts)
}

func (t *fileTx) FindByFilter(ctx context.Context, tableName string, dest interface{}, filter interfaces.Filter, opts interfaces.QueryOptions) error {
	return t.provider.FindByFilter(ctx, tableName, dest, filter, opts)
}

func (t *fileTx) Count(ctx context.Context, tableName string, filter interfaces.Filter) (int64, error) {
	return t.provider.Count(ctx, tableName, filter)
}

func (t *fileTx) Create(ctx context.Context, tableName string, id string, data map[string]interface{}) error {
	t.ops = append(t.ops, txOp{action: "create", tableName: tableName, id: id, data: data})
	return nil
}

func (t *fileTx) Update(ctx context.Context, tableName string, id string, data map[string]interface{}) error {
	t.ops = append(t.ops, txOp{action: "update", tableName: tableName, id: id, data: data})
	return nil
}

func (t *fileTx) Delete(ctx context.Context, tableName string, id string) error {
	t.ops = append(t.ops, txOp{action: "delete", tableName: tableName, id: id})
	return nil
}

func (t *fileTx) BeginTx(ctx context.Context) (interfaces.StoreTx, error) {
	return t, nil
}

func (t *fileTx) Commit() error {
	ctx := context.Background()
	for _, op := range t.ops {
		var err error
		switch op.action {
		case "create":
			err = t.provider.Create(ctx, op.tableName, op.id, op.data)
		case "update":
			err = t.provider.Update(ctx, op.tableName, op.id, op.data)
		case "delete":
			err = t.provider.Delete(ctx, op.tableName, op.id)
		}
		if err != nil {
			return err
		}
	}
	t.committed = true
	return nil
}

func (t *fileTx) Rollback() error {
	t.ops = nil
	return nil
}
