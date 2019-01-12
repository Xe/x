package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Xe/x/internal"
	_ "github.com/Xe/x/tokipona"
	"github.com/Xe/x/web/tokiponatokens"
	_ "github.com/joho/godotenv/autoload"
	tb "gopkg.in/tucnak/telebot.v2"
	"within.website/johaus/parser"
	_ "within.website/johaus/parser/alldialects"
	"within.website/johaus/pretty"
)

const tpapiurl = `https://us-central1-golden-cove-408.cloudfunctions.net/toki-pona-verb-marker`

var (
	telegramToken  = flag.String("telegram-token", "", "telegram bot token")
	tokiPonaAPIURL = flag.String("toki-pona-tokenizer-api-url", tpapiurl, "toki pona tokenizer API URL")
)

func main() {
	internal.HandleStartup()
	b, err := tb.NewBot(tb.Settings{
		Token:  *telegramToken,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})

	if err != nil {
		log.Fatal(err)
		return
	}

	b.Handle("/camxes", parserCommandFor(b, "camxes"))
	b.Handle("/ilmentufa", parserCommandFor(b, "ilmentufa"))
	b.Handle("/maftufa", parserCommandFor(b, "maftufa"))
	b.Handle("/zantufa", parserCommandFor(b, "zantufa"))

	b.Handle("/toki", func(m *tb.Message) {
		defer func() {
			if r := recover(); r != nil {
				fmt.Println("Recovered in f", r)
			}
		}()

		msg := m.Payload
		parts, err := tokiponatokens.Tokenize(*tokiPonaAPIURL, msg)
		if err != nil {
			b.Send(m.Sender, err.Error())
			return
		}

		var sb strings.Builder

		for _, sentence := range parts {
			bracesReply := tokiBraces(sentence)
			log.Printf("%s: %s", m.Sender.Username, bracesReply)
			sb.WriteString(bracesReply)
			sb.WriteRune('\n')
		}

		b.Send(m.Sender, sb.String())
	})

	b.Start()
}

func parserCommandFor(b *tb.Bot, dialect string) func(*tb.Message) {
	return func(m *tb.Message) {
		msg := m.Payload
		tree, err := parser.Parse(dialect, msg)
		if err != nil {
			b.Send(m.Sender, err.Error())
			return
		}

		parser.RemoveMorphology(tree)
		parser.AddElidedTerminators(tree)
		parser.RemoveSpace(tree)
		parser.CollapseLists(tree)

		buf := bytes.NewBuffer(nil)
		pretty.Braces(buf, tree)

		b.Send(m.Sender, buf.String())
	}
}
