package gluaflag

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/yuin/gopher-lua"
)

func captureStdout() func() string {
	old := os.Stdout // keep backup of the real stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outC := make(chan string)
	// copy the output in a separate goroutine so printing can't block indefinitely
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		outC <- buf.String()
	}()

	return func() string {
		w.Close()
		os.Stdout = old // restoring the real stdout
		return <-outC
	}
}

func captureStderr() func() string {
	old := os.Stderr // keep backup of the real Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	outC := make(chan string)
	// copy the output in a separate goroutine so printing can't block indefinitely
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		outC <- buf.String()
	}()

	return func() string {
		w.Close()
		os.Stderr = old // restoring the real stdout
		return <-outC
	}
}

func doString(src string, t *testing.T) (stdout, stderr string) {
	L := lua.NewState()
	defer L.Close()
	L.PreloadModule("flag", Loader)

	restoreStdout := captureStdout()
	restoreStderr := captureStderr()
	err := L.DoString(src)
	stdout = restoreStdout()
	stderr = restoreStderr()
	if err != nil {
		t.Errorf("runtime error: %v", err)
	}

	if len(stdout) > 0 {
		stdout = stdout[0 : len(stdout)-1]
	}

	if len(stderr) > 0 {
		stderr = stderr[0 : len(stderr)-1]
	}

	return stdout, stderr
}

func TestUsage(t *testing.T) {
	// TODO: Fix output
	src := `
	local flag = require('flag')
	arg = {"-foo"}
	arg[0] = "subcmd"
	fs = flag.new("subcommand")
	fs:number("times", 1, "Number help string")
	function fail()
		flags = fs:parse(arg)
	end
	ok, err = pcall(fail)

	print(err)
	`

	expected := strings.Join([]string{
		"usage: subcommand [options]",
		"  -times float",
		"    	Number help string (default 1)",
		"<string>:8: flag provided but not defined: -foo",
	}, "\n")
	expectedStderr := strings.Join([]string{
		"flag provided but not defined: -foo",
	}, "\n")
	got, stderr := doString(src, t)

	if got != expected || stderr != expectedStderr {
		t.Errorf("expected stdout: `%v`\ngot: `%v`\nexpected stderr: `%v`\ngot: `%v`\nsrc: `%v`", expected, got, expectedStderr, stderr, src)
	}
}

func TestNumberFlag(t *testing.T) {
	src := `
	local flag = require('flag')
	arg = {"-times", "2"}
	arg[0] = "subcmd"
	fs = flag.new()
	fs:number("times", 1, "Number help string")
	flags = fs:parse(arg)

	print(flags.times)
	print(type(flags.times))
	`

	expected := "2\nnumber"
	got, _ := doString(src, t)

	if got != expected {
		t.Errorf("expected: `%v`, got: `%v`\nsrc: `%v`", expected, got, src)
	}
}

func TestIntFlag(t *testing.T) {
	src := `
	local flag = require('flag')
	arg = {"-times", "2"}
	arg[0] = "subcmd"
	fs = flag.new()
	fs:int("times", 1, "Number help string")
	flags = fs:parse(arg)

	print(flags.times)
	print(type(flags.times))
	`

	expected := "2\nnumber"
	got, _ := doString(src, t)

	if got != expected {
		t.Errorf("expected: `%v`, got: `%v`\nsrc: `%v`", expected, got, src)
	}
}

func TestIntSliceFlag(t *testing.T) {
	src := `
	local flag = require('flag')
	arg = {"-times", "2", "-times", "4"}
	arg[0] = "subcmd"
	fs = flag.new()
	fs:ints("times", "Number help string")
	flags = fs:parse(arg)

	print(type(flags.times))
	print(table.concat(flags.times, ","))
	`

	expected := "table\n2,4"
	got, _ := doString(src, t)

	if got != expected {
		t.Errorf("expected: `%v`, got: `%v`\nsrc: `%v`", expected, got, src)
	}
}

