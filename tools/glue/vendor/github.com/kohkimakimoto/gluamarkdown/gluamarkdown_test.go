package gluamarkdown

import (
	"github.com/yuin/gopher-lua"
	"testing"
	"os"
	"io/ioutil"
)

func TestDoStringExample(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	L.PreloadModule("markdown", Loader)
	if err := L.DoString(`
local markdown = require("markdown")
local output = markdown.dostring([=[
# glua markdown

Markdown processor for gopher-lua

]=])

print(output)
	`); err != nil {
		t.Error(err)
	}
}

func TestDoString(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	L.PreloadModule("markdown", Loader)
	if err := L.DoString(`
local markdown = require("markdown")
local output = markdown.dostring([[
# h1
## h2
]])

print(output)

assert(output == [=[
<h1>h1</h1>

<h2>h2</h2>
]=])
	`); err != nil {
		t.Error(err)
	}
}

var sampleFile = `# h1
## h2
`

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

	L.PreloadModule("markdown", Loader)
	if err := L.DoString(`
local markdown = require("markdown")
local output = markdown.dofile("` + tmpFile.Name() + `")
print(output)

assert(output == [=[
<h1>h1</h1>

<h2>h2</h2>
]=])
	`); err != nil {
		t.Error(err)
	}
}

