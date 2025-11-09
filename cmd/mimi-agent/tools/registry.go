package tools

import "sync"

var (
	lock  sync.Mutex
	tools map[string]Implementation
)

func Set(name string, impl Implementation) {
	lock.Lock()
	defer lock.Unlock()

	tools[name] = impl
}

func Get(name string) (Implementation, bool) {
	lock.Lock()
	defer lock.Unlock()

	tool, ok := tools[name]
	return tool, ok
}
