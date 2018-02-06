package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/caarlos0/env"
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

type dWebhook struct {
	Content  string `json:"content"`
	Username string `json:"username"`
}

var cfg = struct {
	WebhookURL string `env:"WEBHOOK_URL,required"`
}{}

func main() {
	err := env.Parse(&cfg)
	if err != nil {
		log.Fatal(err)
	}

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

	dw := dWebhook{
		Content:  fmt.Sprintf("Bitcoin value: $%v", bsr.Ask),
		Username: "Buttcoin",
	}

	data, err := json.Marshal(&dw)
	if err != nil {
		log.Fatal(err)
	}

	resp, err = http.Post(cfg.WebhookURL, "application/json", bytes.NewBuffer(data))
	if err != nil {
		log.Fatal(err)
	}

	if resp.StatusCode/100 != 2 {
		log.Fatal(resp.Status)
	}
}
