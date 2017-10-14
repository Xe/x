package gluaflag

import "testing"

func TestNumberArg(t *testing.T) {
	src := `
	local flag = require('flag')
	arg = {"-name", "foo", "2.32"}
	arg[0] = "subcommand"
	fs = flag.new()
	fs:string("name", "foo", "String help string")
	fs:numberArg("title", 1, "Title")
	flags = fs:parse(arg)
	assert(flags.title == 2.32, "expected title to be 2.32")
	`

	doString(src, t)
}

func TestOptionalNumberArg(t *testing.T) {
	src := `
	local flag = require('flag')
	arg = {"-name", "foo", 2.45}
	arg[0] = "subcommand"
	fs = flag.new()
	fs:string("name", "foo", "String help string")
	fs:numberArg("title", "?", "Title")
	flags = fs:parse(arg)
	assert(flags.title == 2.45, "expected flags.title to be '2.45'")
	`
	doString(src, t)
}

func TestOptionalNubersArg(t *testing.T) {
	src := `
	local flag = require('flag')
	arg = {"-name", "foo", 2.55}
	arg[0] = "subcommand"
	fs = flag.new()
	fs:string("name", "foo", "String help string")
	fs:numberArg("title", "*", "Title")
	flags = fs:parse(arg)
	assert(type(flags.title) == "table", "expected flags.title to be a 'table'")
	assert(#flags.title == 1, "expected flags.title to have length == 1")
	assert(flags.title[1] == 2.55, "expected flags.title[1] to be '2.55'")
	`
	doString(src, t)
}

func TestNumbersArg(t *testing.T) {
	src := `
	local flag = require('flag')
	arg = {"-name", "foo", 3.33, 4.44}
	arg[0] = "subcommand"
	fs = flag.new()
	fs:string("name", "foo", "String help string")
	fs:numberArg("title", "+", "Title")
	flags = fs:parse(arg)
	assert(type(flags.title) == "table", "expected flags.title to be a 'table'")
	assert(#flags.title == 2, "expected flags.title to have length == 1")
	assert(flags.title[1] == 3.33, "expected flags.title[1] to be '3.33'")
	assert(flags.title[2] == 4.44, "expected flags.title[2] to be '4.44', got " .. flags.title[2])
	`
	doString(src, t)
}

func TestNumbersArgToFew(t *testing.T) {
	src := `
	local flag = require('flag')
	arg = {"-name", "foo"}
	arg[0] = "subcommand"
	fs = flag.new()
	fs:string("name", "foo", "String help string")
	fs:numberArg("title", "+", "Title")
	ok, err = pcall(function() fs:parse(arg) end)
	print(err)
	`
	expected := "<string>:8: argument title: expected at least one number"
	stdout, _ := doString(src, t)
	if stdout != expected {
		t.Errorf("expected: `%v`\ngot: `%v`\nsrc: `%v`", expected, stdout, src)
	}
}

func TestNNumbersArgument(t *testing.T) {
	src := `
	local flag = require('flag')
	arg = {"-name", "foo", "1.11", "2.22"}
	arg[0] = "subcommand"
	fs = flag.new()
	fs:string("name", "foo", "String help string")
	fs:numberArg("title", 2, "Title")
	flags = fs:parse(arg)
	assert(type(flags.title) == "table", "expected flags.title to be a 'table'")
	assert(#flags.title == 2, "expected flags.title to have length == 1")
	assert(flags.title[1] == 1.11, "expected flags.title[1] to be '1.11'")
	assert(flags.title[2] == 2.22, "expected flags.title[2] to be '2.22', got " .. flags.title[2])
	`
	doString(src, t)
}

func TestNNumbersArgToFew(t *testing.T) {
	src := `
	local flag = require('flag')
	arg = {"-name", "foo", "2.22"}
	arg[0] = "subcommand"
	fs = flag.new()
	fs:string("name", "foo", "String help string")
	fs:numberArg("title", 2, "Title")
	ok, err = pcall(function() fs:parse(arg) end)
	print(err)
	`
	expected := "<string>:8: argument title: expected 2 numbers"
	stdout, _ := doString(src, t)
	if stdout != expected {
		t.Errorf("expected: `%v`\ngot: `%v`\nsrc: `%v`", expected, stdout, src)
	}
}

func TestNumberArgWithString(t *testing.T) {
	src := `
	local flag = require('flag')
	arg = {"-name", "foo", "bar"}
	arg[0] = "subcommand"
	fs = flag.new()
	fs:string("name", "foo", "String help string")
	fs:numberArg("title", 1, "Title")
	ok, err = pcall(function() fs:parse(arg) end)
	print(err)
	`
	expected := "<string>:8: argument title: invalid number value: bar"
	stdout, _ := doString(src, t)
	if stdout != expected {
		t.Errorf("expected: `%v`\ngot: `%v`\nsrc: `%v`", expected, stdout, src)
	}
}
