package main

import (
	"golang.org/x/tools/go/analysis/singlechecker"
	"within.website/x/linters/nosleep"
)

func main() {
	singlechecker.Main(nosleep.Analyzer)
}
