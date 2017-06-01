// See LICENSE file for copyright and license details.

// Package farbfeld implements a farbfeld decoder and encoder.
//
// The farbfeld specification can be found at http://git.2f30.org/farbfeld/
package imagefile

import (
	"encoding/binary"
	"image"
	"image/color"
	"io"
)

const (
	farbfeldHeader string = "farbfeld????????"
)

func init() {
	image.RegisterFormat("farbfeld", farbfeldHeader, Decode, DecodeConfig)
}

// Decode reads a farbfeld from r and returns it as image.NRGBA64.
func Decode(r io.Reader) (image.Image, error) {
	cfg, err := DecodeConfig(r)
	if err != nil {
		return nil, err
	}
	img := image.NewNRGBA64(image.Rect(0, 0, cfg.Width, cfg.Height))
	// image.NRGBA64 is big endian, so is farbfeld → just copy bytes
	_, err = io.ReadFull(r, img.Pix)
	return img, err
}

// DecodeConfig returns dimensions of farbfeld image.
func DecodeConfig(r io.Reader) (image.Config, error) {
	var cfg image.Config
	buff := make([]uint8, 16)
	_, err := io.ReadFull(r, buff)
	cfg.ColorModel = color.NRGBA64Model
	be := binary.BigEndian
	cfg.Width = int(be.Uint32(buff[8:12]))
	cfg.Height = int(be.Uint32(buff[12:16]))
	return cfg, err
}

// Encode writes m to w in farbfeld format. If m is not image.NRGBA64,
// it will be converted lossily.
func Encode(w io.Writer, m image.Image) error {
	header := []uint8(farbfeldHeader)
	be := binary.BigEndian
	width := m.Bounds().Dx()
	height := m.Bounds().Dy()
	be.PutUint32(header[8:12], uint32(width))
	be.PutUint32(header[12:16], uint32(height))
	_, err := w.Write(header)
	if err != nil {
		return err
	}
	switch img := m.(type) {
	case *image.NRGBA64:
		pix := img.Pix
		for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
			_, err = w.Write(pix[:width*8])
			if err != nil {
				return err
			}
			pix = pix[img.Stride:]
		}
	default:
		pix := make([]uint8, width*8)
		for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
			encodeLine(pix, img, y)
			_, err = w.Write(pix)
			if err != nil {
				return err
			}
		}
	}
	return err
}

func encodeLine(pix []uint8, m image.Image, y int) {
	be := binary.BigEndian
	for x := m.Bounds().Min.X; x < m.Bounds().Max.X; x++ {
		c := color.NRGBA64Model.Convert(m.At(x, y)).(color.NRGBA64)
		be.PutUint16(pix, c.R)
		pix = pix[2:]
		be.PutUint16(pix, c.G)
		pix = pix[2:]
		be.PutUint16(pix, c.B)
		pix = pix[2:]
		be.PutUint16(pix, c.A)
		pix = pix[2:]
	}
}
