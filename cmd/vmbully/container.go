package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/luthermonson/go-proxmox"
	"golang.org/x/crypto/ssh"
	"within.website/x/misc/namegen"
)

type containerOptions struct {
	osTemplate  string
	diskStorage string
	diskSize    int
	bridge      string
	memory      int
	cores       int
	nesting     bool
}

func (o *containerOptions) setFlags(f *flag.FlagSet) {
	f.StringVar(&o.osTemplate, "os-template", "", "downloaded Proxmox container archive, e.g. local:vztmpl/ubuntu.tar.zst (required)")
	f.StringVar(&o.diskStorage, "disk-storage", "ssd", "storage for the container root filesystem")
	f.IntVar(&o.diskSize, "disk-size", 32, "root filesystem size in GiB")
	f.StringVar(&o.bridge, "bridge", "vmbr0", "network bridge for DHCP")
	f.IntVar(&o.memory, "memory", 2048, "container memory in MiB")
	f.IntVar(&o.cores, "cores", 2, "container CPU cores")
	f.BoolVar(&o.nesting, "nesting", true, "enable nesting for container systemd compatibility")
}

func (o containerOptions) validate() error {
	if !strings.Contains(o.osTemplate, ":vztmpl/") {
		return errors.New("set --os-template to a downloaded Proxmox container archive (storage:vztmpl/file)")
	}
	if o.diskStorage == "" || strings.ContainsAny(o.diskStorage, ":,\n") || o.bridge == "" || strings.ContainsAny(o.bridge, ",\n") {
		return errors.New("set valid --disk-storage and --bridge names")
	}
	if o.diskSize <= 0 || o.memory <= 0 || o.cores <= 0 {
		return errors.New("--disk-size, --memory and --cores must be positive")
	}
	return nil
}

// createContainer owns only the ID allocated by a successful create request.
// On subsequent errors it deletes that container, even after cancellation.
func createContainer(ctx context.Context, client *proxmox.Client, n *proxmox.Node, o containerOptions, publicKeys string) (ct *proxmox.Container, err error) {
	if err := o.validate(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(publicKeys) == "" {
		return nil, errors.New("container requires an SSH public key")
	}
	cluster, err := client.Cluster(ctx)
	if err != nil {
		return nil, fmt.Errorf("get cluster: %w", err)
	}
	id, err := cluster.NextID(ctx)
	if err != nil {
		return nil, fmt.Errorf("allocate container ID: %w", err)
	}
	name := namegen.Next()
	features := "nesting=0"
	if o.nesting {
		features = "nesting=1"
	}
	task, err := n.NewContainer(ctx, id,
		proxmox.ContainerOption{Name: "hostname", Value: name},
		proxmox.ContainerOption{Name: "ostemplate", Value: o.osTemplate},
		proxmox.ContainerOption{Name: "rootfs", Value: fmt.Sprintf("%s:%d", o.diskStorage, o.diskSize)},
		proxmox.ContainerOption{Name: "net0", Value: "name=eth0,bridge=" + o.bridge + ",ip=dhcp"},
		proxmox.ContainerOption{Name: "memory", Value: o.memory},
		proxmox.ContainerOption{Name: "cores", Value: o.cores},
		proxmox.ContainerOption{Name: "unprivileged", Value: 1},
		proxmox.ContainerOption{Name: "features", Value: features},
		proxmox.ContainerOption{Name: "ssh-public-keys", Value: publicKeys},
	)
	if err != nil {
		return nil, fmt.Errorf("create container %d: %w", id, err)
	}
	slog.Info("creating container", "id", id, "name", name)
	// Finish the accepted create task before cleanup, including on Ctrl-C.
	finishCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Minute)
	defer cancel()
	if err := waitTask(finishCtx, task, time.Second, 5*time.Minute); err != nil {
		if task != nil && task.IsFailed {
			failed, lookupErr := n.Container(finishCtx, id)
			if lookupErr == nil && failed.Name == name {
				return nil, fmt.Errorf("create container %d: %w", id, errors.Join(err, deleteContainer(finishCtx, failed)))
			}
		}
		// A task with unknown status can still be writing its root filesystem.
		// Report the ID rather than deleting a container with an active task.
		return nil, fmt.Errorf("create container %d (inspect its create task before manual cleanup): %w", id, err)
	}
	ct, err = n.Container(finishCtx, id)
	if err != nil {
		return nil, fmt.Errorf("get new container %d (manual cleanup required): %w", id, err)
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, deleteContainer(finishCtx, ct))
			ct = nil
		}
	}()
	task, err = ct.Start(ctx)
	if err != nil {
		return ct, fmt.Errorf("start container %d: %w", id, err)
	}
	if err = waitTask(finishCtx, task, time.Second, 5*time.Minute); err != nil {
		return ct, fmt.Errorf("start container %d: %w", id, err)
	}
	if err = ctx.Err(); err != nil {
		return ct, err
	}
	return ct, nil
}

// deleteContainer is called only for a container created by this invocation.
func deleteContainer(ctx context.Context, ct *proxmox.Container) error {
	if err := ct.Ping(ctx); err != nil {
		return fmt.Errorf("get container %d status for cleanup: %w", ct.VMID, err)
	}
	if ct.Status != "stopped" {
		task, err := ct.Stop(ctx)
		if err != nil {
			return fmt.Errorf("stop container %d: %w", ct.VMID, err)
		}
		if err := waitTask(ctx, task, time.Second, 5*time.Minute); err != nil {
			return fmt.Errorf("stop container %d: %w", ct.VMID, err)
		}
	}
	task, err := ct.Delete(ctx, &proxmox.ContainerDeleteOptions{
		Purge:                    proxmox.IntOrBool(true),
		DestroyUnreferencedDisks: proxmox.IntOrBool(true),
	})
	if err != nil {
		return fmt.Errorf("delete container %d: %w", ct.VMID, err)
	}
	if err := waitTask(ctx, task, time.Second, 5*time.Minute); err != nil {
		return fmt.Errorf("delete container %d: %w", ct.VMID, err)
	}
	slog.Info("container deleted", "id", ct.VMID)
	return nil
}

func containerIPv4(ifaces proxmox.ContainerInterfaces) string {
	for _, iface := range ifaces {
		addr := strings.SplitN(iface.Inet, "/", 2)[0]
		if iface.Name != "lo" && isUsableIPv4(addr) && !net.ParseIP(addr).IsUnspecified() {
			return addr
		}
	}
	return ""
}

func connectContainer(ctx context.Context, ct *proxmox.Container, signer ssh.Signer) (*ssh.Client, error) {
	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = 2 * time.Second
	bo.MaxInterval = 10 * time.Second
	bo.MaxElapsedTime = 5 * time.Minute
	return backoff.RetryNotifyWithData(func() (*ssh.Client, error) {
		ifaces, err := ct.Interfaces(ctx)
		if err != nil {
			return nil, err
		}
		ip := containerIPv4(ifaces)
		if ip == "" {
			return nil, fmt.Errorf("container %d has no usable IPv4 address", ct.VMID)
		}
		return sshDial(ctx, ip, signer)
	}, backoff.WithContext(bo, ctx), func(err error, next time.Duration) {
		slog.Info("container SSH not ready", "id", ct.VMID, "err", err, "retry_in", next)
	})
}
