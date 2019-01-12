package main

import (
	"log"
	"os"
	"strings"

	_ "github.com/Xe/x/internal/tokipona"
	"github.com/Xe/x/web/tokiponatokens"
	"github.com/mndrix/golog"
	line "github.com/peterh/liner"
)

func main() {
	var m golog.Machine
	m = golog.NewMachine()

	for {
		l := line.NewLiner()
		if inp, err := l.Prompt("|: "); err == nil {
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
				sbs, err := SentenceToSelbris(sentence)
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

	os.Exit(0)
}
