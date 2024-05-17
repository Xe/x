package flagfolder

import (
	"flag"
	"testing"
)

func TestFlagFolderSimple(t *testing.T) {
	for _, cs := range []struct {
		flagName  string
		wantValue string
	}{
		{
			flagName:  "foo",
			wantValue: "foo",
		},
		{
			flagName:  "bar",
			wantValue: "bar",
		},
		{
			flagName:  "something-here",
			wantValue: "something here",
		},
		{
			flagName:  "what-is-computer",
			wantValue: "what is computer",
		},
	} {
		t.Run(cs.flagName, func(t *testing.T) {
			fs := flag.NewFlagSet("flagfolder_test", flag.PanicOnError)

			f := fs.String(cs.flagName, "fail", "help for "+cs.flagName)

			if err := ParseSet("./testdata", fs); err != nil {
				t.Errorf("can't parse ./testdata: %v", err)
			}

			if *f != cs.wantValue {
				t.Errorf("wanted --%s to be %q, got: %q", cs.flagName, cs.wantValue, *f)
			}
		})
	}
}
