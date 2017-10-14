package gluaenv

import (
	"github.com/yuin/gopher-lua"
	"testing"
	"io/ioutil"
	"os"
)

func TestSetAndGet(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	L.PreloadModule("env", Loader)
	if err := L.DoString(`
local env = require("env")

env.set("HOGE_KEY", "HOGE_VALUE")

local v = env.get("HOGE_KEY")

assert(v == "HOGE_VALUE")

	`); err != nil {
		t.Error(err)
	}
}

var sampleFile = `
AAA=hogehoge
BBB=bbbbbbbb

# CCC=eeeeee

DDD=ddddddd
`

func TestLoadfile(t *testing.T) {
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
	L.PreloadModule("env", Loader)

	if err := L.DoString(`
local env = require("env")
env.loadfile("` + tmpFile.Name() + `")

assert(env.get("AAA") == "hogehoge")
assert(env.get("BBB") == "bbbbbbbb")
assert(env.get("CCC") == nil)
assert(env.get("DDD") == "ddddddd")

r1, r2 = env.loadfile("` + tmpFile.Name() + `.notfound_file")
assert(r1 == nil)

`); err != nil {
		t.Error(err)
	}
}