func TestNumberFlagCompgen(t *testing.T) {
	src := `
	local flag = require('flag')
	arg = {"-times"}
	arg[0] = "subcommand"
	fs = flag.new()
	local function compgen()
		return "1 2 3"
	end
	fs:number("times", 1, "Number help string", compgen)
	flags = fs:compgen(2, arg)

	print(table.concat(flags, " "))
	`

	expected := strings.Join([]string{
		"1",
		"2",
		"3",
	}, " ")
	got, _ := doString(src, t)

	if got != expected {
		t.Errorf("expected: `%v`, got: `%v`\nsrc: `%v`", expected, got, src)
	}
}

func TestNumberFlagAndArgs(t *testing.T) {
	src := `
	local flag = require('flag')
	arg = {"-times", "2", "foo"}
	arg[0] = "subcmd"
	fs = flag.new()
	fs:number("times", 1, "Number help string")
	flags = fs:parse(arg)

	print(flags.times)
	print(type(flags.times))
	for i, v in ipairs(flags) do
		print(i .. "=" .. v)
	end
	print(flags[1])
	`

	expected := strings.Join([]string{
		"2",
		"number",
		"1=foo",
		"foo",
	}, "\n")
	got, _ := doString(src, t)

	if got != expected {
		t.Errorf("expected: `%v`, got: `%v`\nsrc: `%v`", expected, got, src)
	}
}

func TestNumberSliceFlag(t *testing.T) {
	src := `
	local flag = require('flag')
	arg = {"-times", "2.4", "-times", "4.3"}
	arg[0] = "subcmd"
	fs = flag.new()
	fs:numbers("times", "Number help string")
	flags = fs:parse(arg)

	print(type(flags.times))
	print(table.concat(flags.times, ","))
	`

	expected := "table\n2.4,4.3"
	got, _ := doString(src, t)

	if got != expected {
		t.Errorf("expected: `%v`, got: `%v`\nsrc: `%v`", expected, got, src)
	}
}

func TestStringFlag(t *testing.T) {
	src := `
	local flag = require('flag')
	arg = {"-name", "bar"}
	arg[0] = "subcmd"
	fs = flag.new()
	fs:string("name", "foo", "String help string")
	flags = fs:parse(arg)

	print(flags.name)
	print(type(flags.name))
	`

	expected := strings.Join([]string{
		"bar",
		"string",
	}, "\n")
	got, _ := doString(src, t)

	if got != expected {
		t.Errorf("expected: `%v`, got: `%v`\nsrc: `%v`", expected, got, src)
	}
}

func TestStringFlagCompgen(t *testing.T) {
	src := `
	local flag = require('flag')
	local arg = {"-name"}
	arg[0] = "subcommand"
	local fs = flag.new()
	local function compgen()
		return {"fii", "foo", "fum"}
	end
	fs:string("name", "foo", "String help string", compgen)
	flags = fs:compgen(2, arg)

	print(table.concat(flags, "\n"))
	`

	expected := strings.Join([]string{
		"fii",
		"foo",
		"fum",
	}, "\n")
	got, _ := doString(src, t)

	if got != expected {
		t.Errorf("expected: `%v`, got: `%v`\nsrc: `%v`", expected, got, src)
	}
}

func TestStringSliceFlag(t *testing.T) {
	src := `
	local flag = require('flag')
	arg = {"-times", "foo", "-times", "bar"}
	arg[0] = "subcmd"
	fs = flag.new()
	fs:strings("times", "Number help string")
	flags = fs:parse(arg)

	print(type(flags.times))
	print(table.concat(flags.times, ","))
	`

	expected := "table\nfoo,bar"
	got, _ := doString(src, t)

	if got != expected {
		t.Errorf("expected: `%v`, got: `%v`\nsrc: `%v`", expected, got, src)
	}
}

func TestStringSliceNoValues(t *testing.T) {
	src := `
	local flag = require('flag')
	arg = {}
	arg[0] = "subcmd"
	fs = flag.new()
	fs:strings("times", "Number help string")
	flags = fs:parse(arg)

	print(type(flags.times))
	print(table.concat(flags.times, ","))
	`

	expected := "table\n"
	got, _ := doString(src, t)

	if got != expected {
		t.Errorf("expected: `%v`, got: `%v`\nsrc: `%v`", expected, got, src)
	}
}

