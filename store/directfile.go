package store

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func NewDirectFile(baseDir string) (Interface, error) {
	if baseDir == "" {
		return nil, fmt.Errorf("%w: base directory cannot be empty", ErrBadConfig)
	}

	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("%w: can't create base directory: %w", ErrBadConfig, err)
	}

	return &DirectFile{
		baseDir: baseDir,
	}, nil
}

type DirectFile struct {
	baseDir string
}

func (d *DirectFile) Delete(ctx context.Context, key string) error {
	key, err := d.validateAndCleanKey(key)
	if err != nil {
		return err
	}

	path := d.keyToPath(key)

	err = os.Remove(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%w: %w", ErrNotFound, err)
		}
		return fmt.Errorf("can't delete file: %w", err)
	}

	iopsMetrics.WithLabelValues("directfile", "Delete")
	return nil
}

func (d *DirectFile) Exists(ctx context.Context, key string) error {
	key, err := d.validateAndCleanKey(key)
	if err != nil {
		return err
	}

	path := d.keyToPath(key)

	_, err = os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%w: %w", ErrNotFound, err)
		}
		return err
	}

	iopsMetrics.WithLabelValues("directfile", "Exists")
	return nil
}

func (d *DirectFile) Get(ctx context.Context, key string) ([]byte, error) {
	key, err := d.validateAndCleanKey(key)
	if err != nil {
		return nil, err
	}

	path := d.keyToPath(key)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %w", ErrNotFound, err)
		}
		return nil, fmt.Errorf("can't read file: %w", err)
	}

	iopsMetrics.WithLabelValues("directfile", "Get")
	return data, nil
}

func (d *DirectFile) Set(ctx context.Context, key string, value []byte) error {
	key, err := d.validateAndCleanKey(key)
	if err != nil {
		return err
	}

	path := d.keyToPath(key)

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("can't create directories: %w", err)
	}

	if err := os.WriteFile(path, value, 0644); err != nil {
		return fmt.Errorf("can't write file: %w", err)
	}

	iopsMetrics.WithLabelValues("directfile", "Set")
	return nil
}

func (d *DirectFile) List(ctx context.Context, prefix string) ([]string, error) {
	cleanPrefix := strings.TrimPrefix(prefix, "/")
	basePath := d.baseDir
	if cleanPrefix != "" {
		basePath = filepath.Join(d.baseDir, cleanPrefix)
	}

	var result []string

	err := filepath.WalkDir(basePath, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if entry.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(d.baseDir, path)
		if err != nil {
			return err
		}

		if prefix == "" || strings.HasPrefix(relPath, cleanPrefix) {
			result = append(result, relPath)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("can't walk directory: %w", err)
	}

	iopsMetrics.WithLabelValues("directfile", "List")
	return result, nil
}

func (d *DirectFile) validateAndCleanKey(key string) (string, error) {
	if key == "" {
		return "", fmt.Errorf("%w: key cannot be empty", ErrBadConfig)
	}

	cleanKey := strings.TrimPrefix(key, "/")

	if strings.Contains(cleanKey, "..") {
		return "", fmt.Errorf("%w: key cannot contain '..'", ErrBadConfig)
	}

	return cleanKey, nil
}

func (d *DirectFile) keyToPath(key string) string {
	return filepath.Join(d.baseDir, filepath.FromSlash(key))
}
