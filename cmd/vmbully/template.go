package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/google/subcommands"
	"github.com/luthermonson/go-proxmox"
	"within.website/x/internal/flagenv"
)

type templateCmd struct {
	id            int
	name          string
	diskStorage   string
	importStorage string
	imageURL      string
	imageName     string
	ciUser        string
	ciPassword    string
	sshKeysFile   string
	ciVendor      string
}

func (*templateCmd) Name() string     { return "template" }
func (*templateCmd) Synopsis() string { return "Create a VM template from a cloud image." }
func (*templateCmd) Usage() string {
	return `template [--id] [--name] [--disk-storage] [--ssh-keys-file] [...]:
Download a cloud image to the import storage, create a VM from it, and convert the VM to a template.
`
}
func (t *templateCmd) SetFlags(f *flag.FlagSet) {
	f.IntVar(&t.id, "id", 9000, "VMID of the template to create")
	f.StringVar(&t.name, "name", "ubuntu-26-04-resolute", "name of the template to create")
	f.StringVar(&t.diskStorage, "disk-storage", "ssd", "storage for the disks of the template")
	f.StringVar(&t.importStorage, "import-storage", "local", "storage for the downloaded cloud image, needs the import content type")
	f.StringVar(&t.imageURL, "image-url", "https://cloud-images.ubuntu.com/resolute/current/resolute-server-cloudimg-amd64.img", "cloud image to build the template from")
	f.StringVar(&t.imageName, "image-name", "resolute-server-cloudimg-amd64.qcow2", "file name of the cloud image on the import storage, must end in .qcow2")
	f.StringVar(&t.ciUser, "ci-user", "xe", "cloud-init user")
	f.StringVar(&t.ciPassword, "ci-password", "", "cloud-init password (optional)")
	f.StringVar(&t.sshKeysFile, "ssh-keys-file", "", "file with SSH public keys for the cloud-init user (optional)")
	f.StringVar(&t.ciVendor, "ci-vendor", "", "volume ID of a cloud-init vendor snippet, for example local:snippets/qemu_guest_agent.yaml (optional)")
}

