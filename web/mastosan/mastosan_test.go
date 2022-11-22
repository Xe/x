package mastosan

import (
	"context"
	"fmt"
	"io"
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

const msg = `<p>Tailscale has recently been notified of security vulnerabilities in the Tailscale Windows client which allow a malicious website visited by a device running Tailscale to change the Tailscale daemon configuration and access information in the Tailscale local and peer APIs.</p><p>To patch these vulnerabilities, upgrade Tailscale on your Windows machines to Tailscale v1.32.3 or later, or v1.33.257 or later (unstable).</p><p><a href="https://tailscale.com/blog/windows-security-vulnerabilities/" target="_blank" rel="nofollow noopener noreferrer"><span class="invisible">https://</span><span class="ellipsis">tailscale.com/blog/windows-sec</span><span class="invisible">urity-vulnerabilities/</span></a></p>`

func BenchmarkHTML2Slackdown(b *testing.B) {
	b.RunParallel(benchStep)
}

func benchStep(pb *testing.PB) {
	for pb.Next() {
		result, err := HTML2Slackdown(context.Background(), msg)
		if err != nil {
			panic(err)
		}
		fmt.Fprintln(io.Discard, result)
	}
}
