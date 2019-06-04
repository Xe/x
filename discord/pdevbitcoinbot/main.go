// Command pdevbitcoinbot queries the bitstamp API and stores prices in a HDR
// Histogram. This computes the p95 price of bitcoin.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"within.website/x/internal"
	"within.website/x/web"
	"within.website/x/web/discordwebhook"
	"github.com/codahale/hdrhistogram"
	experrors "golang.org/x/exp/errors"
	"within.website/ln"
	"within.website/ln/opname"
)

type BitstampReply struct {
	High      string  `json:"high"`
	Last      string  `json:"last"`
	Timestamp string  `json:"timestamp"`
	Bid       string  `json:"bid"`
	Vwap      string  `json:"vwap"`
	Volume    string  `json:"volume"`
	Low       string  `json:"low"`
	Ask       string  `json:"ask"`
	Open      float64 `json:"open"`
}

type data struct {
	Snapshot  *hdrhistogram.Snapshot
	LastValue *string
}

var (
	whURL            = flag.String("webhook-url", "", "Discord webhook URL")
	dataFileLocation = flag.String("data", "./data.json", "data file location")
)

func getBitstamp() (*BitstampReply, error) {
	resp, err := http.Get("https://www.bitstamp.net/api/ticker/")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	var bsr BitstampReply
	err = json.NewDecoder(resp.Body).Decode(&bsr)
	if err != nil {
		log.Fatal(err)
	}

	return &bsr, nil
}

func main() {
	internal.HandleStartup()

	ctx := opname.With(context.Background(), "main")

	fin, err := os.Open(*dataFileLocation)
	if err != nil {
		var pe *os.PathError
		if experrors.As(err, &pe) {
			ln.Fatal(ctx, ln.F{
				"err_op":   pe.Op,
				"err_path": pe.Path,
				"err_err":  pe.Err,
			})
		}

		ln.FatalErr(ctx, err)
	}

	var d data
	err = json.NewDecoder(fin).Decode(&d)
	if err != nil {
		ln.FatalErr(ctx, err)
	}
	fin.Close()

	var h *hdrhistogram.Histogram
	if d.Snapshot == nil {
		h = hdrhistogram.New(-400, 3000000000, 5)
	} else {
		h = hdrhistogram.Import(d.Snapshot)
	}

	bsr, err := getBitstamp()
	if err != nil {
		ln.FatalErr(ctx, err)
	}

	bcf, err := strconv.ParseFloat(bsr.Ask, 64)
	if err != nil {
		ln.FatalErr(ctx, err)
	}

	h.RecordValue(int64(bcf))

	var lv = "_shrug_"

	if d.LastValue != nil {
		lv = *d.LastValue
	}

	dw := discordwebhook.Webhook{
		Content:  "BITCOIN PRICE TIEM",
		Username: "Buttcoin",
		Embeds: []discordwebhook.Embeds{discordwebhook.Embeds{
			Footer: discordwebhook.EmbedFooter{
				Text: "powered by pdevbitcoinbot, made by Cadey~#1337",
			},
			Fields: []discordwebhook.EmbedField{
				{
					Name:   "Now",
					Value:  bsr.Ask,
					Inline: true,
				},
				{
					Name:   "Last",
					Value:  lv,
					Inline: true,
				},
				{
					Name:   "P95",
					Value:  fmt.Sprint(h.ValueAtQuantile(95)),
					Inline: true,
				},
			},
		}},
	}

	req := discordwebhook.Send(*whURL, dw)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		ln.FatalErr(ctx, err)
	}
	err = discordwebhook.Validate(resp)
	if err != nil {
		var werr *web.Error
		if experrors.As(err, &werr) {
			ln.Fatal(ctx, werr)
		}

		ln.FatalErr(ctx, err)
	}

	d.LastValue = &bsr.Ask
	d.Snapshot = h.Export()

	fout, err := os.Create("./data.json")
	if err != nil {
		ln.FatalErr(ctx, err)
	}

	err = json.NewEncoder(fout).Encode(&d)
	if err != nil {
		ln.FatalErr(ctx, err)
	}
}
