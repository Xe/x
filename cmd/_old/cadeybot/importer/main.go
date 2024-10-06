package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
)

func main() {
	flag.Parse()

	if flag.NArg() != 1 {
		log.Fatal("usage: importer <messages.csv>")
	}

	fname := flag.Arg(0)
	if fname == "" {
		log.Fatal("usage: importer <messages.csv>")
	}

	fin, err := os.Open(fname)
	if err != nil {
		log.Fatal(err)
	}

	csvReader := csv.NewReader(fin)
	_, err = csvReader.Read() // ignore the first row, it's the index
	if err != nil {
		log.Fatal(err)
	}

	all, err := csvReader.ReadAll()
	if err != nil {
		log.Fatal(err)
	}

	for _, row := range all {
		fmt.Println(row[2])
	}
}
