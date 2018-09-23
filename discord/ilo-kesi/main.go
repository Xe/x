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

			//pretty.Println(parts)

			for _, sent := range parts {
				req, err := parseRequest(sent)
				if err != nil {
					log.Printf("error: %v", err)
					continue
				}

				if req.Address == nil {
					log.Println("ilo Kesi was not addressed")
					continue
				}

				switch req.Action {
				case actionFront:
					if req.Subject == actionWhat {
						st, err := sw.Status(context.Background())
						if err != nil {
							log.Printf("status error: %v", err)
							continue
						}

						qual := TimeToQualifier(st.StartedAt)
						fmt.Printf("ilo Kesi\\ %s la jan %s li lawa insa.\n", qual, withinToToki[st.Front])

						continue
					}

					front := tokiToWithin[req.Subject]

					_, err := sw.Switch(context.Background(), front)
					if err != nil {
						log.Printf("switch error: %v", err)
						continue
					}

					fmt.Printf("ijo Kesi\\ tenpo ni la jan %s li lawa insa.\n", req.Subject)
				case actionWhat:
					switch req.Subject {
					case "tenpo ni":
						fmt.Printf("ilo Kesi\\ ni li tenpo %s\n", time.Now().Format(time.Kitchen))
					}
				}
			}
		} else if err == liner.ErrPromptAborted {
			log.Print("Aborted")
		} else {
			log.Print("Error reading line: ", err)
		}
	}
}
