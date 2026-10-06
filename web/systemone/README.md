# systemone

Go request and response types, a client, and an HTTP handler for the [System One API](https://docs.typesafe.ai/api).

The wire request has `model`, `state`, and named `questions`. State and instructions can be text or structured JSON. Noul questions can describe the `true` and `false` outcomes. Choice criteria is a map of up to 255 options. Score criteria is an ordered slice of 2 to 10 levels. Each answer is returned under its question name.

Ollama also accepts an optional `images` array for vision-capable Clef models. Supply base64-encoded image data without a URL or data URL prefix; images are shared by all questions in order. For example, set `Images: []string{base64.StdEncoding.EncodeToString(imageBytes)}` on the request below.

```go
client := systemone.NewClient("http://kadonomy.local:11434")
// Set client.APIKey when calling the hosted TypeSafe API.

result, err := client.Evaluate(ctx, &systemone.Request{
	Model: "clef",
	State: "Checkout has been failing for every customer for the last hour.",
	Questions: map[string]systemone.Question{
		"urgent": {
			Type: systemone.Noul,
			Instructions: "Is this support request urgent?",
		},
		"team": {
			Type: systemone.Choice,
			Instructions: "Which team should handle this request?",
			Criteria: map[string]any{
				"billing": "Payments, invoices, and refunds",
				"technical": "Outages, errors, and configuration",
				"sales": "Plans and upgrades",
			},
		},
		"severity": {
			Type: systemone.Score,
			Instructions: "How severe is the customer impact?",
			Criteria: []string{"No impact", "Minor", "Major", "Critical"},
		},
	},
})
if err != nil {
	return err
}
fmt.Println(result.Answers["team"].Choice)
```

An evaluator supplies the actual decisions to the HTTP handler. Mount it at `/v1/systemone`:

```go
var evaluator systemone.Evaluator = myEvaluator
mux := http.NewServeMux()
mux.Handle("/v1/systemone", systemone.NewHandler(evaluator))
```

`systemone.Client` also implements `Evaluator`, so it can be used to forward requests to another System One endpoint. The handler checks the request shape and returns JSON errors for invalid requests or evaluator failures.
