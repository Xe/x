package flux

import (
	"fmt"
	"io/ioutil"
	"time"
)

func Example() {
	client := NewClient("http://xe-flux.flycast")

	if _, err := client.HealthCheck(); err != nil {
		fmt.Println("Error health checking:", err)
		panic(err)
	}

	// Example of using the Predict method
	predictionReq := PredictionRequest{
		Input: Input{
			Prompt:            "A beautiful sunrise over the mountains",
			NumOutputs:        1,
			GuidanceScale:     7.5,
			MaxSequenceLength: 256,
			NumInferenceSteps: 50,
			PromptStrength:    0.8,
			OutputFormat:      "png",
			OutputQuality:     90,
		},
		ID:                  "example-prediction-id",
		CreatedAt:           time.Now().Format(time.RFC3339),
		OutputFilePrefix:    "output",
		Webhook:             "http://example.com/webhook",
		WebhookEventsFilter: []string{"start", "completed"},
	}

	predictionResp, err := client.Predict(predictionReq)
	if err != nil {
		fmt.Println("Error predicting:", err)
	} else {
		fmt.Println("PredictionResponse:", predictionResp)
	}

	// Example of using the PredictIdempotent method
	predictionResp, err = client.PredictIdempotent("example-prediction-id", predictionReq)
	if err != nil {
		fmt.Println("Error predicting idempotent:", err)
	} else {
		fmt.Println("PredictIdempotentResponse:", predictionResp)
	}

	// Example of using the CancelPrediction method
	resp, err := client.CancelPrediction("example-prediction-id")
	if err != nil {
		fmt.Println("Error cancelling prediction:", err)
	} else {
		body, _ := ioutil.ReadAll(resp.Body)
		fmt.Println("CancelPrediction response:", string(body))
		resp.Body.Close()
	}
}
