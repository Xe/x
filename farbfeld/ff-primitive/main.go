package main

import (
	"flag"
	"image"
	"log"
	"os"
	"runtime"

	"github.com/fogleman/primitive/primitive"
	farbfeld "github.com/hullerob/go.farbfeld"
)

var (
	shapeCount       = flag.Int("count", 150, "number of shapes used")
	repeatShapeCount = flag.Int("repeat-count", 0, "number of extra shapes drawn in each step")
	alpha            = flag.Int("alpha", 128, "alpha of all shapes")
)

func stepImg(img image.Image, count int) image.Image {
	bg := primitive.MakeColor(primitive.AverageImageColor(img))
	model := primitive.NewModel(img, bg, 512, runtime.NumCPU())

	for range make([]struct{}, count) {
		model.Step(primitive.ShapeTypeTriangle, *alpha, *repeatShapeCount)
	}

	return model.Context.Image()
}

func main() {
	flag.Parse()

	img, err := farbfeld.Decode(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}

	err = farbfeld.Encode(os.Stdout, stepImg(img, *shapeCount))
	if err != nil {
		log.Fatal(err)
	}
}
