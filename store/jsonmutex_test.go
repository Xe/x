package store

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestJSONMutexDBSetGet(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tmpDir := t.TempDir()

	store, err := NewJSONMutexDB(tmpDir)
	if err != nil {
		t.Fatalf("NewJSONMutexDB() error = %v", err)
	}

	err = store.Set(ctx, "foo", []byte("hello world"))
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	data, err := store.Get(ctx, "foo")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if string(data) != "hello world" {
		t.Errorf("Get() = %q, want %q", string(data), "hello world")
	}
}

func TestJSONMutexDBGetNotFound(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tmpDir := t.TempDir()

	store, err := NewJSONMutexDB(tmpDir)
	if err != nil {
		t.Fatalf("NewJSONMutexDB() error = %v", err)
	}

	_, err = store.Get(ctx, "nonexistent")
	if err == nil {
		t.Fatal("Get() expected error, got nil")
	}

	if !IsNotFound(err) {
		t.Errorf("Get() error = %v, want ErrNotFound", err)
	}
}

func TestJSONMutexDBDelete(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tmpDir := t.TempDir()

	store, err := NewJSONMutexDB(tmpDir)
	if err != nil {
		t.Fatalf("NewJSONMutexDB() error = %v", err)
	}

	err = store.Set(ctx, "foo", []byte("data"))
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	err = store.Delete(ctx, "foo")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err = store.Get(ctx, "foo")
	if err == nil {
		t.Fatal("Get() expected error after Delete, got nil")
	}

	if !IsNotFound(err) {
		t.Errorf("Get() error = %v, want ErrNotFound", err)
	}
}

func TestJSONMutexDBDeleteNotFound(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tmpDir := t.TempDir()

	store, err := NewJSONMutexDB(tmpDir)
	if err != nil {
		t.Fatalf("NewJSONMutexDB() error = %v", err)
	}

	err = store.Delete(ctx, "nonexistent")
	if err == nil {
		t.Fatal("Delete() expected error, got nil")
	}

	if !IsNotFound(err) {
		t.Errorf("Delete() error = %v, want ErrNotFound", err)
	}
}

func TestJSONMutexDBExists(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tmpDir := t.TempDir()

	store, err := NewJSONMutexDB(tmpDir)
	if err != nil {
		t.Fatalf("NewJSONMutexDB() error = %v", err)
	}

	err = store.Exists(ctx, "foo")
	if err == nil {
		t.Fatal("Exists() expected error for nonexistent key, got nil")
	}

	if !IsNotFound(err) {
		t.Errorf("Exists() error = %v, want ErrNotFound", err)
	}

	err = store.Set(ctx, "foo", []byte("data"))
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	err = store.Exists(ctx, "foo")
	if err != nil {
		t.Errorf("Exists() error = %v", err)
	}
}

func TestJSONMutexDBList(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tmpDir := t.TempDir()

	store, err := NewJSONMutexDB(tmpDir)
	if err != nil {
		t.Fatalf("NewJSONMutexDB() error = %v", err)
	}

	keys := []string{"foo/a", "foo/b", "bar/c"}
	for _, key := range keys {
		if err := store.Set(ctx, key, []byte("data")); err != nil {
			t.Fatalf("Set(%q) error = %v", key, err)
		}
	}

	list, err := store.List(ctx, "foo/")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(list) != 2 {
		t.Errorf("List() returned %d items, want 2", len(list))
	}

	for _, key := range list {
		if !strings.HasPrefix(key, "foo/") {
			t.Errorf("List() returned key %q without prefix foo/", key)
		}
	}
}

func TestJSONMutexDBListEmpty(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tmpDir := t.TempDir()

	store, err := NewJSONMutexDB(tmpDir)
	if err != nil {
		t.Fatalf("NewJSONMutexDB() error = %v", err)
	}

	keys := []string{"foo/a", "foo/b", "bar/c"}
	for _, key := range keys {
		if err := store.Set(ctx, key, []byte("data")); err != nil {
			t.Fatalf("Set(%q) error = %v", key, err)
		}
	}

	list, err := store.List(ctx, "")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(list) != 3 {
		t.Errorf("List() returned %d items, want 3", len(list))
	}
}

func TestJSONMutexDBEmptyKey(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tmpDir := t.TempDir()

	store, err := NewJSONMutexDB(tmpDir)
	if err != nil {
		t.Fatalf("NewJSONMutexDB() error = %v", err)
	}

	err = store.Set(ctx, "", []byte("data"))
	if err == nil {
		t.Fatal("Set() expected error for empty key, got nil")
	}

	if !IsBadConfig(err) {
		t.Errorf("Set() error = %v, want ErrBadConfig", err)
	}
}

func TestJSONMutexDBIndexPersistence(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tmpDir := t.TempDir()

	store1, err := NewJSONMutexDB(tmpDir)
	if err != nil {
		t.Fatalf("NewJSONMutexDB() error = %v", err)
	}

	err = store1.Set(ctx, "foo", []byte("data"))
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	store2, err := NewJSONMutexDB(tmpDir)
	if err != nil {
		t.Fatalf("NewJSONMutexDB() error = %v", err)
	}

	data, err := store2.Get(ctx, "foo")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if string(data) != "data" {
		t.Errorf("Get() = %q, want %q", string(data), "data")
	}
}

func TestJSONMutexDBConcurrent(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tmpDir := t.TempDir()

	store, err := NewJSONMutexDB(tmpDir)
	if err != nil {
		t.Fatalf("NewJSONMutexDB() error = %v", err)
	}

	var wg sync.WaitGroup
	numGoroutines := 10
	numOps := 50

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				key := string(rune('a' + n)) + "/key" + string(rune('0'+j))
				err := store.Set(ctx, key, []byte("data"))
				if err != nil {
					t.Errorf("Set(%q) error = %v", key, err)
				}

				_, err = store.Get(ctx, key)
				if err != nil {
					t.Errorf("Get(%q) error = %v", key, err)
				}
			}
		}(i)
	}

	wg.Wait()

	list, err := store.List(ctx, "")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	expected := numGoroutines * numOps
	if len(list) != expected {
		t.Errorf("List() returned %d items, want %d", len(list), expected)
	}
}

func TestJSONMutexDBCorruptedIndex(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	indexFile := filepath.Join(tmpDir, "index.json")
	err := os.WriteFile(indexFile, []byte("not valid json"), 0644)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err = NewJSONMutexDB(tmpDir)
	if err == nil {
		t.Fatal("NewJSONMutexDB() expected error for corrupted index, got nil")
	}

	if !IsBadConfig(err) {
		t.Errorf("NewJSONMutexDB() error = %v, want ErrBadConfig", err)
	}
}

func TestJSONMutexDBDataInDataDir(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tmpDir := t.TempDir()

	store, err := NewJSONMutexDB(tmpDir)
	if err != nil {
		t.Fatalf("NewJSONMutexDB() error = %v", err)
	}

	err = store.Set(ctx, "foo", []byte("data"))
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	dataDir := filepath.Join(tmpDir, "data")
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}

	if len(entries) == 0 {
		t.Fatal("No data files found in data directory")
	}

	for _, entry := range entries {
		content, err := os.ReadFile(filepath.Join(dataDir, entry.Name()))
		if err != nil {
			t.Fatalf("ReadFile() error = %v", err)
		}
		if string(content) == "data" {
			return
		}
	}

	t.Fatal("Data file not found in data directory")
}
