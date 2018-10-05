package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/Xe/x/web/discordwebhook"
	"github.com/caarlos0/env"
	"github.com/codahale/hdrhistogram"
	_ "github.com/joho/godotenv/autoload"
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

var cfg = struct {
	WebhookURL string `env:"WEBHOOK_URL,required"`
}{}

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

func sendWebhook(whurl string, dw discordwebhook.Webhook) error {
	data, err := json.Marshal(&dw)
	if err != nil {
		log.Fatal(err)
	}

	resp, err := http.Post(whurl, "application/json", bytes.NewBuffer(data))
	if err != nil {
		log.Fatal(err)
	}

	if resp.StatusCode/100 != 2 {
		io.Copy(os.Stderr, resp.Body)
		resp.Body.Close()
		return fmt.Errorf("status code was %v", resp.StatusCode)
	}

	return nil
}

func main() {
	err := env.Parse(&cfg)
	if err != nil {
		log.Fatal(err)
	}

	fin, err := os.Open("./data.json")
	if err != nil {
		log.Fatal(err)
	}

	var d data
	err = json.NewDecoder(fin).Decode(&d)
	if err != nil {
		log.Fatal(err)
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
		log.Fatal(err)
	}

	bcf, err := strconv.ParseFloat(bsr.Ask, 64)
	if err != nil {
		log.Fatal(err)
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

	req := discordwebhook.Send(cfg.WebhookURL, dw)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	err = discordwebhook.Validate(resp)
	if err != nil {
		log.Fatal(err)
	}

	d.LastValue = &bsr.Ask
	d.Snapshot = h.Export()

	fout, err := os.Create("./data.json")
	if err != nil {
		log.Fatal(err)
	}

	err = json.NewEncoder(fout).Encode(&d)
	if err != nil {
		log.Fatal(err)
	}
}
