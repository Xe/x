package main

import (
	"log"
	"os"
	"strings"

	"github.com/Xe/x/web/tokiponatokens"
	"github.com/mndrix/golog"
	line "github.com/peterh/liner"
)

func main() {
	/*
			m := golog.NewMachine().Consult(`
		toki(jan_Kesi).
		toki(jan_Pola).
		toki(jan_Kesi, jan_Pola).
		toki(jan_Kesi, toki_pona).
		command(ilo_Kesi, toki(ziho, jan_Kesi)).
		`)
			if m.CanProve(`toki(jan_Kesi).`) {
				log.Printf("toki(jan_Kesi). -> jan Kesi li toki.")
			}

			solutions := m.ProveAll(`toki(jan_Kesi, X).`)
			for _, solution := range solutions {
				log.Printf("jan_Kesi li toki e %s", solution.ByName_("X").String())
			}

			solutions = m.ProveAll(`command(X, toki(ziho, jan_Kesi)).`)
			for _, solution := range solutions {
				log.Printf("%s o, toki e jan_Kesi", solution.ByName_("X").String())
			}
	*/

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
