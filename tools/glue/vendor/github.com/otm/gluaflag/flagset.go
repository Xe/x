package gluaflag

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/yuin/gopher-lua"
)

var flagSetFuncs = map[string]lua.LGFunction{
	"number":    number,
	"numbers":   numbers,
	"int":       integer,
	"ints":      integers,
	"string":    str,
	"strings":   strs,
	"bool":      boolean,
	"stringArg": stringArgument,
	"intArg":    intArgument,
	"numberArg": numberArgument,
	"parse":     parse,
	"compgen":   compgen,
	"usage":     usage,
}

// FlagSet is the background userdata component
type FlagSet struct {
	name      string
	fs        *flag.FlagSet
	flags     flgs
	arguments arguments
	output    io.Writer
}

// New returns a new flagset userdata
func New(L *lua.LState, name string) *lua.LUserData {
	f := flag.NewFlagSet(name, flag.ContinueOnError)

	flags := &FlagSet{
		name:      name,
		fs:        f,
		flags:     make(flgs),
		arguments: make(arguments, 0),
		output:    os.Stderr,
	}

	flags.fs.Usage = func() {
		fmt.Fprint(flags.output, flags.Usage())
	}

	ud := L.NewUserData()
	ud.Value = flags
	L.SetMetatable(ud, L.GetTypeMetatable(luaFlagSetTypeName))
	return ud
}

func new(L *lua.LState) int {
	var d lua.LValue = lua.LString("")
	larg := L.GetGlobal("arg")
	if targ, ok := larg.(*lua.LTable); ok {
		d = targ.RawGetInt(0)
	}
	name := L.OptString(1, d.String())

	L.Push(New(L, name))

	return 1
}

// Usage returns the usage message for the flag set
func (fs *FlagSet) Usage() string {
	buff := &bytes.Buffer{}

	buff.WriteString(fmt.Sprintf("usage: %v\n", fs.ShortUsage()))

	fs.fs.SetOutput(buff)
	defer fs.fs.SetOutput(os.Stderr)
	fs.fs.PrintDefaults()

	for _, arg := range fs.arguments {
		buff.WriteString(arg.generateUsage())
	}

	return buff.String()
}

// ShortUsage returns the usage string for a flagset
func (fs *FlagSet) ShortUsage() string {
	buff := &bytes.Buffer{}
	buff.WriteString(fs.name)
	if len(fs.flags) > 0 {
		buff.WriteString(fmt.Sprintf(" [options]"))
	}
	for _, arg := range fs.arguments {
		buff.WriteString(" ")
		buff.WriteString(arg.shortUsage(arg.name))
	}

	return buff.String()
}

// FlagDefaults returns the flagsets help string for the flags
func (fs *FlagSet) FlagDefaults() string {
	buff := &bytes.Buffer{}
	fs.fs.SetOutput(buff)
	defer fs.fs.SetOutput(os.Stderr)
	fs.fs.PrintDefaults()

	return buff.String()
}

// ArgDefaults returns the flagsets help string for the possitional arguments
func (fs *FlagSet) ArgDefaults() string {
	buff := &bytes.Buffer{}
	for _, arg := range fs.arguments {
		buff.WriteString(arg.generateUsage())
	}

	return buff.String()
}

func (fs *FlagSet) printFlags() string {
	var s []string
	fs.fs.VisitAll(func(fl *flag.Flag) {
		s = append(s, "-"+fl.Name)
	})
	return strings.Join(s, "\n")
}

func (fs *FlagSet) getFlags() []string {
	var s []string
	fs.fs.VisitAll(func(fl *flag.Flag) {
		s = append(s, "-"+fl.Name)
	})
	return s
}

