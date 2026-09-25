package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
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
// pre-scale (user space) coordinate space, as passed to DrawString after
// Scale, matching ImageMagick's annotation after scale in the original PHP.
// An empty fill means the title gradient.
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

	dc := gg.NewContextForImage(flattenWhite(imaging.Resize(newPic, canvasWidth, canvasHeight, imaging.Lanczos)))

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

	// gg's Pop restores the matrix and color but deliberately leaves the
	// mask in place (see fogleman/gg Context.Pop), so the clip set above
	// would otherwise leak into every later drawLine call on this
	// context. Reset it to fully opaque (no clipping) before returning.
	dc.ResetClip()

	return nil
}

// crossOut resizes pic to w x h and draws overlay on top, clipped to the
// photo like ImageMagick's COMPOSITE_ATOP.
func crossOut(pic image.Image, w, h int, overlay image.Image) image.Image {
	c := gg.NewContextForImage(flattenWhite(imaging.Resize(pic, w, h, imaging.Lanczos)))
	c.DrawImage(overlay, 0, 0)
	return c.Image()
}

// flattenWhite draws img over an opaque white background. gg composites
// onto whatever background color the source image carries, and a fully
// transparent upload (RGBA zero value) is transparent black, which would
// otherwise render as a black rectangle in the final JPEG (which has no
// alpha channel of its own). Flattening onto white first keeps transparent
// areas looking like blank paper instead.
func flattenWhite(img image.Image) image.Image {
	b := img.Bounds()
	dst := image.NewRGBA(b)
	draw.Draw(dst, b, image.White, image.Point{}, draw.Src)
	draw.Draw(dst, b, img, b.Min, draw.Over)
	return dst
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
