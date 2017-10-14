package gluafs

import (
	"github.com/yuin/gopher-lua"
	"io/ioutil"
	"os"
	"testing"
)

func TestExists(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	L.PreloadModule("fs", Loader)
	if err := L.DoString(`
local fs = require("fs")
assert(fs.exists(".") == true)

	`); err != nil {
		t.Error(err)
	}
}

var tmpFileContent = "aaaaaaaabbbbbbbb"

func TestRead(t *testing.T) {
	tmpFile, err := ioutil.TempFile("", "")
	if err != nil {
		t.Errorf("should not raise error: %v", err)
	}
	if err = ioutil.WriteFile(tmpFile.Name(), []byte(tmpFileContent), 0644); err != nil {
		t.Errorf("should not raise error: %v", err)
	}
	defer func() {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
	}()

	L := lua.NewState()
	defer L.Close()

	L.PreloadModule("fs", Loader)
	if err := L.DoString(`
local fs = require("fs")
local content = fs.read("` + tmpFile.Name() + `")

assert(content == "aaaaaaaabbbbbbbb")

local content2, err = fs.read("` + tmpFile.Name() + `.hoge")
assert(content2 == nil)


	`); err != nil {
		t.Error(err)
	}
}

func TestWrite(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Errorf("should not raise error: %v", err)
	}
	defer func() {
		os.RemoveAll(tmpDir)
	}()

	L := lua.NewState()
	defer L.Close()

	L.PreloadModule("fs", Loader)
	if err := L.DoString(`
local fs = require("fs")
local ret, err = fs.write("` + tmpDir + `/hoge", "aaaaaaaabbbbbbbb", 755)
local content = fs.read("` + tmpDir + `/hoge")

assert(content == "aaaaaaaabbbbbbbb")

ret, err = fs.write("` + tmpDir + `/hoge/aaaa", "aaaaaaaabbbbbbbb")
assert(ret == nil)

	`); err != nil {
		t.Error(err)
	}
}

func TestMkdir(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Errorf("should not raise error: %v", err)
	}
	defer func() {
		os.RemoveAll(tmpDir)
	}()

	L := lua.NewState()
	defer L.Close()

	L.PreloadModule("fs", Loader)
	if err := L.DoString(`
local fs = require("fs")
local ret, err = fs.mkdir("` + tmpDir + `/hoge")
assert(ret == true)

local ret, err = fs.mkdir("` + tmpDir + `/hoge/aaa/bbb", 0777, true)
assert(ret == true)

local ret, err = fs.mkdir("` + tmpDir + `/hoge/bbb/eeee", 0777)
assert(ret == nil)
-- print(err)

	`); err != nil {
		t.Error(err)
	}
}

func TestGlob(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Errorf("should not raise error: %v", err)
	}
	defer func() {
		os.RemoveAll(tmpDir)
	}()

	L := lua.NewState()
	defer L.Close()

	L.PreloadModule("fs", Loader)
	if err := L.DoString(`
local fs = require("fs")
fs.mkdir("` + tmpDir + `/dir01")
fs.mkdir("` + tmpDir + `/dir02")
fs.mkdir("` + tmpDir + `/dir03")
fs.write("` + tmpDir + `/file01", "aaaaaaa")
fs.write("` + tmpDir + `/file02", "bbbbbbb")

local ret, err = fs.glob("` + tmpDir + `/*", function(file)
	print(file.path)
	print(file.realpath)
end)

assert(ret == true)
assert(err == nil)

	`); err != nil {
		t.Error(err)
	}
}