// Compgen returns a string with possible options for the flag
func (fs *FlagSet) Compgen(L *lua.LState, compCWords int, compWords []string) []string {
	if compCWords == 1 && len(compWords) == 1 {
		return fs.getArguments(compCWords, compWords, L)
	}

	if compCWords <= len(compWords) {
		prev := compWords[compCWords-1]
		if string(prev[0]) == "-" {
			fl := fs.fs.Lookup(prev[1:len(prev)])
			v, ok := fs.flags[fl.Name]
			if !ok {
				return []string{}
			}
			switch value := v.value.(type) {
			case *bool:
				if string(compWords[len(compWords)-1][0]) == "-" {
					return fs.getFlags()
				}
				return []string{}
			case *string, *float64, *int:
				word := compWords[len(compWords)-1]
				if compCWords == len(compWords) {
					word = ""
				}

				table := L.NewTable()
				fs.fs.Visit(func(f *flag.Flag) {
					table.RawSetString(f.Name, lua.LString(f.Value.String()))
				})

				raw := L.NewTable()
				for i, word := range compWords {
					if i == 0 {
						raw.RawSet(lua.LNumber(0), lua.LString(word))
						continue
					}
					raw.Append(lua.LString(word))
				}

				stack := L.GetTop()
				if err := L.CallByParam(lua.P{
					Fn:      v.compFn,
					NRet:    -1,
					Protect: true,
				}, lua.LString(word), table, raw); err != nil {
					fmt.Fprintf(os.Stderr, "%v\n", err)
					os.Exit(1)
				}
				stack = L.GetTop() - stack

				if stack == 1 {
					res := L.Get(-1)
					L.Pop(1)
					switch r := res.(type) {
					case *lua.LTable:
						return toStringSlice(r)
					case lua.LString:
						s := string(r)
						if strings.Index(s, "\n") > 0 {
							return strings.Split(s, "\n")
						}
						return []string{s}
					case *lua.LFunction:
						return forEachStrings(L, r)
					default:
						L.RaiseError("unknown type: %T", r)
					}
				}

				res := []string{}
				for i := 1; i <= stack; i++ {
					res = append(res, L.Get(-i).String())
				}
				L.Pop(stack)
				return res

			default:
				L.RaiseError("not implemented type: %T", value)
				return []string{}
			}
		} else if string(compWords[len(compWords)-1][0]) == "-" {
			// current argument starts with "-"
			return fs.getFlags()
		} else { // argument
			return fs.getArguments(compCWords, compWords, L)
		}
	}
	return []string{}
}

func (fs *FlagSet) getArguments(compCWords int, compWords []string, L *lua.LState) []string {
	err := fs.fs.Parse(compWords[1:len(compWords)])
	if err != nil {
		return []string{}
	}
	nargs := fs.fs.NArg()
	if nargs == len(fs.arguments) {
		nargs--
	}
	word := compWords[len(compWords)-1]
	if compCWords == len(compWords) {
		word = ""
	}

	table := L.NewTable()
	fs.fs.Visit(func(f *flag.Flag) {
		table.RawSetString(f.Name, lua.LString(f.Value.String()))
	})

	raw := L.NewTable()
	for i, word := range compWords {
		if i == 0 {
			raw.RawSet(lua.LNumber(0), lua.LString(word))
			continue
		}
		raw.Append(lua.LString(word))
	}

	if nargs >= len(fs.arguments) || nargs < 0 {
		return []string{}
	}

	// stack is needed to know how the stack grows
	stack := L.GetTop()
	if err := L.CallByParam(lua.P{
		Fn:      fs.arguments[nargs].compFn,
		NRet:    -1,
		Protect: true,
	}, lua.LString(word), table, raw); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	stack = L.GetTop() - stack

	if stack == 1 {
		res := L.Get(-1)
		L.Pop(1)
		switch r := res.(type) {
		case *lua.LTable:
			return toStringSlice(r)
		case *lua.LString:
			s := string(*r)
			if strings.Index(s, "\n") > 0 {
				return strings.Split(s, "\n")
			}
			return []string{s}
		case *lua.LFunction:
			return forEachStrings(L, r)
		default:
			L.RaiseError("unknown type: %T", r)
		}
	}

	res := []string{}
	for i := 1; i <= stack; i++ {
		res = append(res, L.Get(-i).String())
	}
	L.Pop(stack)
	return res

}

