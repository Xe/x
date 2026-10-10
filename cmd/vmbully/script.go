package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"slices"
	"strings"
	"time"

	"github.com/google/subcommands"
	"github.com/luthermonson/go-proxmox"
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

// taintedTag marks a VM that a script ran on. Such a VM is never used again.
const taintedTag = "tainted"

type scriptCmd struct {
	user        string
	tag         string
	templateID  int
	diskStorage string
	diskSize    string
}

func (*scriptCmd) Name() string     { return "script" }
func (*scriptCmd) Synopsis() string { return "Run a JavaScript file against a fresh VM." }
func (*scriptCmd) Usage() string {
	return "script [--user] [--tag] [--template-id] [--disk-storage] [--disk-size] <script.js>:\n" +
		"Find a running VM that has the tag and is not tagged \"" + taintedTag + "\".\n" +
		"If there is none, clone a new VM from the template.\n" +
		"Set a random password for the user in the VM, log in with SSH, and run the script.\n" +
		"Then delete the VM and clone a new VM from the template for the next run.\n" +
		"The script has these functions:\n" +
		"  $`command`         run command in the VM and return its stdout, throw on a nonzero exit status\n" +
		"  host$`command`     the same as $, but run command on this machine\n" +
		"  scp(source, dest)  copy the local file source to dest in the VM\n"
}
func (s *scriptCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&s.user, "user", "", "user for the SSH login, empty for the cloud-init user of the VM")
	f.StringVar(&s.tag, "tag", "vmbully", "tag of the VMs that scripts can use")
	f.IntVar(&s.templateID, "template-id", 9010, "VMID of the template to clone the next VM from")
	f.StringVar(&s.diskStorage, "disk-storage", "ssd", "storage for the disks of the next VM")
	f.StringVar(&s.diskSize, "disk-size", "32G", "size of the scsi0 disk of the next VM")
}

