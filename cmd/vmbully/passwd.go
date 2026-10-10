package main

import (
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/google/subcommands"
	"within.website/x/internal/flagenv"
)

type passwdCmd struct {
	user     string
	password string
}

func (*passwdCmd) Name() string     { return "passwd" }
func (*passwdCmd) Synopsis() string { return "Set a new password for a user in a running VM." }
func (*passwdCmd) Usage() string {
	return `passwd [--user] [--password] <vmid>:
Set a new password for a user in a running VM through the QEMU guest agent, and print the password.
`
}
func (p *passwdCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&p.user, "user", "", "user to set the password for, empty for the cloud-init user of the VM")
	f.StringVar(&p.password, "password", "", "new password, empty for a random password")
}

func (p *passwdCmd) Execute(ctx context.Context, f *flag.FlagSet, _ ...any) subcommands.ExitStatus {
	if err := flagenv.ParseSet(flagenv.Prefix, f); err != nil {
		slog.Error("can't read flags from the environment", "err", err)
		return subcommands.ExitUsageError
	}

	if f.NArg() != 1 {
		fmt.Print(p.Usage())
		return subcommands.ExitUsageError
	}

	vmid, err := strconv.Atoi(f.Arg(0))
	if err != nil {
		slog.Error("VMID is not a number", "vmid", f.Arg(0))
		return subcommands.ExitUsageError
	}

	if err := p.run(ctx, vmid); err != nil {
		slog.Error("can't set password", "err", err)
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}

func (p *passwdCmd) run(ctx context.Context, vmid int) error {
	_, n, err := node(ctx)
	if err != nil {
		return err
	}

	vm, err := n.VirtualMachine(ctx, vmid)
	if err != nil {
		return fmt.Errorf("get VM %d: %w", vmid, err)
	}

	user := p.user
	if user == "" && vm.VirtualMachineConfig != nil {
		user = vm.VirtualMachineConfig.CIUser
	}
	if user == "" {
		return fmt.Errorf("VM %d has no cloud-init user, set --user", vmid)
	}

	password := p.password
	if password == "" {
		password = rand.Text()
	}

	if err := vm.AgentSetUserPassword(ctx, password, user); err != nil {
		return fmt.Errorf("set password for %s on VM %d: %w", user, vmid, err)
	}

	fmt.Printf("login: %s password: %s\n", user, password)

	return nil
}
