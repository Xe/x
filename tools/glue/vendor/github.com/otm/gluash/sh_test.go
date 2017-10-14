package gluash

import (
	"bytes"
	"io"
	"io/ioutil"
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
	L.PreloadModule("sh", Loader)

	restoreStdout := captureStdout()
	restoreStderr := captureStderr()
	err := L.DoString(src)
	stdout = restoreStdout()
	stderr = restoreStderr()
	if err != nil {
		t.Errorf("unable to run source: %v", err)
	}

	if len(stdout) > 0 {
		stdout = stdout[0 : len(stdout)-1]
	}

	if len(stderr) > 0 {
		stderr = stderr[0 : len(stderr)-1]
	}

	return stdout, stderr
}

func TestModuleCall(t *testing.T) {
	src := `
    local sh = require('sh')
    sh("echo", "foo", "bar"):print()
  `
	expected := "foo bar"
	got, _ := doString(src, t)

	if got != expected {
		t.Errorf("expected: `%v`, got: `%v`\nsrc: `%v`", expected, got, src)
	}
}

func TestIndexCall(t *testing.T) {
	src := `
    local sh = require('sh')
    sh.echo("foo", "bar"):print()
  `
	expected := "foo bar"
	got, _ := doString(src, t)

	if got != expected {
		t.Errorf("expected: `%v`, got: `%v`\nsrc: `%v`", expected, got, src)
	}
}

func TestOkPrintCall(t *testing.T) {
	src := `
    local sh = require('sh')
    sh.echo("foo", "bar"):ok():print()
  `
	expected := "foo bar"
	got, _ := doString(src, t)

	if got != expected {
		t.Errorf("expected: `%v`, got: `%v`\nsrc: `%v`", expected, got, src)
	}
}

func TestOkFailPrintCall(t *testing.T) {
	src := `
    local sh = require('sh')
		function fail()
			sh.sh("testdata/quickabort_test.sh"):ok():print()
    end
		ok, err = pcall(fail)
    print(ok)
    print(err)
  `
	expected := `false
<string>:4: exit status 1
STDOUT:

STDERR:
stderr text
`
	got, _ := doString(src, t)

	if got != expected {
		t.Errorf("expected: `%v`, got: `%v`\nsrc: `%v`", expected, got, src)
	}
}

func TestPipe(t *testing.T) {
	src := `
    local sh = require('sh')
    sh.echo("foo", "bar\n", "biz", "buz"):grep("foo"):print()
  `
	expected := "foo bar"
	got, _ := doString(src, t)

	if got != expected {
		t.Errorf("expected: `%v`, got: `%v`\nsrc: %v", expected, got, src)
	}
}

func TestPipeWithCmd(t *testing.T) {
	src := `
    local sh = require('sh')
		sh.echo("foo", "bar\n", "biz", "buz"):cmd("grep", "foo"):print()
  `
	expected := "foo bar"
	got, _ := doString(src, t)

	if got != expected {
		t.Errorf("expected: `%v`, got: `%v`\nsrc: %v", expected, got, src)
	}
}

func TestLines(t *testing.T) {
	src := `
    local sh = require('sh')
    for line in sh.echo("foo bar\nbiz", "buz"):lines() do
      print(line)
    end
  `
	expected := "foo bar\nbiz buz"
	got, _ := doString(src, t)

	if got != expected {
		t.Errorf("expected: `%v`, got: `%v`\nsrc: %v", expected, got, src)
	}
}

func TestOK(t *testing.T) {
	src := `
    local sh = require('sh')
    sh.echo("foo"):ok()
    print("ok")
  `
	expected := "ok"
	got, _ := doString(src, t)

	if got != expected {
		t.Errorf("expected: `%v`, got: `%v`\nsrc: %v", expected, got, src)
	}
}

func TestNotOK(t *testing.T) {
	src := `
    local sh = require('sh')
    function fail()
      sh.grep("-d"):ok()
    end

    ok, err = pcall(fail)
    print(ok)
    print(err)
  `
	expected := `false
<string>:4: exit status 2
STDOUT:

STDERR:
grep: option requires an argument -- 'd'
Usage: grep [OPTION]... PATTERN [FILE]...
Try 'grep --help' for more information.
`
	got, _ := doString(src, t)

	if got != expected {
		t.Errorf("expected: `%v`, got: `%v`\nsrc: %v", expected, got, src)
	}
}

