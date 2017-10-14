package gluash

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"sync"
	"syscall"

	"github.com/yuin/gopher-lua"
)

const luaShTypeName = "sh"

type shellCommand struct {
	path    string
	args    []string
	command *exec.Cmd
	stdout  io.ReadCloser
	stderr  io.ReadCloser
	stdin   io.ReadCloser

	waitCalled   bool
	stdoutClosed bool
	stderrClosed bool
}

func newShellCommand(path string, args ...string) (*shellCommand, error) {
	cmd := &shellCommand{
		path: path,
	}

	err := cmd.Command(path, args...)
	if err != nil {
		return nil, err
	}
	return cmd, nil
}

func (s *shellCommand) Command(path string, args ...string) error {
	s.command = exec.Command(path, args...)

	stdout, err := s.command.StdoutPipe()
	if err != nil {
		return err
	}

	stderr, err := s.command.StderrPipe()
	if err != nil {
		return err
	}

	s.stdout = stdout
	s.stderr = stderr

	return nil
}

func (s *shellCommand) Start(wait bool) error {
	if !wait {
		return s.command.Start()
	}
	stderr := &bytes.Buffer{}
	stdout := &bytes.Buffer{}

	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		io.Copy(stdout, s.stdout)
		wg.Done()
	}()

	go func() {
		io.Copy(stderr, s.stderr)
		wg.Done()
	}()

	err := s.command.Start()
	if err != nil {
		return err
	}
	if debug {
		fmt.Println("Waiting for process")
	}
	if err = s.command.Wait(); err != nil && !isExitError(err) {
		return err
	}

	wg.Wait()

	if abort {
		status, ok := s.command.ProcessState.Sys().(syscall.WaitStatus)
		if ok && status.ExitStatus() != 0 {
			return fmt.Errorf(
				"exit -- status %v\nSTDOUT:\n%v\nSTDERR:\n%v}",
				status.ExitStatus(),
				stdout,
				stderr,
			)
		}
	}

	s.stdout = ioutil.NopCloser(stdout)
	s.stderr = ioutil.NopCloser(stderr)
	return nil
}

func (s *shellCommand) UserData(L *lua.LState) *lua.LUserData {
	ud := L.NewUserData()
	ud.Value = s
	L.SetMetatable(ud, L.GetTypeMetatable(luaShTypeName))
	return ud
}

func (s *shellCommand) CloseStdout() {
	//s.stdout.Close()
	s.stdoutClosed = true
}

func (s *shellCommand) CloseStderr() {
	//s.stderr.Close()
	s.stderrClosed = true
}

func (s *shellCommand) Close(std string) {
	switch std {
	case "stderr":
		s.CloseStderr()
	case "stdout":
		s.CloseStdout()
	default:
		log.Fatalf("unable to close stream: %v", std)
	}
}

func (s *shellCommand) IsClosed(std string) bool {
	switch std {
	case "stderr":
		return s.stderrClosed
	case "stdout":
		return s.stdoutClosed
	default:
		log.Fatalf("Unknown option: %v", std)
	}
	return false
}

// shIndex checks if it is a predefined method or if it should be interprited as
// shell command.
func shIndex(L *lua.LState) int {
	index := L.CheckString(2)

	switch index {
	case "cmd":
		L.Push(L.NewFunction(shPathCmd(L)))
		return 1
	case "print":
		L.Push(L.NewFunction(shPrint))
		return 1
	case "ok":
		L.Push(L.NewFunction(shOk))
		return 1
	case "lines":
		L.Push(L.NewFunction(shLines))
		return 1
	case "success":
		L.Push(L.NewFunction(shSuccess))
		return 1
	case "exitcode":
		L.Push(L.NewFunction(shExitCode))
		return 1
	case "stdout", "stderr", "combinedOutput":
		L.Push(L.NewFunction(shOutput(index)))
		return 1
	default:
		return shCmd(L)
	}

}

func shCmd(L *lua.LState) int {
	shellCmd := checkShellCmd(L)
	index := L.CheckString(2)

	cmd := &shellCommand{
		path:  index,
		stdin: shellCmd.stdout,
	}

	L.Push(cmd.UserData(L))
	return 1
}

func shPathCmd(L *lua.LState) lua.LGFunction {
	shellCmd := checkShellCmd(L)

	return func(L *lua.LState) int {
		path := L.CheckString(2)
		args := checkStrings(L, 3)

		cmd := &shellCommand{
			path: path,
		}

		err := cmd.Command(path, args...)
		checkError(L, err)

		cmd.command.Stdin = shellCmd.stdout

		err = cmd.Start(true)
		checkError(L, err)

		L.Push(cmd.UserData(L))
		return 1
	}
}

// shCall executes the shell command and returns it self
func shCall(L *lua.LState) int {
	ud := L.CheckUserData(1)
	args := checkStrings(L, 2)
	shellCmd := checkShellCmd(L)
	err := shellCmd.Command(shellCmd.path, args...)
	checkError(L, err)

	if shellCmd.stdin != nil {
		shellCmd.command.Stdin = shellCmd.stdin
	}

	err = shellCmd.Start(shouldWait)
	checkError(L, err)

	L.Push(ud)
	return 1
}

func shOutput(std string) lua.LGFunction {
	return func(L *lua.LState) int {
		shellCmd := checkShellCmd(L)
		file := L.OptString(2, "")

		if shellCmd.waitCalled {
			L.RaiseError("Do not call `ok` or `success` before print")
		}

		stream := io.MultiReader(shellCmd.stdout, shellCmd.stderr)
		if std == "stderr" {
			stream = shellCmd.stderr
		}
		if std == "stdout" {
			stream = shellCmd.stdout
		}

		buf := new(bytes.Buffer)
		_, err := buf.ReadFrom(stream)
		if err != nil {
			L.RaiseError("Unable to read from `%v` several times", std)
		}

		if std == "stderr" || std == "stdout" {
			shellCmd.Close(std)
		} else {
			shellCmd.CloseStderr()
			shellCmd.CloseStdout()
		}

		if file != "" {
			err := ioutil.WriteFile(file, buf.Bytes(), 0644)
			checkError(L, err)
		}

		out := buf.String()
		L.Push(lua.LString(out))
		return 1
	}
}

