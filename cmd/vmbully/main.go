package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/google/subcommands"
	"github.com/luthermonson/go-proxmox"
	"within.website/x/internal"
)

var (
	proxmoxHost    = flag.String("proxmox-host", "", "proxmox host")
	proxmoxTokenID = flag.String("proxmox-token-id", "", "proxmox token ID")
	proxmoxSecret  = flag.String("proxmox-secret", "", "proxmox API secret")
	proxmoxNode    = flag.String("proxmox-node", "shachi", "proxmox node to create LXC containers on")
)

func main() {
	subcommands.Register(subcommands.HelpCommand(), "")
	subcommands.Register(subcommands.FlagsCommand(), "")
	subcommands.Register(subcommands.CommandsCommand(), "")

	subcommands.Register(&createCmd{}, "LXC containers")
	subcommands.Register(&scriptCmd{}, "LXC containers")

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

var taskWarningStatus = regexp.MustCompile(`^WARNINGS: [0-9]+$`)

// waitTask accepts completed tasks with OK or WARNINGS statuses.
// The Proxmox client labels warnings as failures, unlike Proxmox itself.
func waitTask(ctx context.Context, task *proxmox.Task, interval, max time.Duration) error {
	if task == nil {
		return fmt.Errorf("Proxmox returned no task ID")
	}
	if err := task.Wait(ctx, interval, max); err != nil {
		return err
	}

	if task.Status != "stopped" {
		return fmt.Errorf("task %s has unexpected status %q", task.UPID, task.Status)
	}
	if task.ExitStatus == "OK" {
		return nil
	}
	if taskWarningStatus.MatchString(task.ExitStatus) {
		slog.Warn("Proxmox task completed with warnings", "task", task.UPID, "status", task.ExitStatus)
		lines, err := task.Log(ctx, 0, 100)
		if err != nil {
			slog.Warn("can't read task warnings", "task", task.UPID, "err", err)
			return nil
		}
		var numbers []int
		for number := range lines {
			numbers = append(numbers, number)
		}
		slices.Sort(numbers)
		for _, number := range numbers {
			if line := lines[number]; strings.HasPrefix(strings.TrimSpace(line), "WARN:") {
				slog.Warn("Proxmox task warning", "task", task.UPID, "warning", line)
			}
		}
		return nil
	}
	return fmt.Errorf("task %s failed: %s", task.UPID, task.ExitStatus)
}
