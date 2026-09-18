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
			name:   "empty names",
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
