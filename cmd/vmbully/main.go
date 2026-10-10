package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/google/subcommands"
	"github.com/luthermonson/go-proxmox"
	"within.website/x/internal"
)

var (
	proxmoxHost    = flag.String("proxmox-host", "", "proxmox host")
	proxmoxTokenID = flag.String("proxmox-token-id", "", "proxmox token ID")
	proxmoxSecret  = flag.String("proxmox-secret", "", "proxmox API secret")
	proxmoxNode    = flag.String("proxmox-node", "shachi", "proxmox node to create VMs on")
)

func main() {
	subcommands.Register(subcommands.HelpCommand(), "")
	subcommands.Register(subcommands.FlagsCommand(), "")
	subcommands.Register(subcommands.CommandsCommand(), "")

	subcommands.Register(&cloneCmd{}, "VM manipulation")
	subcommands.Register(&passwdCmd{}, "VM manipulation")
	subcommands.Register(&scriptCmd{}, "VM manipulation")
	subcommands.Register(&templateCmd{}, "VM manipulation")

	internal.HandleStartup()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	os.Exit(int(subcommands.Execute(ctx)))
}

// node connects to the Proxmox API and returns the client and the target node.
func node(ctx context.Context) (*proxmox.Client, *proxmox.Node, error) {
	client := proxmox.NewClient(*proxmoxHost,
		proxmox.WithAPIToken(*proxmoxTokenID, *proxmoxSecret),
		proxmox.WithInsecureSkipVerify(),    // lab only
		proxmox.WithTimeout(30*time.Second), // http.DefaultClient has no timeout
	)

	n, err := client.Node(ctx, *proxmoxNode)
	if err != nil {
		return nil, nil, fmt.Errorf("get node %s: %w", *proxmoxNode, err)
	}

	return client, n, nil
}

// waitTask waits for task and returns an error when the task fails.
// task.Wait alone returns nil for a task that ends with a failure.
func waitTask(ctx context.Context, task *proxmox.Task, interval, max time.Duration) error {
	if err := task.Wait(ctx, interval, max); err != nil {
		return err
	}

	if task.IsFailed {
		return fmt.Errorf("task %s failed: %s", task.UPID, task.ExitStatus)
	}

	return nil
}
