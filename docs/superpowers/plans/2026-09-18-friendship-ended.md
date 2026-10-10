# friendship-ended Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build `cmd/friendship-ended`, a Go web app that makes "Friendship ended with X, now Y is my best friend" meme images with gg and stores renders and uploads in Tigris.

**Architecture:** One `main` package. `render.go` is a pure image function (gg plus embedded assets). `store.go` hides Tigris behind a small `Store` interface so handlers are tested against an in-memory fake. `handlers.go` has four routes that render templ pages styled with the xe-design-system CSS.

**Tech Stack:** Go 1.25, `github.com/fogleman/gg` v1.3.0, `golang.org/x/image/font/opentype`, `github.com/disintegration/imaging`, `github.com/a-h/templ`, `aws-sdk-go-v2/service/s3` through `within.website/x/tigris`, `github.com/google/uuid`, `golang.org/x/sync/errgroup`.

**Spec:** `docs/superpowers/specs/2026-09-18-friendship-ended-design.md`

## Global Constraints

- Binaries call `internal.HandleStartup()` and never call `flag.Parse()` themselves.
- Log with `log/slog`, key `"err"` for errors. No `log` package.
- Every commit uses `git commit --signoff`, a Conventional Commits message, and the footer `Assisted-by: Claude Opus 5 via Claude Code`.
- `x1.png`, `x2.png`, `treasure.gif` are GPL-2.0 (Leigh Aucoin, https://github.com/lmaucoin/friendship-ended) and must stay marked as such.
- Comic Sans MS Bold is extracted from `https://downloads.sourceforge.net/corefonts/comic32.exe`. The user confirmed permission to embed it.
- Upload limits: 32 MB per request, 10 MB per file, 40 megapixels per image, names 1 to 64 runes after trimming.
- Only GIF, JPEG and PNG are accepted, detected by content, never by client MIME type.
- Tigris keys: `friendships/<uuidv7>/{result.jpg,new.<ext>,old1.<ext>,old2.<ext>,meta.json}`. `<ext>` is `gif`, `jpg` or `png`.
- ASCII only in code, comments and copy.
- The scratchpad for downloads is `/tmp/claude-1000/-home-xe-Code-Xe-x/6da801a3-7bd2-438e-9d6a-747f9259cf5c/scratchpad`. Do not put downloads in the repo tree except the final `comicbd.ttf`.
- Work happens on branch `Xe/friendship-ended`.
- `npm run generate` currently rewrites unrelated files (`cmd/hdrwtch/static/css/styles.css`, `gen/**`). Never stage those. Always `git add` explicit paths.

## File Map

```text
cmd/friendship-ended/
  main.go              flags, wiring                       (Task 3)
  doc.go               package doc, go:generate            (Task 2)
  embed.go             //go:embed assets static            (Task 1)
  render.go            renderer, Render                     (Task 1)
  render_test.go       render tests + snapshot              (Task 1)
  store.go             Store, ErrNotFound, s3Store          (Task 2)
  memstore_test.go     memStore fake                        (Task 2)
  handlers.go          Server, routes, upload validation    (Task 2)
  handlers_test.go     handler tests                        (Task 2)
  web.templ            base, indexPage, resultPage, errorPage (Task 2)
  web_templ.go         generated                            (Task 2)
  assets/              GPL assets + font                    (Task 1)
    x1.png x2.png treasure.gif LICENSE README.md comicbd.ttf FONT.md
  static/              design system CSS + fonts            (Task 2)
    colors_and_type.css site.css fonts/*.woff2
  testdata/.gitkeep                                         (Task 1)
```

---

### Task 1: Assets and renderer

**Files:**

- Create: `cmd/friendship-ended/embed.go`
- Create: `cmd/friendship-ended/render.go`
- Create: `cmd/friendship-ended/render_test.go`
- Create: `cmd/friendship-ended/assets/{x1.png,x2.png,treasure.gif,LICENSE,README.md,comicbd.ttf,FONT.md}`
- Create: `cmd/friendship-ended/static/.gitkeep` (so the embed pattern compiles; Task 2 fills it)
- Create: `cmd/friendship-ended/testdata/.gitkeep`
- Modify: `go.mod`, `go.sum`

**Interfaces:**

- Consumes: nothing.
- Produces:
  - `var content embed.FS` holding `assets/` and `static/`.
  - `type renderer struct` (unexported fields).
  - `func newRenderer() (*renderer, error)`
  - `func (r *renderer) Render(oldName, newName string, newPic, old1, old2 image.Image) (image.Image, error)`. The output is always 800x600, and the method is safe for concurrent use.

- [ ] **Step 1: Copy the GPL assets and the license**

```bash
cd /home/xe/Code/Xe/x
mkdir -p cmd/friendship-ended/assets cmd/friendship-ended/static cmd/friendship-ended/testdata
cp var/friendship-ended/x1.png var/friendship-ended/x2.png var/friendship-ended/treasure.gif var/friendship-ended/LICENSE cmd/friendship-ended/assets/
touch cmd/friendship-ended/static/.gitkeep cmd/friendship-ended/testdata/.gitkeep
```

- [ ] **Step 2: Fetch Comic Sans MS Bold**

```bash
SCRATCH=/tmp/claude-1000/-home-xe-Code-Xe-x/6da801a3-7bd2-438e-9d6a-747f9259cf5c/scratchpad
cd $SCRATCH
curl -fsSLo comic32.exe https://downloads.sourceforge.net/corefonts/comic32.exe
7z e -y comic32.exe comicbd.ttf
sha256sum comic32.exe comicbd.ttf
file comicbd.ttf
cp comicbd.ttf /home/xe/Code/Xe/x/cmd/friendship-ended/assets/comicbd.ttf
```

Expected: `file` reports `TrueType Font data`. Record both SHA-256 values for Step 3.

- [ ] **Step 3: Write `assets/README.md` and `assets/FONT.md`**

`cmd/friendship-ended/assets/README.md`:

```markdown
# Assets

`x1.png`, `x2.png`, and `treasure.gif` come from
[friendship-ended](https://github.com/lmaucoin/friendship-ended) by Leigh
Aucoin. They are licensed under the GNU General Public License, version 2.
See `LICENSE` in this folder.

`comicbd.ttf` is Comic Sans MS Bold. See `FONT.md`.
```

`cmd/friendship-ended/assets/FONT.md` (paste in the real hashes from Step 2):

```markdown
# comicbd.ttf

Comic Sans MS Bold, copyright Microsoft Corporation.

Source: https://downloads.sourceforge.net/corefonts/comic32.exe, extracted
with `7z e comic32.exe comicbd.ttf`.

- comic32.exe SHA-256: <hash from step 2>
- comicbd.ttf SHA-256: <hash from step 2>
```

- [ ] **Step 4: Add gg**

```bash
cd /home/xe/Code/Xe/x && go get github.com/fogleman/gg@v1.3.0
```

- [ ] **Step 5: Write `embed.go`**

```go
package main

import "embed"

// content holds the render assets (GPL-2.0, see assets/README.md) and the
// static files served to browsers.
//
//go:embed assets static
var content embed.FS
```

- [ ] **Step 6: Write the failing tests in `render_test.go`**

```go
package main

import (
	"flag"
	"image"
	"image/color"
	"image/color/palette"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "write testdata/out.png for visual inspection")

func solid(w, h int, c color.Color) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), image.NewUniform(c), image.Point{}, draw.Src)
	return img
}

func paletted(w, h int) image.Image {
	img := image.NewPaletted(image.Rect(0, 0, w, h), palette.Plan9)
	for i := range img.Pix {
		img.Pix[i] = uint8(i % len(palette.Plan9))
	}
	return img
}

func testRenderer(t *testing.T) *renderer {
	t.Helper()
	r, err := newRenderer()
	if err != nil {
		t.Fatalf("newRenderer: %v", err)
	}
	return r
}

func TestRender(t *testing.T) {
	r := testRenderer(t)
	white := color.White

	for _, tt := range []struct {
		name             string
		oldName, newName string
		newPic, old1     image.Image
		old2             image.Image
	}{
		{
			name:    "normal",
			oldName: "mudasir", newName: "salman",
			newPic: solid(1024, 768, white), old1: solid(300, 400, white), old2: solid(300, 400, white),
		},
		{
			name:    "tiny inputs",
			oldName: "a", newName: "b",
			newPic: solid(1, 1, white), old1: solid(1, 1, white), old2: solid(1, 1, white),
		},
		{
			name:    "huge inputs",
			oldName: "big", newName: "bigger",
			newPic: solid(4000, 3000, white), old1: solid(3000, 4000, white), old2: solid(3000, 4000, white),
		},
		{
			name:    "paletted gif input",
			oldName: "gif", newName: "jif",
			newPic: paletted(640, 480), old1: paletted(100, 100), old2: paletted(100, 100),
		},
		{
			name:    "empty names",
			newPic: solid(800, 600, white), old1: solid(10, 10, white), old2: solid(10, 10, white),
		},
		{
			name:    "long names",
			oldName: strings.Repeat("W", 64), newName: strings.Repeat("M", 64),
			newPic: solid(800, 600, white), old1: solid(10, 10, white), old2: solid(10, 10, white),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := r.Render(tt.oldName, tt.newName, tt.newPic, tt.old1, tt.old2)
			if err != nil {
				t.Fatalf("Render: %v", err)
			}
			if want := image.Rect(0, 0, 800, 600); got.Bounds() != want {
				t.Fatalf("bounds = %v, want %v", got.Bounds(), want)
			}
		})
	}
}

// TestRenderDrawsText checks that the title and caption land where the
// original put them: title in the top band, captions starting at x=300.
func TestRenderDrawsText(t *testing.T) {
	r := testRenderer(t)
	white := color.White

	got, err := r.Render("mudasir", "salman", solid(800, 600, white), solid(10, 10, white), solid(10, 10, white))
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	for _, tt := range []struct {
		name string
		area image.Rectangle
		min  int
	}{
		{name: "title band", area: image.Rect(0, 0, 800, 100), min: 2000},
		{name: "caption column", area: image.Rect(300, 100, 800, 380), min: 2000},
		{name: "left of captions stays clean", area: image.Rect(0, 120, 280, 330), min: 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			n := countNonWhite(got, tt.area)
			if tt.min == 0 && n != 0 {
				t.Fatalf("%d non-white pixels in %v, want 0", n, tt.area)
			}
			if n < tt.min {
				t.Fatalf("%d non-white pixels in %v, want at least %d", n, tt.area, tt.min)
			}
		})
	}
}

func countNonWhite(img image.Image, area image.Rectangle) int {
	n := 0
	for y := area.Min.Y; y < area.Max.Y; y++ {
		for x := area.Min.X; x < area.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			if r < 0xf000 || g < 0xf000 || b < 0xf000 {
				n++
			}
		}
	}
	return n
}

// TestRenderSnapshot writes testdata/out.png with -update so a human can
// compare it to the original meme layout.
func TestRenderSnapshot(t *testing.T) {
	if !*update {
		t.Skip("pass -update to write testdata/out.png")
	}
	r := testRenderer(t)

	got, err := r.Render("mudasir", "salman",
		solid(800, 600, color.RGBA{0x9f, 0xc5, 0xe8, 0xff}),
		solid(172, 259, color.RGBA{0xd9, 0xb3, 0x8c, 0xff}),
		solid(221, 242, color.RGBA{0xb3, 0x8c, 0x66, 0xff}),
	)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	fout, err := os.Create(filepath.Join("testdata", "out.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer fout.Close()
	if err := png.Encode(fout, got); err != nil {
		t.Fatal(err)
	}
}
```

- [ ] **Step 7: Run the tests and confirm they fail**

Run: `go test ./cmd/friendship-ended/ -run TestRender`
Expected: build failure, `undefined: newRenderer`.

- [ ] **Step 8: Write `render.go`**

```go
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/fogleman/gg"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

const (
	canvasWidth  = 800
	canvasHeight = 600
	strokeColor  = "#006488"
)

// strokeOffsets fakes a 1px text stroke by drawing the text in the stroke
// color at each neighboring offset before drawing the fill on top.
var strokeOffsets = [][2]float64{
	{-1, -1}, {0, -1}, {1, -1},
	{-1, 0}, {1, 0},
	{-1, 1}, {0, 1}, {1, 1},
}

// renderer draws friendship ended images. It is safe for concurrent use:
// the parsed font and overlays are read-only and font faces are made per
// call.
type renderer struct {
	font   *opentype.Font
	x1, x2 image.Image
}

// textLine is one piece of text in the image. Positions are in the
// coordinate space after scaling, matching ImageMagick's annotation after
// scale in the original PHP. An empty fill means the title gradient.
type textLine struct {
	text           string
	size           float64
	scaleX, scaleY float64
	x, y           float64
	fill           string
}

func newRenderer() (*renderer, error) {
	fontData, err := content.ReadFile("assets/comicbd.ttf")
	if err != nil {
		return nil, fmt.Errorf("can't read font: %w", err)
	}

	f, err := opentype.Parse(fontData)
	if err != nil {
		return nil, fmt.Errorf("can't parse font: %w", err)
	}

	x1, err := loadPNG("assets/x1.png")
	if err != nil {
		return nil, err
	}

	x2, err := loadPNG("assets/x2.png")
	if err != nil {
		return nil, err
	}

	return &renderer{font: f, x1: x1, x2: x2}, nil
}

func loadPNG(name string) (image.Image, error) {
	fin, err := content.Open(name)
	if err != nil {
		return nil, fmt.Errorf("can't open %s: %w", name, err)
	}
	defer fin.Close()

	img, err := png.Decode(fin)
	if err != nil {
		return nil, fmt.Errorf("can't decode %s: %w", name, err)
	}

	return img, nil
}

// Render makes the final 800x600 image. Names are trimmed and uppercased.
func (r *renderer) Render(oldName, newName string, newPic, old1, old2 image.Image) (image.Image, error) {
	oldName = strings.ToUpper(strings.TrimSpace(oldName))
	newName = strings.ToUpper(strings.TrimSpace(newName))

	dc := gg.NewContextForImage(imaging.Resize(newPic, canvasWidth, canvasHeight, imaging.Lanczos))

	lines := []textLine{
		{text: "Friendship ended with " + oldName, size: 58, scaleX: 0.8, scaleY: 2, x: 0, y: 40},
		{text: "Now", size: 38, scaleX: 1, scaleY: 2, x: 300, y: 70, fill: "#DF0676"},
		{text: newName, size: 38, scaleX: 1, scaleY: 2, x: 300, y: 110, fill: "#AB5955"},
		{text: "is my", size: 38, scaleX: 1, scaleY: 2, x: 300, y: 150, fill: "#7C9535"},
		{text: "best friend", size: 38, scaleX: 1, scaleY: 2, x: 300, y: 185, fill: "#4CBF1F"},
	}

	for _, l := range lines {
		if err := r.drawLine(dc, l); err != nil {
			return nil, err
		}
	}

	dc.DrawImage(crossOut(old1, 172, 259, r.x1), 0, 341)
	dc.DrawImage(crossOut(old2, 221, 242, r.x2), 579, 358)

	return dc.Image(), nil
}

func (r *renderer) drawLine(dc *gg.Context, l textLine) error {
	face, err := opentype.NewFace(r.font, &opentype.FaceOptions{
		Size:    l.size,
		DPI:     72,
		Hinting: font.HintingNone,
	})
	if err != nil {
		return fmt.Errorf("can't make %v point font face: %w", l.size, err)
	}
	defer face.Close()

	drawText := func(c *gg.Context, fill color.Color, dx, dy float64) {
		c.Push()
		defer c.Pop()
		c.Scale(l.scaleX, l.scaleY)
		c.SetFontFace(face)
		c.SetColor(fill)
		c.DrawString(l.text, l.x+dx, l.y+dy)
	}

	stroke := hexColor(strokeColor)
	for _, off := range strokeOffsets {
		drawText(dc, stroke, off[0], off[1])
	}

	if l.fill != "" {
		drawText(dc, hexColor(l.fill), 0, 0)
		return nil
	}

	// gg only draws text in a solid color, so render the glyphs into a
	// mask and paint the gradient through it. The original used a 50x120
	// gradient pattern in pre-scale space, so the gradient spans 120 *
	// scaleY device pixels.
	mask := gg.NewContext(dc.Width(), dc.Height())
	drawText(mask, color.White, 0, 0)

	grad := gg.NewLinearGradient(0, 0, 0, 120*l.scaleY)
	grad.AddColorStop(0, hexColor("#CF4E09"))
	grad.AddColorStop(1, hexColor("#00B92C"))

	dc.Push()
	defer dc.Pop()
	if err := dc.SetMask(mask.AsMask()); err != nil {
		return fmt.Errorf("can't set gradient mask: %w", err)
	}
	dc.SetFillStyle(grad)
	dc.DrawRectangle(0, 0, float64(dc.Width()), float64(dc.Height()))
	dc.Fill()

	return nil
}

// crossOut resizes pic to w x h and draws overlay on top, clipped to the
// photo like ImageMagick's COMPOSITE_ATOP.
func crossOut(pic image.Image, w, h int, overlay image.Image) image.Image {
	c := gg.NewContextForImage(imaging.Resize(pic, w, h, imaging.Lanczos))
	c.DrawImage(overlay, 0, 0)
	return c.Image()
}

// hexColor parses #RRGGBB. It panics on bad input because it is only
// called with constants.
func hexColor(s string) color.Color {
	var r, g, b uint8
	if _, err := fmt.Sscanf(s, "#%02x%02x%02x", &r, &g, &b); err != nil {
		panic(fmt.Sprintf("bad hex color %q: %v", s, err))
	}
	return color.RGBA{R: r, G: g, B: b, A: 0xff}
}
```

- [ ] **Step 9: Run the tests and confirm they pass**

Run: `go test ./cmd/friendship-ended/ -run TestRender -v`
Expected: `TestRender` (6 subtests) and `TestRenderDrawsText` (3 subtests) PASS. `TestRenderSnapshot` SKIP.

If the "left of captions stays clean" subtest fails, the text is landing somewhere unexpected. Debug the scale and position math before loosening the test.

- [ ] **Step 10: Look at the snapshot**

Run: `go test ./cmd/friendship-ended/ -run TestRenderSnapshot -update`
Then open `cmd/friendship-ended/testdata/out.png` with the Read tool.

Check:

- The title "Friendship ended with MUDASIR" spans the top, is squished horizontally and tall, and has an orange-to-green gradient.
- The caption column at x=300 reads Now / SALMAN / is my / best friend in pink, brown, olive and green. Every line has a thin teal outline.
- Old friend boxes sit bottom-left and bottom-right, each with a green X.

Do not commit `out.png`.

- [ ] **Step 11: Tidy and vet**

Run: `go mod tidy && go vet ./cmd/friendship-ended/`
Expected: no output from vet. `go.mod` has `github.com/fogleman/gg v1.3.0` in the direct require block.

- [ ] **Step 12: Commit**

```bash
git add go.mod go.sum cmd/friendship-ended/embed.go cmd/friendship-ended/render.go cmd/friendship-ended/render_test.go cmd/friendship-ended/assets cmd/friendship-ended/static/.gitkeep cmd/friendship-ended/testdata/.gitkeep
git commit --signoff -m "feat(friendship-ended): add gg renderer and GPL assets

Port the ImageMagick compositing from the PHP friendship-ended
generator to fogleman/gg. Reuse the original GPL-2.0 overlay and
treasure assets and embed Comic Sans MS Bold for the text.

Assisted-by: Claude Opus 5 via Claude Code"
```

---

### Task 2: Store, handlers, and pages

**Files:**

- Create: `cmd/friendship-ended/store.go`
- Create: `cmd/friendship-ended/memstore_test.go`
- Create: `cmd/friendship-ended/handlers.go`
- Create: `cmd/friendship-ended/handlers_test.go`
- Create: `cmd/friendship-ended/web.templ` (and generated `web_templ.go`)
- Create: `cmd/friendship-ended/static/colors_and_type.css`, `static/site.css`, `static/fonts/*.woff2`
- Delete: `cmd/friendship-ended/static/.gitkeep`
- Create: `cmd/friendship-ended/doc.go` (package doc and `go:generate`, so `templ generate` works before `main.go` exists)

**Interfaces:**

- Consumes: `content`, `newRenderer`, `(*renderer).Render` from Task 1.
- Produces:
  - `var ErrNotFound = errors.New("store: key not found")`
  - `type Store interface { Put(ctx context.Context, key, contentType string, data []byte) error; Get(ctx context.Context, key string) ([]byte, error); Presign(ctx context.Context, key string) (string, error) }`
  - `func newS3Store(client *s3.Client, bucket string, expiry time.Duration) *s3Store`
  - `func newServer(store Store, rend *renderer, baseURL string) *Server`
  - `func (s *Server) Handler() http.Handler`

- [ ] **Step 1: Copy the design system**

```bash
cd /home/xe/Code/Xe/x
DS=/home/xe/.claude/plugins/cache/xe-agent-plugins/xe-design/9b01305736ef/skills/xe-design-system
mkdir -p cmd/friendship-ended/static/fonts
cp $DS/fonts/*.woff2 cmd/friendship-ended/static/fonts/
sed -E '/format\("woff2"\),$/s/,$/;/; /format\("truetype-variations"\);/d' $DS/colors_and_type.css > cmd/friendship-ended/static/colors_and_type.css
git rm -q cmd/friendship-ended/static/.gitkeep
grep -n "url(" cmd/friendship-ended/static/colors_and_type.css
```

Expected: exactly three `url(` lines. Each ends in `format("woff2");` and none mention `.ttf`.

- [ ] **Step 2: Write `static/site.css`**

```css
.page {
  max-width: 52rem;
  margin: 0 auto;
  padding: var(--space-7) var(--space-4) var(--space-6);
}

.lead {
  font-size: var(--step-1);
  color: var(--text-muted);
}

.friend-form {
  display: grid;
  gap: var(--space-2);
  max-width: 36rem;
}

.friend-form label {
  font-weight: 600;
  margin-top: var(--space-3);
}

.friend-form input[type="text"] {
  font: inherit;
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg-hard);
  color: var(--text);
}

.friend-form .help {
  margin: 0;
  font-size: var(--step--1);
  color: var(--text-muted);
}

.friend-form button {
  justify-self: start;
  margin-top: var(--space-4);
}

.adm {
  display: flex;
  margin: 0 0 var(--space-4);
  background: var(--bg-0);
  border: 1px solid var(--bg-3);
  border-radius: var(--radius-md);
  overflow: hidden;
  box-shadow: var(--shadow-sm);
}

.adm .bar {
  flex: 0 0 12px;
}

.adm--warn .bar {
  background: var(--red-bright);
}

.adm .body {
  padding: var(--space-3) var(--space-4);
}

.adm h4 {
  font-family: var(--font-sans);
  font-size: var(--step-0);
  font-weight: 700;
  margin: 0 0 var(--space-1);
}

.adm p {
  margin: 0;
}

.result {
  margin: 0 0 var(--space-5);
}

.result img,
.treasure {
  display: block;
  max-width: 100%;
  height: auto;
}

.treasure {
  margin: 0 auto var(--space-5);
}

.page-footer {
  max-width: 52rem;
  margin: 0 auto;
  padding: var(--space-4);
  border-top: 1px solid var(--border);
  font-size: var(--step--1);
  color: var(--text-muted);
}
```

- [ ] **Step 3: Write `doc.go`**

```go
// Command friendship-ended makes "Friendship ended with X, now Y is my best
// friend" images and stores them in Tigris.
//
// It is a port of https://github.com/lmaucoin/friendship-ended by Leigh
// Aucoin.
package main

//go:generate go tool templ generate
```

- [ ] **Step 4: Write `web.templ`**

```templ
package main

import "strings"

type formState struct {
	OldName string
	NewName string
	Error   string
}

templ base(title string, head templ.Component) {
	<!DOCTYPE html>
	<html lang="en">
		<head>
			<meta charset="utf-8"/>
			<meta name="viewport" content="width=device-width, initial-scale=1"/>
			<title>{ title }</title>
			<meta name="description" content="Your friendship has ended and I am so sorry."/>
			<link rel="stylesheet" href="/static/colors_and_type.css"/>
			<link rel="stylesheet" href="/static/site.css"/>
			if head != nil {
				@head
			}
		</head>
		<body>
			<main class="page">
				{ children... }
			</main>
			<footer class="page-footer">
				<p>
					A port of <a href="https://github.com/lmaucoin/friendship-ended">friendship-ended</a> by Leigh Aucoin.
					The overlay and treasure images are GPL-2.0.
				</p>
			</footer>
		</body>
	</html>
}

templ indexPage(f formState) {
	@base("Friendship Ended Generator", nil) {
		<h1>Friendship Ended Generator</h1>
		<p class="lead">
			Dedicated to the <a href="http://internet.gawker.com/facebook-user-asif-ends-friendship-with-mudasir-welc-1731160013">new friendship of ASIF and SALMAN</a>.
		</p>
		<p><em>All fields are required. Just like in <strong>friendship</strong>...</em></p>
		if f.Error != "" {
			<div class="adm adm--warn" role="alert">
				<div class="bar"></div>
				<div class="body">
					<h4>That didn't work</h4>
					<p>{ f.Error }</p>
				</div>
			</div>
		}
		<form class="xe-card friend-form" method="post" action="/" enctype="multipart/form-data">
			<label for="old-friend-name">BYE</label>
			<input type="text" id="old-friend-name" name="old-friend-name" placeholder="Old friend's name" maxlength="64" value={ f.OldName } required/>
			<label for="new-friend-name">HI</label>
			<input type="text" id="new-friend-name" name="new-friend-name" placeholder="New friend's name" maxlength="64" value={ f.NewName } required/>
			<label for="new-friend-pic">A picture of your new friend</label>
			<input type="file" id="new-friend-pic" name="new-friend-pic" accept="image/gif,image/jpeg,image/png" required/>
			<p class="help">You can be in it but you don't have to be.</p>
			<label for="old-friend-1">A picture of your old friend</label>
			<input type="file" id="old-friend-1" name="old-friend-1" accept="image/gif,image/jpeg,image/png" required/>
			<p class="help">Hopefully, an unflattering one.</p>
			<label for="old-friend-2">Another picture of your old friend</label>
			<input type="file" id="old-friend-2" name="old-friend-2" accept="image/gif,image/jpeg,image/png" required/>
			<p class="help">They are no longer your friend.</p>
			<button type="submit" class="xe-btn xe-btn--primary">FRIEND</button>
		</form>
	}
}

templ resultHead(baseURL, id string, m meta) {
	<meta property="og:type" content="website"/>
	<meta property="og:title" content={ "Friendship ended with " + strings.ToUpper(m.OldName) }/>
	<meta property="og:description" content={ "Now " + strings.ToUpper(m.NewName) + " is my best friend." }/>
	<meta property="og:url" content={ baseURL + "/f/" + id }/>
	<meta property="og:image" content={ baseURL + "/f/" + id + "/image.jpg" }/>
	<meta property="og:image:width" content="800"/>
	<meta property="og:image:height" content="600"/>
	<meta name="twitter:card" content="summary_large_image"/>
}

templ resultPage(baseURL, id string, m meta) {
	@base("Friendship Ended Generator", resultHead(baseURL, id, m)) {
		<h1>Bless your new friendship with <em>{ strings.ToUpper(m.NewName) }</em></h1>
		<figure class="xe-card result">
			<img src={ "/f/" + id + "/image.jpg" } width="800" height="600" alt={ "Friendship ended with " + strings.ToUpper(m.OldName) + ". Now " + strings.ToUpper(m.NewName) + " is my best friend." }/>
		</figure>
		<img class="treasure" src="/treasure.gif" width="350" height="350" alt="An animated treasure chest full of gold coins"/>
		<p><em><a href="/">Do you have another new friend...........</a></em></p>
	}
}

templ errorPage(title, message string) {
	@base(title, nil) {
		<h1>{ title }</h1>
		<p>{ message }</p>
		<p><a href="/">Go back and make a new friend</a></p>
	}
}
```

- [ ] **Step 5: Write `store.go`**

```go
package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// ErrNotFound is returned by Store.Get when the key does not exist.
var ErrNotFound = errors.New("store: key not found")

// Store is the object storage the handlers need.
type Store interface {
	Put(ctx context.Context, key, contentType string, data []byte) error
	Get(ctx context.Context, key string) ([]byte, error)
	Presign(ctx context.Context, key string) (string, error)
}

type s3Store struct {
	client  *s3.Client
	presign *s3.PresignClient
	bucket  string
	expiry  time.Duration
}

func newS3Store(client *s3.Client, bucket string, expiry time.Duration) *s3Store {
	return &s3Store{
		client:  client,
		presign: s3.NewPresignClient(client),
		bucket:  bucket,
		expiry:  expiry,
	}
}

func (s *s3Store) Put(ctx context.Context, key, contentType string, data []byte) error {
	if _, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(key),
		Body:          bytes.NewReader(data),
		ContentType:   aws.String(contentType),
		ContentLength: aws.Int64(int64(len(data))),
	}); err != nil {
		return fmt.Errorf("store: can't put %s: %w", key, err)
	}

	return nil
}

func (s *s3Store) Get(ctx context.Context, key string) ([]byte, error) {
	obj, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var nsk *types.NoSuchKey
		if errors.As(err, &nsk) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("store: can't get %s: %w", key, err)
	}
	defer obj.Body.Close()

	data, err := io.ReadAll(obj.Body)
	if err != nil {
		return nil, fmt.Errorf("store: can't read %s: %w", key, err)
	}

	return data, nil
}

func (s *s3Store) Presign(ctx context.Context, key string) (string, error) {
	req, err := s.presign.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(s.expiry))
	if err != nil {
		return "", fmt.Errorf("store: can't presign %s: %w", key, err)
	}

	return req.URL, nil
}
```

- [ ] **Step 6: Write `memstore_test.go`**

```go
package main

import (
	"context"
	"errors"
	"sync"
)

type memObject struct {
	contentType string
	data        []byte
}

// memStore is an in-memory Store for tests. Set failPut to make every Put
// fail.
type memStore struct {
	mu      sync.Mutex
	objects map[string]memObject
	failPut bool
}

func newMemStore() *memStore {
	return &memStore{objects: map[string]memObject{}}
}

func (m *memStore) Put(_ context.Context, key, contentType string, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failPut {
		return errors.New("memstore: put failed on purpose")
	}

	m.objects[key] = memObject{contentType: contentType, data: append([]byte(nil), data...)}
	return nil
}

func (m *memStore) Get(_ context.Context, key string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	obj, ok := m.objects[key]
	if !ok {
		return nil, ErrNotFound
	}

	return obj.data, nil
}

func (m *memStore) Presign(_ context.Context, key string) (string, error) {
	return "https://tigris.example/" + key + "?X-Amz-Signature=fake", nil
}
```

- [ ] **Step 7: Write the failing tests in `handlers_test.go`**

```go
package main

import (
	"bytes"
	"encoding/json"
	"image/color"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"golang.org/x/image/bmp"
)

const testBaseURL = "https://friends.example"

func encodePNG(t *testing.T, w, h int) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, solid(w, h, color.RGBA{0x80, 0x40, 0x20, 0xff})); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func encodeBMP(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := bmp.Encode(&buf, solid(4, 4, color.White)); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

type upload struct {
	field string
	data  []byte
}

func multipartBody(t *testing.T, fields map[string]string, files []upload) (io.Reader, string) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for k, v := range fields {
		if err := mw.WriteField(k, v); err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range files {
		fw, err := mw.CreateFormFile(f.field, f.field+".bin")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := fw.Write(f.data); err != nil {
			t.Fatal(err)
		}
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	return &buf, mw.FormDataContentType()
}

func testServer(t *testing.T) (*Server, *memStore) {
	t.Helper()
	store := newMemStore()
	return newServer(store, testRenderer(t), testBaseURL), store
}

var locationRE = regexp.MustCompile(`^/f/([0-9a-f-]{36})$`)

func TestCreate(t *testing.T) {
	goodNames := map[string]string{"old-friend-name": "mudasir", "new-friend-name": "salman"}
	pic := encodePNG(t, 64, 48)
	goodFiles := []upload{{"new-friend-pic", pic}, {"old-friend-1", pic}, {"old-friend-2", pic}}

	withFile := func(field string, data []byte) []upload {
		out := []upload{}
		for _, f := range goodFiles {
			if f.field == field {
				if data == nil {
					continue
				}
				f.data = data
			}
			out = append(out, f)
		}
		return out
	}

	for _, tt := range []struct {
		name       string
		fields     map[string]string
		files      []upload
		failPut    bool
		wantStatus int
		wantBody   string
	}{
		{name: "happy path", fields: goodNames, files: goodFiles, wantStatus: http.StatusSeeOther},
		{
			name:       "missing old name",
			fields:     map[string]string{"new-friend-name": "salman"},
			files:      goodFiles,
			wantStatus: http.StatusBadRequest,
			wantBody:   "old friend needs a name",
		},
		{
			name:       "blank new name",
			fields:     map[string]string{"old-friend-name": "mudasir", "new-friend-name": "   "},
			files:      goodFiles,
			wantStatus: http.StatusBadRequest,
			wantBody:   "new friend needs a name",
		},
		{
			name:       "name too long",
			fields:     map[string]string{"old-friend-name": strings.Repeat("a", 65), "new-friend-name": "salman"},
			files:      goodFiles,
			wantStatus: http.StatusBadRequest,
			wantBody:   "old friend needs a name",
		},
		{
			name:       "missing file",
			fields:     goodNames,
			files:      withFile("old-friend-2", nil),
			wantStatus: http.StatusBadRequest,
			wantBody:   "please attach a picture",
		},
		{
			name:       "not an image",
			fields:     goodNames,
			files:      withFile("old-friend-1", []byte("hello, this is text")),
			wantStatus: http.StatusBadRequest,
			wantBody:   "isn&#39;t a GIF, JPEG, or PNG",
		},
		{
			name:       "bmp is rejected",
			fields:     goodNames,
			files:      withFile("new-friend-pic", encodeBMP(t)),
			wantStatus: http.StatusBadRequest,
			wantBody:   "isn&#39;t a GIF, JPEG, or PNG",
		},
		{
			name:       "file too big",
			fields:     goodNames,
			files:      withFile("new-friend-pic", make([]byte, maxFileSize+1)),
			wantStatus: http.StatusBadRequest,
			wantBody:   "bigger than 10 MB",
		},
		{
			name:       "storage failure",
			fields:     goodNames,
			files:      goodFiles,
			failPut:    true,
			wantStatus: http.StatusInternalServerError,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			srv, store := testServer(t)
			store.failPut = tt.failPut

			body, ct := multipartBody(t, tt.fields, tt.files)
			req := httptest.NewRequest(http.MethodPost, "/", body)
			req.Header.Set("Content-Type", ct)
			rec := httptest.NewRecorder()

			srv.Handler().ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantBody != "" && !strings.Contains(rec.Body.String(), tt.wantBody) {
				t.Fatalf("body does not contain %q:\n%s", tt.wantBody, rec.Body.String())
			}
			if tt.wantStatus != http.StatusSeeOther {
				if len(store.objects) != 0 {
					t.Fatalf("stored %d objects on failure, want 0", len(store.objects))
				}
				return
			}

			m := locationRE.FindStringSubmatch(rec.Header().Get("Location"))
			if m == nil {
				t.Fatalf("Location = %q, want /f/<uuid>", rec.Header().Get("Location"))
			}
			prefix := "friendships/" + m[1] + "/"

			for key, wantType := range map[string]string{
				"result.jpg": "image/jpeg",
				"new.png":    "image/png",
				"old1.png":   "image/png",
				"old2.png":   "image/png",
				"meta.json":  "application/json",
			} {
				obj, ok := store.objects[prefix+key]
				if !ok {
					t.Errorf("missing object %s", prefix+key)
					continue
				}
				if obj.contentType != wantType {
					t.Errorf("%s content type = %q, want %q", key, obj.contentType, wantType)
				}
			}
			if len(store.objects) != 5 {
				t.Errorf("stored %d objects, want 5", len(store.objects))
			}

			var got meta
			if err := json.Unmarshal(store.objects[prefix+"meta.json"].data, &got); err != nil {
				t.Fatal(err)
			}
			if got.OldName != "mudasir" || got.NewName != "salman" || got.CreatedAt.IsZero() {
				t.Errorf("meta = %+v", got)
			}
		})
	}
}

func TestRead(t *testing.T) {
	const id = "0192f0c1-8a2b-7c3d-9e4f-5a6b7c8d9e0f"
	const unknown = "0192f0c1-8a2b-7c3d-9e4f-000000000000"

	for _, tt := range []struct {
		name         string
		path         string
		wantStatus   int
		wantBody     []string
		wantLocation string
		wantType     string
	}{
		{name: "form", path: "/", wantStatus: http.StatusOK, wantBody: []string{`name="old-friend-name"`, "FRIEND"}},
		{
			name:       "result page",
			path:       "/f/" + id,
			wantStatus: http.StatusOK,
			wantBody: []string{
				"SALMAN",
				`content="` + testBaseURL + "/f/" + id + `/image.jpg"`,
				`src="/f/` + id + `/image.jpg"`,
			},
		},
		{name: "unknown id", path: "/f/" + unknown, wantStatus: http.StatusNotFound},
		{name: "bad id", path: "/f/not-a-uuid", wantStatus: http.StatusNotFound},
		{
			name:         "image redirect",
			path:         "/f/" + id + "/image.jpg",
			wantStatus:   http.StatusFound,
			wantLocation: "https://tigris.example/friendships/" + id + "/result.jpg?X-Amz-Signature=fake",
		},
		{name: "image bad id", path: "/f/nope/image.jpg", wantStatus: http.StatusNotFound},
		{name: "treasure", path: "/treasure.gif", wantStatus: http.StatusOK, wantType: "image/gif"},
		{name: "design css", path: "/static/colors_and_type.css", wantStatus: http.StatusOK, wantType: "text/css; charset=utf-8"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			srv, store := testServer(t)
			metaJSON, err := json.Marshal(meta{OldName: "mudasir", NewName: "salman", CreatedAt: time.Now()})
			if err != nil {
				t.Fatal(err)
			}
			store.objects["friendships/"+id+"/meta.json"] = memObject{contentType: "application/json", data: metaJSON}

			rec := httptest.NewRecorder()
			srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			for _, want := range tt.wantBody {
				if !strings.Contains(rec.Body.String(), want) {
					t.Errorf("body does not contain %q", want)
				}
			}
			if tt.wantLocation != "" && rec.Header().Get("Location") != tt.wantLocation {
				t.Errorf("Location = %q, want %q", rec.Header().Get("Location"), tt.wantLocation)
			}
			if tt.wantType != "" && rec.Header().Get("Content-Type") != tt.wantType {
				t.Errorf("Content-Type = %q, want %q", rec.Header().Get("Content-Type"), tt.wantType)
			}
		})
	}
}
```

- [ ] **Step 8: Generate templates and confirm the tests fail**

Run: `go generate ./cmd/friendship-ended/ && go test ./cmd/friendship-ended/ -run 'TestCreate|TestRead'`
Expected: build failure, `undefined: newServer` and `undefined: meta`.

- [ ] **Step 9: Write `handlers.go`**

```go
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/a-h/templ"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

const (
	maxUploadSize  = 32 << 20
	maxFileSize    = 10 << 20
	maxImagePixels = 40_000_000
	maxNameRunes   = 64
)

// User-facing upload errors. Their text is shown on the form.
var (
	errNoFile        = errors.New("please attach a picture")
	errFileTooBig    = errors.New("that picture is bigger than 10 MB")
	errNotImage      = errors.New("that file isn't a GIF, JPEG, or PNG")
	errImageTooLarge = errors.New("that picture has too many pixels")
)

// meta is stored as meta.json next to each render.
type meta struct {
	OldName   string    `json:"old_name"`
	NewName   string    `json:"new_name"`
	CreatedAt time.Time `json:"created_at"`
}

// Server serves the friendship ended web app.
type Server struct {
	store    Store
	renderer *renderer
	baseURL  string
}

func newServer(store Store, rend *renderer, baseURL string) *Server {
	return &Server{
		store:    store,
		renderer: rend,
		baseURL:  strings.TrimSuffix(baseURL, "/"),
	}
}

// Handler returns the HTTP routes for the app.
func (s *Server) Handler() http.Handler {
	static, err := fs.Sub(content, "static")
	if err != nil {
		panic(fmt.Sprintf("static files missing from embed: %v", err))
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.index)
	mux.HandleFunc("POST /{$}", s.create)
	mux.HandleFunc("GET /f/{id}", s.show)
	mux.HandleFunc("GET /f/{id}/image.jpg", s.image)
	mux.HandleFunc("GET /treasure.gif", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFileFS(w, r, content, "assets/treasure.gif")
	})
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(static)))

	return mux
}

func (s *Server) index(w http.ResponseWriter, r *http.Request) {
	templ.Handler(indexPage(formState{})).ServeHTTP(w, r)
}

func (s *Server) formError(w http.ResponseWriter, r *http.Request, f formState) {
	templ.Handler(indexPage(f), templ.WithStatus(http.StatusBadRequest)).ServeHTTP(w, r)
}

func (s *Server) internalError(w http.ResponseWriter, r *http.Request, msg string, err error) {
	slog.ErrorContext(r.Context(), msg, "err", err)
	templ.Handler(
		errorPage("Something broke", "Something went wrong on our end. Your friendship is still over, though."),
		templ.WithStatus(http.StatusInternalServerError),
	).ServeHTTP(w, r)
}

func (s *Server) notFound(w http.ResponseWriter, r *http.Request) {
	templ.Handler(
		errorPage("Friendship not found", "That friendship either never existed or was never written down."),
		templ.WithStatus(http.StatusNotFound),
	).ServeHTTP(w, r)
}

func (s *Server) create(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		s.formError(w, r, formState{Error: "That upload was too big or broken. Each picture can be at most 10 MB."})
		return
	}
	defer r.MultipartForm.RemoveAll()

	f := formState{
		OldName: strings.TrimSpace(r.FormValue("old-friend-name")),
		NewName: strings.TrimSpace(r.FormValue("new-friend-name")),
	}

	if !validName(f.OldName) {
		f.Error = fmt.Sprintf("Your old friend needs a name, %d characters or less.", maxNameRunes)
		s.formError(w, r, f)
		return
	}

	if !validName(f.NewName) {
		f.Error = fmt.Sprintf("Your new friend needs a name, %d characters or less.", maxNameRunes)
		s.formError(w, r, f)
		return
	}

	uploads := map[string]*imageUpload{}
	for _, field := range []struct{ name, label string }{
		{"new-friend-pic", "New friend picture"},
		{"old-friend-1", "First old friend picture"},
		{"old-friend-2", "Second old friend picture"},
	} {
		up, err := readUpload(r, field.name)
		if err != nil {
			f.Error = fmt.Sprintf("%s: %v.", field.label, err)
			s.formError(w, r, f)
			return
		}
		uploads[field.name] = up
	}

	img, err := s.renderer.Render(f.OldName, f.NewName,
		uploads["new-friend-pic"].img, uploads["old-friend-1"].img, uploads["old-friend-2"].img)
	if err != nil {
		s.internalError(w, r, "can't render image", err)
		return
	}

	var result bytes.Buffer
	if err := jpeg.Encode(&result, img, &jpeg.Options{Quality: 90}); err != nil {
		s.internalError(w, r, "can't encode jpeg", err)
		return
	}

	id, err := uuid.NewV7()
	if err != nil {
		s.internalError(w, r, "can't make id", err)
		return
	}
	prefix := "friendships/" + id.String() + "/"

	objects := []struct {
		key, contentType string
		data             []byte
	}{
		{prefix + "result.jpg", "image/jpeg", result.Bytes()},
		{prefix + "new." + uploads["new-friend-pic"].ext, uploads["new-friend-pic"].contentType, uploads["new-friend-pic"].data},
		{prefix + "old1." + uploads["old-friend-1"].ext, uploads["old-friend-1"].contentType, uploads["old-friend-1"].data},
		{prefix + "old2." + uploads["old-friend-2"].ext, uploads["old-friend-2"].contentType, uploads["old-friend-2"].data},
	}

	g, gctx := errgroup.WithContext(r.Context())
	for _, obj := range objects {
		g.Go(func() error {
			return s.store.Put(gctx, obj.key, obj.contentType, obj.data)
		})
	}
	if err := g.Wait(); err != nil {
		s.internalError(w, r, "can't store images", err)
		return
	}

	// meta.json goes last so a result page never points at missing images.
	metaJSON, err := json.Marshal(meta{OldName: f.OldName, NewName: f.NewName, CreatedAt: time.Now().UTC()})
	if err != nil {
		s.internalError(w, r, "can't marshal metadata", err)
		return
	}
	if err := s.store.Put(r.Context(), prefix+"meta.json", "application/json", metaJSON); err != nil {
		s.internalError(w, r, "can't store metadata", err)
		return
	}

	slog.InfoContext(r.Context(), "friendship ended", "id", id.String())
	http.Redirect(w, r, "/f/"+id.String(), http.StatusSeeOther)
}

func validName(name string) bool {
	n := utf8.RuneCountInString(name)
	return n > 0 && n <= maxNameRunes
}

type imageUpload struct {
	img         image.Image
	data        []byte
	ext         string
	contentType string
}

// readUpload reads and decodes one uploaded picture. Errors it returns are
// safe to show to the user.
func readUpload(r *http.Request, field string) (*imageUpload, error) {
	fin, hdr, err := r.FormFile(field)
	if err != nil {
		return nil, errNoFile
	}
	defer fin.Close()

	if hdr.Size > maxFileSize {
		return nil, errFileTooBig
	}

	data, err := io.ReadAll(io.LimitReader(fin, maxFileSize+1))
	if err != nil {
		return nil, errNotImage
	}
	if len(data) > maxFileSize {
		return nil, errFileTooBig
	}

	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, errNotImage
	}

	var ext string
	switch format {
	case "gif", "png":
		ext = format
	case "jpeg":
		ext = "jpg"
	default:
		return nil, errNotImage
	}

	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width*cfg.Height > maxImagePixels {
		return nil, errImageTooLarge
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, errNotImage
	}

	return &imageUpload{img: img, data: data, ext: ext, contentType: "image/" + format}, nil
}

func parseID(r *http.Request) (string, bool) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		return "", false
	}
	return id.String(), true
}

func (s *Server) show(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r)
	if !ok {
		s.notFound(w, r)
		return
	}

	data, err := s.store.Get(r.Context(), "friendships/"+id+"/meta.json")
	if errors.Is(err, ErrNotFound) {
		s.notFound(w, r)
		return
	}
	if err != nil {
		s.internalError(w, r, "can't load metadata", err)
		return
	}

	var m meta
	if err := json.Unmarshal(data, &m); err != nil {
		s.internalError(w, r, "can't parse metadata", err)
		return
	}

	templ.Handler(resultPage(s.baseURL, id, m)).ServeHTTP(w, r)
}

func (s *Server) image(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r)
	if !ok {
		s.notFound(w, r)
		return
	}

	u, err := s.store.Presign(r.Context(), "friendships/"+id+"/result.jpg")
	if err != nil {
		s.internalError(w, r, "can't presign image", err)
		return
	}

	w.Header().Set("Cache-Control", "private, max-age=300")
	http.Redirect(w, r, u, http.StatusFound)
}
```

Note: `imaging`, imported by `render.go`, registers BMP and TIFF decoders. The `switch format` whitelist is what keeps them out. The "bmp is rejected" test covers it.

- [ ] **Step 10: Run the tests and confirm they pass**

Run: `go generate ./cmd/friendship-ended/ && go test ./cmd/friendship-ended/ -v`
Expected: `TestCreate` (9 subtests), `TestRead` (8 subtests), `TestRender` and `TestRenderDrawsText` PASS.

If "not an image" or "bmp is rejected" fails on the body check, look at the actual body in the failure output. templ escapes `'` as `&#39;`. Update `wantBody` only if templ uses a different escape.

- [ ] **Step 11: Tidy and vet**

Run: `go mod tidy && go vet ./cmd/friendship-ended/`
Expected: vet prints nothing.

- [ ] **Step 12: Commit**

```bash
git add go.mod go.sum cmd/friendship-ended/doc.go cmd/friendship-ended/store.go cmd/friendship-ended/memstore_test.go cmd/friendship-ended/handlers.go cmd/friendship-ended/handlers_test.go cmd/friendship-ended/web.templ cmd/friendship-ended/web_templ.go cmd/friendship-ended/static
git commit --signoff -m "feat(friendship-ended): add web handlers and Tigris storage

Serve the upload form and unlisted result pages styled with the
xe-design-system. Renders and the original uploads go to Tigris, and
result images are served through presigned URL redirects.

Assisted-by: Claude Opus 5 via Claude Code"
```

---

### Task 3: main.go and a live run

**Files:**

- Create: `cmd/friendship-ended/main.go`

**Interfaces:**

- Consumes: `newRenderer`, `newS3Store`, `newServer`, `(*Server).Handler`, `tigris.Client(ctx) (*s3.Client, error)` from `within.website/x/tigris`.
- Produces: the `friendship-ended` binary.

- [ ] **Step 1: Write `main.go`**

```go
package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"time"

	"within.website/x/internal"
	"within.website/x/tigris"
)

var (
	bind          = flag.String("bind", ":3000", "HTTP address to bind to")
	bucket        = flag.String("bucket", "friendship-ended", "Tigris bucket to store friendships in")
	baseURL       = flag.String("base-url", "http://localhost:3000", "public URL of this service, used in OpenGraph tags")
	presignExpiry = flag.Duration("presign-expiry", time.Hour, "how long presigned image URLs stay valid")
)

func main() {
	internal.HandleStartup()

	ctx := context.Background()

	s3c, err := tigris.Client(ctx)
	if err != nil {
		slog.Error("can't create Tigris client", "err", err)
		os.Exit(1)
	}

	rend, err := newRenderer()
	if err != nil {
		slog.Error("can't load renderer", "err", err)
		os.Exit(1)
	}

	srv := newServer(newS3Store(s3c, *bucket, *presignExpiry), rend, *baseURL)

	hs := &http.Server{
		Addr:              *bind,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	slog.Info("listening", "bind", *bind, "bucket", *bucket, "base-url", *baseURL)
	if err := hs.ListenAndServe(); err != nil {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 2: Build and vet**

Run: `go build -o /tmp/claude-1000/-home-xe-Code-Xe-x/6da801a3-7bd2-438e-9d6a-747f9259cf5c/scratchpad/friendship-ended ./cmd/friendship-ended/ && go vet ./cmd/friendship-ended/`
Expected: no output.

- [ ] **Step 3: Live run against Tigris**

This needs a real bucket and credentials. If `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` are not set, stop and ask the user for a bucket name and for them to export credentials. Do not create a bucket without asking.

```bash
/tmp/claude-1000/-home-xe-Code-Xe-x/6da801a3-7bd2-438e-9d6a-747f9259cf5c/scratchpad/friendship-ended --bucket <bucket> --bind :3000
```

In another shell, from `/home/xe/Code/Xe/x`:

```bash
P=cmd/friendship-ended/assets
curl -si -F old-friend-name=mudasir -F new-friend-name=salman \
  -F new-friend-pic=@$P/x1.png -F old-friend-1=@$P/x2.png -F old-friend-2=@$P/treasure.gif \
  http://localhost:3000/ | grep -i '^location'
```

Expected: `Location: /f/<uuid>`. Then:

- `curl -s http://localhost:3000/f/<uuid> | grep og:image` shows `http://localhost:3000/f/<uuid>/image.jpg`.
- `curl -sI http://localhost:3000/f/<uuid>/image.jpg` gives `302` with a `Location` on the Tigris host containing `X-Amz-Signature`.
- `curl -s -o $SCRATCH/live.jpg "<that Location>"` downloads a JPEG. Open it with the Read tool and confirm it looks like the snapshot from Task 1.

Stop the server afterwards.

- [ ] **Step 4: Run the full package tests one more time**

Run: `go test ./cmd/friendship-ended/`
Expected: `ok`.

- [ ] **Step 5: Commit**

```bash
git add cmd/friendship-ended/main.go
git commit --signoff -m "feat(friendship-ended): add server entrypoint

Wire the renderer, Tigris store, and handlers together behind flags
for bind address, bucket, public base URL, and presign expiry.

Assisted-by: Claude Opus 5 via Claude Code"
```
