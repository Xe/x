package cursed

import "sync"

// UseState is a copy of the useState monad in React/Preact.
//
// This is a thread-safe stateful container that allows you to set
// and get a shared value.
//
// Consumers should take care to NOT mutate any values in the fetched
// variable.
func UseState[T any](initial T) (func() T, func(T)) {
	var (
		lock sync.Mutex
		data T = initial

		set = func(upda T) {
			lock.Lock()
			defer lock.Unlock()
			data = upda
		}

		get = func() T {
			lock.Lock()
			defer lock.Unlock()
			return data
		}
	)

	return get, set
}
