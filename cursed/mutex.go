package cursed

import "sync"

// Mutex is a generic locking container for Go much like Rust's std::sync::Mutex<T>.
//
// It differs from a normal sync.Mutex because it guards a value instead of just
// being something you lock and unlock to guard another value. When you are done with
// the value, call the function return to re-lock the mutex.
type Mutex[T any] struct {
	val  T
	lock sync.Mutex
}

func NewMutex[T any](val T) *Mutex[T] {
	return &Mutex[T]{val: val}
}

func (mu *Mutex[T]) Unlock() (T, func()) {
	mu.lock.Lock()
	return mu.val, func() { mu.lock.Unlock() }
}
