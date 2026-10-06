package systemone

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"within.website/x/valid"
)

var (
	_ valid.Interface = (*Request)(nil)
	_ valid.Interface = Question{}
)

func solidImage(width, height int, fill color.Color) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), image.NewUniform(fill), image.Point{}, draw.Src)
	return img
}

func pngBase64(t *testing.T, img image.Image) string {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

func TestClientEvaluate(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/systemone" {
			t.Errorf("request = %s %s, want POST /v1/systemone", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer secret" {
			t.Errorf("Authorization = %q, want Bearer secret", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if got := body["model"]; got != "clef-flash" {
			t.Errorf("model = %v, want clef-flash", got)
		}
		state, ok := body["state"].(map[string]any)
		if !ok || state["message"] != "Checkout failed" {
			t.Errorf("state = %v, want structured message", body["state"])
		}
		images, ok := body["images"].([]any)
		if !ok || len(images) != 3 {
			t.Fatalf("images = %v, want three encoded images", body["images"])
		}
		for i, want := range []struct {
			width, height int
			red           bool
		}{{1280, 720, true}, {640, 480, false}, {360, 720, true}} {
			encoded, ok := images[i].(string)
			if !ok {
				t.Fatalf("image %d = %T, want base64 string", i, images[i])
			}
			data, err := base64.StdEncoding.DecodeString(encoded)
			if err != nil {
				t.Fatalf("decode image %d base64: %v", i, err)
			}
			img, format, err := image.Decode(bytes.NewReader(data))
			if err != nil {
				t.Fatalf("decode image %d: %v", i, err)
			}
			if format != "jpeg" || img.Bounds().Dx() != want.width || img.Bounds().Dy() != want.height {
				t.Errorf("image %d: format %q, bounds %v; want jpeg %dx%d", i, format, img.Bounds(), want.width, want.height)
			}
			red, _, blue, _ := img.At(img.Bounds().Min.X, img.Bounds().Min.Y).RGBA()
			if (red > blue) != want.red {
				t.Errorf("image %d color order incorrect: red %d, blue %d", i, red, blue)
			}
		}
		questions, ok := body["questions"].(map[string]any)
		if !ok || len(questions) != 3 {
			t.Errorf("questions = %v, want three", body["questions"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"model":"clef-flash","answers":{"urgent":{"type":"noul","noul":0},"team":{"type":"choice","choice":"technical","probabilities":{"technical":1},"confidence":0},"severity":{"type":"score","score":0,"legend":{"0":"No impact","1":"Critical"},"probabilities":{"0":1,"1":0},"confidence":0}},"usage":{"input_tokens":346,"output_tokens":0}}`)
	}))
	defer server.Close()

	client := NewClient(server.URL + "/")
	client.APIKey = "secret"
	input := &Request{
		Model: "clef-flash",
		State: map[string]any{"message": "Checkout failed"},
		Images: []image.Image{
			solidImage(1600, 900, color.RGBA{R: 255, A: 255}),
			solidImage(640, 480, color.RGBA{B: 255, A: 255}),
			solidImage(600, 1200, color.RGBA{R: 255, A: 255}),
		},
		Questions: map[string]Question{
			"urgent":   {Type: Noul, Instructions: "Is it urgent?", Criteria: map[string]any{"true": "Needs action now", "false": nil}},
			"team":     {Type: Choice, Instructions: map[string]any{"question": "Which team?"}, Criteria: map[string]any{"billing": nil, "technical": "Outages"}},
			"severity": {Type: Score, Instructions: "How severe?", Criteria: []any{"No impact", "Critical"}},
		},
	}
	got, err := client.Evaluate(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if got.Usage.InputTokens != 346 || got.Usage.OutputTokens != 0 {
		t.Errorf("usage = %+v, want 346 input and 0 output", got.Usage)
	}
	if got.Answers["urgent"].Noul == nil || *got.Answers["urgent"].Noul != 0 {
		t.Errorf("noul = %+v, want zero", got.Answers["urgent"].Noul)
	}
	if got.Answers["severity"].Score == nil || *got.Answers["severity"].Score != 0 {
		t.Errorf("score = %+v, want zero", got.Answers["severity"].Score)
	}
	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{`"noul":0`, `"score":0`, `"confidence":0`} {
		if !strings.Contains(string(encoded), field) {
			t.Errorf("response %s lacks %s", encoded, field)
		}
	}
}

func TestRequestValid(t *testing.T) {
	t.Parallel()
	base := func() Request {
		return Request{Model: "clef", State: "text", Questions: map[string]Question{"q": {Type: Noul, Instructions: "True?"}}}
	}
	tests := []struct {
		name    string
		change  func(*Request)
		wantErr string
	}{
		{name: "valid noul", change: func(*Request) {}},
		{name: "missing model", change: func(r *Request) { r.Model = "" }, wantErr: "model is required"},
		{name: "scalar state", change: func(r *Request) { r.State = 4 }, wantErr: "state must be"},
		{name: "no questions", change: func(r *Request) { r.Questions = nil }, wantErr: "at least one question"},
		{name: "unknown primitive", change: func(r *Request) { r.Questions["q"] = Question{Type: "other"} }, wantErr: "unknown type"},
		{name: "choice criteria array", change: func(r *Request) { r.Questions["q"] = Question{Type: Choice, Criteria: []string{"one", "two"}} }, wantErr: "choice criteria must be an object"},
		{name: "score too short", change: func(r *Request) { r.Questions["q"] = Question{Type: Score, Criteria: []string{"one"}} }, wantErr: "2 to 10 levels"},
		{name: "score too long", change: func(r *Request) { r.Questions["q"] = Question{Type: Score, Criteria: make([]string, 11)} }, wantErr: "2 to 10 levels"},
		{name: "invalid instructions", change: func(r *Request) { r.Questions["q"] = Question{Type: Noul, Instructions: 3} }, wantErr: "instructions"},
		{name: "nil image", change: func(r *Request) { r.Images = []image.Image{nil} }, wantErr: "image 0: is nil"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			input := base()
			tt.change(&input)
			err := input.Valid()
			if tt.wantErr == "" && err != nil {
				t.Errorf("Valid() error = %v, want nil", err)
			}
			if tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)) {
				t.Errorf("Valid() error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestHandler(t *testing.T) {
	t.Parallel()
	first := pngBase64(t, solidImage(8, 4, color.RGBA{R: 255, A: 255}))
	second := pngBase64(t, solidImage(4, 8, color.RGBA{B: 255, A: 255}))
	called := false
	handler := NewHandler(EvaluatorFunc(func(_ context.Context, input *Request) (*Response, error) {
		called = true
		if input.Model != "clef" || input.Questions["urgent"].Type != Noul {
			t.Errorf("request = %+v, want clef with urgent noul", input)
		}
		if len(input.Images) != 2 || input.Images[0].Bounds().Size() != image.Pt(8, 4) || input.Images[1].Bounds().Size() != image.Pt(4, 8) {
			t.Errorf("images = %v, want two decoded images in order", input.Images)
		}
		zero := 0.0
		return &Response{Model: "clef", Answers: map[string]Answer{"urgent": {Type: Noul, Noul: &zero}}}, nil
	}))
	body, err := json.Marshal(wireRequest{Model: "clef", State: "Checkout failed", Images: []string{first, second}, Questions: map[string]Question{"urgent": {Type: Noul, Instructions: "Urgent?"}}})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/systemone", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !called {
		t.Errorf("status = %d, evaluator called = %v, want 200 and true", w.Code, called)
	}
	var got Response
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Answers["urgent"].Noul == nil || *got.Answers["urgent"].Noul != 0 {
		t.Errorf("noul = %v, want zero", got.Answers["urgent"].Noul)
	}
}

func TestHandlerErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		method     string
		body       string
		evaluator  Evaluator
		wantStatus int
	}{
		{name: "wrong method", method: http.MethodGet, wantStatus: http.StatusMethodNotAllowed},
		{name: "invalid JSON", method: http.MethodPost, body: `{`, wantStatus: http.StatusUnprocessableEntity},
		{name: "multiple JSON objects", method: http.MethodPost, body: `{} {}`, wantStatus: http.StatusUnprocessableEntity},
		{name: "invalid question", method: http.MethodPost, body: `{"model":"clef","state":"text","questions":{"q":{"type":"score","criteria":["one"]}}}`, wantStatus: http.StatusUnprocessableEntity},
		{name: "invalid image", method: http.MethodPost, body: `{"model":"clef","state":"text","images":["not base64"],"questions":{"q":{"type":"noul"}}}`, wantStatus: http.StatusUnprocessableEntity},
		{name: "evaluator failure", method: http.MethodPost, body: `{"model":"clef","state":"text","questions":{"q":{"type":"noul"}}}`, evaluator: EvaluatorFunc(func(context.Context, *Request) (*Response, error) { return nil, errors.New("backend failed") }), wantStatus: http.StatusBadGateway},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			evaluator := tt.evaluator
			if evaluator == nil {
				evaluator = EvaluatorFunc(func(context.Context, *Request) (*Response, error) { return &Response{}, nil })
			}
			w := httptest.NewRecorder()
			NewHandler(evaluator).ServeHTTP(w, httptest.NewRequest(tt.method, "/v1/systemone", strings.NewReader(tt.body)))
			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d; body = %s", w.Code, tt.wantStatus, w.Body.String())
			}
			if got := w.Header().Get("Content-Type"); got != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", got)
			}
		})
	}
}