func shOk(L *lua.LState) int {
	ud := L.CheckUserData(1)
	shellCmd := checkShellCmd(L)

	exitcode, err := wait(L, shellCmd)
	checkError(L, err)

	if exitcode != 0 {
		buf := new(bytes.Buffer)
		buf.ReadFrom(shellCmd.stdout)
		so := buf.String()
		buf.Reset()
		buf.ReadFrom(shellCmd.stderr)
		se := buf.String()
		L.RaiseError(
			"exit status %v\nSTDOUT:\n%v\nSTDERR:\n%v",
			exitcode,
			so,
			se,
		)
		//L.RaiseError("exit status %v", exitcode)
	}

	L.Push(ud)
	return 1
}

func shSuccess(L *lua.LState) int {
	shellCmd := checkShellCmd(L)

	errorCode, err := wait(L, shellCmd)
	checkError(L, err)

	L.Push(lua.LBool(errorCode == 0))
	return 1
}

func shExitCode(L *lua.LState) int {
	shellCmd := checkShellCmd(L)
	exitcode, err := wait(L, shellCmd)
	checkError(L, err)
	L.Push(lua.LNumber(exitcode))
	return 1
}

func shLines(L *lua.LState) int {
	shellCmd := checkShellCmd(L)
	std := L.OptString(2, "stdout")

	if !(std == "stdout" || std == "stderr") {
		L.RaiseError("lines: illigal file handle `%v`", std)
	}
	if shellCmd.IsClosed(std) {
		L.RaiseError("Unable to read from `%v` several times", std)
	}

	stream := shellCmd.stdout
	if std == "stderr" {
		stream = shellCmd.stderr
	}

	scanner := bufio.NewScanner(stream)
	scanner.Split(bufio.ScanLines)
	iterator := func(L *lua.LState) int {
		if scanner.Scan() {
			L.Push(lua.LString(scanner.Text()))
			return 1
		}

		shellCmd.Close(std)
		checkError(L, scanner.Err())
		return 0
	}

	L.Push(L.NewFunction(iterator))
	return 1
}

func shPrint(L *lua.LState) int {
	ud := L.CheckUserData(1)
	shellCmd := checkShellCmd(L)
	if shellCmd.stderrClosed || shellCmd.stdoutClosed {
		L.RaiseError("Unable to read from `%v` several times", "stdout/stderr")
	}

	if shellCmd.waitCalled {
		L.RaiseError("Do not call `ok` or `success` before print")
	}

	combo := io.MultiReader(shellCmd.stdout, shellCmd.stderr)

	_, err := io.Copy(os.Stdout, combo)
	//fmt.Println("A")
	checkError(L, err)
	//fmt.Println("B")
	_, err = wait(L, shellCmd)
	if err != nil && !isExitError(err) {
		L.RaiseError("Error while waiting for command to finish: %v", err)
	}

	shellCmd.CloseStderr()
	shellCmd.CloseStdout()
	L.Push(ud)
	return 1
}

// check if it is a shellCmd userdata as the first parmeter
func checkShellCmd(L *lua.LState) *shellCommand {
	ud := L.CheckUserData(1)
	shellCmd, ok := ud.Value.(*shellCommand)
	if !ok {
		L.Error(lua.LString("Expected the user data should be a shell command"), 0)
		return nil
	}
	return shellCmd
}

// converts all input parameters to strings from the n:th element
func checkStrings(L *lua.LState, n int) []string {
	params := L.GetTop()
	if n > params {
		return []string{}
	}
	args := make([]string, 0, (params - n + 1))
	for i := n; i <= params; i++ {
		p := L.Get(i)
		if p.Type() == lua.LTUserData {
			continue
		}
		L.CheckTypes(i, lua.LTString, lua.LTNumber)
		args = append(args, p.String())
	}
	return args
}

func checkError(L *lua.LState, err error) {
	if err != nil {
		L.RaiseError("%v", err)
	}
}

func isExitError(err error) bool {
	_, ok := err.(*exec.ExitError)
	return ok
}

func wait(L *lua.LState, shellCmd *shellCommand) (exitcode int, err error) {

	defer func() {
		if abort && (err != nil || exitcode != 0) {
			L.RaiseError("exit status %v, err: %v", exitcode, err)
		}
	}()

	if shellCmd.command.ProcessState == nil {
		stderr := &bytes.Buffer{}
		stdout := &bytes.Buffer{}

		wg := sync.WaitGroup{}
		wg.Add(2)

		go func() {
			io.Copy(stdout, shellCmd.stdout)
			wg.Done()
		}()

		go func() {
			io.Copy(stderr, shellCmd.stderr)
			wg.Done()
		}()

		err := shellCmd.command.Wait()
		shellCmd.stdout = ioutil.NopCloser(stdout)
		shellCmd.stderr = ioutil.NopCloser(stderr)

		if err != nil && !isExitError(err) {
			return 0, err
		}
	}

	if shellCmd.command.ProcessState.Success() {
		return 0, nil
	}

	if status, ok := shellCmd.command.ProcessState.Sys().(syscall.WaitStatus); ok {
		return status.ExitStatus(), nil
	}

	err = fmt.Errorf("`%v`: error retreiving exit code", shellCmd.command.Args)
	return 0, err
}
