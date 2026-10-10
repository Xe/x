package main

import (
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"log/slog"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/google/subcommands"
	"github.com/luthermonson/go-proxmox"
	"within.website/x/internal/flagenv"
	"within.website/x/misc/namegen"
)

type cloneCmd struct {
	templateID  int
	diskStorage string
	diskSize    string
	tag         string
	sshUser     string
}

func (*cloneCmd) Name() string     { return "clone" }
func (*cloneCmd) Synopsis() string { return "Clone the template into a new VM and start it." }
func (*cloneCmd) Usage() string {
	return `clone [--template-id] [--disk-storage] [--disk-size] [--tag] [--ssh-user]:
Clone the template into a new VM, resize its disk, start it, and print its IPv4 addresses.
Then log in with SSH and a password made for this VM, and print the output of "uname -av".
`
}
func (c *cloneCmd) SetFlags(f *flag.FlagSet) {
	f.IntVar(&c.templateID, "template-id", 9000, "VMID of the template to clone")
	f.StringVar(&c.diskStorage, "disk-storage", "ssd", "storage for the disks of the new VM")
	f.StringVar(&c.diskSize, "disk-size", "32G", "size of the scsi0 disk of the new VM")
	f.StringVar(&c.tag, "tag", "vmbully", "tag to add to the new VM, empty for no tag")
	f.StringVar(&c.sshUser, "ssh-user", "", "user for the SSH login, empty for the cloud-init user of the template")
}

