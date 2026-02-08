package agent

import (
	"testing"
	"time"
)

func TestMetricsReset(t *testing.T) {
	m := Metrics{
		Iterations:   5,
		TotalTokens:  1000,
		InputTokens:  500,
		OutputTokens: 500,
		ToolsCalled:  2,
		Duration:     time.Second,
		ErrorCount:   1,
	}

	m.Reset()

	if m.Iterations != 0 {
		t.Errorf("expected Iterations to be 0 after reset, got %d", m.Iterations)
	}
	if m.TotalTokens != 0 {
		t.Errorf("expected TotalTokens to be 0 after reset, got %d", m.TotalTokens)
	}
	if m.ToolsCalled != 0 {
		t.Errorf("expected ToolsCalled to be 0 after reset, got %d", m.ToolsCalled)
	}
}

func TestMetricsTokenUsage(t *testing.T) {
	m := Metrics{
		TotalTokens:  1000,
		InputTokens:  600,
		OutputTokens: 400,
	}

	usage := m.TokenUsage()

	if usage.Total != 1000 {
		t.Errorf("expected Total to be 1000, got %d", usage.Total)
	}
	if usage.Input != 600 {
		t.Errorf("expected Input to be 600, got %d", usage.Input)
	}
	if usage.Output != 400 {
		t.Errorf("expected Output to be 400, got %d", usage.Output)
	}
}

func TestMetricsSuccessRate(t *testing.T) {
	tests := []struct {
		name     string
		metrics  Metrics
		wantRate float64
	}{
		{
			name: "no errors",
			metrics: Metrics{
				Iterations: 10,
				ErrorCount: 0,
			},
			wantRate: 100.0,
		},
		{
			name: "half errors",
			metrics: Metrics{
				Iterations: 10,
				ErrorCount: 5,
			},
			wantRate: 50.0,
		},
		{
			name: "all errors",
			metrics: Metrics{
				Iterations: 10,
				ErrorCount: 10,
			},
			wantRate: 0.0,
		},
		{
			name:     "no iterations",
			metrics:  Metrics{},
			wantRate: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rate := tt.metrics.SuccessRate()
			if rate != tt.wantRate {
				t.Errorf("SuccessRate() = %f, want %f", rate, tt.wantRate)
			}
		})
	}
}

func TestMetricsAverageTokensPerIteration(t *testing.T) {
	tests := []struct {
		name    string
		metrics Metrics
		wantAvg float64
	}{
		{
			name: "simple average",
			metrics: Metrics{
				Iterations:  10,
				TotalTokens: 1000,
			},
			wantAvg: 100.0,
		},
		{
			name: "no iterations",
			metrics: Metrics{
				TotalTokens: 1000,
			},
			wantAvg: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			avg := tt.metrics.AverageTokensPerIteration()
			if avg != tt.wantAvg {
				t.Errorf("AverageTokensPerIteration() = %f, want %f", avg, tt.wantAvg)
			}
		})
	}
}

func TestTokenUsageCost(t *testing.T) {
	tests := []struct {
		name     string
		usage    TokenUsage
		wantCost float64
	}{
		{
			name:     "no tokens",
			usage:    TokenUsage{},
			wantCost: 0.0,
		},
		{
			name: "only input tokens",
			usage: TokenUsage{
				Input: 1_000_000,
			},
			wantCost: 2.50,
		},
		{
			name: "only output tokens",
			usage: TokenUsage{
				Output: 1_000_000,
			},
			wantCost: 10.00,
		},
		{
			name: "mixed tokens",
			usage: TokenUsage{
				Input:  500_000,
				Output: 500_000,
			},
			wantCost: 6.25,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost := tt.usage.Cost()
			if cost != tt.wantCost {
				t.Errorf("Cost() = %f, want %f", cost, tt.wantCost)
			}
		})
	}
}

func TestMetricsSnapshot(t *testing.T) {
	m := &Metrics{
		Iterations:  5,
		TotalTokens: 1000,
	}

	snapshot := m.Snapshot()

	if snapshot.Metrics.Iterations != 5 {
		t.Errorf("expected snapshot Iterations to be 5, got %d", snapshot.Metrics.Iterations)
	}
	if snapshot.Timestamp.IsZero() {
		t.Error("expected snapshot timestamp to be set")
	}
}

func TestMetricsDelta(t *testing.T) {
	from := MetricsSnapshot{
		Metrics: Metrics{
			Iterations:  5,
			TotalTokens: 1000,
			ToolsCalled: 2,
		},
		Timestamp: time.Now(),
	}

	to := MetricsSnapshot{
		Metrics: Metrics{
			Iterations:  10,
			TotalTokens: 2000,
			ToolsCalled: 4,
		},
		Timestamp: from.Timestamp.Add(time.Second),
	}

	delta := Delta(from, to)

	if delta.Iterations != 5 {
		t.Errorf("expected delta Iterations to be 5, got %d", delta.Iterations)
	}
	if delta.TotalTokens != 1000 {
		t.Errorf("expected delta TotalTokens to be 1000, got %d", delta.TotalTokens)
	}
	if delta.ToolsCalled != 2 {
		t.Errorf("expected delta ToolsCalled to be 2, got %d", delta.ToolsCalled)
	}
	if delta.Duration != time.Second {
		t.Errorf("expected delta Duration to be 1s, got %v", delta.Duration)
	}
}

func TestMetricsString(t *testing.T) {
	m := Metrics{
		Iterations:   5,
		TotalTokens:  1000,
		InputTokens:  600,
		OutputTokens: 400,
		ToolsCalled:  2,
		Duration:     time.Second,
		ErrorCount:   1,
	}

	str := m.String()
	if str == "" {
		t.Error("expected non-empty string")
	}
}

func TestTokenUsageString(t *testing.T) {
	usage := TokenUsage{
		Total:  1000,
		Input:  600,
		Output: 400,
	}

	str := usage.String()
	if str == "" {
		t.Error("expected non-empty string")
	}
}

func TestMetricsDeltaString(t *testing.T) {
	delta := MetricsDelta{
		Iterations:  5,
		TotalTokens: 1000,
		ToolsCalled: 2,
		Duration:    time.Second,
	}

	str := delta.String()
	if str == "" {
		t.Error("expected non-empty string")
	}
}
