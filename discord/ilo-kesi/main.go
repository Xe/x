package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/Xe/x/web/switchcounter"
	"github.com/Xe/x/web/tokiponatokens"
	"github.com/joeshaw/envdecode"
	_ "github.com/joho/godotenv/autoload"
	"github.com/peterh/liner"
)

// lipuSona is the configuration.
type lipuSona struct {
	//DiscordToken            string   `env:"DISCORD_TOKEN,required"` // lipu pi lukin ala
	TokiPonaTokenizerAPIURL string `env:"TOKI_PONA_TOKENIZER_API_URL,default=https://us-central1-golden-cove-408.cloudfunctions.net/function-1"`
	SwitchCounterWebhook    string `env:"SWITCH_COUNTER_WEBHOOK,required"`
	IloNimi                 string `env:"ILO_NIMI,default=Kesi"`
}

func init() {
	rand.Seed(time.Now().UnixNano())
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

	chain := NewChain(3)
	err = chain.Load("cadey.gob")
	if err != nil {
		log.Fatal(err)
	}

	line.SetCtrlCAborts(true)

	for {
		if inp, err := line.Prompt("|: "); err == nil {
			if inp == "" {
				return
			}

			line.AppendHistory(inp)

			parts, err := tokiponatokens.Tokenize(cfg.TokiPonaTokenizerAPIURL, inp)
			if err != nil {
				log.Printf("Can't parse: %v", err)
			}

			for _, sent := range parts {
				req, err := parseRequest(sent)
				if err != nil {
					log.Printf("error: %v", err)
					continue
				}

				if len(req.Address) != 2 {
					log.Println("ilo Kesi was not addressed")
					continue
				}

				if req.Address[0] != "ilo" {
					log.Println("Addressed non-ilo")
					continue
				}

				if req.Address[1] != cfg.IloNimi {
					log.Printf("ilo %s was addressed, not ilo %s", req.Address[1], cfg.IloNimi)
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

					fmt.Printf("ilo Kesi\\ tenpo ni la jan %s li lawa insa.\n", req.Subject)
				case actionWhat:
					switch req.Subject {
					case "tenpo ni":
						fmt.Printf("ilo Kesi\\ ni li tenpo %s\n", time.Now().Format(time.Kitchen))
						continue
					}
				}

				switch req.Subject {
				case "sitelen pakala":
					fmt.Printf("ilo Kesi\\ %s\n", chain.Generate(20))
					continue
				}
			}
		} else if err == liner.ErrPromptAborted {
			log.Print("Aborted")
		} else {
			log.Print("Error reading line: ", err)
		}
	}
}
