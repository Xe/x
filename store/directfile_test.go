package store

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDirectFileSetGet(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tmpDir := t.TempDir()

	store, err := NewDirectFile(tmpDir)
	if err != nil {
		t.Fatalf("NewDirectFile() error = %v", err)
	}

	err = store.Set(ctx, "foo/bar/baz", []byte("hello world"))
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	data, err := store.Get(ctx, "foo/bar/baz")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if string(data) != "hello world" {
		t.Errorf("Get() = %q, want %q", string(data), "hello world")
	}
}

func TestDirectFileGetNotFound(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tmpDir := t.TempDir()

	store, err := NewDirectFile(tmpDir)
	if err != nil {
		t.Fatalf("NewDirectFile() error = %v", err)
	}

	_, err = store.Get(ctx, "nonexistent")
	if err == nil {
		t.Fatal("Get() expected error, got nil")
	}

	if !IsNotFound(err) {
		t.Errorf("Get() error = %v, want ErrNotFound", err)
	}
}

func TestDirectFileDelete(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tmpDir := t.TempDir()

	store, err := NewDirectFile(tmpDir)
	if err != nil {
		t.Fatalf("NewDirectFile() error = %v", err)
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

func TestDirectFileDeleteNotFound(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tmpDir := t.TempDir()

	store, err := NewDirectFile(tmpDir)
	if err != nil {
		t.Fatalf("NewDirectFile() error = %v", err)
	}

	err = store.Delete(ctx, "nonexistent")
	if err == nil {
		t.Fatal("Delete() expected error, got nil")
	}

	if !IsNotFound(err) {
		t.Errorf("Delete() error = %v, want ErrNotFound", err)
	}
}

func TestDirectFileExists(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tmpDir := t.TempDir()

	store, err := NewDirectFile(tmpDir)
	if err != nil {
		t.Fatalf("NewDirectFile() error = %v", err)
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

func TestDirectFileList(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tmpDir := t.TempDir()

	store, err := NewDirectFile(tmpDir)
	if err != nil {
		t.Fatalf("NewDirectFile() error = %v", err)
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

func TestDirectFileListEmpty(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tmpDir := t.TempDir()

	store, err := NewDirectFile(tmpDir)
	if err != nil {
		t.Fatalf("NewDirectFile() error = %v", err)
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

func TestDirectFileEmptyKey(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tmpDir := t.TempDir()

	store, err := NewDirectFile(tmpDir)
	if err != nil {
		t.Fatalf("NewDirectFile() error = %v", err)
	}

	err = store.Set(ctx, "", []byte("data"))
	if err == nil {
		t.Fatal("Set() expected error for empty key, got nil")
	}

	if !IsBadConfig(err) {
		t.Errorf("Set() error = %v, want ErrBadConfig", err)
	}
}

func TestDirectFileKeyWithDotDot(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tmpDir := t.TempDir()

	store, err := NewDirectFile(tmpDir)
	if err != nil {
		t.Fatalf("NewDirectFile() error = %v", err)
	}

	err = store.Set(ctx, "../escape", []byte("data"))
	if err == nil {
		t.Fatal("Set() expected error for key with .., got nil")
	}

	if !IsBadConfig(err) {
		t.Errorf("Set() error = %v, want ErrBadConfig", err)
	}
}

func TestDirectFileLeadingSlash(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tmpDir := t.TempDir()

	store, err := NewDirectFile(tmpDir)
	if err != nil {
		t.Fatalf("NewDirectFile() error = %v", err)
	}

	err = store.Set(ctx, "/foo/bar", []byte("data"))
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	data, err := store.Get(ctx, "/foo/bar")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if string(data) != "data" {
		t.Errorf("Get() = %q, want %q", string(data), "data")
	}

	expectedPath := filepath.Join(tmpDir, "foo/bar")
	if _, err := os.Stat(expectedPath); err != nil {
		t.Errorf("File not created at expected path %s: %v", expectedPath, err)
	}
}

func TestDirectFileNestedDirectories(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tmpDir := t.TempDir()

	store, err := NewDirectFile(tmpDir)
	if err != nil {
		t.Fatalf("NewDirectFile() error = %v", err)
	}

	deepKey := "a/b/c/d/e/f/g"
	err = store.Set(ctx, deepKey, []byte("deep"))
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	data, err := store.Get(ctx, deepKey)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if string(data) != "deep" {
		t.Errorf("Get() = %q, want %q", string(data), "deep")
	}

	expectedPath := filepath.Join(tmpDir, deepKey)
	if _, err := os.Stat(expectedPath); err != nil {
		t.Errorf("File not created at expected path %s: %v", expectedPath, err)
	}
}

func IsNotFound(err error) bool {
	return err != nil && strings.Contains(err.Error(), "not found")
}

func IsBadConfig(err error) bool {
	return err != nil && strings.Contains(err.Error(), "configuration is invalid")
}
