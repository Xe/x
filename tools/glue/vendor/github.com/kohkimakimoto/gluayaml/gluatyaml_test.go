package gluayaml

import (
	"github.com/yuin/gopher-lua"
	"testing"
)

func TestLoader(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	L.PreloadModule("yaml", Loader)
	if err := L.DoString(`
local yaml = require("yaml")
	`); err != nil {
		t.Error(err)
	}
}

func TestParse(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	L.PreloadModule("yaml", Loader)
	if err := L.DoString(`
local yaml = require("yaml")

local str = [==[
- aaa
- bbb
- ccc
]==]

local tb, err = yaml.parse(str)
if tb == nil then
	error("parse error" .. err)
end

assert(tb[1] == "aaa")
assert(tb[2] == "bbb")
assert(tb[3] == "ccc")

	`); err != nil {
		t.Error(err)
	}
}

func TestParse2(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	L.PreloadModule("yaml", Loader)
	if err := L.DoString(`
local yaml = require("yaml")

local str = [==[
aaa: bbb
bbb: ccc
]==]

local tb, err = yaml.parse(str)
if tb == nil then
	error("parse error" .. err)
end

assert(tb.aaa == "bbb")
assert(tb.bbb == "ccc")

	`); err != nil {
		t.Error(err)
	}
}

func TestParse3(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	L.PreloadModule("yaml", Loader)
	if err := L.DoString(`
local yaml = require("yaml")

local str = [==[
aaa: true
bbb: false
]==]

local tb, err = yaml.parse(str)
if tb == nil then
	error("parse error" .. err)
end

assert(tb.aaa == true)
assert(tb.bbb == false)

	`); err != nil {
		t.Error(err)
	}
}

func TestParse4(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	L.PreloadModule("yaml", Loader)
	if err := L.DoString(`
local yaml = require("yaml")

local str = [==[
aaa: 1234
bbb: 1234.567
ccc: [aa, bb, cc]
ddd:
  -  aa
  -  bb
eee:
  aa : bb
  bb : cc

]==]

local tb, err = yaml.parse(str)
if tb == nil then
	error("parse error" .. err)
end

assert(tb.aaa == 1234)
assert(tb.bbb == 1234.567)
assert(tb.ccc[1] == "aa")
assert(tb.ccc[2] == "bb")
assert(tb.ccc[3] == "cc")
assert(tb.ddd[1] == "aa")
assert(tb.ddd[2] == "bb")
assert(tb.eee.aa == "bb")
assert(tb.eee.bb == "cc")

	`); err != nil {
		t.Error(err)
	}
}

func TestParseError(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	L.PreloadModule("yaml", Loader)
	if err := L.DoString(`
local yaml = require("yaml")

local str = [==[
- aaa
aaaaa
- bbb
- ccc
]==]

local tb, err = yaml.parse(str)
assert(tb == nil)


	`); err != nil {
		t.Error(err)
	}
}

func TestDump(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	L.PreloadModule("yaml", Loader)
	if err := L.DoString(`
local yaml = require("yaml")

	`); err != nil {
		t.Error(err)
	}
}
