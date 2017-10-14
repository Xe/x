package gluaflag

import (
	"fmt"
	"strconv"

	"github.com/yuin/gopher-lua"
)

type numberslice []float64

// String implements the stringer interface
func (i *numberslice) String() string {
	return fmt.Sprintf("%v", *i)
}

// Set implements the flag interface
func (i *numberslice) Set(value string) error {
	tmp, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return err
	}
	*i = append(*i, tmp)
	return nil
}

// Table converts the slice to a lua.LTable
func (i *numberslice) Table(L *lua.LState) *lua.LTable {
	t := L.NewTable()
	for _, v := range *i {
		t.Append(lua.LNumber(v))
	}
	return t
}

type intslice []int

// String implements the stringer interface
func (i *intslice) String() string {
	return fmt.Sprintf("%d", *i)
}

// Set implements the flag interface
func (i *intslice) Set(value string) error {
	tmp, err := strconv.Atoi(value)
	if err != nil {
		return err
	}
	*i = append(*i, tmp)
	return nil
}

func (i *intslice) Table(L *lua.LState) *lua.LTable {
	t := L.NewTable()
	for _, v := range *i {
		t.Append(lua.LNumber(v))
	}
	return t
}

type stringslice []string

// String implements the stringer interface
func (s *stringslice) String() string {
	return fmt.Sprintf("%v", *s)
}

// Set implements the flag interface
func (s *stringslice) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func (s *stringslice) Table(L *lua.LState) *lua.LTable {
	t := L.NewTable()
	for _, v := range *s {
		t.Append(lua.LString(v))
	}
	return t
}
