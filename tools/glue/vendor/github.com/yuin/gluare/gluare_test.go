package gluare

import (
	"fmt"
	"github.com/yuin/gopher-lua"
	"os"
	"testing"
)

func testScriptDir(t *testing.T, tests []string, directory string) {
	if err := os.Chdir(directory); err != nil {
		t.Error(err)
	}
	defer os.Chdir("..")
	for _, script := range tests {
		fmt.Printf("testing %s/%s\n", directory, script)
		L := lua.NewState(lua.Options{
			RegistrySize:        1024 * 20,
			CallStackSize:       1024,
			IncludeGoStackTrace: true,
		})
		L.PreloadModule("re", Loader)
		if err := L.DoFile(script); err != nil {
			t.Error(err)
		}
		L.Close()
	}
}

func TestGluaRe(t *testing.T) {
	testScriptDir(t, []string{"test.lua"}, "_tests")
}
