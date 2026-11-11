package sqlite

import (
	"context"
	"path/filepath"
	"testing"
)

type pragmas struct {
	journalMode string
	busyTimeout int
	foreignKeys int
}

func connectAndReadPragmas(t *testing.T, dsn string) pragmas {
	t.Helper()
	provider := New()
	provider.SetDSN(dsn)
	ctx := context.Background()
	if err := provider.Connect(ctx); err != nil {
		t.Fatalf("connect %q: %v", dsn, err)
	}
	defer provider.Close()

	var p pragmas
	if err := provider.QueryRow(ctx, "PRAGMA journal_mode").Scan(&p.journalMode); err != nil {
		t.Fatalf("journal_mode: %v", err)
	}
	if err := provider.QueryRow(ctx, "PRAGMA busy_timeout").Scan(&p.busyTimeout); err != nil {
		t.Fatalf("busy_timeout: %v", err)
	}
	if err := provider.QueryRow(ctx, "PRAGMA foreign_keys").Scan(&p.foreignKeys); err != nil {
		t.Fatalf("foreign_keys: %v", err)
	}
	return p
}

func TestConnect_FileDatabaseDefaults(t *testing.T) {
	got := connectAndReadPragmas(t, filepath.Join(t.TempDir(), "app.db"))

	if got.journalMode != "wal" {
		t.Errorf("journal_mode: expected wal, got %q", got.journalMode)
	}
	if got.busyTimeout != defaultBusyTimeoutMs {
		t.Errorf("busy_timeout: expected %d, got %d", defaultBusyTimeoutMs, got.busyTimeout)
	}
	if got.foreignKeys != 1 {
		t.Errorf("foreign_keys: expected 1, got %d", got.foreignKeys)
	}
}

func TestConnect_FileURIWithExistingParams(t *testing.T) {
	dsn := "file:" + filepath.Join(t.TempDir(), "app.db") + "?cache=private"
	got := connectAndReadPragmas(t, dsn)

	if got.journalMode != "wal" || got.foreignKeys != 1 {
		t.Errorf("expected wal + foreign keys, got %+v", got)
	}
}

func TestConnect_InMemorySkipsWAL(t *testing.T) {
	got := connectAndReadPragmas(t, ":memory:")

	if got.journalMode != "memory" {
		t.Errorf("journal_mode: expected memory, got %q", got.journalMode)
	}
	if got.foreignKeys != 1 {
		t.Errorf("foreign_keys: expected 1, got %d", got.foreignKeys)
	}
}

func TestConnect_ExplicitDSNParamsWin(t *testing.T) {
	dsn := filepath.Join(t.TempDir(), "app.db") + "?_journal_mode=DELETE&_busy_timeout=1234&_fk=0"
	got := connectAndReadPragmas(t, dsn)

	want := pragmas{journalMode: "delete", busyTimeout: 1234, foreignKeys: 0}
	if got != want {
		t.Errorf("expected %+v, got %+v", want, got)
	}
}
