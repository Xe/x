package main

import (
	"errors"
	"slices"
	"testing"

	"modernc.org/quickjs"
)

func TestScriptVM(t *testing.T) {
	for _, tt := range []struct {
		name     string
		script   string
		runErr   error
		hostErr  error
		wantCmds []string
		wantHost []string
		wantCopy []string
		wantRes  any
		wantErr  bool
	}{
		{
			name:     "plain command returns stdout",
			script:   "$`uname -a`",
			wantCmds: []string{"uname -a"},
			wantRes:  "out:uname -a",
		},
		{
			name:     "value is one shell word",
			script:   "const f = \"a b; rm -rf /\"; $`cat ${f}`",
			wantCmds: []string{"cat 'a b; rm -rf /'"},
			wantRes:  "out:cat 'a b; rm -rf /'",
		},
		{
			name:     "single quote in value",
			script:   "$`echo ${\"it's\"}`",
			wantCmds: []string{`echo 'it'\''s'`},
			wantRes:  `out:echo 'it'\''s'`,
		},
		{
			name:     "array is one word for each element",
			script:   "$`ls ${[\"a\", \"b c\"]} ${1}`",
			wantCmds: []string{"ls 'a' 'b c' '1'"},
			wantRes:  "out:ls 'a' 'b c' '1'",
		},
		{
			name:     "failed command throws",
			script:   "$`false`; $`true`",
			runErr:   errors.New("exit status 1"),
			wantCmds: []string{"false"},
			wantErr:  true,
		},
		{
			name:     "script can catch a failed command",
			script:   "let r = \"ok\"; try { $`false` } catch (e) { r = \"caught\" }; r",
			runErr:   errors.New("exit status 1"),
			wantCmds: []string{"false"},
			wantRes:  "caught",
		},
		{
			name:     "host$ runs on the host with the same quoting",
			script:   "const out = $`hostname`; host$`echo ${out} ${[\"a b\"]}`",
			wantCmds: []string{"hostname"},
			wantHost: []string{"echo 'out:hostname' 'a b'"},
			wantRes:  "host:echo 'out:hostname' 'a b'",
		},
		{
			name:     "failed host command throws",
			script:   "host$`false`; $`true`",
			hostErr:  errors.New("exit status 1"),
			wantHost: []string{"false"},
			wantErr:  true,
		},
		{
			name:     "scp passes source and destination",
			script:   "scp(\"./a.txt\", \"/tmp/a.txt\"); 1",
			wantCopy: []string{"./a.txt", "/tmp/a.txt"},
			wantRes:  1,
		},
		{
			name:    "scp with one argument throws",
			script:  "scp(\"./a.txt\")",
			wantErr: true,
		},
		{
			name:    "script throw is an error",
			script:  "throw new Error(\"nope\")",
			wantErr: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var cmds, hostCmds, copied []string

			js, err := newScriptVM(
				func(cmd string) (string, error) {
					cmds = append(cmds, cmd)
					return "out:" + cmd, tt.runErr
				},
				func(cmd string) (string, error) {
					hostCmds = append(hostCmds, cmd)
					return "host:" + cmd, tt.hostErr
				},
				func(source, dest string) error {
					copied = append(copied, source, dest)
					return nil
				},
			)
			if err != nil {
				t.Fatal(err)
			}
			defer js.Close()

			res, err := js.Eval(tt.script, quickjs.EvalGlobal)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Eval error = %v, wantErr %v", err, tt.wantErr)
			}
			if !slices.Equal(cmds, tt.wantCmds) {
				t.Errorf("commands = %q, want %q", cmds, tt.wantCmds)
			}
			if !slices.Equal(hostCmds, tt.wantHost) {
				t.Errorf("host commands = %q, want %q", hostCmds, tt.wantHost)
			}
			if !slices.Equal(copied, tt.wantCopy) {
				t.Errorf("copied = %q, want %q", copied, tt.wantCopy)
			}
			if !tt.wantErr && res != tt.wantRes {
				t.Errorf("result = %#v, want %#v", res, tt.wantRes)
			}
		})
	}
}

func TestHostRun(t *testing.T) {
	out, err := hostRun(t.Context(), "printf 'a b'")
	if err != nil {
		t.Fatal(err)
	}
	if out != "a b" {
		t.Errorf("stdout = %q, want %q", out, "a b")
	}

	if _, err := hostRun(t.Context(), "exit 3"); err == nil {
		t.Error("a nonzero exit status must be an error")
	}
}
