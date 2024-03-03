package ollama

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func mkFakeStreamedOllamaChat(response string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		enc := json.NewEncoder(w)
		enc.Encode(CompleteResponse{
			Model: "fake/for:testing",
			Message: Message{
				Content: response,
			},
		})

		enc.Encode(CompleteResponse{
			Done: true,
		})
	}
}

func mkBetterFakeStreamedOllamaChat(responses []string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		enc := json.NewEncoder(w)
		for _, resp := range responses {
			enc.Encode(CompleteResponse{
				Model: "fake/for:testing",
				Message: Message{
					Content: resp,
				},
			})
		}

		enc.Encode(CompleteResponse{
			Done: true,
		})
	}
}

type fakeMessage struct {
	Content string `json:"content"`
}

func (fm fakeMessage) Valid() error {
	if fm.Content != "hello, world" {
		return fmt.Errorf("expected %q, got %q", "hello, world", fm.Content)
	}

	return nil
}

func TestHallucinateSimple(t *testing.T) {
	srv := httptest.NewServer(mkFakeStreamedOllamaChat(`{"content": "hello, world"}`))
	defer srv.Close()

	c := NewClient(srv.URL)

	resp, err := Hallucinate[fakeMessage](context.Background(), c, HallucinateOpts{
		Model: "fake/for:testing",
		Messages: []Message{
			{
				Content: "hello",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if resp.Content != "hello, world" {
		t.Fatalf("expected %q, got %q", "hello, world", resp.Content)
	}
}

func TestHallucinateMultiple(t *testing.T) {
	srv := httptest.NewServer(mkBetterFakeStreamedOllamaChat([]string{
		`{`,
		`"`,
		`content`,
		`"`,
		`:`,
		`"`,
		`hello`,
		`, `,
		`world`,
		`"`,
		`}`,
	}))
	defer srv.Close()

	c := NewClient(srv.URL)

	resp, err := Hallucinate[fakeMessage](context.Background(), c, HallucinateOpts{
		Model: "fake/for:testing",
		Messages: []Message{
			{
				Content: "hello",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if resp.Content != "hello, world" {
		t.Fatalf("expected %q, got %q", "hello, world", resp.Content)
	}
}
