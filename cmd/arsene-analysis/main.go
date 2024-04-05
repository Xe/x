package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strconv"
	"time"

	"within.website/x/internal"
	"within.website/x/web/ollama"
)

var (
	foutName    = flag.String("out", "enriched.csv", "output file name")
	ollamaHost  = flag.String("ollama-host", "http://xe-inference.flycast", "ollama host")
	ollamaModel = flag.String("ollama-model", "nous-hermes2-mixtral:8x7b-dpo-q5_K_M", "ollama model")
	subsetFile  = flag.String("subset", "", "subset CSV file to use")
)

type sentimentResponse struct {
	Sentiment string `json:"sentiment"`
}

func (sr sentimentResponse) Valid() error {
	if sr.Sentiment != "positive" && sr.Sentiment != "negative" && sr.Sentiment != "neutral" {
		return fmt.Errorf("invalid sentiment %q", sr.Sentiment)
	}

	return nil
}

func main() {
	internal.HandleStartup()

	fin, err := os.Open(*subsetFile)
	if err != nil {
		log.Fatal(err)
	}
	defer fin.Close()

	fout, err := os.Create(*foutName)
	if err != nil {
		log.Fatal(err)
	}
	defer fout.Close()

	w := csv.NewWriter(fout)
	w.Write([]string{"id", "price_change", "sentiment"})

	cli := ollama.NewClient(*ollamaHost)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Hour)
	defer cancel()

	r := csv.NewReader(fin)
	for {
		row, err := r.Read()
		if err != nil {
			break
		}

		//slog.Debug("got", "row", row)

		sr, err := ParseSubsetRow(row)
		if err != nil {
			slog.Error("failed to parse row", "err", err)
			continue
		}

		sen, err := ollama.Hallucinate[sentimentResponse](ctx, cli, ollama.HallucinateOpts{
			Model: *ollamaModel,
			Messages: []ollama.Message{
				{
					Role: "system",
					Content: `Rate the sentiment of the following text. If the sentiment is positive, return this JSON object:
{"sentiment":"positive"}
If the sentiment is negative, return this JSON object:
{"sentiment":"negative"}
If there is neither a positive nor a negative sentiment, return this JSON object:
{"sentiment":"neutral"}
DO NOT send any whitespace or newlines in the JSON object.`,
				},
				{
					Role:    "user",
					Content: sr.Body,
				},
			},
		})
		if err != nil {
			slog.Error("failed to chat", "err", err)
			continue
		}

		priceChange := ""

		if sr.PrevPrice > sr.AfterPrice {
			priceChange = "negative"
		} else if sr.PrevPrice < sr.AfterPrice {
			priceChange = "positive"
		} else {
			priceChange = "neutral"
		}

		w.Write([]string{
			strconv.Itoa(sr.ID),
			priceChange,
			sen.Sentiment,
		})
		w.Flush()
	}

	w.Flush()
}

type SubsetRow struct {
	ID         int     `json:"id"`
	Title      string  `json:"title"`
	Body       string  `json:"body"`
	PrevPrice  float64 `json:"prev_price"`
	AfterPrice float64 `json:"after_price"`
}

func ParseSubsetRow(data []string) (*SubsetRow, error) {
	if len(data) != 5 {
		return nil, fmt.Errorf("expected 5 fields, got %d", len(data))
	}

	id, err := strconv.Atoi(data[0])
	if err != nil {
		return nil, fmt.Errorf("id: %w", err)
	}

	prevPrice, err := strconv.ParseFloat(data[3], 64)
	if err != nil {
		return nil, fmt.Errorf("prev_price: %w", err)
	}

	afterPrice, err := strconv.ParseFloat(data[4], 64)
	if err != nil {
		return nil, fmt.Errorf("after_price: %w", err)
	}

	return &SubsetRow{
		ID:         id,
		Title:      data[1],
		Body:       data[2],
		PrevPrice:  prevPrice,
		AfterPrice: afterPrice,
	}, nil
}
