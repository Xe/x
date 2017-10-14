package luar

import (
	"reflect"
	"testing"

	"github.com/yuin/gopher-lua"
	"strings"
)

func Test_config(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	config := GetConfig(L)
	if config == nil {
		t.Fatal("expecting non-nil config")
	}

	if config.FieldNames != nil {
		t.Fatal("expected config.FieldName to be nil")
	}

	if config.MethodNames != nil {
		t.Fatal("expected config.MethodName to be nil")
	}
}

func Test_config_fieldnames(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	config := GetConfig(L)
	config.FieldNames = func(s reflect.Type, f reflect.StructField) []string {
		return []string{strings.ToLower(f.Name)}
	}

	type S struct {
		Name string
		Age  int `luar:"AGE"`
	}
	s := S{
		Name: "Tim",
		Age:  89,
	}

	L.SetGlobal("s", New(L, &s))

	testReturn(t, L, `return s.Name`, `nil`)
	testReturn(t, L, `return s.name`, `Tim`)
	testReturn(t, L, `return s.AGE`, `nil`)
	testReturn(t, L, `return s.age`, `89`)
}

type TestConfigMethodnames []string

func (t TestConfigMethodnames) Len() int {
	return len(t)
}

func Test_config_methodnames(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	config := GetConfig(L)
	config.MethodNames = func(s reflect.Type, m reflect.Method) []string {
		return []string{strings.ToLower(m.Name) + "gth"}
	}

	v := TestConfigMethodnames{
		"hello",
		"world",
	}

	L.SetGlobal("v", New(L, v))

	testError(t, L, `return v:len()`, `attempt to call a non-function object`)
	testReturn(t, L, `return v:length()`, `2`)
}
