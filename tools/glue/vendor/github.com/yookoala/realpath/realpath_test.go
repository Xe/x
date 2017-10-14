package realpath

import (
	"testing"
)

var symlinkTests []symlinkTest

type symlinkTest struct {
	path   []byte
	start  int
	link   string
	after  string
	expect []byte
}

func (t symlinkTest) expecting(result []byte) bool {
	return string(result) == string(t.expect)
}

func init() {
	symlinkTests = []symlinkTest{
		symlinkTest{
			path:   []byte("/absolute/path/link/test"),
			start:  14,
			link:   "/another/absolute/path",
			after:  "/test",
			expect: []byte("/another/absolute/path/test"),
		},
		symlinkTest{
			path:   []byte("/absolute/path/link/test"),
			start:  14,
			link:   "./relative/path2",
			after:  "/test",
			expect: []byte("/absolute/path/relative/path2/test"),
		},
		symlinkTest{
			path:   []byte("/absolute/path/link/test"),
			start:  14,
			link:   "../relative/path2",
			after:  "/test",
			expect: []byte("/absolute/relative/path2/test"),
		},
	}
}

func TestSwitchSymlinkCom(t *testing.T) {
	for i, tt := range symlinkTests {
		if r := switchSymlinkCom(tt.path, tt.start, tt.link, tt.after); !tt.expecting(r) {
			t.Errorf("Failed test %d.\nExpected: \"%s\"\nActually: \"%s\"",
				i, tt.expect, r)
		}
	}
}
