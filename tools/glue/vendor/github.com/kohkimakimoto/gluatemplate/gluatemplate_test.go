package gluatemplate

import (
	"github.com/yuin/gopher-lua"
	"io/ioutil"
	"os"
	"testing"
)

func TestDoString(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	L.PreloadModule("template", Loader)
	if err := L.DoString(`
local template = require("template")

local output = template.dostring([[
This is a text template library.
Created by {{.first_name}} {{.last_name}}
]], {
	first_name = "kohki",
	last_name = "makimoto",
})

-- print(output)
--
-- This is a text template library.
-- Created by kohki makimoto
--

output = template.dostring("{{.a}}{{.b}}{{.c}}", {
	a = "aaa",
	b = "bbb",
	c = "ccc",

})
assert(output == "aaabbbccc")

output = template.dostring("{{len .b}}{{range $i, $v := .b}}{{$v}}{{end}}", {
	a = "aaa",
	b = {
		"hogehoge",
		"foobarfoobar",
	},

})

assert(output == "2hogehogefoobarfoobar")

	`); err != nil {
		t.Error(err)
	}
}

func TestDoStringError(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	L.PreloadModule("template", Loader)
	if err := L.DoString(`
local template = require("template")

output, err = template.dostring("{{end}}{{.aaa}}{{.b}}{{.c}}", {
	a = "aaa",
	b = "bbb",
	c = "ccc",

})
assert(output == nil)
print(err)

	`); err != nil {
		t.Error(err)
	}
}

var sampleFile = "{{.a}}{{.b}}{{.c}}"

func TestDoFile(t *testing.T) {
	tmpFile, err := ioutil.TempFile("", "")
	if err != nil {
		t.Errorf("should not raise error: %v", err)
	}
	if err = ioutil.WriteFile(tmpFile.Name(), []byte(sampleFile), 0644); err != nil {
		t.Errorf("should not raise error: %v", err)
	}
	defer func() {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
	}()

	L := lua.NewState()
	defer L.Close()

	L.PreloadModule("template", Loader)
	if err := L.DoString(`
local template = require("template")

local output = template.dofile("` + tmpFile.Name() + `", {
	a = "aaa",
	b = "bbb",
	c = "ccc",
})

assert(output == "aaabbbccc")


	`); err != nil {
		t.Error(err)
	}
}

func TestDoFileError(t *testing.T) {
	tmpFile, err := ioutil.TempFile("", "")
	if err != nil {
		t.Errorf("should not raise error: %v", err)
	}
	if err = ioutil.WriteFile(tmpFile.Name(), []byte(sampleFile), 0644); err != nil {
		t.Errorf("should not raise error: %v", err)
	}
	defer func() {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
	}()

	L := lua.NewState()
	defer L.Close()

	L.PreloadModule("template", Loader)
	if err := L.DoString(`
local template = require("template")

local output = template.dofile("` + tmpFile.Name() + `.hoge", {
	a = "aaa",
	b = "bbb",
	c = "ccc",
})

assert(output == nil)

	`); err != nil {
		t.Error(err)
	}
}
