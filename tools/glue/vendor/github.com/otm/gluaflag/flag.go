package gluaflag

import (
	"fmt"
	"strconv"

	"github.com/yuin/gopher-lua"
)

var flagFuncs = map[string]lua.LGFunction{}

type flg struct {
	name     string
	value    interface{}
	usage    string
	required bool
	compFn   *lua.LFunction
}

func (f *flg) userdata(L *lua.LState) lua.LValue {
	ud := L.NewUserData()
	ud.Value = f
	L.SetMetatable(ud, L.GetTypeMetatable(luaFlagTypeName))
	return ud
}

type flgs map[string]*flg

type argument struct {
	name       string
	times      int
	glob       bool
	optional   bool
	usage      string
	value      lua.LValue
	typ        string
	parser     parser
	shortUsage shortUsage
	compFn     *lua.LFunction
}

func (a *argument) parse(args []string, L *lua.LState) ([]string, error) {
	args, value, err := a.parser(args, L)
	a.value = value
	return args, err
}

func (a *argument) toLValue(L *lua.LState) lua.LValue {
	return a.value
}

func (a *argument) generateUsage() string {
	typ := a.typ
	if typ == "" {
		typ = "string"
	}

	return fmt.Sprintf("  %v %v\n    \t%v\n", a.name, typ, a.usage)
}

type arguments []*argument

type parser func([]string, *lua.LState) ([]string, lua.LValue, error)

type shortUsage func(string) string

// string parsers
func parseString(args []string, L *lua.LState) ([]string, lua.LValue, error) {
	if len(args) < 1 {
		return args, lua.LNil, fmt.Errorf("expected string")
	}
	return parseOptionalString(args, L)
}

func parseOptionalString(args []string, L *lua.LState) ([]string, lua.LValue, error) {
	if len(args) < 1 {
		return args, lua.LNil, nil
	}
	return args[1:len(args)], lua.LString(args[0]), nil
}

func parseNStrings(n int) parser {
	return func(args []string, L *lua.LState) ([]string, lua.LValue, error) {
		if len(args) < n {
			return args, lua.LNil, fmt.Errorf("expected %v strings", n)
		}
		table := L.NewTable()
		for i := 0; i < n; i++ {
			table.Append(lua.LString(args[i]))
		}
		return args[n:len(args)], table, nil
	}
}

func parseStrings(args []string, L *lua.LState) ([]string, lua.LValue, error) {
	if len(args) < 1 {
		return args, L.NewTable(), fmt.Errorf("expected at least one string")
	}

	return parseOptionalStrings(args, L)
}

func parseOptionalStrings(args []string, L *lua.LState) ([]string, lua.LValue, error) {
	table := L.NewTable()
	for i := 0; i < len(args); i++ {
		table.Append(lua.LString(args[i]))
	}
	return make([]string, 0), table, nil
}

// integer parsers
func parseInt(args []string, L *lua.LState) ([]string, lua.LValue, error) {
	if len(args) < 1 {
		return args, lua.LNil, fmt.Errorf("expected string")
	}
	return parseOptionalInt(args, L)
}

func parseOptionalInt(args []string, L *lua.LState) ([]string, lua.LValue, error) {
	if len(args) < 1 {
		return args, lua.LNil, nil
	}
	i, err := strconv.Atoi(args[0])
	if err != nil {
		return args[1:len(args)], lua.LNumber(0), fmt.Errorf("invalid integer value: %v", args[0])
	}
	return args[1:len(args)], lua.LNumber(i), nil
}

func parseNInts(n int) parser {
	return func(args []string, L *lua.LState) ([]string, lua.LValue, error) {
		if len(args) < n {
			return args, lua.LNil, fmt.Errorf("expected %v integers", n)
		}
		table := L.NewTable()
		for i := 0; i < n; i++ {
			v, err := strconv.Atoi(args[i])
			if err != nil {
				return args[1:len(args)], lua.LNumber(0), fmt.Errorf("invalid integer value: %v", args[i])
			}
			table.Append(lua.LNumber(v))
		}
		return args[n:len(args)], table, nil
	}
}

func parseInts(args []string, L *lua.LState) ([]string, lua.LValue, error) {
	if len(args) < 1 {
		return args, L.NewTable(), fmt.Errorf("expected at least one integer")
	}

	return parseOptionalInts(args, L)
}