func TestBoolFlag(t *testing.T) {
	src := `
	local flag = require('flag')
	arg = {"-q"}
	arg[0] = "subcmd"
	fs = flag.new()
	fs:bool("q", false, "Bool help string")
	flags = fs:parse(arg)

	print(flags.q)
	print(type(flags.q))
	`

	expected := strings.Join([]string{
		"true",
		"boolean",
	}, "\n")
	got, _ := doString(src, t)

	if got != expected {
		t.Errorf("expected: `%v`, got: `%v`\nsrc: `%v`", expected, got, src)
	}
}

func TestStringArg(t *testing.T) {
	src := `
	local flag = require('flag')
	arg = {"test", "-name", "foo", "mr"}
	fs = flag.new()
	local function compgen()
		return "fii foo fum"
	end
	fs:string("name", "foo", "String help string", compgen)
	fs:stringArg("title", 1, "Title", function()
		return "mr miss mrs"
	end)
	flags = fs:parse(arg)

	function pairsByKeys(t)
		local a = {}
		for n in pairs(t) do table.insert(a, n) end
		table.sort(a)
		local i = 0			-- iterator variable
		local iter = function ()	 -- iterator function
			i = i + 1
			if a[i] == nil then return nil
			else return a[i], t[a[i]]
			end
		end
		return iter
	end

	for k, v in pairsByKeys(flags) do
		print(k .. " " .. v)
	end
	`

	expected := strings.Join([]string{
		"name foo",
		"title mr",
	}, "\n")
	got, _ := doString(src, t)

	if got != expected {
		t.Errorf("expected: `%v`, got: `%v`\nsrc: `%v`", expected, got, src)
	}
}

func TestOptionalStringArg(t *testing.T) {
	src := `
	local flag = require('flag')
	arg = {"-name", "foo", "bar"}
	arg[0] = "subcommand"
	fs = flag.new()
	fs:string("name", "foo", "String help string")
	fs:stringArg("title", "?", "Title")
	flags = fs:parse(arg)
	assert(flags.title == "bar", "expected flags.title to be 'bar'")
	`
	doString(src, t)
}

func TestOptionalStringsArg(t *testing.T) {
	src := `
	local flag = require('flag')
	arg = {"-name", "foo", "bar"}
	arg[0] = "subcommand"
	fs = flag.new()
	fs:string("name", "foo", "String help string")
	fs:stringArg("title", "*", "Title")
	flags = fs:parse(arg)
	assert(type(flags.title) == "table", "expected flags.title to be a 'table'")
	assert(#flags.title == 1, "expected flags.title to have length == 1")
	assert(flags.title[1] == "bar", "expected flags.title[1] to be 'bar'")
	`
	doString(src, t)
}

func TestStringsArg(t *testing.T) {
	src := `
	local flag = require('flag')
	arg = {"-name", "foo", "bar"}
	arg[0] = "subcommand"
	fs = flag.new()
	fs:string("name", "foo", "String help string")
	fs:stringArg("title", "+", "Title")
	flags = fs:parse(arg)
	assert(type(flags.title) == "table", "expected flags.title to be a 'table'")
	assert(#flags.title == 1, "expected flags.title to have length == 1")
	assert(flags.title[1] == "bar", "expected flags.title[1] to be 'bar'")
	`
	doString(src, t)
}

func TestStringsArgToFew(t *testing.T) {
	src := `
	local flag = require('flag')
	arg = {"-name", "foo"}
	arg[0] = "subcommand"
	fs = flag.new()
	fs:string("name", "foo", "String help string")
	fs:stringArg("title", "+", "Title")
	ok, err = pcall(function() fs:parse(arg) end)
	print(err)
	`
	expected := "<string>:8: argument title: expected at least one string"
	stdout, _ := doString(src, t)
	if stdout != expected {
		t.Errorf("expected: `%v`\ngot: `%v`\nsrc: `%v`", expected, stdout, src)
	}
}

