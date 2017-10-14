package luar // import "layeh.com/gopher-luar"

import (
	"reflect"

	"github.com/yuin/gopher-lua"
)

// New creates and returns a new lua.LValue for the given value.
//
// The following table shows how Go types are converted to Lua types:
//  Kind            gopher-lua Type  Custom Metatable
//  --------------------------------------------------
//  nil             LNil             No
//  Bool            LBool            No
//  Int             LNumber          No
//  Int8            LNumber          No
//  Int16           LNumber          No
//  Int32           LNumber          No
//  Int64           LNumber          No
//  Uint            LNumber          No
//  Uint8           LNumber          No
//  Uint16          LNumber          No
//  Uint32          LNumber          No
//  Uint64          LNumber          No
//  Uintptr         *LUserData       No
//  Float32         LNumber          No
//  Float64         LNumber          No
//  Complex64       *LUserData       No
//  Complex128      *LUserData       No
//  Array           *LUserData       Yes
//  Chan            *LUserData       Yes
//  Func            *lua.LFunction   No
//  Map             *LUserData       Yes
//  Ptr             *LUserData       Yes
//  Slice           *LUserData       Yes
//  String          LString          No
//  Struct          *LUserData       Yes
//  UnsafePointer   *LUserData       No
func New(L *lua.LState, value interface{}) lua.LValue {
	if value == nil {
		return lua.LNil
	}
	if lval, ok := value.(lua.LValue); ok {
		return lval
	}

	val := reflect.ValueOf(value)
	switch val.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		if val.IsNil() {
			return lua.LNil
		}
	}

	switch val.Kind() {
	case reflect.Bool:
		return lua.LBool(val.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return lua.LNumber(float64(val.Int()))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return lua.LNumber(float64(val.Uint()))
	case reflect.Float32, reflect.Float64:
		return lua.LNumber(val.Float())
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Ptr, reflect.Slice, reflect.Struct:
		ud := L.NewUserData()
		ud.Value = val.Interface()
		ud.Metatable = getMetatableFromValue(L, val)
		return ud
	case reflect.Func:
		return funcWrapper(L, val, false)
	case reflect.String:
		return lua.LString(val.String())
	default:
		ud := L.NewUserData()
		ud.Value = val.Interface()
		return ud
	}
}

// NewType returns a new type creator for the given value's type.
//
// When the lua.LValue is called, a new value will be created that is the
// same type as value's type.
func NewType(L *lua.LState, value interface{}) lua.LValue {
	val := reflect.TypeOf(value)
	ud := L.NewUserData()
	ud.Value = val
	ud.Metatable = getTypeMetatable(L, val)

	return ud
}

func lValueToReflect(L *lua.LState, v lua.LValue, hint reflect.Type, tryConvertPtr *bool) reflect.Value {
	if hint.Implements(refTypeLuaLValue) {
		return reflect.ValueOf(v)
	}

	isPtr := false

	switch converted := v.(type) {
	case lua.LBool:
		var val reflect.Value
		if hint.Kind() == reflect.String {
			val = reflect.ValueOf(converted.String())
		} else {
			val = reflect.ValueOf(bool(converted))
		}
		return val.Convert(hint)
	case lua.LChannel:
		return reflect.ValueOf(converted).Convert(hint)
	case lua.LNumber:
		var val reflect.Value
		if hint.Kind() == reflect.String {
			val = reflect.ValueOf(converted.String())
		} else {
			val = reflect.ValueOf(converted)
		}
		return val.Convert(hint)
	case *lua.LFunction:
		fn := func(args []reflect.Value) []reflect.Value {
			L.Push(converted)

			varadicCount := 0

			for i, arg := range args {
				if hint.IsVariadic() && i+1 == len(args) {
					// arg is the varadic slice
					varadicCount = arg.Len()
					for j := 0; j < varadicCount; j++ {
						arg := arg.Index(j)
						if !arg.CanInterface() {
							L.Pop(i + j + 1)
							L.RaiseError("unable to Interface argument %d", i+j)
						}
						L.Push(New(L, arg.Interface()))
					}
					// recount for varadic slice that appeared
					varadicCount--
					break
				}

				if !arg.CanInterface() {
					L.Pop(i + 1)
					L.RaiseError("unable to Interface argument %d", i)
				}
				L.Push(New(L, arg.Interface()))
			}

			L.Call(len(args)+varadicCount, hint.NumOut())
			defer L.Pop(hint.NumOut())

			ret := make([]reflect.Value, hint.NumOut())

			for i := 0; i < hint.NumOut(); i++ {
				outHint := hint.Out(i)
				ret[i] = lValueToReflect(L, L.Get(-hint.NumOut()+i), outHint, nil)
			}

			return ret
		}
		return reflect.MakeFunc(hint, fn)
	case *lua.LNilType:
		switch hint.Kind() {
		case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice, reflect.UnsafePointer:
			return reflect.Zero(hint)
		default:
			L.RaiseError("cannot convert nil to %s", hint.String())
		}
	case *lua.LState:
		return reflect.ValueOf(converted).Convert(hint)
	case lua.LString:
		return reflect.ValueOf(string(converted)).Convert(hint)
	case *lua.LTable:
		switch {
		case hint.Kind() == reflect.Slice:
			elemType := hint.Elem()
			length := converted.Len()
			s := reflect.MakeSlice(hint, length, length)

			for i := 0; i < length; i++ {
				value := converted.RawGetInt(i + 1)
				elemValue := lValueToReflect(L, value, elemType, nil)
				s.Index(i).Set(elemValue)
			}

			return s

		case hint.Kind() == reflect.Map:
			keyType := hint.Key()
			elemType := hint.Elem()
			s := reflect.MakeMap(hint)

			converted.ForEach(func(key, value lua.LValue) {
				if _, ok := key.(lua.LString); !ok {
					return
				}

				lKey := lValueToReflect(L, key, keyType, nil)
				lValue := lValueToReflect(L, value, elemType, nil)
				s.SetMapIndex(lKey, lValue)
			})

			return s

		case hint.Kind() == reflect.Ptr && hint.Elem().Kind() == reflect.Struct:
			hint = hint.Elem()
			isPtr = true
			fallthrough
		case hint.Kind() == reflect.Struct:
			s := reflect.New(hint)
			t := s.Elem()

			mt := &Metatable{
				LTable: getMetatable(L, hint),
			}

			converted.ForEach(func(key, value lua.LValue) {
				if _, ok := key.(lua.LString); !ok {
					return
				}

				fieldName := key.String()
				index := mt.fieldIndex(fieldName)
				if index == nil {
					L.RaiseError("invalid field %s", fieldName)
				}
				field := hint.FieldByIndex(index)

				lValue := lValueToReflect(L, value, field.Type, nil)
				t.FieldByIndex(field.Index).Set(lValue)
			})

			if isPtr {
				return s
			}

			return t

		default:
			return reflect.ValueOf(converted).Convert(hint)
		}
	case *lua.LUserData:
		val := reflect.ValueOf(converted.Value)
		if tryConvertPtr != nil && val.Kind() != reflect.Ptr && hint.Kind() == reflect.Ptr && val.Type() == hint.Elem() {
			newVal := reflect.New(hint.Elem())
			newVal.Elem().Set(val)
			val = newVal
			*tryConvertPtr = true
		} else {
			val = val.Convert(hint)
			if tryConvertPtr != nil {
				*tryConvertPtr = false
			}
		}
		return val
	}
	L.RaiseError("fatal lValueToReflect error")
	return reflect.Value{} // never returns
}
