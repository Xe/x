//go:build ignore
// +build ignore

package main

import (
	"encoding/csv"
	"io"
	"os"

	. "github.com/dave/jennifer/jen"
)

func main() {
	fin, err := os.Open("./TokiPonaRelex.csv")
	if err != nil {
		panic(err)
	}
	defer fin.Close()
	relexFile := NewFile("tokipona")
	_ = relexFile
	r := csv.NewReader(fin)

	fout, err := os.Create("relex_gen.go")
	if err != nil {
		panic(err)
	}
	defer fout.Close()

	relexFile.Comment("generated by generate.go; DO NOT EDIT")

	d := Dict{}

	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}

		d[Lit(record[1])] = Lit(record[2])
	}

	relexFile.Var().Id("relexMap").Op("=").Map(String()).String().Values(d)
	err = relexFile.Render(fout)
	if err != nil {
		panic(err)
	}
}
