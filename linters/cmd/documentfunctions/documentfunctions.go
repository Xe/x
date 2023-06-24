package main

import (
	"golang.org/x/tools/go/analysis/singlechecker"
	"within.website/x/linters/documentfunctions"
)

func main() {
	singlechecker.Main(documentfunctions.Analyzer)
}
