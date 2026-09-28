package repository

import (
	"context"
	"testing"

	"github.com/gnemade360/go-dal/dal/providers/file"
)

type TestEntity struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

func (e TestEntity) GetID() string     { return e.ID }
func (e TestEntity) SetID(id string)   { e.ID = id }
func (e TestEntity) TableName() string { return "test_entities" }

func TestStoreRepository_CRUD(t *testing.T) {
	dir := t.TempDir()
	store := file.New(file.WithDir(dir))

	repo := NewStoreRepository[TestEntity](store)
	ctx := context.Background()

	entity := &TestEntity{ID: "e-1", Name: "First", Status: "active"}
	if err := repo.Create(ctx, entity); err != nil {
		t.Fatalf("create: %v", err)
	}

	found, err := repo.FindByID(ctx, "e-1")
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if found.Name != "First" {
		t.Fatalf("expected 'First', got %q", found.Name)
	}

	found.Name = "Updated"
	if err := repo.Update(ctx, found); err != nil {
		t.Fatalf("update: %v", err)
	}

	found2, err := repo.FindByID(ctx, "e-1")
	if err != nil {
		t.Fatalf("find after update: %v", err)
	}
	if found2.Name != "Updated" {
		t.Fatalf("expected 'Updated', got %q", found2.Name)
	}

	if err := repo.Delete(ctx, "e-1"); err != nil {
		t.Fatalf("delete: %v", err)
	}

	_, err = repo.FindByID(ctx, "e-1")
	if err == nil {
		t.Fatal("expected not found after delete")
	}
}

func TestStoreRepository_FindAll(t *testing.T) {
	dir := t.TempDir()
	store := file.New(file.WithDir(dir))

	repo := NewStoreRepository[TestEntity](store)
	ctx := context.Background()

	repo.Create(ctx, &TestEntity{ID: "1", Name: "Alpha", Status: "active"})
	repo.Create(ctx, &TestEntity{ID: "2", Name: "Beta", Status: "inactive"})
	repo.Create(ctx, &TestEntity{ID: "3", Name: "Charlie", Status: "active"})

	all, err := repo.FindAll(ctx)
	if err != nil {
		t.Fatalf("find all: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("expected 3, got %d", len(all))
	}
}

func TestStoreRepository_FindWithOptions(t *testing.T) {
	dir := t.TempDir()
	store := file.New(file.WithDir(dir))

	repo := NewStoreRepository[TestEntity](store)
	ctx := context.Background()

	repo.Create(ctx, &TestEntity{ID: "1", Name: "Alpha", Status: "active"})
	repo.Create(ctx, &TestEntity{ID: "2", Name: "Beta", Status: "inactive"})
	repo.Create(ctx, &TestEntity{ID: "3", Name: "Charlie", Status: "active"})

	spec := NewFieldSpec("status", OpEqual, "active")
	results, err := repo.FindWithOptions(ctx, spec)
	if err != nil {
		t.Fatalf("find with options: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 active, got %d", len(results))
	}
}

func TestStoreRepository_Count(t *testing.T) {
	dir := t.TempDir()
	store := file.New(file.WithDir(dir))

	repo := NewStoreRepository[TestEntity](store)
	ctx := context.Background()

	repo.Create(ctx, &TestEntity{ID: "1", Name: "A", Status: "x"})
	repo.Create(ctx, &TestEntity{ID: "2", Name: "B", Status: "y"})
	repo.Create(ctx, &TestEntity{ID: "3", Name: "C", Status: "x"})

	count, err := repo.Count(ctx, nil)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected 3, got %d", count)
	}

	spec := NewFieldSpec("status", OpEqual, "x")
	count, err = repo.Count(ctx, spec)
	if err != nil {
		t.Fatalf("count filtered: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2, got %d", count)
	}
}

func TestStoreRepository_Exists(t *testing.T) {
	dir := t.TempDir()
	store := file.New(file.WithDir(dir))

	repo := NewStoreRepository[TestEntity](store)
	ctx := context.Background()

	repo.Create(ctx, &TestEntity{ID: "1", Name: "A", Status: "active"})

	spec := NewFieldSpec("status", OpEqual, "active")
	exists, err := repo.Exists(ctx, spec)
	if err != nil {
		t.Fatalf("exists: %v", err)
	}
	if !exists {
		t.Fatal("expected exists=true")
	}

	spec2 := NewFieldSpec("status", OpEqual, "missing")
	exists2, err := repo.Exists(ctx, spec2)
	if err != nil {
		t.Fatalf("exists: %v", err)
	}
	if exists2 {
		t.Fatal("expected exists=false")
	}
}

func TestStoreRepository_Batch(t *testing.T) {
	dir := t.TempDir()
	store := file.New(file.WithDir(dir))

	repo := NewStoreRepository[TestEntity](store)
	ctx := context.Background()

	entities := []*TestEntity{
		{ID: "b-1", Name: "Batch 1"},
		{ID: "b-2", Name: "Batch 2"},
		{ID: "b-3", Name: "Batch 3"},
	}

	if err := repo.CreateBatch(ctx, entities); err != nil {
		t.Fatalf("batch create: %v", err)
	}

	all, err := repo.FindAll(ctx)
	if err != nil {
		t.Fatalf("find all: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("expected 3, got %d", len(all))
	}

	if err := repo.DeleteBatch(ctx, []string{"b-1", "b-2", "b-3"}); err != nil {
		t.Fatalf("batch delete: %v", err)
	}

	all, _ = repo.FindAll(ctx)
	if len(all) != 0 {
		t.Fatalf("expected 0 after delete, got %d", len(all))
	}
}
