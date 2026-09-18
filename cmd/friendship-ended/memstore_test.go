package main

import (
	"context"
	"errors"
	"sync"
)

type memObject struct {
	contentType string
	data        []byte
}

// memStore is an in-memory Store for tests. Set failPut to make every Put
// fail.
type memStore struct {
	mu      sync.Mutex
	objects map[string]memObject
	failPut bool
}

func newMemStore() *memStore {
	return &memStore{objects: map[string]memObject{}}
}

func (m *memStore) Put(_ context.Context, key, contentType string, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failPut {
		return errors.New("memstore: put failed on purpose")
	}

	m.objects[key] = memObject{contentType: contentType, data: append([]byte(nil), data...)}
	return nil
}

func (m *memStore) Get(_ context.Context, key string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	obj, ok := m.objects[key]
	if !ok {
		return nil, ErrNotFound
	}

	return obj.data, nil
}

func (m *memStore) Presign(_ context.Context, key string) (string, error) {
	return "https://tigris.example/" + key + "?X-Amz-Signature=fake", nil
}
