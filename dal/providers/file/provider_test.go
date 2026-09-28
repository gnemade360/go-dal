package file

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/gnemade360/go-dal/dal/interfaces"
)

func setupTestProvider(t *testing.T) *FileProvider {
	t.Helper()
	dir := t.TempDir()
	return New(WithDir(dir))
}

func TestCreate_And_FindByID(t *testing.T) {
	p := setupTestProvider(t)
	ctx := context.Background()

	data := map[string]interface{}{
		"id":   "item-1",
		"name": "Test Item",
		"age":  float64(25),
	}

	if err := p.Create(ctx, "items", "item-1", data); err != nil {
		t.Fatalf("create: %v", err)
	}

	var result map[string]interface{}
	if err := p.FindByID(ctx, "items", "item-1", &result); err != nil {
		t.Fatalf("find: %v", err)
	}

	if result["name"] != "Test Item" {
		t.Fatalf("expected name 'Test Item', got %v", result["name"])
	}
	if result["age"] != float64(25) {
		t.Fatalf("expected age 25, got %v", result["age"])
	}
}

func TestCreate_Duplicate(t *testing.T) {
	p := setupTestProvider(t)
	ctx := context.Background()

	data := map[string]interface{}{"id": "dup-1", "name": "first"}
	if err := p.Create(ctx, "items", "dup-1", data); err != nil {
		t.Fatalf("first create: %v", err)
	}

	err := p.Create(ctx, "items", "dup-1", data)
	if err == nil {
		t.Fatal("expected duplicate error")
	}
}

func TestFindByID_NotFound(t *testing.T) {
	p := setupTestProvider(t)
	ctx := context.Background()

	var result map[string]interface{}
	err := p.FindByID(ctx, "items", "nonexistent", &result)
	if err == nil {
		t.Fatal("expected not found error")
	}
}

func TestUpdate(t *testing.T) {
	p := setupTestProvider(t)
	ctx := context.Background()

	data := map[string]interface{}{"id": "u-1", "name": "original"}
	if err := p.Create(ctx, "items", "u-1", data); err != nil {
		t.Fatalf("create: %v", err)
	}

	updated := map[string]interface{}{"id": "u-1", "name": "updated"}
	if err := p.Update(ctx, "items", "u-1", updated); err != nil {
		t.Fatalf("update: %v", err)
	}

	var result map[string]interface{}
	if err := p.FindByID(ctx, "items", "u-1", &result); err != nil {
		t.Fatalf("find: %v", err)
	}
	if result["name"] != "updated" {
		t.Fatalf("expected 'updated', got %v", result["name"])
	}
}

func TestUpdate_NotFound(t *testing.T) {
	p := setupTestProvider(t)
	ctx := context.Background()

	err := p.Update(ctx, "items", "missing", map[string]interface{}{"id": "missing"})
	if err == nil {
		t.Fatal("expected not found error")
	}
}

func TestDelete(t *testing.T) {
	p := setupTestProvider(t)
	ctx := context.Background()

	data := map[string]interface{}{"id": "d-1", "name": "deleteme"}
	if err := p.Create(ctx, "items", "d-1", data); err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := p.Delete(ctx, "items", "d-1"); err != nil {
		t.Fatalf("delete: %v", err)
	}

	var result map[string]interface{}
	err := p.FindByID(ctx, "items", "d-1", &result)
	if err == nil {
		t.Fatal("expected not found after delete")
	}
}

func TestDelete_NotFound(t *testing.T) {
	p := setupTestProvider(t)
	ctx := context.Background()

	err := p.Delete(ctx, "items", "nonexistent")
	if err == nil {
		t.Fatal("expected not found error")
	}
}

func TestFindAll(t *testing.T) {
	p := setupTestProvider(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		data := map[string]interface{}{
			"id":   fmt.Sprintf("item-%d", i),
			"name": fmt.Sprintf("Item %d", i),
			"seq":  float64(i),
		}
		if err := p.Create(ctx, "items", data["id"].(string), data); err != nil {
			t.Fatalf("create %d: %v", i, err)
		}
	}

	var results []map[string]interface{}
	if err := p.FindAll(ctx, "items", &results, interfaces.QueryOptions{}); err != nil {
		t.Fatalf("find all: %v", err)
	}

	if len(results) != 5 {
		t.Fatalf("expected 5 items, got %d", len(results))
	}
}

func TestFindAll_WithLimit(t *testing.T) {
	p := setupTestProvider(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		data := map[string]interface{}{
			"id":   fmt.Sprintf("item-%d", i),
			"name": fmt.Sprintf("Item %d", i),
		}
		p.Create(ctx, "items", data["id"].(string), data)
	}

	var results []map[string]interface{}
	if err := p.FindAll(ctx, "items", &results, interfaces.QueryOptions{Limit: 3}); err != nil {
		t.Fatalf("find: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 items, got %d", len(results))
	}
}

func TestFindAll_WithOffset(t *testing.T) {
	p := setupTestProvider(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		data := map[string]interface{}{
			"id":   fmt.Sprintf("item-%d", i),
			"name": fmt.Sprintf("Item %d", i),
		}
		p.Create(ctx, "items", data["id"].(string), data)
	}

	var results []map[string]interface{}
	if err := p.FindAll(ctx, "items", &results, interfaces.QueryOptions{Offset: 3}); err != nil {
		t.Fatalf("find: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 items, got %d", len(results))
	}
}