func usage(L *lua.LState) int {
	gf := checkFlagSet(L, 1)
	L.Push(lua.LString(gf.Usage()))
	return 1
}

func compgen(L *lua.LState) int {
	ud := L.CheckUserData(1)
	compCWords := L.CheckInt(2)
	compWords := L.CheckTable(3)

	gf, ok := ud.Value.(*FlagSet)
	if !ok {
		L.RaiseError("Expected gluaflag userdata, got `%T`", ud.Value)
	}

	comp := gf.Compgen(L, compCWords, toStringSlice(compWords))

	L.Push(toTable(L, comp))
	return 1
}

func number(L *lua.LState) int {
	ud := L.CheckUserData(1)
	name := L.CheckString(2)
	value := L.CheckNumber(3)
	usage := L.CheckString(4)
	cf := L.OptFunction(5, L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString(""))
		return 1
	}))

	gf, ok := ud.Value.(*FlagSet)
	if !ok {
		L.RaiseError("Expected gluaflag userdata, got `%T`", ud.Value)
	}

	f := gf.fs.Float64(name, float64(value), usage)
	gf.flags[name] = &flg{
		name:   name,
		value:  f,
		usage:  usage,
		compFn: cf,
	}

	L.Push(gf.flags[name].userdata(L))
	return 1
}

func numbers(L *lua.LState) int {
	ud := L.CheckUserData(1)
	name := L.CheckString(2)
	usage := L.CheckString(3)
	cf := L.OptFunction(4, L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString(""))
		return 1
	}))

	gf, ok := ud.Value.(*FlagSet)
	if !ok {
		L.RaiseError("Expected gluaflag userdata, got `%T`", ud.Value)
	}

	var numbers numberslice
	gf.fs.Var(&numbers, name, usage)
	gf.flags[name] = &flg{
		name:   name,
		value:  &numbers,
		usage:  usage,
		compFn: cf,
	}

	return 0
}

func integer(L *lua.LState) int {
	ud := L.CheckUserData(1)
	name := L.CheckString(2)
	value := L.CheckInt(3)
	usage := L.CheckString(4)
	cf := L.OptFunction(5, L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString(""))
		return 1
	}))

	gf, ok := ud.Value.(*FlagSet)
	if !ok {
		L.RaiseError("Expected gluaflag userdata, got `%T`", ud.Value)
	}

	f := gf.fs.Int(name, int(value), usage)
	gf.flags[name] = &flg{
		name:   name,
		value:  f,
		usage:  usage,
		compFn: cf,
	}

	return 0
}

func integers(L *lua.LState) int {
	ud := L.CheckUserData(1)
	name := L.CheckString(2)
	usage := L.CheckString(3)
	cf := L.OptFunction(4, L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString(""))
		return 1
	}))

	gf, ok := ud.Value.(*FlagSet)
	if !ok {
		L.RaiseError("Expected gluaflag userdata, got `%T`", ud.Value)
	}

	var ints intslice
	gf.fs.Var(&ints, name, usage)
	gf.flags[name] = &flg{
		name:   name,
		value:  &ints,
		usage:  usage,
		compFn: cf,
	}

	return 0
}

func str(L *lua.LState) int {
	ud := L.CheckUserData(1)
	name := L.CheckString(2)
	value := L.CheckString(3)
	usage := L.CheckString(4)
	cf := L.OptFunction(5, L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString(""))
		return 1
	}))

	gf, ok := ud.Value.(*FlagSet)
	if !ok {
		L.RaiseError("Expected gluaflag userdata, got `%T`", ud.Value)
	}

	f := gf.fs.String(name, value, usage)
	gf.flags[name] = &flg{
		name:   name,
		value:  f,
		usage:  usage,
		compFn: cf,
	}

	return 0
}

