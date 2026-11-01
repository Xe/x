package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/luthermonson/go-proxmox"
)

type containerAPI struct {
	t            *testing.T
	created      []map[string]any
	operations   []string
	status       string
	name         string
	failCreate   bool
	warnCreate   bool
	failStart    bool
	failDelete   bool
	rejectCreate bool
	onStart      func()
	onInterfaces func()
}

func (a *containerAPI) serve(w http.ResponseWriter, r *http.Request) {
	a.t.Helper()
	path := r.URL.Path
	var data any
	switch {
	case path == "/nodes/shachi/status":
		data = map[string]any{}
	case path == "/cluster/status":
		data = []any{}
	case path == "/cluster/nextid":
		data = "101"
	case path == "/nodes/shachi/lxc" && r.Method == http.MethodPost:
		if a.rejectCreate {
			http.Error(w, "ID already exists", http.StatusConflict)
			return
		}
		var opts map[string]any
		if err := json.NewDecoder(r.Body).Decode(&opts); err != nil {
			a.t.Errorf("decode create body: %v", err)
		}
		a.created = append(a.created, opts)
		a.name, _ = opts["hostname"].(string)
		a.status = "stopped"
		a.operations = append(a.operations, "create")
		data = "UPID:shachi:1:1:1:vzcreate:101:root@pam:"
	case strings.Contains(path, "/tasks/") && strings.HasSuffix(path, "/log"):
		data = []map[string]any{{"n": 1, "t": "WARN: Systemd 255 detected. You may need to enable nesting."}}
	case strings.Contains(path, "/tasks/"):
		exit := "OK"
		if a.warnCreate && strings.Contains(path, ":vzcreate:") {
			exit = "WARNINGS: 1"
		}
		if (a.failStart && strings.Contains(path, ":vzstart:")) || (a.failCreate && strings.Contains(path, ":vzcreate:")) {
			exit = "test task failed"
		}
		data = map[string]any{"status": "stopped", "exitstatus": exit, "upid": strings.Split(path, "/")[4], "node": "shachi"}
	case path == "/nodes/shachi/lxc/101/status/current":
		data = map[string]any{"vmid": 101, "name": a.name, "status": a.status}
	case path == "/nodes/shachi/lxc/101/config":
		data = map[string]any{"hostname": a.name}
	case path == "/nodes/shachi/lxc/101/status/start":
		a.operations = append(a.operations, "start")
		if !a.failStart {
			a.status = "running"
		}
		if a.onStart != nil {
			a.onStart()
		}
		data = "UPID:shachi:1:1:1:vzstart:101:root@pam:"
	case path == "/nodes/shachi/lxc/101/interfaces":
		if a.onInterfaces != nil {
			a.onInterfaces()
		}
		data = []any{}
	case path == "/nodes/shachi/lxc/101/status/stop":
		a.operations = append(a.operations, "stop")
		a.status = "stopped"
		data = "UPID:shachi:1:1:1:vzstop:101:root@pam:"
	case path == "/nodes/shachi/lxc/101" && r.Method == http.MethodDelete:
		a.operations = append(a.operations, "delete")
		if a.failDelete {
			http.Error(w, "cleanup failed", http.StatusInternalServerError)
			return
		}
		if r.URL.Query().Get("purge") != "1" || r.URL.Query().Get("destroy-unreferenced-disks") != "1" {
			a.t.Errorf("delete query = %s", r.URL.RawQuery)
		}
		data = "UPID:shachi:1:1:1:vzdestroy:101:root@pam:"
	default:
		a.t.Errorf("unexpected API request: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected request", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"data": data})
}

func testContainerOptions() containerOptions {
	return containerOptions{osTemplate: "local:vztmpl/ubuntu.tar.zst", diskStorage: "ssd", diskSize: 32, bridge: "vmbr0", memory: 2048, cores: 2, nesting: true}
}

