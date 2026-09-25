package ollaya

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
	"time"

	"within.website/x/web"
)

func sampleRequest() *DecideRequest {
	return &DecideRequest{
		Model: "laya",
		State: "I was charged twice for my subscription this month. Please refund the second charge.",
		Questions: map[string]Question{
			"department": Choice("Which team should handle this ticket?", map[string]string{
				"billing":   "Payments, invoices and refunds",
				"technical": "Bugs, errors and outages",
				"account":   "Login, profile and settings",
			}),
			"urgency": Score("How urgent is this ticket?",
				"Can wait", "Needs attention this week", "Needs attention today"),
			"refund": Noul("The customer asks for money back."),
		},
	}
}

func readJSON(t *testing.T, fname string) any {
	t.Helper()

	data, err := os.ReadFile(fname)
	if err != nil {
		t.Fatal(err)
	}

	var result any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}

	return result
}

func TestDecide(t *testing.T) {
	t.Parallel()

	wantReq := readJSON(t, "testdata/request.json")
	respBody, err := os.ReadFile("testdata/response.json")
	if err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/decide" {
			http.Error(w, "wrong route", http.StatusNotFound)
			return
		}

		var gotReq any
		if err := json.NewDecoder(r.Body).Decode(&gotReq); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if !reflect.DeepEqual(gotReq, wantReq) {
			t.Logf("want: %v", wantReq)
			t.Logf("got:  %v", gotReq)
			t.Error("request body does not match testdata/request.json")
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(respBody)
	}))
	defer srv.Close()

	resp, err := NewClient(srv.URL).Decide(t.Context(), sampleRequest())
	if err != nil {
		t.Fatal(err)
	}

	want := &DecideResponse{
		Model: "laya:en",
		Answers: map[string]Answer{
			"department": {
				Type:          TypeChoice,
				Choice:        "billing",
				Confidence:    0.7781,
				Probabilities: map[string]float64{"billing": 0.8521, "technical": 0.0611, "account": 0.0868},
			},
			"urgency": {
				Type:          TypeScore,
				Score:         1.1982,
				Confidence:    0.3418,
				Legend:        map[string]string{"0": "Can wait", "1": "Needs attention this week", "2": "Needs attention today"},
				Probabilities: map[string]float64{"0": 0.1203, "1": 0.5612, "2": 0.3185},
			},
			"refund": {Type: TypeNoul, Noul: 0.9127},
		},
		Usage:          Usage{InputTokens: 118, OutputTokens: 0},
		Routing:        &Routing{Router: "laya:latest", Model: "laya:en", Route: "english", Reason: "English Latin text"},
		StateTruncated: false,
		DoneReason:     "decide",
		CreatedAt:      time.Date(2026, 9, 24, 9, 30, 12, 418_000_000, time.UTC),
		TotalDuration:  18734512 * time.Nanosecond,
		LoadDuration:   0,
		EvalDuration:   16302117 * time.Nanosecond,
	}

	if !reflect.DeepEqual(resp, want) {
		t.Logf("want: %+v", want)
		t.Logf("got:  %+v", resp)
		t.Error("response does not match")
	}
}

func TestDecideHTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"model not found"}`, http.StatusNotFound)
	}))
	defer srv.Close()

	_, err := NewClient(srv.URL).Decide(t.Context(), sampleRequest())

	var webErr *web.Error
	if !errors.As(err, &webErr) {
		t.Fatalf("want *web.Error, got: %v", err)
	}

	if webErr.GotStatus != http.StatusNotFound {
		t.Errorf("want status %d, got: %d", http.StatusNotFound, webErr.GotStatus)
	}
}

func TestDecideRequestValid(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name string
		edit func(*DecideRequest)
		err  error
	}{
		{name: "valid", edit: func(*DecideRequest) {}},
		{name: "no model", edit: func(dr *DecideRequest) { dr.Model = "" }, err: ErrNoModel},
		{name: "no state", edit: func(dr *DecideRequest) { dr.State = "" }, err: ErrNoState},
		{name: "no questions", edit: func(dr *DecideRequest) { dr.Questions = nil }, err: ErrNoQuestions},
		{
			name: "no instructions",
			edit: func(dr *DecideRequest) { dr.Questions["refund"] = Noul("") },
			err:  ErrNoInstructions,
		},
		{
			name: "one choice",
			edit: func(dr *DecideRequest) {
				dr.Questions["department"] = Choice("Which team?", map[string]string{"billing": "Money"})
			},
			err: ErrNotEnoughCriteria,
		},
		{
			name: "one score level",
			edit: func(dr *DecideRequest) { dr.Questions["urgency"] = Score("How urgent?", "Can wait") },
			err:  ErrNotEnoughCriteria,
		},
		{
			name: "unknown type",
			edit: func(dr *DecideRequest) { dr.Questions["weird"] = Question{Type: "vibes", Instructions: "Vibe check."} },
			err:  ErrUnknownType,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dr := sampleRequest()
			tt.edit(dr)

			err := dr.Valid()
			if !errors.Is(err, tt.err) || (tt.err == nil && err != nil) {
				t.Logf("want: %v", tt.err)
				t.Logf("got:  %v", err)
				t.Error("got wrong error")
			}
		})
	}
}
