package gluaflag

import "testing"

func TestIntArg(t *testing.T) {
	src := `
	local flag = require('flag')
	arg = {"test", "-name", "foo", "2"}
	fs = flag.new()
	fs:string("name", "foo", "String help string")
	fs:intArg("title", 1, "Title")
	flags = fs:parse(arg)
	assert(flags.title == 2, "expected title to be 2")
	`

	doString(src, t)
}

func TestOptionalIntArg(t *testing.T) {
	src := `
	local flag = require('flag')
	arg = {"-name", "foo", 2}
	arg[0] = "subcommand"
	fs = flag.new()
	fs:string("name", "foo", "String help string")
	fs:intArg("title", "?", "Title")
	flags = fs:parse(arg)
	assert(flags.title == 2, "expected flags.title to be '2'")
	`
	doString(src, t)
}

func TestOptionalIntsArg(t *testing.T) {
	src := `
	local flag = require('flag')
	arg = {"-name", "foo", 2}
	arg[0] = "subcommand"
	fs = flag.new()
	fs:string("name", "foo", "String help string")
	fs:intArg("title", "*", "Title")
	flags = fs:parse(arg)
	assert(type(flags.title) == "table", "expected flags.title to be a 'table'")
	assert(#flags.title == 1, "expected flags.title to have length == 1")
	assert(flags.title[1] == 2, "expected flags.title[1] to be '2'")
	`
	doString(src, t)
}

func TestIntsArg(t *testing.T) {
	src := `
	local flag = require('flag')
	arg = {"-name", "foo", 3, 4}
	arg[0] = "subcommand"
	fs = flag.new()
	fs:string("name", "foo", "String help string")
	fs:intArg("title", "+", "Title")
	flags = fs:parse(arg)
	assert(type(flags.title) == "table", "expected flags.title to be a 'table'")
	assert(#flags.title == 2, "expected flags.title to have length == 1")
	assert(flags.title[1] == 3, "expected flags.title[1] to be '3'")
	assert(flags.title[2] == 4, "expected flags.title[2] to be '4', got " .. flags.title[2])
	`
	doString(src, t)
}

func TestIntsArgToFew(t *testing.T) {
	src := `
	local flag = require('flag')
	arg = {"-name", "foo"}
	arg[0] = "subcommand"
	fs = flag.new()
	fs:string("name", "foo", "String help string")
	fs:intArg("title", "+", "Title")
	ok, err = pcall(function() fs:parse(arg) end)
	print(err)
	`
	expected := "<string>:8: argument title: expected at least one integer"
	stdout, _ := doString(src, t)
	if stdout != expected {
		t.Errorf("expected: `%v`\ngot: `%v`\nsrc: `%v`", expected, stdout, src)
	}
}

func TestNIntsArgument(t *testing.T) {
	src := `
	local flag = require('flag')
	arg = {"-name", "foo", "1", "2"}
	arg[0] = "subcommand"
	fs = flag.new()
	fs:string("name", "foo", "String help string")
	fs:intArg("title", 2, "Title")
	flags = fs:parse(arg)
	assert(type(flags.title) == "table", "expected flags.title to be a 'table'")
	assert(#flags.title == 2, "expected flags.title to have length == 1")
	assert(flags.title[1] == 1, "expected flags.title[1] to be '1'")
	assert(flags.title[2] == 2, "expected flags.title[2] to be '2', got " .. flags.title[2])
	`
	doString(src, t)
}

func TestNIntsArgToFew(t *testing.T) {
	src := `
	local flag = require('flag')
	arg = {"-name", "foo", "2"}
	arg[0] = "subcommand"
	fs = flag.new()
	fs:string("name", "foo", "String help string")
	fs:intArg("title", 2, "Title")
	ok, err = pcall(function() fs:parse(arg) end)
	print(err)
	`
	expected := "<string>:8: argument title: expected 2 integers"
	stdout, _ := doString(src, t)
	if stdout != expected {
		t.Errorf("expected: `%v`\ngot: `%v`\nsrc: `%v`", expected, stdout, src)
	}
}

func TestIntArgWithString(t *testing.T) {
	src := `
	local flag = require('flag')
	arg = {"-name", "foo", "bar"}
	arg[0] = "subcommand"
	fs = flag.new()
	fs:string("name", "foo", "String help string")
	fs:intArg("title", 1, "Title")
	ok, err = pcall(function() fs:parse(arg) end)
	print(err)
	`
	expected := "<string>:8: argument title: invalid integer value: bar"
	stdout, _ := doString(src, t)
	if stdout != expected {
		t.Errorf("expected: `%v`\ngot: `%v`\nsrc: `%v`", expected, stdout, src)
	}
}
