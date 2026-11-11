package repository

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	dalerrors "github.com/gnemade360/go-dal/dal/errors"
	"github.com/gnemade360/go-dal/dal/interfaces"
	"github.com/gnemade360/go-dal/dal/providers/sqlite"
	_ "github.com/mattn/go-sqlite3"
)

const txTestTimeout = 2 * time.Second

var errAbort = errors.New("abort")

type account struct {
	ID   string `db:"id" json:"id"`
	Name string `db:"name" json:"name"`
}

func (a account) GetID() string     { return a.ID }
func (a account) SetID(id string)   {}
func (a account) TableName() string { return "accounts" }

func newSQLiteProvider(t *testing.T) *sqlite.SQLiteProvider {
	t.Helper()
	provider := sqlite.New()
	provider.SetDSN(filepath.Join(t.TempDir(), "repo.db"))
	if err := provider.Connect(context.Background()); err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { provider.Close() })

	if _, err := provider.Exec(context.Background(), "CREATE TABLE accounts (id TEXT PRIMARY KEY, name TEXT)"); err != nil {
		t.Fatalf("create table: %v", err)
	}
	return provider
}

// runBounded fails the test instead of hanging when a write inside a
// transaction waits for the single pooled connection held by that transaction.
func runBounded(t *testing.T, fn func(ctx context.Context) error) error {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), txTestTimeout)
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- fn(ctx) }()

	select {
	case err := <-done:
		if ctx.Err() != nil {
			t.Fatalf("blocked until context deadline (operation bypassed the transaction): %v", err)
		}
		return err
	case <-time.After(txTestTimeout + time.Second):
		t.Fatal("hung past context deadline")
		return nil
	}
}

func countAccounts(t *testing.T, provider *sqlite.SQLiteProvider) int {
	t.Helper()
	var n int
	if err := provider.QueryRow(context.Background(), "SELECT COUNT(*) FROM accounts").Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	return n
}

func TestBaseRepository_TransactionRollbackDiscardsWrites(t *testing.T) {
	provider := newSQLiteProvider(t)
	repo := NewBaseRepository[account](provider, nil)

	err := runBounded(t, func(ctx context.Context) error {
		return repo.Transaction(ctx, func(ctx context.Context, txRepo Repository[account]) error {
			if err := txRepo.Create(ctx, &account{ID: "a", Name: "Alpha"}); err != nil {
				return err
			}
			if _, err := txRepo.FindByID(ctx, "a"); err != nil {
				return err
			}
			return errAbort
		})
	})
	if !errors.Is(err, errAbort) {
		t.Fatalf("expected errAbort, got %v", err)
	}
	if n := countAccounts(t, provider); n != 0 {
		t.Fatalf("expected rollback to discard write, found %d rows", n)
	}
}

func TestBaseRepository_TransactionCommitKeepsWrites(t *testing.T) {
	provider := newSQLiteProvider(t)
	repo := NewBaseRepository[account](provider, nil)

	err := runBounded(t, func(ctx context.Context) error {
		return repo.Transaction(ctx, func(ctx context.Context, txRepo Repository[account]) error {
			if err := txRepo.CreateBatch(ctx, []*account{{ID: "a", Name: "Alpha"}, {ID: "b", Name: "Beta"}}); err != nil {
				return err
			}
			if err := txRepo.Update(ctx, &account{ID: "a", Name: "Alpha2"}); err != nil {
				return err
			}
			if err := txRepo.Delete(ctx, "b"); err != nil {
				return err
			}
			all, err := txRepo.FindAll(ctx)
			if err != nil {
				return err
			}
			if len(all) != 1 {
				return errors.New("expected one row visible inside transaction")
			}
			n, err := txRepo.Count(ctx, nil)
			if err != nil {
				return err
			}
			if n != 1 {
				return errors.New("expected count 1 inside transaction")
			}
			return nil
		})
	})
	if err != nil {
		t.Fatalf("transaction: %v", err)
	}

	got, err := repo.FindByID(context.Background(), "a")
	if err != nil || got.Name != "Alpha2" {
		t.Fatalf("expected committed Alpha2, got %v (%+v)", err, got)
	}
	if n := countAccounts(t, provider); n != 1 {
		t.Fatalf("expected 1 row after commit, found %d", n)
	}
}

type ctxTx struct {
	context.Context
	interfaces.Transaction
}

func TestBaseRepository_WithTransactionUsesTx(t *testing.T) {
	provider := newSQLiteProvider(t)
	repo := NewBaseRepository[account](provider, nil)
	ctx := context.Background()

	tx, err := provider.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	txRepo := repo.WithTransaction(ctxTx{Context: ctx, Transaction: tx})

	err = runBounded(t, func(ctx context.Context) error {
		return txRepo.Create(ctx, &account{ID: "a", Name: "Alpha"})
	})
	if err != nil {
		tx.Rollback()
		t.Fatalf("create in tx: %v", err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatalf("rollback: %v", err)
	}
	if n := countAccounts(t, provider); n != 0 {
		t.Fatalf("expected rollback to discard write, found %d rows", n)
	}
}

func TestStoreRepository_TransactionRollbackDiscardsWrites(t *testing.T) {
	provider := newSQLiteProvider(t)
	repo := NewStoreRepository[account](sqlite.NewStoreAdapter(provider))

	err := runBounded(t, func(ctx context.Context) error {
		return repo.Transaction(ctx, func(ctx context.Context, txRepo Repository[account]) error {
			if err := txRepo.Create(ctx, &account{ID: "a", Name: "Alpha"}); err != nil {
				return err
			}
			if _, err := txRepo.FindByID(ctx, "a"); err != nil {
				return err
			}
			return errAbort
		})
	})
	if !errors.Is(err, errAbort) {
		t.Fatalf("expected errAbort, got %v", err)
	}
	if _, err := repo.FindByID(context.Background(), "a"); !dalerrors.IsNotFound(err) {
		t.Fatalf("expected rollback to discard write, got %v", err)
	}
}

func TestStoreRepository_TransactionCommitKeepsWrites(t *testing.T) {
	provider := newSQLiteProvider(t)
	repo := NewStoreRepository[account](sqlite.NewStoreAdapter(provider))

	err := runBounded(t, func(ctx context.Context) error {
		return repo.Transaction(ctx, func(ctx context.Context, txRepo Repository[account]) error {
			return txRepo.Create(ctx, &account{ID: "a", Name: "Alpha"})
		})
	})
	if err != nil {
		t.Fatalf("transaction: %v", err)
	}

	got, err := repo.FindByID(context.Background(), "a")
	if err != nil || got.Name != "Alpha" {
		t.Fatalf("expected committed Alpha, got %v (%+v)", err, got)
	}
}

type ctxStoreTx struct {
	context.Context
	interfaces.StoreTx
}

func TestStoreRepository_WithTransactionUsesTx(t *testing.T) {
	provider := newSQLiteProvider(t)
	store := sqlite.NewStoreAdapter(provider)
	repo := NewStoreRepository[account](store)
	ctx := context.Background()

	tx, err := store.BeginTx(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	txRepo := repo.WithTransaction(ctxStoreTx{Context: ctx, StoreTx: tx})

	err = runBounded(t, func(ctx context.Context) error {
		return txRepo.Create(ctx, &account{ID: "a", Name: "Alpha"})
	})
	if err != nil {
		tx.Rollback()
		t.Fatalf("create in tx: %v", err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatalf("rollback: %v", err)
	}
	if n := countAccounts(t, provider); n != 0 {
		t.Fatalf("expected rollback to discard write, found %d rows", n)
	}
}
