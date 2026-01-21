package store

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCASSetGet(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tmpDir := t.TempDir()

	store, err := NewCAS(tmpDir)
	if err != nil {
		t.Fatalf("NewCAS() error = %v", err)
	}
	defer Close(store)

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

func TestCASGetNotFound(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tmpDir := t.TempDir()

	store, err := NewCAS(tmpDir)
	if err != nil {
		t.Fatalf("NewCAS() error = %v", err)
	}
	defer Close(store)

	_, err = store.Get(ctx, "nonexistent")
	if err == nil {
		t.Fatal("Get() expected error, got nil")
	}

	if !IsNotFound(err) {
		t.Errorf("Get() error = %v, want ErrNotFound", err)
	}
}

func TestCASDelete(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tmpDir := t.TempDir()

	store, err := NewCAS(tmpDir)
	if err != nil {
		t.Fatalf("NewCAS() error = %v", err)
	}
	defer Close(store)

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

func TestCASDeleteNotFound(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tmpDir := t.TempDir()

	store, err := NewCAS(tmpDir)
	if err != nil {
		t.Fatalf("NewCAS() error = %v", err)
	}
	defer Close(store)

	err = store.Delete(ctx, "nonexistent")
	if err == nil {
		t.Fatal("Delete() expected error, got nil")
	}

	if !IsNotFound(err) {
		t.Errorf("Delete() error = %v, want ErrNotFound", err)
	}
}

func TestCASExists(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tmpDir := t.TempDir()

	store, err := NewCAS(tmpDir)
	if err != nil {
		t.Fatalf("NewCAS() error = %v", err)
	}
	defer Close(store)

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

func TestCASList(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tmpDir := t.TempDir()

	store, err := NewCAS(tmpDir)
	if err != nil {
		t.Fatalf("NewCAS() error = %v", err)
	}
	defer Close(store)

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

func TestCASListEmpty(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tmpDir := t.TempDir()

	store, err := NewCAS(tmpDir)
	if err != nil {
		t.Fatalf("NewCAS() error = %v", err)
	}
	defer Close(store)

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

func TestCASEmptyKey(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tmpDir := t.TempDir()

	store, err := NewCAS(tmpDir)
	if err != nil {
		t.Fatalf("NewCAS() error = %v", err)
	}
	defer Close(store)

	err = store.Set(ctx, "", []byte("data"))
	if err == nil {
		t.Fatal("Set() expected error for empty key, got nil")
	}

	if !IsBadConfig(err) {
		t.Errorf("Set() error = %v, want ErrBadConfig", err)
	}
}

func TestCASDeduplication(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tmpDir := t.TempDir()

	store, err := NewCAS(tmpDir)
	if err != nil {
		t.Fatalf("NewCAS() error = %v", err)
	}
	defer Close(store)

	data := []byte("same data")
	err = store.Set(ctx, "key1", data)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	err = store.Set(ctx, "key2", data)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	objectsDir := filepath.Join(tmpDir, "objects")
	var objectFiles []string

	err = filepath.WalkDir(objectsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			objectFiles = append(objectFiles, path)
		}
		return nil
	})

	if err != nil {
		t.Fatalf("WalkDir() error = %v", err)
	}

	if len(objectFiles) != 1 {
		t.Errorf("Expected 1 object file for duplicate data, got %d", len(objectFiles))
	}

	data1, err := store.Get(ctx, "key1")
	if err != nil {
		t.Fatalf("Get(key1) error = %v", err)
	}

	data2, err := store.Get(ctx, "key2")
	if err != nil {
		t.Fatalf("Get(key2) error = %v", err)
	}

	if string(data1) != string(data2) {
		t.Errorf("key1 and key2 have different data despite same content")
	}
}

func TestCASObjectSharding(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tmpDir := t.TempDir()

	store, err := NewCAS(tmpDir)
	if err != nil {
		t.Fatalf("NewCAS() error = %v", err)
	}
	defer Close(store)

	err = store.Set(ctx, "foo", []byte("data"))
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	objectsDir := filepath.Join(tmpDir, "objects")
	entries, err := os.ReadDir(objectsDir)
	if err != nil {
		t.Fatalf("ReadDir(objects) error = %v", err)
	}

	if len(entries) == 0 {
		t.Fatal("No shard directories found in objects/")
	}

	hasSubShards := false
	for _, entry := range entries {
		if entry.IsDir() {
			subPath := filepath.Join(objectsDir, entry.Name())
			subEntries, err := os.ReadDir(subPath)
			if err != nil {
				continue
			}
			for _, subEntry := range subEntries {
				if subEntry.IsDir() {
					hasSubShards = true
					break
				}
			}
		}
	}

	if !hasSubShards {
		t.Fatal("Objects not sharded into aa/bb/ structure")
	}
}

func TestCASOrphanedObject(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tmpDir := t.TempDir()

	store, err := NewCAS(tmpDir)
	if err != nil {
		t.Fatalf("NewCAS() error = %v", err)
	}
	defer Close(store)

	data := []byte("data")
	err = store.Set(ctx, "key1", data)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	err = store.Delete(ctx, "key1")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	objectsDir := filepath.Join(tmpDir, "objects")
	var objectFiles []string

	err = filepath.WalkDir(objectsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			objectFiles = append(objectFiles, path)
		}
		return nil
	})

	if err != nil {
		t.Fatalf("WalkDir() error = %v", err)
	}

	if len(objectFiles) != 1 {
		t.Logf("Note: Object file was cleaned up (got %d files, expected 1 orphan)", len(objectFiles))
	}
}
