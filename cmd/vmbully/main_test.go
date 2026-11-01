package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/luthermonson/go-proxmox"
)

func TestWaitTask(t *testing.T) {
	for _, tt := range []struct {
		name, exit string
		logError   bool
		wantErr    bool
	}{
		{name: "success", exit: "OK"},
		{name: "warning", exit: "WARNINGS: 1"},
		{name: "warnings with inaccessible log", exit: "WARNINGS: 2", logError: true},
		{name: "actual failure", exit: "unable to create CT", wantErr: true},
		{name: "failure mentioning warnings", exit: "WARNINGS: 1; creation failed", wantErr: true},
		{name: "missing exit status", wantErr: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			upid := proxmox.UPID("UPID:shachi:1:1:1:vzcreate:101:root@pam:")
			logRequests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if strings.HasSuffix(r.URL.Path, "/log") {
					logRequests++
					if tt.logError {
						http.Error(w, "log unavailable", http.StatusForbidden)
						return
					}
					json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{{"n": 1, "t": "WARN: Systemd 255 detected. You may need to enable nesting."}}})
					return
				}
				json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"upid": upid, "node": "shachi", "status": "stopped", "exitstatus": tt.exit}})
			}))
			defer server.Close()
			task := proxmox.NewTask(upid, proxmox.NewClient(server.URL))
			err := waitTask(t.Context(), task, time.Millisecond, time.Second)
			if (err != nil) != tt.wantErr {
				t.Errorf("waitTask error = %v, wantErr %v", err, tt.wantErr)
			}
			if strings.HasPrefix(tt.exit, "WARNINGS: ") && !tt.wantErr && logRequests != 1 {
				t.Errorf("warning log requests = %d, want 1", logRequests)
			}
		})
	}
}
