package sigv4a

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// vectorContext mirrors the context.json in each aws-c-auth v4a test vector.
type vectorContext struct {
	Credentials struct {
		AccessKeyID     string `json:"access_key_id"`
		SecretAccessKey string `json:"secret_access_key"`
	} `json:"credentials"`
	Region    string `json:"region"`
	Service   string `json:"service"`
	Timestamp string `json:"timestamp"`
}

func loadVectorContext(t *testing.T, dir string) vectorContext {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "v4a", dir, "context.json"))
	if err != nil {
		t.Fatalf("read context.json: %v", err)
	}
	var vc vectorContext
	if err := json.Unmarshal(b, &vc); err != nil {
		t.Fatalf("parse context.json: %v", err)
	}
	return vc
}

func (vc vectorContext) signingTime(t *testing.T) time.Time {
	t.Helper()
	ts, err := time.Parse(time.RFC3339, vc.Timestamp)
	if err != nil {
		t.Fatalf("parse timestamp: %v", err)
	}
	return ts
}

// readVectorFile returns the named file from a vector directory verbatim.
func readVectorFile(t *testing.T, dir, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "v4a", dir, name))
	if err != nil {
		t.Fatalf("read %s/%s: %v", dir, name, err)
	}
	return string(b)
}

// vectorPublicKey parses the vector's public-key.json (hex X/Y coordinates).
func vectorPublicKey(t *testing.T, dir string) *ecdsa.PublicKey {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "v4a", dir, "public-key.json"))
	if err != nil {
		t.Fatalf("read public-key.json: %v", err)
	}
	var pk struct{ X, Y string }
	if err := json.Unmarshal(b, &pk); err != nil {
		t.Fatalf("parse public-key.json: %v", err)
	}
	x, ok := new(big.Int).SetString(pk.X, 16)
	if !ok {
		t.Fatalf("bad X coordinate %q", pk.X)
	}
	y, ok := new(big.Int).SetString(pk.Y, 16)
	if !ok {
		t.Fatalf("bad Y coordinate %q", pk.Y)
	}
	return &ecdsa.PublicKey{Curve: elliptic.P256(), X: x, Y: y}
}

// vectorDirs lists every vector directory under testdata/v4a.
func vectorDirs(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join("testdata", "v4a"))
	if err != nil {
		t.Fatalf("read testdata/v4a: %v", err)
	}
	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, e.Name())
		}
	}
	if len(dirs) == 0 {
		t.Fatal("no test vectors found")
	}
	return dirs
}