func parseOptionalInts(args []string, L *lua.LState) ([]string, lua.LValue, error) {
	table := L.NewTable()
	for i := 0; i < len(args); i++ {
		v, err := strconv.Atoi(args[i])
		if err != nil {
			return args[1:len(args)], lua.LNumber(0), fmt.Errorf("invalid integer value: %v", args[i])
		}
		table.Append(lua.LNumber(v))
	}
	return make([]string, 0), table, nil
}

// number parsers
func parseNumber(args []string, L *lua.LState) ([]string, lua.LValue, error) {
	if len(args) < 1 {
		return args, lua.LNil, fmt.Errorf("expected string")
	}
	return parseOptionalNumber(args, L)
}

func parseOptionalNumber(args []string, L *lua.LState) ([]string, lua.LValue, error) {
	if len(args) < 1 {
		return args, lua.LNil, nil
	}
	i, err := strconv.ParseFloat(args[0], 64)
	if err != nil {
		return args[1:len(args)], lua.LNumber(0), fmt.Errorf("invalid number value: %v", args[0])
	}
	return args[1:len(args)], lua.LNumber(i), nil
}

func parseNNumbers(n int) parser {
	return func(args []string, L *lua.LState) ([]string, lua.LValue, error) {
		if len(args) < n {
			return args, lua.LNil, fmt.Errorf("expected %v numbers", n)
		}
		table := L.NewTable()
		for i := 0; i < n; i++ {
			v, err := strconv.ParseFloat(args[i], 64)
			if err != nil {
				return args[1:len(args)], lua.LNumber(0), fmt.Errorf("invalid number value: %v", args[i])
			}
			table.Append(lua.LNumber(v))
		}
		return args[n:len(args)], table, nil
	}
}

func parseNumbers(args []string, L *lua.LState) ([]string, lua.LValue, error) {
	if len(args) < 1 {
		return args, L.NewTable(), fmt.Errorf("expected at least one number")
	}

	return parseOptionalNumbers(args, L)
}

func parseOptionalNumbers(args []string, L *lua.LState) ([]string, lua.LValue, error) {
	table := L.NewTable()
	for i := 0; i < len(args); i++ {
		v, err := strconv.ParseFloat(args[i], 64)
		if err != nil {
			return args[1:len(args)], lua.LNumber(0), fmt.Errorf("invalid number value: %v", args[i])
		}
		table.Append(lua.LNumber(v))
	}
	return make([]string, 0), table, nil
}

func getParser(typ string, option lua.LValue) (parser, error) {
	parsers := map[string]map[string]parser{
		"string": {
			"+": parseStrings,
			"*": parseOptionalStrings,
			"?": parseOptionalString,
			"1": parseString,
		},
		"int": {
			"+": parseInts,
			"*": parseOptionalInts,
			"?": parseOptionalInt,
			"1": parseInt,
		},
		"number": {
			"+": parseNumbers,
			"*": parseOptionalNumbers,
			"?": parseOptionalNumber,
			"1": parseNumber,
		},
	}

	nParsers := map[string]func(int) parser{
		"string": parseNStrings,
		"int":    parseNInts,
		"number": parseNNumbers,
	}

	switch t := option.(type) {
	case lua.LString:
		if p, ok := parsers[typ][string(t)]; ok {
			return p, nil
		}
	case lua.LNumber:
		switch {
		case int(t) == 1:
			return parsers[typ]["1"], nil
		case int(t) > 1:
			return nParsers[typ](int(t)), nil
		}
	}

	return nil, fmt.Errorf("nargs should be an integer or one of '?', '*', or '+'")
}

func getShortUsageFn(option lua.LValue) (shortUsage, error) {
	su := map[string]shortUsage{
		"?": func(name string) string {
			return fmt.Sprintf("[%v] ", name)
		},
		"1": func(name string) string {
			return fmt.Sprintf("%v ", name)
		},

		"*": func(name string) string {
			return fmt.Sprintf("[%v...] ", name)
		},
		"+": func(name string) string {
			return fmt.Sprintf("%v [%v...] ", name, name)
		},
	}

	shortNUsage := func(n int) shortUsage {
		return func(name string) string {
			usage := ""
			for i := 0; i < n; i++ {
				usage = fmt.Sprintf("%v %v ", usage, name)
			}
			return usage
		}
	}

	switch t := option.(type) {
	case lua.LString:
		if p, ok := su[string(t)]; ok {
			return p, nil
		}
	case lua.LNumber:
		switch {
		case int(t) == 1:
			return su["1"], nil
		case int(t) > 1:
			return shortNUsage(int(t)), nil
		}
	}

	return nil, fmt.Errorf("nargs should be an integer or one of '?', '*', or '+'")
}
