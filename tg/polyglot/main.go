package main

import (
	"bytes"
	"log"
	"os"
	"strings"
	"time"

	"github.com/Xe/johaus/parser"
	_ "github.com/Xe/johaus/parser/alldialects"
	"github.com/Xe/johaus/pretty"
	_ "github.com/joho/godotenv/autoload"
	tb "gopkg.in/tucnak/telebot.v2"
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

	b.Start()
}

func parserCommandFor(b *tb.Bot, dialect string) func(*tb.Message) {
	return func(m *tb.Message) {
		msg := strings.Join(strings.Split(m.Payload, " ")[1:], " ")
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