func TestSuccess(t *testing.T) {
	src := `
    local sh = require('sh')
    ok = sh.echo("foo"):success()
    print(ok)
  `
	expected := "true"
	got, _ := doString(src, t)

	if got != expected {
		t.Errorf("expected: `%v`, got: `%v`\nsrc: `%v`", expected, got, src)
	}
}

func TestNotSuccess(t *testing.T) {
	src := `
    local sh = require('sh')
    ok = sh.grep("-d"):success()

    print(ok)
  `
	expected := "false"
	got, _ := doString(src, t)

	if got != expected {
		t.Errorf("expected: `%v`, got: `%v`\nsrc: %v", expected, got, src)
	}
}

func TestExitcode(t *testing.T) {
	src := `
    local sh = require('sh')
    exitcode = sh.echo("foo"):exitcode()
    print(exitcode)
  `
	expected := "0"
	got, _ := doString(src, t)

	if got != expected {
		t.Errorf("expected: `%v`, got: `%v`\nsrc: %v", expected, got, src)
	}
}

func TestNotExitcode(t *testing.T) {
	src := `
    local sh = require('sh')
    exitcode = sh.grep("-d"):exitcode()

    print(exitcode)
  `
	expected := "2"
	got, _ := doString(src, t)

	if got != expected {
		t.Errorf("expected: `%v`, got: `%v`\nsrc: %v", expected, got, src)
	}
}

func TestStdout(t *testing.T) {
	src := `
    local sh = require('sh')
    out = sh.echo("foo"):stdout()
    print(out)
  `
	expected := "foo" + "\n"
	got, _ := doString(src, t)

	if got != expected {
		t.Errorf("expected: `%v`, got: `%v`\nsrc: %v", expected, got, src)
	}
}

func TestStderr(t *testing.T) {
	src := `
    local sh = require('sh')
    out = sh("./testdata/stderr.test.sh"):stderr()
    print(out)
  `
	expected := "foo" + "\n"
	got, _ := doString(src, t)

	if got != expected {
		t.Errorf("expected: `%v`, got: `%v`\nsrc: %v", expected, got, src)
	}
}

func TestWriteStdoutToFile(t *testing.T) {
	src := `
    local sh = require('sh')
    tmp = "./remove.me"
    out = sh.echo("foo"):stdout(tmp)
    print(out)
  `
	expected := "foo" + "\n"
	file := "./remove.me"
	defer os.Remove(file)
	got, _ := doString(src, t)

	if got != expected {
		t.Errorf("expected stdout: `%v`, got: `%v`\nsrc: %v", expected, got, src)
	}
	dat, err := ioutil.ReadFile(file)
	if err != nil {
		t.Errorf("unable to read file: `%v`", file)
	}
	if string(dat) != expected {
		t.Errorf("expected file: `%v`, got: `%v`\nsrc: %v", expected, string(dat), src)
	}
}

func TestWriteStderrToFile(t *testing.T) {
	src := `
    local sh = require('sh')
    tmp = "./remove.me"
    out = sh("./testdata/stderr.test.sh"):stderr(tmp)
    print(out)
  `
	expected := "foo" + "\n"
	file := "./remove.me"
	defer os.Remove(file)
	got, _ := doString(src, t)

	if got != expected {
		t.Errorf("expected stdout: `%v`, got: `%v`\nsrc: %v", expected, got, src)
	}
	dat, err := ioutil.ReadFile(file)
	if err != nil {
		t.Errorf("unable to read file: `%v`", file)
	}
	if string(dat) != expected {
		t.Errorf("expected file: `%v`, got: `%v`\nsrc: %v", expected, string(dat), src)
	}
}

func TestCombindedOutput(t *testing.T) {
	src := `
    local sh = require('sh')
    out = sh("./testdata/stderr.test.sh"):combinedOutput()
    print(out)
  `
	expected := "foo" + "\n"
	got, _ := doString(src, t)

	if got != expected {
		t.Errorf("expected stdout: `%v`, got: `%v`\nsrc: %v", expected, got, src)
	}
}

