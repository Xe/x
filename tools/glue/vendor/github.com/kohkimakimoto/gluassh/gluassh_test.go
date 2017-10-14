package gluassh

import(
	"testing"
	"github.com/yuin/gopher-lua"
	"os"
	"path/filepath"
)

var keynopass string
var keypass string

func TestRunWithKeynopass(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	L.PreloadModule("ssh", Loader)

	if err := L.DoString(`
local ssh = require "ssh"

conn_user_keynopass = {
	host = "192.168.56.81",
	port = "22",
	user = "user_keynopass",
	key = "` + keynopass + `",
}

local ret = ssh.run(conn_user_keynopass, "hostname")

assert(ret.out == "gluassh-test-server\n")
	`); err != nil {
		t.Error(err)
	}
}

func TestRunWithKeynopassSudo(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	L.PreloadModule("ssh", Loader)

	if err := L.DoString(`
local ssh = require "ssh"

conn_user_keynopass = {
	host = "192.168.56.81",
	port = "22",
	user = "user_keynopass",
	key = "` + keynopass + `",
}

ret = ssh.run(conn_user_keynopass, {sudo = true}, "whoami")
assert(ret.out == "root\n")
	`); err != nil {
		t.Error(err)
	}
}

func TestRunWithKeypass(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	L.PreloadModule("ssh", Loader)

	if err := L.DoString(`
local ssh = require "ssh"

conn_user_keypass = {
	host = "192.168.56.81",
	port = "22",
	user = "user_keypass",
	key = "` + keypass + `",
	key_passphrase = "hogehoge",
}

local ret = ssh.run(conn_user_keypass, "hostname")

assert(ret.out == "gluassh-test-server\n")
	`); err != nil {
		t.Error(err)
	}
}

func TestRunWithKeynopassFunc(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	L.PreloadModule("ssh", Loader)

	if err := L.DoString(`
local ssh = require "ssh"

conn_user_keynopass = {
	host = "192.168.56.81",
	port = "22",
	user = "user_keynopass",
	key = "` + keynopass + `",
}

local ret = ssh.run(conn_user_keynopass, {}, "hostname", function(stdout, stderr)

	print(stdout)

end)

	`); err != nil {
		t.Error(err)
	}
}

func init() {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	keynopass = filepath.Join(wd, "_tests", "keys", "id_rsa.nopass")
	keypass = filepath.Join(wd, "_tests", "keys", "id_rsa.pass")
}