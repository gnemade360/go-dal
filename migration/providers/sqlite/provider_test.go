package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func newConnectedProvider(t *testing.T, dsn string) *Provider {
	t.Helper()
	p := NewProvider(dsn, "")
	if err := p.Connect(context.Background()); err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { p.Disconnect(context.Background()) })
	return p
}

func TestAcquireLock_ExcludesSecondMigrator(t *testing.T) {
	dsn := filepath.Join(t.TempDir(), "migrate.db")
	first := newConnectedProvider(t, dsn)
	second := newConnectedProvider(t, dsn)
	ctx := context.Background()

	if err := first.AcquireLock(ctx, time.Second); err != nil {
		t.Fatalf("first acquire: %v", err)
	}

	if err := second.AcquireLock(ctx, 300*time.Millisecond); err == nil {
		t.Fatal("second migrator acquired the lock while the first still holds it")
	}

	if err := first.ReleaseLock(ctx); err != nil {
		t.Fatalf("first release: %v", err)
	}
	if err := second.AcquireLock(ctx, time.Second); err != nil {
		t.Fatalf("second acquire after release: %v", err)
	}
	if err := second.ReleaseLock(ctx); err != nil {
		t.Fatalf("second release: %v", err)
	}
}

func TestAcquireLock_WaitsForRelease(t *testing.T) {
	dsn := filepath.Join(t.TempDir(), "migrate.db")
	first := newConnectedProvider(t, dsn)
	second := newConnectedProvider(t, dsn)
	ctx := context.Background()

	if err := first.AcquireLock(ctx, time.Second); err != nil {
		t.Fatalf("first acquire: %v", err)
	}
	go func() {
		time.Sleep(200 * time.Millisecond)
		first.ReleaseLock(ctx)
	}()

	if err := second.AcquireLock(ctx, 3*time.Second); err != nil {
		t.Fatalf("second acquire should succeed once the first releases: %v", err)
	}
	second.ReleaseLock(ctx)
}

func TestAcquireLock_RespectsContextCancel(t *testing.T) {
	dsn := filepath.Join(t.TempDir(), "migrate.db")
	first := newConnectedProvider(t, dsn)
	second := newConnectedProvider(t, dsn)

	if err := first.AcquireLock(context.Background(), time.Second); err != nil {
		t.Fatalf("first acquire: %v", err)
	}
	defer first.ReleaseLock(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	start := time.Now()
	if err := second.AcquireLock(ctx, 10*time.Second); err == nil {
		t.Fatal("expected acquire to fail when context is cancelled")
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("acquire ignored context cancellation, took %v", elapsed)
	}
}

func TestReleaseLock_OnlyReleasesOwnLock(t *testing.T) {
	dsn := filepath.Join(t.TempDir(), "migrate.db")
	first := newConnectedProvider(t, dsn)
	second := newConnectedProvider(t, dsn)
	ctx := context.Background()

	if err := first.AcquireLock(ctx, time.Second); err != nil {
		t.Fatalf("first acquire: %v", err)
	}
	defer first.ReleaseLock(ctx)

	second.ReleaseLock(ctx)
	if err := second.AcquireLock(ctx, 300*time.Millisecond); err == nil {
		t.Fatal("a non-holder's ReleaseLock freed someone else's lock")
	}
}
