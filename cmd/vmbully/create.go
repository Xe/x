package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/google/subcommands"
	"golang.org/x/crypto/ssh"
	"within.website/x/internal/flagenv"
)

type createCmd struct {
	containerOptions
	sshKeysFile string
}

func (*createCmd) Name() string     { return "create" }
func (*createCmd) Synopsis() string { return "Create and start an LXC container for manual use." }
func (*createCmd) Usage() string {
	return "create --os-template=<storage:vztmpl/archive> --ssh-keys-file=<file> [flags]:\nCreate an LXC container and leave it running. Delete it manually when finished.\n"
}
func (c *createCmd) SetFlags(f *flag.FlagSet) {
	c.containerOptions.setFlags(f)
	f.StringVar(&c.sshKeysFile, "ssh-keys-file", "", "file with SSH public keys for root (required)")
}
func (c *createCmd) Execute(ctx context.Context, f *flag.FlagSet, _ ...any) subcommands.ExitStatus {
	if err := flagenv.ParseSet(flagenv.Prefix, f); err != nil {
		slog.Error("can't read flags from the environment", "err", err)
		return subcommands.ExitUsageError
	}
	if f.NArg() != 0 || c.sshKeysFile == "" {
		fmt.Print(c.Usage())
		return subcommands.ExitUsageError
	}
	if err := c.run(ctx); err != nil {
		slog.Error("can't create container", "err", err)
		return subcommands.ExitFailure
	}
	return subcommands.ExitSuccess
}

func (c *createCmd) run(ctx context.Context) error {
	if err := c.containerOptions.validate(); err != nil {
		return err
	}
	data, err := os.ReadFile(c.sshKeysFile)
	if err != nil {
		return fmt.Errorf("read SSH keys: %w", err)
	}
	keys := strings.TrimSpace(string(data))
	if keys == "" {
		return errors.New("SSH keys file is empty")
	}
	for rest := []byte(keys); len(rest) > 0; {
		_, _, _, remaining, err := ssh.ParseAuthorizedKey(rest)
		if err != nil {
			return fmt.Errorf("parse SSH keys: %w", err)
		}
		rest = remaining
	}
	client, n, err := node(ctx)
	if err != nil {
		return err
	}
	ct, err := createContainer(ctx, client, n, c.containerOptions, keys)
	if err != nil {
		return err
	}
	fmt.Printf("container: %d on %s (%s)\n", ct.VMID, ct.Node, ct.Name)
	fmt.Println("Container left running. Use the Proxmox interface to find its DHCP address and delete it when finished.")
	return nil
}