func TestNStringsArgument(t *testing.T) {
	src := `
	local flag = require('flag')
	arg = {"-name", "foo", "bar", "baz"}
	arg[0] = "subcommand"
	fs = flag.new()
	fs:string("name", "foo", "String help string")
	fs:stringArg("title", 2, "Title")
	flags = fs:parse(arg)
	assert(type(flags.title) == "table", "expected flags.title to be a 'table'")
	assert(#flags.title == 2, "expected flags.title to have length == 1")
	assert(flags.title[1] == "bar", "expected flags.title[1] to be 'bar'")
	assert(flags.title[2] == "baz", "expected flags.title[1] to be 'baz'")
	`
	doString(src, t)
}

func TestNStringsArgToFew(t *testing.T) {
	src := `
	local flag = require('flag')
	arg = {"-name", "foo", "bar"}
	arg[0] = "subcommand"
	fs = flag.new()
	fs:string("name", "foo", "String help string")
	fs:stringArg("title", 2, "Title")
	ok, err = pcall(function() fs:parse(arg) end)
	print(err)
	`
	expected := "<string>:8: argument title: expected 2 strings"
	stdout, _ := doString(src, t)
	if stdout != expected {
		t.Errorf("expected: `%v`\ngot: `%v`\nsrc: `%v`", expected, stdout, src)
	}
}

func TestStringArgumentCompgen(t *testing.T) {
	src := `
	local flag = require('flag')

	fs = flag.new("subcommand")
	local function compgen()
		return "fii", "foo", "fum"
	end

	res = {}
	fs:string("name", "foo", "String help string", compgen)
	fs:stringArg("mr", 1, "Title", function(arg, flags, raw)
		res.arg = arg
		res.flags = flags
		res.raw = raw
		return "mr", "miss", "mrs"
	end)

	local arg = {"-name", "foo", "m"}
	arg[0] = "subcommand"
	flags = fs:compgen(3, arg)
	assert(res.arg == "m", "expected arg to be 'm', got " .. res.arg)
	assert(res.flags.name == "foo", "expected flag.names to be 'foo', got " .. res.flags.name)
	assert(res.raw[0] == "subcommand", "expected raw[0] to be 'subcommand', got " .. res.raw[0])
	assert(res.raw[1] == "-name", "expected raw[1] to be '-name', got " .. res.raw[1])

	print(table.concat(flags, " "))

	`

	expected := strings.Join([]string{
		"mrs",
		"miss",
		"mr",
	}, " ")
	got, _ := doString(src, t)

	if got != expected {
		t.Errorf("expected: `%v`, got: `%v`\nsrc: `%v`", expected, got, src)
	}
}

func TeststringArgumentCompgen2(t *testing.T) {
	src := `
	local flag = require('flag')
	arg = {"-name", "foo", "m"}
	arg[0] = "subcommand"
	fs = flag.new()
	local function compgen()
		return "fii foo fum"
	end
	fs:string("name", "foo", "String help string", compgen)
	fs:stringArgument("mr", 1, "Title", function()
		return "mr miss mrs"
	end)
	flags = fs:compgen(3, arg)

	print(flags)
	`

	expected := strings.Join([]string{
		"mr",
		"miss",
		"mrs",
	}, " ")
	got, _ := doString(src, t)

	if got != expected {
		t.Errorf("expected: `%v`, got: `%v`\nsrc: `%v`", expected, got, src)
	}
}

func TestStringArgumentUsage(t *testing.T) {
	src := `
	local flag = require('flag')
	fs = flag.new("subcommand")
	fs:stringArg("title", 1, "Your title")

	print(fs:usage())
	`

	expected := strings.Join([]string{
		"usage: subcommand title ",
		"  title string",
		"    \tYour title\n",
	}, "\n")
	got, _ := doString(src, t)

	if got != expected {
		t.Errorf("expected stdout: `%v`\ngot: `%v`\nsrc: `%v`", expected, got, src)
	}
}