func TestWriteCombindedOutputToFile(t *testing.T) {
	src := `
    local sh = require('sh')
    tmp = "./remove.me"
    out = sh("./testdata/stderr.test.sh"):combinedOutput(tmp)
    print(out)
  `
	expected := "foo" + "\n"
	file := "./remove.me"
	defer os.Remove(file)
	got, _ := doString(src, t)

	if got != expected {
		t.Errorf("expected stdout: `%v`, got: `%v`\nsrc: %v", expected, got, src)
	}
	dat, err := ioutil.ReadFile(file)
	if err != nil {
		t.Errorf("unable to read file: `%v`", file)
	}
	if string(dat) != expected {
		t.Errorf("expected file: `%v`, got: `%v`\nsrc: %v", expected, string(dat), src)
	}
}

func TestSetGlobalAbort(t *testing.T) {
	src := `
    local sh = require('sh')

		conf = sh{}
		print(conf.abort)

    sh{abort=true}

    conf = sh{}
		print(conf.abort)
    `
	expected := "false\ntrue"
	got, _ := doString(src, t)

	if got != expected {
		t.Errorf("expected stdout: `%v`, got: `%v`\nsrc: %v", expected, got, src)
	}
}

func TestGlobalAbort(t *testing.T) {
	src := `
    local sh = require('sh')
    sh{abort=true}

    function fail()
      print("this should print")
      sh("false"):print()
      print("but not this")
    end

    function fail2()
      print("this should print")
      sh("false"):ok()
      print("but not this")
    end

    function fail3()
      print("this should print")
      sh("false"):success()
      print("but not this")
    end

    function fail4()
      print("this should print")
      sh("false")
      print("but not this")
    end

    ok, err = pcall(fail)
    ok, err = pcall(fail2)
    ok, err = pcall(fail3)
    ok, err = pcall(fail4)
		sh{abort=false}
    `
	expected := strings.TrimSpace(strings.Repeat("this should print\n", 4))
	got, _ := doString(src, t)

	if got != expected {
		t.Errorf("expected stdout: `%v`, got: `%v`\nsrc: %v", expected, got, src)
	}
}

func TestSetGlobalWait(t *testing.T) {
	src := `
    local sh = require('sh')

		conf = sh{}
		print(conf.wait)

    sh{wait=true}

    conf = sh{}
		print(conf.wait)
		sh{wait=false}
    `
	expected := "false\ntrue"
	got, _ := doString(src, t)

	if got != expected {
		t.Errorf("expected stdout: `%v`, got: `%v`\nsrc: %v", expected, got, src)
	}
}

func TestGlobalWaitWithAbort(t *testing.T) {
	src := `
    local sh = require('sh')
    sh{wait=true, abort=true}
		sh("/bin/sh", "./testdata/abort.test.sh")
		print("bar")
    `
	expected := `<string>:4: exit status 1
STDOUT:
stdout text

STDERR:
stderr text
}
stack traceback:
	[G]: in sh
	<string>:4: in function 'main chunk'
	[G]: ?`
	L := lua.NewState()
	defer L.Close()
	L.PreloadModule("sh", Loader)

	err := L.DoString(src)

	if err.Error() != expected {
		t.Errorf(
			"expected: `%v`\ngot: `%v`\nsrc: %v",
			expected,
			err,
			src,
		)
	}
}

func TestModulePipeWithAbort(t *testing.T) {
	src := `
    local sh = require('sh')
		sh{abort=true}
    sh("echo", "foo", "bar\n", "biz", "buz\n", "fii foo"):grep("foo"):print()
  `
	expected := "foo bar\n fii foo"
	got, _ := doString(src, t)

	if got != expected {
		t.Errorf("expected: `%v`, got: `%v`\nsrc: %v", expected, got, src)
	}
}

func TestPipeWithAbort(t *testing.T) {
	src := `
    local sh = require('sh')
		sh{abort=true}
    sh.echo("foo", "bar\n", "biz", "buz\n", "fii foo"):grep("foo"):print()
  `
	expected := "foo bar\n fii foo"
	got, _ := doString(src, t)

	if got != expected {
		t.Errorf("expected: `%v`, got: `%v`\nsrc: %v", expected, got, src)
	}
}

func TestPipeGlob(t *testing.T) {
	src := `
    local sh = require('sh')
		sh.echo(sh.glob('*.md')):print()
  `
	expected := "README.md"
	got, _ := doString(src, t)

	if got != expected {
		t.Errorf("expected: `%v`, got: `%v`\nsrc: %v", expected, got, src)
	}
}
