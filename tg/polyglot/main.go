package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/Xe/x/web/tokiponatokens"
	_ "github.com/joho/godotenv/autoload"
	tb "gopkg.in/tucnak/telebot.v2"
	"within.website/johaus/parser"
	_ "within.website/johaus/parser/alldialects"
	"within.website/johaus/pretty"
)

func main() {
	b, err := tb.NewBot(tb.Settings{
		Token:  os.Getenv("TELEGRAM_TOKEN"),
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
		parts, err := tokiponatokens.Tokenize(os.Getenv("TOKI_PONA_TOKENIZER_API_URL"), msg)
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