func TestContainerLifecycle(t *testing.T) {
	for _, tt := range []struct {
		name                                                         string
		failCreate, failStart, rejectCreate, cancelStart, warnCreate bool
		want                                                         []string
		wantErr                                                      bool
	}{
		{name: "warning still starts container", warnCreate: true, want: []string{"create", "start", "stop", "delete"}},
		{name: "fresh create and delete", want: []string{"create", "start", "stop", "delete"}},
		{name: "failed start deletes stopped container", failStart: true, want: []string{"create", "start", "delete"}, wantErr: true},
		{name: "failed create task deletes partial container", failCreate: true, want: []string{"create", "delete"}, wantErr: true},
		{name: "rejected create never deletes existing ID", rejectCreate: true, wantErr: true},
		{name: "interrupt still stops and deletes", cancelStart: true, want: []string{"create", "start", "stop", "delete"}, wantErr: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			a := &containerAPI{t: t, failCreate: tt.failCreate, failStart: tt.failStart, rejectCreate: tt.rejectCreate, warnCreate: tt.warnCreate}
			if tt.cancelStart {
				a.onStart = cancel
			}
			server := httptest.NewServer(http.HandlerFunc(a.serve))
			defer server.Close()
			client := proxmox.NewClient(server.URL)
			n, err := client.Node(ctx, "shachi")
			if err != nil {
				t.Fatal(err)
			}
			ct, err := createContainer(ctx, client, n, testContainerOptions(), "ssh-ed25519 test")
			if (err != nil) != tt.wantErr {
				t.Fatalf("create error = %v, wantErr %v", err, tt.wantErr)
			}
			if ct != nil {
				if err := deleteContainer(ctx, ct); err != nil {
					t.Fatal(err)
				}
			}
			if !slices.Equal(a.operations, tt.want) {
				t.Errorf("operations = %v, want %v", a.operations, tt.want)
			}
			if len(a.created) > 0 {
				opts := a.created[0]
				for key, want := range map[string]any{"ostemplate": "local:vztmpl/ubuntu.tar.zst", "rootfs": "ssd:32", "unprivileged": float64(1), "features": "nesting=1", "net0": "name=eth0,bridge=vmbr0,ip=dhcp", "ssh-public-keys": "ssh-ed25519 test"} {
					if opts[key] != want {
						t.Errorf("%s = %v, want %v", key, opts[key], want)
					}
				}
			}
		})
	}
}

func TestScriptSetupFailureCleanup(t *testing.T) {
	for _, failDelete := range []bool{false, true} {
		t.Run(fmt.Sprintf("delete failure %v", failDelete), func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			a := &containerAPI{t: t, onInterfaces: cancel, failDelete: failDelete}
			server := httptest.NewServer(http.HandlerFunc(a.serve))
			defer server.Close()
			oldHost, oldNode := *proxmoxHost, *proxmoxNode
			*proxmoxHost, *proxmoxNode = server.URL, "shachi"
			t.Cleanup(func() { *proxmoxHost, *proxmoxNode = oldHost, oldNode })
			path := filepath.Join(t.TempDir(), "script.js")
			if err := os.WriteFile(path, []byte("$`uname -a`"), 0600); err != nil {
				t.Fatal(err)
			}
			s := &scriptCmd{containerOptions: testContainerOptions()}
			err := s.run(ctx, path)
			if !errors.Is(err, context.Canceled) {
				t.Errorf("error = %v, want cancellation", err)
			}
			if failDelete && !strings.Contains(err.Error(), "delete container 101") {
				t.Errorf("cleanup failure missing from %v", err)
			}
			if want := []string{"create", "start", "stop", "delete"}; !slices.Equal(a.operations, want) {
				t.Errorf("operations = %v, want %v", a.operations, want)
			}
			if len(a.created) != 1 {
				t.Errorf("created %d containers, want 1", len(a.created))
			}
			key, _ := a.created[0]["ssh-public-keys"].(string)
			if !strings.HasPrefix(key, "ssh-ed25519 ") {
				t.Errorf("no temporary SSH public key: %q", key)
			}
		})
	}
}

func TestContainerIPv4(t *testing.T) {
	for _, tt := range []struct{ name, inet, want string }{
		{"CIDR", "192.0.2.10/24", "192.0.2.10"},
		{"plain address", "192.0.2.10", "192.0.2.10"},
		{"loopback", "127.0.0.1/8", ""},
		{"link local", "169.254.1.1/16", ""},
		{"unspecified", "0.0.0.0", ""},
		{"IPv6", "2001:db8::1/64", ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := containerIPv4(proxmox.ContainerInterfaces{&proxmox.ContainerInterface{Name: "eth0", Inet: tt.inet}})
			if got != tt.want {
				t.Errorf("IPv4 = %q, want %q", got, tt.want)
			}
		})
	}
}