func (s *scriptCmd) Execute(ctx context.Context, f *flag.FlagSet, _ ...any) subcommands.ExitStatus {
	if err := flagenv.ParseSet(flagenv.Prefix, f); err != nil {
		slog.Error("can't read flags from the environment", "err", err)
		return subcommands.ExitUsageError
	}

	if f.NArg() != 1 || s.tag == "" {
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

func (s *scriptCmd) run(ctx context.Context, scriptPath string) error {
	src, err := os.ReadFile(scriptPath)
	if err != nil {
		return fmt.Errorf("read script: %w", err)
	}

	client, n, err := node(ctx)
	if err != nil {
		return err
	}

	// created is true when this run made the VM. Such a VM is tainted from
	// the start, so that no other run takes it.
	created := false
	vm, err := findFreeVM(ctx, client, s.tag)
	if errors.Is(err, errNoFreeVM) {
		slog.Info("no free VM, cloning a new one", "template", s.templateID)
		created = true
		vm, err = cloneVM(ctx, n, cloneOptions{
			templateID:  s.templateID,
			diskStorage: s.diskStorage,
			diskSize:    s.diskSize,
			tags:        []string{s.tag, taintedTag},
			password:    rand.Text(), // evalOn sets its own password
		})
	}
	if err != nil {
		return err
	}
	slog.Info("using VM", "id", vm.VMID, "name", vm.Name)

	tainted, err := s.evalOn(ctx, vm, string(src))
	if !tainted && !created {
		// The VM is not used, so it stays in the pool.
		return err
	}

	// Clean up even after an interrupt, so that no tainted VM stays behind.
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Minute)
	defer cancel()

	if !tainted {
		// The script did not start. Delete the VM that this run made, but
		// do not put a VM in the pool for a setup that does not work.
		return errors.Join(err, deleteVM(cleanupCtx, vm))
	}

	return errors.Join(err, deleteVM(cleanupCtx, vm), s.cloneNext(cleanupCtx, n))
}

// evalOn logs in to vm and evaluates the script src. tainted reports if vm got
// the tainted tag, which happens immediately before the script starts.
func (s *scriptCmd) evalOn(ctx context.Context, vm *proxmox.VirtualMachine, src string) (tainted bool, err error) {
	user := s.user
	if user == "" && vm.VirtualMachineConfig != nil {
		user = vm.VirtualMachineConfig.CIUser
	}
	if user == "" {
		return false, fmt.Errorf("VM %d has no cloud-init user, set --user", vm.VMID)
	}

	// A new VM can still be in its first boot.
	ifaces, err := waitForAgent(ctx, vm)
	if err != nil {
		return false, err
	}

	var ip string
	for _, iface := range ifaces {
		for _, addr := range iface.IPAddresses {
			if ip == "" && isUsableIPv4(addr.IPAddress) {
				ip = addr.IPAddress
			}
		}
	}
	if ip == "" {
		return false, fmt.Errorf("VM %d has no usable IPv4 address", vm.VMID)
	}

	password := rand.Text()
	if err := vm.AgentSetUserPassword(ctx, password, user); err != nil {
		return false, fmt.Errorf("set password for %s on VM %d: %w", user, vm.VMID, err)
	}

	sshClient, err := sshDial(ctx, ip, user, password)
	if err != nil {
		return false, fmt.Errorf("ssh to %s@%s: %w", user, ip, err)
	}
	defer sshClient.Close()
	slog.Info("connected to VM", "id", vm.VMID, "user", user, "ip", ip)

	js, err := newScriptVM(
		func(cmd string) (string, error) { return sshRun(sshClient, cmd) },
		func(cmd string) (string, error) { return hostRun(ctx, cmd) },
		func(source, dest string) error { return sshCopy(sshClient, source, dest) },
	)
	if err != nil {
		return false, err
	}
	defer js.Close()

	// From here the VM is used. It gets no second script, even when this
	// script fails.
	if err := addTag(ctx, vm, taintedTag); err != nil {
		return false, err
	}

	_, err = js.Eval(src, quickjs.EvalGlobal)
	return true, err
}

// cloneNext clones and starts the VM for the next script run.
func (s *scriptCmd) cloneNext(ctx context.Context, n *proxmox.Node) error {
	next, err := cloneVM(ctx, n, cloneOptions{
		templateID:  s.templateID,
		diskStorage: s.diskStorage,
		diskSize:    s.diskSize,
		tags:        []string{s.tag},
		password:    rand.Text(), // the next run sets its own password
	})
	if err != nil {
		return fmt.Errorf("clone next VM: %w", err)
	}
	slog.Info("next VM ready", "id", next.VMID, "name", next.Name)

	return nil
}

// errNoFreeVM means that the pool has no VM that a script can use.
var errNoFreeVM = errors.New("no running VM has the pool tag without the tainted tag")

// findFreeVM returns the running VM with the lowest VMID that has the tag and
// is not tainted. It returns errNoFreeVM when there is none.
func findFreeVM(ctx context.Context, client *proxmox.Client, tag string) (*proxmox.VirtualMachine, error) {
	cluster, err := client.Cluster(ctx)
	if err != nil {
		return nil, fmt.Errorf("get cluster: %w", err)
	}
	resources, err := cluster.Resources(ctx, "vm")
	if err != nil {
		return nil, fmt.Errorf("list cluster VMs: %w", err)
	}

	var found *proxmox.ClusterResource
	for _, r := range resources {
		if r.Type != "qemu" || r.Template != 0 || r.Status != proxmox.StatusVirtualMachineRunning {
			continue
		}
		tags := splitTags(r.Tags)
		if !slices.Contains(tags, tag) || slices.Contains(tags, taintedTag) {
			continue
		}
		if found == nil || r.VMID < found.VMID {
			found = r
		}
	}
	if found == nil {
		return nil, errNoFreeVM
	}

	n, err := client.Node(ctx, found.Node)
	if err != nil {
		return nil, fmt.Errorf("get node %s: %w", found.Node, err)
	}

	vm, err := n.VirtualMachine(ctx, int(found.VMID))
	if err != nil {
		return nil, fmt.Errorf("get VM %d: %w", found.VMID, err)
	}

	return vm, nil
}

// splitTags splits the tag list that the cluster resource API returns.
func splitTags(tags string) []string {
	return strings.FieldsFunc(tags, func(r rune) bool { return r == ';' || r == ',' || r == ' ' })
}

// deleteVM stops vm and deletes it with its disks. It refuses a VM that is not
// tainted.
func deleteVM(ctx context.Context, vm *proxmox.VirtualMachine) error {
	if !vm.HasTag(taintedTag) {
		return fmt.Errorf("VM %d is not tagged %q, refusing to delete it", vm.VMID, taintedTag)
	}

	task, err := vm.Stop(ctx)
	if err != nil {
		return fmt.Errorf("stop VM %d: %w", vm.VMID, err)
	}
	if err := waitTask(ctx, task, time.Second, 5*time.Minute); err != nil {
		return fmt.Errorf("stop VM %d: %w", vm.VMID, err)
	}

	task, err = vm.Delete(ctx, &proxmox.VirtualMachineDeleteOptions{
		Purge:                    proxmox.IntOrBool(true),
		DestroyUnreferencedDisks: proxmox.IntOrBool(true),
	})
	if err != nil {
		return fmt.Errorf("delete VM %d: %w", vm.VMID, err)
	}
	if err := waitTask(ctx, task, time.Second, 5*time.Minute); err != nil {
		return fmt.Errorf("delete VM %d: %w", vm.VMID, err)
	}
	slog.Info("VM deleted", "id", vm.VMID)

	return nil
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
