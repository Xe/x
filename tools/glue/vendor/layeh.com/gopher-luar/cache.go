package luar // import "layeh.com/gopher-luar"

import (
	"container/list"
	"reflect"

	"github.com/yuin/gopher-lua"
)

const (
	cacheKey = "github.com/layeh/gopher-luar"
	tagName  = "luar"
)

type mtCache struct {
	regular, types map[reflect.Type]*lua.LTable
}

func newMTCache() *mtCache {
	return &mtCache{
		regular: make(map[reflect.Type]*lua.LTable),
		types:   make(map[reflect.Type]*lua.LTable),
	}
}

func getMTCache(L *lua.LState) *mtCache {
	registry, ok := L.Get(lua.RegistryIndex).(*lua.LTable)
	if !ok {
		L.RaiseError("gopher-luar: corrupt lua registry")
	}
	lCache, ok := registry.RawGetString(cacheKey).(*lua.LUserData)
	if !ok {
		lCache = L.NewUserData()
		lCache.Value = newMTCache()
		registry.RawSetString(cacheKey, lCache)
	}
	cache, ok := lCache.Value.(*mtCache)
	if !ok {
		L.RaiseError("gopher-luar: corrupt luar metatable cache")
	}
	return cache
}

func addMethods(L *lua.LState, vtype reflect.Type, tbl *lua.LTable, ptrReceiver bool) {
	for i := 0; i < vtype.NumMethod(); i++ {
		method := vtype.Method(i)
		if method.PkgPath != "" {
			continue
		}
		fn := funcWrapper(L, method.Func, ptrReceiver)
		tbl.RawSetString(method.Name, fn)
		tbl.RawSetString(getUnexportedName(method.Name), fn)
	}
}

func addFields(L *lua.LState, vtype reflect.Type, tbl *lua.LTable) {
	type element struct {
		Type  reflect.Type
		Index []int
	}

	queue := list.New()
	queue.PushFront(element{
		Type: vtype,
	})

	for queue.Len() > 0 {
		e := queue.Back()
		elem := e.Value.(element)
		vtype := elem.Type
	fields:
		for i := 0; i < vtype.NumField(); i++ {
			field := vtype.Field(i)
			if field.PkgPath != "" && !field.Anonymous {
				continue
			}
			var names []string
			tag := field.Tag.Get(tagName)
			if tag == "-" {
				continue
			}
			if tag != "" {
				names = []string{
					tag,
				}
			} else {
				names = []string{
					field.Name,
					getUnexportedName(field.Name),
				}
			}
			for _, key := range names {
				if tbl.RawGetString(key) != lua.LNil {
					continue fields
				}
			}
			index := make([]int, len(elem.Index)+1)
			copy(index, elem.Index)
			index[len(elem.Index)] = i

			ud := L.NewUserData()
			ud.Value = index
			for _, key := range names {
				tbl.RawSetString(key, ud)
			}
			if field.Anonymous {
				t := field.Type
				if field.Type.Kind() != reflect.Struct {
					if field.Type.Kind() != reflect.Ptr || field.Type.Elem().Kind() != reflect.Struct {
						continue
					}
					t = field.Type.Elem()
				}
				index := make([]int, len(elem.Index)+1)
				copy(index, elem.Index)
				index[len(elem.Index)] = i
				queue.PushFront(element{
					Type:  t,
					Index: index,
				})
			}
		}

		queue.Remove(e)
	}
}

func getMetatable(L *lua.LState, vtype reflect.Type) *lua.LTable {
	cache := getMTCache(L)

	if vtype.Kind() == reflect.Ptr {
		vtype = vtype.Elem()
	}
	if v := cache.regular[vtype]; v != nil {
		return v
	}

	mt := L.NewTable()
	mt.RawSetString("__tostring", L.NewFunction(tostring))
	mt.RawSetString("__metatable", L.NewTable())
	mt.RawSetString("__pow", L.NewFunction(ptrPow))
	mt.RawSetString("__unm", L.NewFunction(ptrUnm))

	ptrMethods := L.NewTable()
	methods := L.NewTable()

	switch vtype.Kind() {
	case reflect.Array:
		mt.RawSetString("__index", L.NewFunction(arrayIndex))
		mt.RawSetString("__newindex", L.NewFunction(arrayNewIndex))
		mt.RawSetString("__len", L.NewFunction(arrayLen))
		mt.RawSetString("__call", L.NewFunction(arrayCall))
		mt.RawSetString("__eq", L.NewFunction(eq))
	case reflect.Chan:
		methods.RawSetString("send", L.NewFunction(chanSend))
		methods.RawSetString("receive", L.NewFunction(chanReceive))
		methods.RawSetString("close", L.NewFunction(chanClose))

		mt.RawSetString("__index", L.NewFunction(chanIndex))
		mt.RawSetString("__len", L.NewFunction(chanLen))
		mt.RawSetString("__eq", L.NewFunction(chanEq))
	case reflect.Map:
		mt.RawSetString("__index", L.NewFunction(mapIndex))
		mt.RawSetString("__newindex", L.NewFunction(mapNewIndex))
		mt.RawSetString("__len", L.NewFunction(mapLen))
		mt.RawSetString("__call", L.NewFunction(mapCall))
		mt.RawSetString("__eq", L.NewFunction(mapEq))
	case reflect.Slice:
		methods.RawSetString("capacity", L.NewFunction(sliceCapacity))
		methods.RawSetString("append", L.NewFunction(sliceAppend))

		mt.RawSetString("__index", L.NewFunction(sliceIndex))
		mt.RawSetString("__newindex", L.NewFunction(sliceNewIndex))
		mt.RawSetString("__len", L.NewFunction(sliceLen))
		mt.RawSetString("__call", L.NewFunction(sliceCall))
		mt.RawSetString("__eq", L.NewFunction(sliceEq))
	case reflect.Struct:
		fields := L.NewTable()
		addFields(L, vtype, fields)
		mt.RawSetString("fields", fields)

		mt.RawSetString("__index", L.NewFunction(structIndex))
		mt.RawSetString("__newindex", L.NewFunction(structNewIndex))
		mt.RawSetString("__eq", L.NewFunction(eq))
	default:
		mt.RawSetString("__index", L.NewFunction(ptrIndex))
		mt.RawSetString("__eq", L.NewFunction(ptrEq))
	}

	addMethods(L, reflect.PtrTo(vtype), ptrMethods, true)
	mt.RawSetString("ptr_methods", ptrMethods)

	addMethods(L, vtype, methods, false)
	mt.RawSetString("methods", methods)

	mt.RawSetString("original", L.NewTable())

	cache.regular[vtype] = mt
	return mt
}

func getMetatableFromValue(L *lua.LState, value reflect.Value) *lua.LTable {
	vtype := value.Type()
	return getMetatable(L, vtype)
}

func getTypeMetatable(L *lua.LState, t reflect.Type) *lua.LTable {
	cache := getMTCache(L)

	if v := cache.types[t]; v != nil {
		return v
	}

	mt := L.NewTable()
	mt.RawSetString("__call", L.NewFunction(typeCall))
	mt.RawSetString("__eq", L.NewFunction(typeEq))

	cache.types[t] = mt
	return mt
}
