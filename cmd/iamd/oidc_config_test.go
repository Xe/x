package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOIDCConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(path, []byte("upstream-secret\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name             string
		enabled          bool
		issuer, id, file string
		wantErr          bool
	}{
		{name: "disabled"},
		{name: "enabled", enabled: true, issuer: "https://iamd.xeserv.us", id: "iamd", file: path},
		{name: "missing client", enabled: true, issuer: "https://iamd.xeserv.us", file: path, wantErr: true},
		{name: "missing secret", enabled: true, issuer: "https://iamd.xeserv.us", id: "iamd", wantErr: true},
		{name: "insecure issuer", enabled: true, issuer: "http://iamd.xeserv.us", id: "iamd", file: path, wantErr: true},
		{name: "issuer path", enabled: true, issuer: "https://iamd.xeserv.us/path", id: "iamd", file: path, wantErr: true},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := loadOIDCConfig(tt.enabled, tt.issuer, tt.id, tt.file)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, tt.wantErr)
			}
			if tt.enabled && !tt.wantErr && cfg.UpstreamClientSecret != "upstream-secret" {
				t.Fatal("secret not loaded")
			}
		})
	}
}
