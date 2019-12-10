package main

import (
	"io"
	"os"
	"strings"

	"within.website/x/internal"
	"within.website/x/writer"
)

func main() {
	internal.HandleStartup()

	prefix := strings.Join(os.Args[1:], " ") + " | "
	wr := writer.LineSplitting(writer.PrefixWriter(prefix, os.Stdout))
	_, err := io.Copy(wr, os.Stdin)
	if err != nil {
		panic(err)
	}
}
