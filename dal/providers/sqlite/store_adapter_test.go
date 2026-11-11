package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	dalerrors "github.com/gnemade360/go-dal/dal/errors"
	"github.com/gnemade360/go-dal/dal/interfaces"
	_ "github.com/mattn/go-sqlite3"
)

const opTimeout = 2 * time.Second

type item struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func newTestAdapter(t *testing.T) *StoreAdapter {
	t.Helper()
	provider := New()
	provider.SetDSN(filepath.Join(t.TempDir(), "store.db"))
	if err := provider.Connect(context.Background()); err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { provider.Close() })

	if _, err := provider.Exec(context.Background(), "CREATE TABLE items (id TEXT PRIMARY KEY, name TEXT)"); err != nil {
		t.Fatalf("create table: %v", err)
	}
	return NewStoreAdapter(provider)
}

// withTimeout fails the test instead of hanging when fn blocks on the
// single pooled connection. The context deadline unblocks database/sql
// waiters; the outer select guards against anything that ignores ctx.
func withTimeout(t *testing.T, name string, fn func(ctx context.Context) error) error {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), opTimeout)
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- fn(ctx) }()

	select {
	case err := <-done:
		if ctx.Err() != nil {
			t.Fatalf("%s: blocked until context deadline (deadlock on single connection): %v", name, err)
		}
		return err
	case <-time.After(opTimeout + time.Second):
		t.Fatalf("%s: hung past context deadline", name)
		return nil
	}
}

func TestStoreAdapter_FindByID_DoesNotDeadlock(t *testing.T) {
	adapter := newTestAdapter(t)
	ctx := context.Background()
	if err := adapter.Create(ctx, "items", "a", map[string]interface{}{"id": "a", "name": "Alpha"}); err != nil {
		t.Fatalf("create: %v", err)
	}

	var got item
	err := withTimeout(t, "FindByID", func(ctx context.Context) error {
		return adapter.FindByID(ctx, "items", "a", &got)
	})
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if got.Name != "Alpha" {
		t.Fatalf("expected Alpha, got %q", got.Name)
	}
}

func TestStoreAdapter_FindByID_NotFound(t *testing.T) {
	adapter := newTestAdapter(t)

	var got item
	err := withTimeout(t, "FindByID", func(ctx context.Context) error {
		return adapter.FindByID(ctx, "items", "missing", &got)
	})
	if !dalerrors.IsNotFound(err) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestStoreTx_OperationsRunInsideTransaction(t *testing.T) {
	adapter := newTestAdapter(t)
	ctx := context.Background()

	tx, err := adapter.BeginTx(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer tx.Rollback()

	err = withTimeout(t, "tx.Create", func(ctx context.Context) error {
		return tx.Create(ctx, "items", "a", map[string]interface{}{"id": "a", "name": "Alpha"})
	})
	if err != nil {
		t.Fatalf("create in tx: %v", err)
	}

	err = withTimeout(t, "tx.Update", func(ctx context.Context) error {
		return tx.Update(ctx, "items", "a", map[string]interface{}{"name": "Beta"})
	})
	if err != nil {
		t.Fatalf("update in tx: %v", err)
	}

	var got item
	err = withTimeout(t, "tx.FindByID", func(ctx context.Context) error {
		return tx.FindByID(ctx, "items", "a", &got)
	})
	if err != nil {
		t.Fatalf("find in tx: %v", err)
	}
	if got.Name != "Beta" {
		t.Fatalf("expected uncommitted Beta inside tx, got %q", got.Name)
	}

	var all []item
	err = withTimeout(t, "tx.FindAll", func(ctx context.Context) error {
		return tx.FindAll(ctx, "items", &all, interfaces.QueryOptions{})
	})
	if err != nil || len(all) != 1 {
		t.Fatalf("find all in tx: %v, %d rows", err, len(all))
	}

	var count int64
	err = withTimeout(t, "tx.Count", func(ctx context.Context) error {
		var cerr error
		count, cerr = tx.Count(ctx, "items", interfaces.Filter{})
		return cerr
	})
	if err != nil || count != 1 {
		t.Fatalf("count in tx: %v, %d", err, count)
	}

	err = withTimeout(t, "tx.Delete", func(ctx context.Context) error {
		return tx.Delete(ctx, "items", "a")
	})
	if err != nil {
		t.Fatalf("delete in tx: %v", err)
	}
}

func TestStoreTx_RollbackDiscardsWrites(t *testing.T) {
	adapter := newTestAdapter(t)
	ctx := context.Background()

	tx, err := adapter.BeginTx(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	err = withTimeout(t, "tx.Create", func(ctx context.Context) error {
		return tx.Create(ctx, "items", "a", map[string]interface{}{"id": "a", "name": "Alpha"})
	})
	if err != nil {
		tx.Rollback()
		t.Fatalf("create in tx: %v", err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatalf("rollback: %v", err)
	}

	var got item
	err = withTimeout(t, "FindByID", func(ctx context.Context) error {
		return adapter.FindByID(ctx, "items", "a", &got)
	})
	if !dalerrors.IsNotFound(err) {
		t.Fatalf("expected row to be rolled back, got %v (%+v)", err, got)
	}
}

func TestStoreTx_CommitKeepsWrites(t *testing.T) {
	adapter := newTestAdapter(t)
	ctx := context.Background()

	tx, err := adapter.BeginTx(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	err = withTimeout(t, "tx.Create", func(ctx context.Context) error {
		return tx.Create(ctx, "items", "a", map[string]interface{}{"id": "a", "name": "Alpha"})
	})
	if err != nil {
		tx.Rollback()
		t.Fatalf("create in tx: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}

	var got item
	err = withTimeout(t, "FindByID", func(ctx context.Context) error {
		return adapter.FindByID(ctx, "items", "a", &got)
	})
	if err != nil || got.Name != "Alpha" {
		t.Fatalf("expected committed row, got %v (%+v)", err, got)
	}
}
