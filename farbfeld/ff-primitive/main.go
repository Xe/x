package main

import (
	"flag"
	"image"
	"log"
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/fogleman/primitive/primitive"
	farbfeld "github.com/hullerob/go.farbfeld"
)

var (
	shapeCount       = flag.Int("count", 150, "number of shapes used")
	repeatShapeCount = flag.Int("repeat-count", 0, "number of extra shapes drawn in each step")
	alpha            = flag.Int("alpha", 128, "alpha of all shapes")
	cpuprofile       = flag.String("cpuprofile", "", "write cpu profile to file")
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

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	img, err := farbfeld.Decode(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}

	err = farbfeld.Encode(os.Stdout, stepImg(img, *shapeCount))
	if err != nil {
		log.Fatal(err)
	}
}
