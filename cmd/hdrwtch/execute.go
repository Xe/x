package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"within.website/x/web/useragent"
)

func checkURL(ctx context.Context, probe Probe) (*ProbeResult, error) {
	result := &ProbeResult{
		ProbeID: probe.ID,
		Region:  *region,
	}

	userAgent := useragent.GenUserAgent("hdrwtch/1.0", fmt.Sprintf("https://%s/docs/why-in-logs", *domain))

	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, "GET", probe.URL, nil)
	if err != nil {
		result.Success = false
		result.Remark = fmt.Sprintf("failed to create request: %v", err)
		return result, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", userAgent)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		result.Success = false
		result.Remark = fmt.Sprintf("failed to execute request: %v", err)
		return result, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	result.Success = true
	result.StatusCode = resp.StatusCode
	result.Duration = time.Since(start)

	return &ProbeResult{
		ProbeID:    probe.ID,
		StatusCode: resp.StatusCode,
		Duration:   time.Since(start),
	}, nil
}