func strs(L *lua.LState) int {
	ud := L.CheckUserData(1)
	name := L.CheckString(2)
	usage := L.CheckString(3)
	cf := L.OptFunction(4, L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString(""))
		return 1
	}))

	gf, ok := ud.Value.(*FlagSet)
	if !ok {
		L.RaiseError("Expected gluaflag userdata, got `%T`", ud.Value)
	}

	var strs stringslice
	gf.fs.Var(&strs, name, usage)
	gf.flags[name] = &flg{
		name:   name,
		value:  &strs,
		usage:  usage,
		compFn: cf,
	}

	return 0
}

func boolean(L *lua.LState) int {
	ud := L.CheckUserData(1)
	name := L.CheckString(2)
	value := L.CheckBool(3)
	usage := L.CheckString(4)

	gf, ok := ud.Value.(*FlagSet)
	if !ok {
		L.RaiseError("Expected gluaflag userdata, got `%T`", ud.Value)
	}

	f := gf.fs.Bool(name, value, usage)
	gf.flags[name] = &flg{
		name:   name,
		value:  f,
		usage:  usage,
		compFn: nil,
	}

	return 0
}

func stringArgument(L *lua.LState) int {
	ud := L.CheckUserData(1)
	name := L.CheckString(2)
	times := L.CheckAny(3)
	usage := L.CheckString(4)
	cf := L.OptFunction(5, L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString(""))
		return 1
	}))

	a := &argument{
		name:   name,
		usage:  usage,
		compFn: cf,
		typ:    "string",
	}

	parser, err := getParser("string", times)
	if err != nil {
		L.RaiseError(err.Error())
	}
	a.parser = parser

	su, err := getShortUsageFn(times)
	if err != nil {
		L.RaiseError(err.Error())
	}
	a.shortUsage = su

	gf, ok := ud.Value.(*FlagSet)
	if !ok {
		L.RaiseError("Expected gluaflag userdata, got `%T`", ud.Value)
	}

	udPossitionalArgument := L.NewUserData()
	udPossitionalArgument.Value = a
	L.SetMetatable(ud, L.GetTypeMetatable(luaFlagSetTypeName))

	gf.arguments = append(gf.arguments, a)

	L.Push(udPossitionalArgument)
	return 1
}

func intArgument(L *lua.LState) int {
	ud := L.CheckUserData(1)
	name := L.CheckString(2)
	times := L.CheckAny(3)
	usage := L.CheckString(4)
	cf := L.OptFunction(5, L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString(""))
		return 1
	}))

	a := &argument{
		name:   name,
		usage:  usage,
		compFn: cf,
		typ:    "int",
	}

	parser, err := getParser("int", times)
	if err != nil {
		L.RaiseError(err.Error())
	}
	a.parser = parser

	su, err := getShortUsageFn(times)
	if err != nil {
		L.RaiseError(err.Error())
	}
	a.shortUsage = su

	gf, ok := ud.Value.(*FlagSet)
	if !ok {
		L.RaiseError("Expected gluaflag userdata, got `%T`", ud.Value)
	}

	udPossitionalArgument := L.NewUserData()
	udPossitionalArgument.Value = a
	L.SetMetatable(ud, L.GetTypeMetatable(luaFlagSetTypeName))

	gf.arguments = append(gf.arguments, a)

	L.Push(udPossitionalArgument)
	return 1
}

func numberArgument(L *lua.LState) int {
	ud := L.CheckUserData(1)
	name := L.CheckString(2)
	times := L.CheckAny(3)
	usage := L.CheckString(4)
	cf := L.OptFunction(5, L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString(""))
		return 1
	}))

	a := &argument{
		name:   name,
		usage:  usage,
		compFn: cf,
		typ:    "number",
	}

	parser, err := getParser("number", times)
	if err != nil {
		L.RaiseError(err.Error())
	}
	a.parser = parser

	su, err := getShortUsageFn(times)
	if err != nil {
		L.RaiseError(err.Error())
	}
	a.shortUsage = su

	gf, ok := ud.Value.(*FlagSet)
	if !ok {
		L.RaiseError("Expected gluaflag userdata, got `%T`", ud.Value)
	}

	udPossitionalArgument := L.NewUserData()
	udPossitionalArgument.Value = a
	L.SetMetatable(ud, L.GetTypeMetatable(luaFlagSetTypeName))

	gf.arguments = append(gf.arguments, a)

	L.Push(udPossitionalArgument)
	return 1
}