func TestFindByFilter(t *testing.T) {
	p := setupTestProvider(t)
	ctx := context.Background()

	p.Create(ctx, "items", "a", map[string]interface{}{"id": "a", "status": "active", "name": "Alpha"})
	p.Create(ctx, "items", "b", map[string]interface{}{"id": "b", "status": "inactive", "name": "Beta"})
	p.Create(ctx, "items", "c", map[string]interface{}{"id": "c", "status": "active", "name": "Charlie"})

	var results []map[string]interface{}
	filter := interfaces.Filter{Field: "status", Op: "=", Value: "active"}
	if err := p.FindByFilter(ctx, "items", &results, filter, interfaces.QueryOptions{}); err != nil {
		t.Fatalf("filter: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 active items, got %d", len(results))
	}
}

func TestFindByFilter_Like(t *testing.T) {
	p := setupTestProvider(t)
	ctx := context.Background()

	p.Create(ctx, "items", "1", map[string]interface{}{"id": "1", "name": "hello world"})
	p.Create(ctx, "items", "2", map[string]interface{}{"id": "2", "name": "goodbye world"})
	p.Create(ctx, "items", "3", map[string]interface{}{"id": "3", "name": "hello there"})

	var results []map[string]interface{}
	filter := interfaces.Filter{Field: "name", Op: "LIKE", Value: "hello%"}
	if err := p.FindByFilter(ctx, "items", &results, filter, interfaces.QueryOptions{}); err != nil {
		t.Fatalf("filter: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results for 'hello%%', got %d", len(results))
	}
}

func TestCount(t *testing.T) {
	p := setupTestProvider(t)
	ctx := context.Background()

	p.Create(ctx, "items", "a", map[string]interface{}{"id": "a", "type": "x"})
	p.Create(ctx, "items", "b", map[string]interface{}{"id": "b", "type": "y"})
	p.Create(ctx, "items", "c", map[string]interface{}{"id": "c", "type": "x"})

	count, err := p.Count(ctx, "items", interfaces.Filter{})
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected 3, got %d", count)
	}

	count, err = p.Count(ctx, "items", interfaces.Filter{Field: "type", Op: "=", Value: "x"})
	if err != nil {
		t.Fatalf("count filtered: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2, got %d", count)
	}
}

func TestCount_EmptyTable(t *testing.T) {
	p := setupTestProvider(t)
	ctx := context.Background()

	count, err := p.Count(ctx, "empty_table", interfaces.Filter{})
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0, got %d", count)
	}
}

func TestTransaction_Commit(t *testing.T) {
	p := setupTestProvider(t)
	ctx := context.Background()

	tx, err := p.BeginTx(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}

	tx.Create(ctx, "items", "tx-1", map[string]interface{}{"id": "tx-1", "name": "txn item"})
	tx.Create(ctx, "items", "tx-2", map[string]interface{}{"id": "tx-2", "name": "txn item 2"})

	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}

	var result map[string]interface{}
	if err := p.FindByID(ctx, "items", "tx-1", &result); err != nil {
		t.Fatalf("find after commit: %v", err)
	}
}

func TestTransaction_Rollback(t *testing.T) {
	p := setupTestProvider(t)
	ctx := context.Background()

	tx, err := p.BeginTx(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}

	tx.Create(ctx, "items", "rb-1", map[string]interface{}{"id": "rb-1", "name": "rollback item"})

	if err := tx.Rollback(); err != nil {
		t.Fatalf("rollback: %v", err)
	}

	var result map[string]interface{}
	err = p.FindByID(ctx, "items", "rb-1", &result)
	if err == nil {
		t.Fatal("expected not found after rollback")
	}
}

func TestFileCreation(t *testing.T) {
	dir := t.TempDir()
	p := New(WithDir(dir))
	ctx := context.Background()

	data := map[string]interface{}{"id": "fs-1", "key": "value"}
	if err := p.Create(ctx, "things", "fs-1", data); err != nil {
		t.Fatalf("create: %v", err)
	}

	path := filepath.Join(dir, "things", "fs-1.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file at %s: %v", path, err)
	}
}

func TestFindAll_OrderBy(t *testing.T) {
	p := setupTestProvider(t)
	ctx := context.Background()

	p.Create(ctx, "items", "c", map[string]interface{}{"id": "c", "name": "Charlie"})
	p.Create(ctx, "items", "a", map[string]interface{}{"id": "a", "name": "Alpha"})
	p.Create(ctx, "items", "b", map[string]interface{}{"id": "b", "name": "Beta"})

	var results []map[string]interface{}
	opts := interfaces.QueryOptions{
		OrderBy: []interfaces.OrderClause{{Field: "name", Direction: "ASC"}},
	}
	if err := p.FindAll(ctx, "items", &results, opts); err != nil {
		t.Fatalf("find: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3, got %d", len(results))
	}
	if results[0]["name"] != "Alpha" {
		t.Fatalf("expected first to be Alpha, got %v", results[0]["name"])
	}
	if results[2]["name"] != "Charlie" {
		t.Fatalf("expected last to be Charlie, got %v", results[2]["name"])
	}
}
