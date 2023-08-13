package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/google/subcommands"
	"github.com/rodaine/table"
	"within.website/x/web"
)

var (
	sanguisugaURL = flag.String("url", "http://sanguisuga", "Base sanguisuga URL")
)

type Show struct {
	Title    string `json:"title"`
	DiskPath string `json:"diskPath"`
	Quality  string `json:"quality"`
}

type sanguisugaAnimeList struct{}

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

type sanguisugaAnimeTrack struct{}

func (*sanguisugaAnimeTrack) Name() string { return "anime-track" }
func (*sanguisugaAnimeTrack) Synopsis() string {
	return "Add a new anime to the list of anime to track."
}
func (*sanguisugaAnimeTrack) Usage() string {
	return `anime-track <title> <dataDir>:
  Add a new anime to the tracklist for XDCC.`
}

func (sal *sanguisugaAnimeTrack) SetFlags(f *flag.FlagSet) {}

func (sal *sanguisugaAnimeTrack) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if f.NArg() != 2 {
		fmt.Println(sal.Usage())
		return subcommands.ExitFailure
	}

	show := Show{
		Title:    f.Arg(0),
		DiskPath: f.Arg(1),
		Quality:  "1080p",
	}

	data, err := json.Marshal(show)
	if err != nil {
		log.Fatal(err)
	}

	resp, err := http.Post(fmt.Sprintf("%s/api/anime/track", *sanguisugaURL), "application/json", bytes.NewBuffer(data))
	if err != nil {
		log.Fatal(err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Fatal(web.NewError(http.StatusOK, resp))
	}

	return subcommands.ExitSuccess
}
