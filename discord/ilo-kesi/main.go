package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Xe/x/web/switchcounter"
	"github.com/joeshaw/envdecode"
	_ "github.com/joho/godotenv/autoload"
	"github.com/peterh/liner"
)

// lipuSona is the configuration.
type lipuSona struct {
	//DiscordToken            string   `env:"DISCORD_TOKEN,required"` // lipu pi lukin ala
	TokiPonaTokenizerAPIURL string   `env:"TOKI_PONA_TOKENIZER_API_URL,default=https://us-central1-golden-cove-408.cloudfunctions.net/function-1"`
	SwitchCounterWebhook    string   `env:"SWITCH_COUNTER_WEBHOOK,required"`
	IloNimi                 []string `env:"IJO_NIMI,default=ke;si"`
}

func main() {
	var cfg lipuSona
	err := envdecode.StrictDecode(&cfg)
	if err != nil {
		log.Fatal(err)
	}

	//pretty.Println(cfg)

	sw := switchcounter.NewHTTPClient(http.DefaultClient, cfg.SwitchCounterWebhook)

	line := liner.NewLiner()
	defer line.Close()

	line.SetCtrlCAborts(true)

	for {
		if inp, err := line.Prompt("|: "); err == nil {
			if inp == "" {
				return
			}

			line.AppendHistory(inp)

			parts, err := TokenizeTokiPona(cfg.TokiPonaTokenizerAPIURL, inp)
			if err != nil {
				log.Printf("Can't parse: %v", err)
			}

			for _, sent := range parts {
				req, err := parseRequest(sent)
				if err != nil {
					log.Printf("error: %v", err)
					continue
				}

				switch req.Action {
				case actionFront:
					if req.Subject == nil {
						st, err := sw.Status(context.Background())
						if err != nil {
							log.Printf("status error: %v", err)
							continue
						}

						fmt.Printf("ilo Kesi\\ jan %s li lawa insa.\n", withinToToki[st.Front])

						log.Printf("Started at: %s (%s ago)", st.StartedAt, time.Since(st.StartedAt))
						continue
					}

					log.Printf("setting front not implemented yet :(")
				}
			}
		} else if err == liner.ErrPromptAborted {
			log.Print("Aborted")
		} else {
			log.Print("Error reading line: ", err)
		}
	}
}
