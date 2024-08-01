package python

import (
	"context"
	"testing"
)

func TestRun(t *testing.T) {
	var code = `import sys

print(f"Python {sys.version} running in {sys.platform}/wazero.")`

	dir := t.TempDir()

	res, err := Run(context.Background(), dir, code)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("stdout: %s", res.Stdout)
	t.Logf("stderr: %s", res.Stderr)
}
