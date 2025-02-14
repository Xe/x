package main

import (
	"sync"
	"time"
)

func zilch[T any]() T {
	var zero T
	return zero
}

type DecayMap[K, V comparable] struct {
	data map[K]DecayMapEntry[V]
	lock sync.RWMutex
}

type DecayMapEntry[V comparable] struct {
	Value  V
	expiry time.Time
}

func NewDecayMap[K, V comparable]() *DecayMap[K, V] {
	return &DecayMap[K, V]{
		data: make(map[K]DecayMapEntry[V]),
	}
}

func (m *DecayMap[K, V]) Get(key K) (V, bool) {
	m.lock.RLock()
	value, ok := m.data[key]
	m.lock.RUnlock()

	if !ok {
		return zilch[V](), false
	}

	if time.Now().After(value.expiry) {
		m.lock.Lock()
		delete(m.data, key)
		m.lock.Unlock()

		return zilch[V](), false
	}

	return value.Value, true
}

func (m *DecayMap[K, V]) Set(key K, value V, ttl time.Duration) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.data[key] = DecayMapEntry[V]{
		Value:  value,
		expiry: time.Now().Add(ttl),
	}
}
