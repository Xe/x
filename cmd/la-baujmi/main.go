// Command la-baujmi is a simple language understander for Toki Pona.
package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/mndrix/golog"
	line "github.com/peterh/liner"
	"within.website/x/internal"
	_ "within.website/x/internal/tokipona"
	"within.website/x/web/tokiponatokens"
)

var (
	historyFname = flag.String("history-file", filepath.Join(os.Getenv("HOME"), ".la-baujmi-history"), "location of history file")
)

func main() {
	internal.HandleStartup()
	var m golog.Machine
	m = golog.NewMachine()
	l := line.NewLiner()

	fin, err := os.Open(*historyFname)
	if err == nil {
		l.ReadHistory(fin)
		fin.Close()
	}

	for {
		if inp, err := l.Prompt("|toki: "); err == nil {
			if inp == "" {
				return
			}

			l.AppendHistory(inp)

			parts, err := tokiponatokens.Tokenize(tokiPonaAPIURL, inp)
			if err != nil {
				log.Printf("error: %v", err)
				continue
			}

			for _, sentence := range parts {
				sbs, err := SentenceToBridis(sentence)
				if err != nil {
					log.Printf("can't derive facts: %v", err)
					continue
				}

				for _, sb := range sbs {
					f := sb.Fact()

					if strings.Contains(inp, "?") {
						log.Printf("Query: %s", f)

						solutions := m.ProveAll(f)
						for _, solution := range solutions {
							log.Println("found", strings.Replace(f, "A", solution.ByName_("A").String(), -1))
						}

						continue
					}

					log.Printf("registering fact: %s", f)
					m = m.Consult(f)
				}
			}
		} else if err == line.ErrPromptAborted {
			log.Print("Aborted")
			break
		} else {
			log.Print("Error reading line: ", err)
			break
		}
	}

	fout, err := os.Create(*historyFname)
	if err != nil {
		panic(err)
	}
	l.WriteHistory(fout)
	fout.Close()
}
