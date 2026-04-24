// Package xess vendors a copy of Xess and makes it available at /.xess/xess.css
//
// This is intended to be used as a vendored package in other projects.
package xess

import (
	"embed"
	"net/http"

	"github.com/a-h/templ"
	"within.website/x"
	"within.website/x/internal"
)

//go:generate go tool templ generate

var (
	//go:embed xess.css static
	Static embed.FS

	URL = "/.within.website/x/xess/xess.css"
)

func init() {
	Mount(http.DefaultServeMux)

	URL = URL + "?cachebuster=" + x.Version
}

func Mount(mux *http.ServeMux) {
	mux.Handle("/.within.website/x/xess/", internal.UnchangingCache(http.StripPrefix("/.within.website/x/xess/", http.FileServerFS(Static))))
}

func buttonClass(v ButtonVariant) string {
	if v == "" {
		v = BtnPrimary
	}
	return "xe-btn " + string(v)
}

func admonitionClass(k AdmonitionKind) string {
	if k == "" {
		k = AdmonitionInfo
	}
	return "xe-admonition xe-admonition--" + string(k)
}

func badgeClass(k BadgeKind) string {
	if k == "" {
		k = BadgeNeutral
	}
	return "xe-badge xe-badge--" + string(k)
}

func toastClass(k ToastKind) string {
	if k == "" {
		k = ToastInfo
	}
	return "xe-toast xe-toast--" + string(k)
}

// StickerURL builds a stickers.xeiaso.net URL for the given character and mood.
func StickerURL(character, mood string) string {
	return "https://stickers.xeiaso.net/sticker/" + character + "/" + mood
}

func NotFound(w http.ResponseWriter, r *http.Request) {
	templ.Handler(
		Simple("Not found: "+r.URL.Path, fourohfour(r.URL.Path)),
		templ.WithStatus(http.StatusNotFound),
	)
}
