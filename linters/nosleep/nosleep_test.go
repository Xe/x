package nosleep

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestNosleep ensures that the nosleep analyzer catches use of time.Sleep in testing code.
func TestNosleep(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), Analyzer)
}
