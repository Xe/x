package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/google/subcommands"
	"golang.org/x/crypto/ssh"
	"modernc.org/quickjs"
	"within.website/x/internal/flagenv"
)

// scriptPrelude defines $ and host$ on top of the __vmbully_run and
// __vmbully_host_run host functions. Each interpolated value becomes one shell
// word. An array becomes one word for each element.
const scriptPrelude = `{
  const quote = (v) => Array.isArray(v)
    ? v.map(quote).join(" ")
    : "'" + String(v).replace(/'/g, "'\\''") + "'";
  const tag = (run) => (strings, ...values) => {
    let cmd = strings[0];
    values.forEach((v, i) => { cmd += quote(v) + strings[i + 1]; });
    return run(cmd);
  };
  globalThis.$ = tag(__vmbully_run);
  globalThis.host$ = tag(__vmbully_host_run);
}
`

type scriptCmd struct {
	containerOptions
}

func (*scriptCmd) Name() string     { return "script" }
func (*scriptCmd) Synopsis() string { return "Run JavaScript in a new LXC container, then delete it." }
func (*scriptCmd) Usage() string {
	return "script --os-template=<storage:vztmpl/archive> [flags] <script.js>:\n" +
		"Create and start an LXC container, run the script as root over SSH, then delete the container.\n" +
		"  $`command`         run command in the container and return stdout\n" +
		"  host$`command`     run command on this machine\n" +
		"  scp(source, dest)  copy a local file into the container\n"
}
func (s *scriptCmd) SetFlags(f *flag.FlagSet) {
	s.containerOptions.setFlags(f)
}

func (s *scriptCmd) Execute(ctx context.Context, f *flag.FlagSet, _ ...any) subcommands.ExitStatus {
	if err := flagenv.ParseSet(flagenv.Prefix, f); err != nil {
		slog.Error("can't read flags from the environment", "err", err)
		return subcommands.ExitUsageError
	}

	if f.NArg() != 1 {
		fmt.Print(s.Usage())
		return subcommands.ExitUsageError
	}

	if err := s.run(ctx, f.Arg(0)); err != nil {
		var jsErr *quickjs.Error
		if errors.As(err, &jsErr) && jsErr.Stack != "" {
			slog.Error("script failed", "err", err, "stack", jsErr.Stack)
		} else {
			slog.Error("script failed", "err", err)
		}
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}

func (s *scriptCmd) run(ctx context.Context, scriptPath string) (err error) {
	if err := s.containerOptions.validate(); err != nil {
		return err
	}
	src, err := os.ReadFile(scriptPath)
	if err != nil {
		return fmt.Errorf("read script: %w", err)
	}
	_, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("generate SSH key: %w", err)
	}
	signer, err := ssh.NewSignerFromKey(key)
	if err != nil {
		return err
	}
	client, n, err := node(ctx)
	if err != nil {
		return err
	}
	ct, err := createContainer(ctx, client, n, s.containerOptions, string(ssh.MarshalAuthorizedKey(signer.PublicKey())))
	if err != nil {
		return err
	}
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Minute)
		defer cancel()
		err = errors.Join(err, deleteContainer(cleanupCtx, ct))
	}()

	sshClient, err := connectContainer(ctx, ct, signer)
	if err != nil {
		return err
	}
	defer sshClient.Close()
	js, err := newScriptVM(
		func(cmd string) (string, error) { return sshRun(sshClient, cmd) },
		func(cmd string) (string, error) { return hostRun(ctx, cmd) },
		func(source, dest string) error { return sshCopy(sshClient, source, dest) },
	)
	if err != nil {
		return err
	}
	defer js.Close()
	_, err = js.Eval(string(src), quickjs.EvalGlobal)
	return err
}

// hostRun runs cmd with sh on this machine and returns its stdout. The stderr
// of cmd goes to the local stderr. A nonzero exit status is an error.
func hostRun(ctx context.Context, cmd string) (string, error) {
	c := exec.CommandContext(ctx, "sh", "-c", cmd)
	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = io.MultiWriter(os.Stderr, &stderr)

	if err := c.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return stdout.String(), fmt.Errorf("host command %q: exit status %d: %s", cmd, exitErr.ExitCode(), strings.TrimSpace(stderr.String()))
		}
		return stdout.String(), fmt.Errorf("host command %q: %w", cmd, err)
	}

	return stdout.String(), nil
}

// newScriptVM returns a JavaScript VM with $, host$, scp, print and console
// defined. run, hostRun and copy do the work of $, host$ and scp. An error from
// one of them becomes a JavaScript exception.
func newScriptVM(run, hostRun func(cmd string) (string, error), copy func(source, dest string) error) (*quickjs.VM, error) {
	js, err := quickjs.NewVM()
	if err != nil {
		return nil, fmt.Errorf("create JavaScript VM: %w", err)
	}

	if err := js.StdAddHelpers(); err != nil {
		js.Close()
		return nil, fmt.Errorf("add console to JavaScript VM: %w", err)
	}

	for name, fn := range map[string]func(cmd string) (string, error){
		"__vmbully_run":      run,
		"__vmbully_host_run": hostRun,
	} {
		err := js.RegisterHostFunc(name, func(args []any) (any, error) {
			if len(args) != 1 {
				return nil, errors.New("the command tag needs one command")
			}
			cmd, ok := args[0].(string)
			if !ok {
				return nil, errors.New("the command tag needs a string command")
			}
			return fn(cmd)
		})
		if err != nil {
			js.Close()
			return nil, fmt.Errorf("register %s: %w", name, err)
		}
	}

	err = js.RegisterHostFunc("scp", func(args []any) (any, error) {
		if len(args) != 2 {
			return nil, errors.New("scp needs a source and a destination")
		}
		source, ok1 := args[0].(string)
		dest, ok2 := args[1].(string)
		if !ok1 || !ok2 {
			return nil, errors.New("scp needs a string source and a string destination")
		}
		return nil, copy(source, dest)
	})
	if err != nil {
		js.Close()
		return nil, fmt.Errorf("register scp: %w", err)
	}

	if _, err := js.Eval(scriptPrelude, quickjs.EvalGlobal); err != nil {
		js.Close()
		return nil, fmt.Errorf("define $ and host$: %w", err)
	}

	return js, nil
}
