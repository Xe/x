package main

import (
	"bytes"
	"context"
	"expvar"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"git.xeserv.us/xena/jvozba"
	_ "github.com/joho/godotenv/autoload"
	tb "gopkg.in/tucnak/telebot.v2"
	"within.website/johaus/parser"
	_ "within.website/johaus/parser/alldialects"
	"within.website/johaus/pretty"
	"within.website/ln"
	"within.website/ln/opname"
	"within.website/x/internal"
	_ "within.website/x/tokipona"
	"within.website/x/web/tokiponatokens"
)

const (
	tpapiurl = `https://us-central1-golden-cove-408.cloudfunctions.net/toki-pona-verb-marker`
	selfURL  = `http://10.0.0.240:5009/`
)

var (
	telegramToken  = flag.String("telegram-token", "", "telegram bot token")
	tokiPonaAPIURL = flag.String("toki-pona-tokenizer-api-url", tpapiurl, "toki pona tokenizer API URL")
	port           = flag.String("port", "5009", "HTTP port for statistics")
)

func main() {
	internal.HandleStartup()
	b, err := tb.NewBot(tb.Settings{
		Token:  *telegramToken,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})
	errCount := expvar.NewInt("errors")

	if err != nil {
		log.Fatal(err)
		return
	}

	b.Handle("/camxes", parserCommandFor(b, "camxes"))
	b.Handle("/ilmentufa", parserCommandFor(b, "ilmentufa"))
	b.Handle("/maftufa", parserCommandFor(b, "maftufa"))
	b.Handle("/zantufa", parserCommandFor(b, "zantufa"))

	tokiCount := expvar.NewInt("toki")
	b.Handle("/toki", func(m *tb.Message) {
		defer func() {
			if r := recover(); r != nil {
				fmt.Println("Recovered in f", r)
				errCount.Add(1)
			}
		}()
		defer tokiCount.Add(1)

		msg := m.Payload
		parts, err := tokiponatokens.Tokenize(*tokiPonaAPIURL, msg)
		if err != nil {
			b.Send(m.Sender, err.Error())
			errCount.Add(1)
			return
		}

		var sb strings.Builder

		for _, sentence := range parts {
			bracesReply := tokiBraces(sentence)
			sb.WriteString(bracesReply)
			sb.WriteRune('\n')
		}

		b.Send(m.Sender, sb.String())
	})

	b.Handle("/lujvo", func(m *tb.Message) {
		msg := m.Payload
		jvo, err := jvozba.Jvozba(msg)
		if err != nil {
			b.Send(m.Sender, err.Error())
			errCount.Add(1)
			return
		}

		b.Send(m.Sender, jvo)
	})

	ln.Log(opname.With(context.Background(), "main"), ln.Info("starting HTTP server"), ln.F{"port": *port, "using": "idpmiddleware", "self_url": selfURL})
	go http.ListenAndServe(":"+*port, http.DefaultServeMux)

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
