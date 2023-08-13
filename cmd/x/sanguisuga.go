package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/google/subcommands"
	"github.com/rodaine/table"
)

var (
	sanguisugaURL = flag.String("url", "http://sanguisuga", "Base sanguisuga URL")
)

type Show struct {
	Title    string `json:"title"`
	DiskPath string `json:"diskPath"`
	Quality  string `json:"quality"`
}

type sanguisugaAnimeList struct {
	URL string
}

func (*sanguisugaAnimeList) Name() string     { return "anime-list" }
func (*sanguisugaAnimeList) Synopsis() string { return "Print list of anime tracked by sanguisuga." }
func (*sanguisugaAnimeList) Usage() string {
	return `anime-list [--url]:
  Print list of anime tracked by sanguisuga.`
}

func (sal *sanguisugaAnimeList) SetFlags(f *flag.FlagSet) {}

func (sal *sanguisugaAnimeList) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	resp, err := http.Get(fmt.Sprintf("%s/api/anime/list", *sanguisugaURL))
	if err != nil {
		log.Fatal(err)
	}

	var shows []Show

	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&shows); err != nil {
		log.Fatal(err)
	}

	tbl := table.New("Name", "Disk Path", "Quality")

	for _, show := range shows {
		tbl.AddRow(show.Title, show.DiskPath, show.Quality)
	}

	tbl.Print()

	return subcommands.ExitSuccess
}