func (c *cloneCmd) Execute(ctx context.Context, f *flag.FlagSet, _ ...any) subcommands.ExitStatus {
	if err := flagenv.ParseSet(flagenv.Prefix, f); err != nil {
		slog.Error("can't read flags from the environment", "err", err)
		return subcommands.ExitUsageError
	}

	if err := c.run(ctx); err != nil {
		slog.Error("can't clone template", "err", err)
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}

func (c *cloneCmd) run(ctx context.Context) error {
	_, n, err := node(ctx)
	if err != nil {
		return err
	}

	var tags []string
	if c.tag != "" {
		tags = append(tags, c.tag)
	}

	password := rand.Text()
	newVM, err := cloneVM(ctx, n, cloneOptions{
		templateID:  c.templateID,
		diskStorage: c.diskStorage,
		diskSize:    c.diskSize,
		tags:        tags,
		password:    password,
	})
	if err != nil {
		return err
	}
	newID := int(newVM.VMID)

	sshUser := c.sshUser
	if sshUser == "" && newVM.VirtualMachineConfig != nil {
		sshUser = newVM.VirtualMachineConfig.CIUser
	}
	if sshUser == "" {
		return fmt.Errorf("template %d has no cloud-init user, set --ssh-user", c.templateID)
	}

	ifaces, err := waitForAgent(ctx, newVM)
	if err != nil {
		return err
	}

	var sshIP string
	for _, iface := range ifaces {
		for _, addr := range iface.IPAddresses {
			if !isUsableIPv4(addr.IPAddress) {
				continue
			}

			fmt.Printf("%s: %s\n", iface.Name, addr.IPAddress)
			if sshIP == "" {
				sshIP = addr.IPAddress
			}
		}
	}

	if sshIP == "" {
		return fmt.Errorf("VM %d has no usable IPv4 address", newID)
	}
	fmt.Printf("login: %s@%s password: %s\n", sshUser, sshIP, password)

	sshClient, err := sshDial(ctx, sshIP, sshUser, password)
	if err != nil {
		return fmt.Errorf("ssh to %s@%s: %w", sshUser, sshIP, err)
	}
	defer sshClient.Close()

	out, err := sshRun(sshClient, "uname -av")
	if err != nil {
		return err
	}
	fmt.Print(out)

	return nil
}

type cloneOptions struct {
	templateID  int
	diskStorage string
	diskSize    string
	tags        []string
	password    string // cloud-init password of the new VM
}

// cloneVM clones the template into a new VM on node n, then tags, resizes and
// starts the new VM. It does not wait for the guest agent.
func cloneVM(ctx context.Context, n *proxmox.Node, opts cloneOptions) (*proxmox.VirtualMachine, error) {
	template, err := n.VirtualMachine(ctx, opts.templateID)
	if err != nil {
		return nil, fmt.Errorf("get template %d: %w", opts.templateID, err)
	}

	newID, task, err := template.Clone(ctx, &proxmox.VirtualMachineCloneOptions{
		Name:    namegen.Next(),
		Full:    proxmox.IntOrBool(true),
		Storage: opts.diskStorage,
	})
	if err != nil {
		return nil, fmt.Errorf("clone template %d: %w", opts.templateID, err)
	}
	slog.Info("new vm created", "id", newID)

	if err := waitTask(ctx, task, time.Second, 5*time.Minute); err != nil {
		return nil, err
	}

	newVM, err := n.VirtualMachine(ctx, newID)
	if err != nil {
		return nil, err
	}

	for _, tag := range opts.tags {
		if err := addTag(ctx, newVM, tag); err != nil {
			return nil, err
		}
	}

	// Each VM gets its own password, set before the first boot so that
	// cloud-init applies it.
	if err := newVM.ConfigSync(ctx, proxmox.VirtualMachineOption{Name: "cipassword", Value: opts.password}); err != nil {
		return nil, fmt.Errorf("set VM password: %w", err)
	}

	task, err = newVM.ResizeDisk(ctx, "scsi0", opts.diskSize)
	if err != nil {
		return nil, fmt.Errorf("resize disk: %w", err)
	}
	if err := waitTask(ctx, task, time.Second, 5*time.Minute); err != nil {
		return nil, err
	}
	slog.Info("VM disk resized", "id", newID)

	task, err = newVM.Start(ctx)
	if err != nil {
		return nil, fmt.Errorf("start VM: %w", err)
	}
	if err := waitTask(ctx, task, time.Second, 5*time.Minute); err != nil {
		return nil, err
	}
	slog.Info("VM started", "id", newID)

	return newVM, nil
}

// addTag adds tag to vm and keeps the tags that vm already has.
func addTag(ctx context.Context, vm *proxmox.VirtualMachine, tag string) error {
	if vm.VirtualMachineConfig == nil {
		vm.VirtualMachineConfig = &proxmox.VirtualMachineConfig{}
	}

	tags := tag
	if cfg := vm.VirtualMachineConfig; cfg.Tags != "" {
		tags = cfg.Tags
		if !vm.HasTag(tag) {
			tags += proxmox.TagSeperator + tag
		}
	}

	if err := vm.ConfigSync(ctx, proxmox.VirtualMachineOption{Name: "tags", Value: tags}); err != nil {
		return fmt.Errorf("tag VM %d as %s: %w", vm.VMID, tag, err)
	}
	vm.VirtualMachineConfig.Tags = tags
	vm.VirtualMachineConfig.TagsSlice = nil
	slog.Info("VM tagged", "id", vm.VMID, "tags", tags)

	return nil
}

// waitForAgent waits with exponential backoff until the guest agent of vm
// answers, and returns the network interfaces of vm.
func waitForAgent(ctx context.Context, vm *proxmox.VirtualMachine) ([]*proxmox.AgentNetworkIface, error) {
	// The guest agent is not ready until the VM boots and cloud-init
	// installs it, so every error counts as "not ready yet".
	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = 2 * time.Second
	bo.MaxInterval = 30 * time.Second
	bo.MaxElapsedTime = 5 * time.Minute

	ifaces, err := backoff.RetryNotifyWithData(func() ([]*proxmox.AgentNetworkIface, error) {
		return vm.AgentGetNetworkIFaces(ctx)
	}, backoff.WithContext(bo, ctx), func(err error, next time.Duration) {
		slog.Info("guest agent not ready", "id", vm.VMID, "err", err, "retry_in", next.String())
	})
	if err != nil {
		return nil, fmt.Errorf("get VM network interfaces: %w", err)
	}

	return ifaces, nil
}
