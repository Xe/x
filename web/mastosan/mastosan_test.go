package mastosan

import (
	"context"
	"testing"
	"time"
)

func TestHTML2Slackdown(t *testing.T) {
	for _, tt := range []struct {
		name     string
		inp, out string
	}{
		{"basic mention", `<p>test mention <span class="h-card"><a href="https://vt.social/@xe" class="u-url mention">@<span>xe</span></a></span> so I can see what HTML mastodon makes</p>`, "test mention <https://vt.social/@xe|@xe> so I can see what HTML mastodon makes"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			result, err := HTML2Slackdown(ctx, tt.inp)
			if err != nil {
				t.Fatal(err)
			}

			if tt.out != result {
				t.Logf("inp: %s", tt.inp)
				t.Logf("out: %q", tt.out)
				t.Logf("got: %q", result)
				t.Fatal("output did not match what was expected")
			}
		})
	}
}