func (t *templateCmd) Execute(ctx context.Context, f *flag.FlagSet, _ ...any) subcommands.ExitStatus {
	if err := flagenv.ParseSet(flagenv.Prefix, f); err != nil {
		slog.Error("can't read flags from the environment", "err", err)
		return subcommands.ExitUsageError
	}

	if err := t.run(ctx); err != nil {
		slog.Error("can't create template", "err", err)
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}

func (t *templateCmd) run(ctx context.Context) error {
	if !strings.HasSuffix(t.imageName, ".qcow2") {
		return fmt.Errorf("--image-name %q must end in .qcow2", t.imageName)
	}

	var sshKeys string
	if t.sshKeysFile != "" {
		data, err := os.ReadFile(t.sshKeysFile)
		if err != nil {
			return fmt.Errorf("read SSH keys: %w", err)
		}
		sshKeys = strings.TrimSpace(string(data))
	}

	client, n, err := node(ctx)
	if err != nil {
		return err
	}

	// Check the VMID before the download so a taken ID fails fast.
	cluster, err := client.Cluster(ctx)
	if err != nil {
		return fmt.Errorf("get cluster: %w", err)
	}
	resources, err := cluster.Resources(ctx, "vm")
	if err != nil {
		return fmt.Errorf("list cluster VMs: %w", err)
	}
	for _, r := range resources {
		if r.VMID == uint64(t.id) {
			return fmt.Errorf("VMID %d is in use by %q on node %s", t.id, r.Name, r.Node)
		}
	}

	// PVE pulls the image onto the storage itself. import-from only takes a
	// volume from a storage with the import content type when the caller is
	// not root@pam.
	storage, err := n.Storage(ctx, t.importStorage)
	if err != nil {
		return fmt.Errorf("get storage %s: %w", t.importStorage, err)
	}

	imageVolID := fmt.Sprintf("%s:import/%s", t.importStorage, t.imageName)
	content, err := storage.GetContent(ctx)
	if err != nil {
		return fmt.Errorf("list content of storage %s: %w", t.importStorage, err)
	}

	if hasVolume(content, imageVolID) {
		slog.Info("image already present, skipping download", "volid", imageVolID)
	} else {
		slog.Info("downloading image", "url", t.imageURL, "volid", imageVolID)
		task, err := storage.DownloadURL(ctx, "import", t.imageName, t.imageURL)
		if err != nil {
			return fmt.Errorf("download image: %w", err)
		}
		if err := waitTask(ctx, task, 5*time.Second, 20*time.Minute); err != nil {
			return fmt.Errorf("download image: %w", err)
		}
	}

	options := []proxmox.VirtualMachineOption{
		{Name: "name", Value: t.name},
		{Name: "ostype", Value: "l26"},
		{Name: "bios", Value: "ovmf"},
		{Name: "net0", Value: "virtio,bridge=vmbr0"},
		// Cloud images ship a serial console; point the display at it.
		{Name: "serial0", Value: "socket"},
		{Name: "vga", Value: "serial0"},
		{Name: "memory", Value: 2048},
		{Name: "cores", Value: 2},
		{Name: "cpu", Value: "host"},
		{Name: "balloon", Value: 512},
		// "storage:0" takes the disk size from the import source.
		{Name: "scsi0", Value: fmt.Sprintf("%s:0,import-from=%s,discard=on", t.diskStorage, imageVolID)},
		{Name: "boot", Value: "order=scsi0"},
		{Name: "scsihw", Value: "virtio-scsi-pci"},
		{Name: "agent", Value: "enabled=1,fstrim_cloned_disks=1"},
		{Name: "scsi2", Value: fmt.Sprintf("%s:cloudinit", t.diskStorage)},
		{Name: "ipconfig0", Value: "ip=dhcp"},
		{Name: "ciuser", Value: t.ciUser},
		{Name: "efidisk0", Value: fmt.Sprintf("%s:1,format=raw,efitype=4m,pre-enrolled-keys=1", t.diskStorage)},
	}
	if t.ciPassword != "" {
		options = append(options, proxmox.VirtualMachineOption{Name: "cipassword", Value: t.ciPassword})
	}
	if sshKeys != "" {
		// PVE rejects Go's default query escaping for this field.
		options = append(options, proxmox.VirtualMachineOption{Name: "sshkeys", Value: proxmox.EncodeSSHKeys(sshKeys)})
	}
	if t.ciVendor != "" {
		options = append(options, proxmox.VirtualMachineOption{Name: "cicustom", Value: "vendor=" + t.ciVendor})
	}

	slog.Info("creating VM", "id", t.id, "name", t.name)
	task, err := n.NewVirtualMachine(ctx, t.id, options...)
	if err != nil {
		return fmt.Errorf("create VM %d: %w", t.id, err)
	}
	if err := waitTask(ctx, task, time.Second, 5*time.Minute); err != nil {
		return fmt.Errorf("create VM %d: %w", t.id, err)
	}

	vm, err := n.VirtualMachine(ctx, t.id)
	if err != nil {
		return fmt.Errorf("get VM %d: %w", t.id, err)
	}

	task, err = vm.ConvertToTemplate(ctx)
	if err != nil {
		return fmt.Errorf("convert VM %d to template: %w", t.id, err)
	}
	if err := waitTask(ctx, task, time.Second, 2*time.Minute); err != nil {
		return fmt.Errorf("convert VM %d to template: %w", t.id, err)
	}

	slog.Info("template ready", "id", t.id, "name", t.name)

	return nil
}

func hasVolume(content []*proxmox.StorageContent, volid string) bool {
	for _, c := range content {
		if c.Volid == volid {
			return true
		}
	}
	return false
}