func possitionalInt(L *lua.LState) int {
	ud := L.CheckUserData(1)
	name := L.CheckString(2)
	times := L.CheckAny(3)
	usage := L.CheckString(4)
	cf := L.OptFunction(5, L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString(""))
		return 1
	}))

	a := &argument{
		name:   name,
		usage:  usage,
		typ:    "int",
		compFn: cf,
	}

	switch t := times.(type) {
	case lua.LString:
		switch t {
		case "+":
			a.glob = true
		case "*":
			a.glob = true
			a.optional = true
		case "?":
			a.times = 1
			a.optional = true
		default:
			L.RaiseError("nargs should be an integer or one of '?', '*', or '+'")
		}
	case lua.LNumber:
		if int(t) < 1 {
			L.RaiseError("nargs should be an integer or one of '?', '*', or '+'")
		}
		a.times = int(t)
	default:
		L.RaiseError("nargs should be an integer or one of '?', '*', or '+'")
	}

	gf, ok := ud.Value.(*FlagSet)
	if !ok {
		L.RaiseError("Expected gluaflag userdata, got `%T`", ud.Value)
	}

	udPossitionalArgument := L.NewUserData()
	udPossitionalArgument.Value = a
	L.SetMetatable(ud, L.GetTypeMetatable(luaFlagSetTypeName))

	gf.arguments = append(gf.arguments, a)

	L.Push(udPossitionalArgument)
	return 1
}

// Parse the command line parameters
func Parse(L *lua.LState, ud *lua.LUserData, args []string) (*lua.LTable, error) {
	gf, ok := ud.Value.(*FlagSet)
	if !ok {
		L.RaiseError("Expected gluaflag userdata, got `%T`", ud.Value)
		return nil, ErrUserDataType
	}

	gf.fs.SetOutput(ioutil.Discard)
	gf.output = ioutil.Discard
	err := gf.fs.Parse(args)
	if err != nil {
		return nil, err
	}

	t := L.NewTable()
	for f, v := range gf.flags {
		switch value := v.value.(type) {
		case *float64:
			t.RawSetString(f, lua.LNumber(*value))
		case *string:
			t.RawSetString(f, lua.LString(*value))
		case *bool:
			t.RawSetString(f, lua.LBool(*value))
		case *int:
			t.RawSetString(f, lua.LNumber(*value))
		case *intslice:
			t.RawSetString(f, value.Table(L))
		case *numberslice:
			t.RawSetString(f, value.Table(L))
		case *stringslice:
			t.RawSetString(f, value.Table(L))
		default:
			L.RaiseError("unknown type: `%T`", v)
		}
	}

	// nothing defined for possitional arguments, just copy them
	if len(gf.arguments) == 0 {
		for _, v := range gf.fs.Args() {
			t.Append(lua.LString(v))
		}
		return t, nil
	}

	// TODO: refactor to a function in arguments
	args = gf.fs.Args()
	for _, arg := range gf.arguments {
		args, err = arg.parse(args, L)
		if err != nil {
			return nil, fmt.Errorf("argument %v: %v", arg.name, err.Error())
		}
		t.RawSetString(arg.name, arg.toLValue(L))
	}

	if len(args) > 0 {
		L.RaiseError("unknown argument: %v", args)
	}

	return t, nil
}

func parse(L *lua.LState) int {
	ud := L.CheckUserData(1)
	args := L.CheckTable(2)

	a := toStringSlice(args)

	t, err := Parse(L, ud, a[1:len(a)])
	if err != nil {
		L.RaiseError("%v", err)
	}

	L.Push(t)
	return 1
}
