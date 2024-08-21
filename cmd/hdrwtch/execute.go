package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"runtime"
	"time"

	"golang.org/x/sync/errgroup"
	"within.website/x/web/useragent"
)

func checkURL(ctx context.Context, probe Probe) *ProbeResult {
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
		return result
	}

	req.Header.Set("User-Agent", userAgent)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		result.Success = false
		result.Remark = fmt.Sprintf("failed to execute request: %v", err)
		return result
	}
	defer resp.Body.Close()

	result.Success = true
	result.StatusCode = resp.StatusCode
	result.Duration = time.Since(start)
	result.LastModified = resp.Header.Get("Last-Modified")

	return result
}

func (s *Server) cron() {
	ctx, cancel := context.WithTimeout(context.Background(), 14*time.Minute) // 1 minute less than the cron interval
	defer cancel()

	tx := s.dao.db.Begin().WithContext(ctx)
	defer tx.Rollback()

	var probes []Probe

	if err := s.dao.db.Preload("LastResult").Find(&probes).Error; err != nil {
		slog.Error("failed to get probes", "err", err)
		return
	}

	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(runtime.NumCPU())

	for _, probe := range probes {
		probe := probe

		g.Go(func() error {
			result := checkURL(gCtx, probe)

			if result.LastModified != probe.LastResult.LastModified {
				slog.Info("probe result changed", "probe", probe.ID, "old", probe.LastResult, "new", result)

				user, err := s.dao.GetUser(gCtx, probe.UserID)
				if err != nil {
					slog.Error("failed to get user", "err", err)
					return err
				}

				if err := s.messageUser(user, fmt.Sprintf("*%s*:\n\nLast modified: %s\nRegion: %s\nStatus code: %d\nRemark: %s", probe.Name, result.LastModified, result.Region, result.StatusCode, result.Remark)); err != nil {
					slog.Error("failed to message user", "err", err)
					return err
				}
			}

			if err := s.dao.CreateProbeResult(gCtx, tx, probe, result); err != nil {
				slog.Error("failed to create probe result", "err", err, "probe", probe, "result", result)
				return err
			}

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		slog.Error("failed to check probes", "err", err)
	}

	if err := tx.Commit().Error; err != nil {
		slog.Error("failed to commit transaction", "err", err)
	}

	slog.Info("checked probes", "count", len(probes))
}
