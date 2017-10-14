package luar // import "layeh.com/gopher-luar"

import (
	"reflect"

	"github.com/yuin/gopher-lua"
)

// Metatable holds the Lua metatable for a Go type.
type Metatable struct {
	*lua.LTable

	l *lua.LState
}

// MT returns the metatable for value's type. nil is returned if value's type
// does not use a custom metatable.
func MT(L *lua.LState, value interface{}) *Metatable {
	val := reflect.ValueOf(value)
	switch val.Type().Kind() {
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Ptr, reflect.Slice, reflect.Struct:
		return &Metatable{
			LTable: getMetatableFromValue(L, val),
			l:      L,
		}
	default:
		return nil
	}
}

func (m *Metatable) original() *lua.LTable {
	original, ok := m.RawGetString("original").(*lua.LTable)
	if !ok {
		m.l.RaiseError("gopher-luar: corrupt luar metatable")
	}
	return original
}

// Reset resets the metatable, restoring any fields or methods removed through
// whitelisting or blacklisting.
func (m *Metatable) Reset() {
	original := m.original()
	for _, name := range []string{"methods", "ptr_methods", "fields"} {
		val := original.RawGetString(name)
		if val != lua.LNil {
			original.RawSetString(name, lua.LNil)
			m.RawSetString(name, val)
		}
	}
}

func (m *Metatable) list(whitelist bool, names ...string) {
	m.Reset()

	set := make(map[string]struct{}, len(names))
	for _, name := range names {
		set[name] = struct{}{}
	}

	original := m.original()

	for _, tblName := range []string{"methods", "ptr_methods", "fields"} {
		tbl, ok := m.RawGetString(tblName).(*lua.LTable)
		if !ok {
			continue
		}
		original.RawSetString(tblName, tbl)
		newTbl := m.l.NewTable()
		m.RawSetString(tblName, newTbl)
		tbl.ForEach(func(key, value lua.LValue) {
			keyStr := lua.LVAsString(key)
			if _, ok := set[keyStr]; ok == whitelist {
				newTbl.RawSetString(keyStr, value)
			}
		})
	}
}

// Whitelist sets only the gvein fields or methods to be accessed from Lua.
//
// Before the whitelist is applied, the metatable is reset to its original
// state (i.e. any previously applied whitelist or blacklist is reverted).
func (m *Metatable) Whitelist(names ...string) {
	m.list(true, names...)
}

// Blacklist disallows the given fields or methods to be accessed from Lua.
//
// Before the blacklist is applied, the metatable is reset to its original
// state (i.e. any previously applied whitelist or blacklist is reverted).
func (m *Metatable) Blacklist(names ...string) {
	m.list(false, names...)
}

func (m *Metatable) method(name string) lua.LValue {
	methods := m.RawGetString("methods").(*lua.LTable)
	if fn := methods.RawGetString(name); fn != lua.LNil {
		return fn
	}
	return nil
}

func (m *Metatable) ptrMethod(name string) lua.LValue {
	methods := m.RawGetString("ptr_methods").(*lua.LTable)
	if fn := methods.RawGetString(name); fn != lua.LNil {
		return fn
	}
	return nil
}

func (m *Metatable) fieldIndex(name string) []int {
	fields := m.RawGetString("fields").(*lua.LTable)
	if index := fields.RawGetString(name); index != lua.LNil {
		return index.(*lua.LUserData).Value.([]int)
	}
	return nil
}
