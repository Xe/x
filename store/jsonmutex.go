package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

func NewJSONMutexDB(baseDir string) (Interface, error) {
	if baseDir == "" {
		return nil, fmt.Errorf("%w: base directory cannot be empty", ErrBadConfig)
	}

	dataDir := filepath.Join(baseDir, "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("%w: can't create data directory: %w", ErrBadConfig, err)
	}

	indexFile := filepath.Join(baseDir, "index.json")

	index := make(map[string]string)
	if data, err := os.ReadFile(indexFile); err == nil {
		if err := json.Unmarshal(data, &index); err != nil {
			return nil, fmt.Errorf("%w: can't decode index: %w", ErrBadConfig, err)
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("%w: can't read index: %w", ErrBadConfig, err)
	}

	return &JSONMutexDB{
		base:      baseDir,
		dataDir:   dataDir,
		indexFile: indexFile,
		index:     index,
	}, nil
}

type JSONMutexDB struct {
	mu        sync.RWMutex
	base      string
	dataDir   string
	indexFile string
	index     map[string]string
}

func (j *JSONMutexDB) Close() error {
	return nil
}

func (j *JSONMutexDB) Delete(ctx context.Context, key string) error {
	if key == "" {
		return fmt.Errorf("%w: key cannot be empty", ErrBadConfig)
	}

	j.mu.Lock()
	defer j.mu.Unlock()

	filename, ok := j.index[key]
	if !ok {
		return fmt.Errorf("%w: key not found", ErrNotFound)
	}

	delete(j.index, key)

	if err := j.writeIndex(); err != nil {
		return err
	}

	dataPath := filepath.Join(j.dataDir, filename)
	if err := os.Remove(dataPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("can't delete data file: %w", err)
	}

	iopsMetrics.WithLabelValues("jsonmutex", "Delete")
	return nil
}

func (j *JSONMutexDB) Exists(ctx context.Context, key string) error {
	if key == "" {
		return fmt.Errorf("%w: key cannot be empty", ErrBadConfig)
	}

	j.mu.RLock()
	defer j.mu.RUnlock()

	_, ok := j.index[key]
	if !ok {
		return fmt.Errorf("%w: key not found", ErrNotFound)
	}

	iopsMetrics.WithLabelValues("jsonmutex", "Exists")
	return nil
}

func (j *JSONMutexDB) Get(ctx context.Context, key string) ([]byte, error) {
	if key == "" {
		return nil, fmt.Errorf("%w: key cannot be empty", ErrBadConfig)
	}

	j.mu.RLock()
	defer j.mu.RUnlock()

	filename, ok := j.index[key]
	if !ok {
		return nil, fmt.Errorf("%w: key not found", ErrNotFound)
	}

	dataPath := filepath.Join(j.dataDir, filename)
	data, err := os.ReadFile(dataPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: data file missing", ErrNotFound)
		}
		return nil, fmt.Errorf("can't read data file: %w", err)
	}

	iopsMetrics.WithLabelValues("jsonmutex", "Get")
	return data, nil
}

func (j *JSONMutexDB) Set(ctx context.Context, key string, value []byte) error {
	if key == "" {
		return fmt.Errorf("%w: key cannot be empty", ErrBadConfig)
	}

	j.mu.Lock()
	defer j.mu.Unlock()

	filename := j.index[key]
	if filename == "" {
		filename = sanitizeFilename(key)
		j.index[key] = filename
	}

	dataPath := filepath.Join(j.dataDir, filename)
	if err := os.WriteFile(dataPath, value, 0644); err != nil {
		return fmt.Errorf("can't write data file: %w", err)
	}

	if err := j.writeIndex(); err != nil {
		return err
	}

	iopsMetrics.WithLabelValues("jsonmutex", "Set")
	return nil
}

func (j *JSONMutexDB) List(ctx context.Context, prefix string) ([]string, error) {
	j.mu.RLock()
	defer j.mu.RUnlock()

	var result []string
	for key := range j.index {
		if prefix == "" || strings.HasPrefix(key, prefix) {
			result = append(result, key)
		}
	}

	iopsMetrics.WithLabelValues("jsonmutex", "List")
	return result, nil
}

func (j *JSONMutexDB) writeIndex() error {
	data, err := json.MarshalIndent(j.index, "", "  ")
	if err != nil {
		return fmt.Errorf("can't encode index: %w", err)
	}

	if err := os.WriteFile(j.indexFile, data, 0644); err != nil {
		return fmt.Errorf("can't write index file: %w", err)
	}

	return nil
}

func sanitizeFilename(key string) string {
	key = strings.ReplaceAll(key, "/", "_")
	key = strings.ReplaceAll(key, "\\", "_")
	key = strings.ReplaceAll(key, ":", "_")
	return key
}
