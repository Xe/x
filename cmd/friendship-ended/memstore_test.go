package main

import (
	"context"
	"errors"
	"strings"
	"sync"
)

type memObject struct {
	contentType string
	data        []byte
}

// memStore is an in-memory Store for tests. Set failPut to make every Put
// fail, or failSuffix to make only Puts whose key ends with that suffix
// fail (for example "old2.png" to fail one specific upload).
type memStore struct {
	mu         sync.Mutex
	objects    map[string]memObject
	failPut    bool
	failSuffix string
}

func newMemStore() *memStore {
	return &memStore{objects: map[string]memObject{}}
}

func (m *memStore) Put(_ context.Context, key, contentType string, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failPut || (m.failSuffix != "" && strings.HasSuffix(key, m.failSuffix)) {
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
