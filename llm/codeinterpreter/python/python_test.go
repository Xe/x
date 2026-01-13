package python

import (
	"context"
	"testing"
)

func TestRun(t *testing.T) {
	var code = `import sys

print(f"Python {sys.version} running in {sys.platform}/wazero.")`

	res, err := Run(context.Background(), nil, code)
	if err != nil {
		t.Logf("stdout: %s", res.Stdout)
		t.Logf("stderr: %s", res.Stderr)
		t.Logf("platform error: %s", res.PlatformError)
		t.Fatal(err)
	}

	t.Logf("stdout: %s", res.Stdout)
	t.Logf("stderr: %s", res.Stderr)
}
