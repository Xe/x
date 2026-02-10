package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"within.website/x/cmd/markdownlang-telemetryd/internal/notifications"
	"within.website/x/cmd/markdownlang-telemetryd/internal/report"
	"within.website/x/cmd/markdownlang-telemetryd/internal/storage"
	"within.website/x/cmd/markdownlang-telemetryd/internal/users"
	"within.website/x/internal"
)

var (
	httpAddr = flag.String("http-addr", ":9100", "HTTP listen address")
)

func main() {
	internal.HandleStartup()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Initialize storage
	st, err := storage.New(ctx)
	if err != nil {
		log.Fatalf("failed to create storage: %v", err)
	}

	// Initialize user tracker with S3 backing
	tracker, err := users.New(ctx, st.Interface())
	if err != nil {
		log.Fatalf("failed to create user tracker: %v", err)
	}

	// Ingest handler
	http.HandleFunc("/ingest", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Read and parse request body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			slog.Error("failed to read request body", "error", err)
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		var rep report.Report
		if err := json.Unmarshal(body, &rep); err != nil {
			slog.Error("failed to parse request body", "error", err)
			http.Error(w, "Failed to parse request body", http.StatusBadRequest)
			return
		}

		// Validate report
		if err := rep.Validate(); err != nil {
			slog.Error("validation failed", "error", err, "email", rep.GitUserEmail)
			http.Error(w, fmt.Sprintf("Validation failed: %v", err), http.StatusBadRequest)
			return
		}

		// Check if this is a new user
		if tracker.IsNewUser(rep.GitUserName, rep.GitUserEmail) {
			tracker.RecordUser(rep.GitUserName, rep.GitUserEmail)
			if err := tracker.Save(ctx); err != nil {
				slog.Error("failed to save user data", "error", err)
			}

			// Send Discord notification for new user
			if err := notifications.NotifyNewUser(rep.GitUserName, rep.GitUserEmail); err != nil {
				slog.Error("failed to send new user notification", "error", err, "email", rep.GitUserEmail)
			} else {
				slog.Info("new user detected and notified", "name", rep.GitUserName, "email", rep.GitUserEmail)
			}
		}

		// Increment execution count
		count := tracker.IncrementExecution(rep.GitUserEmail)

		// Persist count after each increment
		if err := tracker.Save(ctx); err != nil {
			slog.Error("failed to save execution count", "error", err)
		}

		// Check if user has exceeded quota
		if count > users.QuotaLimit {
			// Notify Xe about quota exceeded (only once when crossing the threshold)
			if count == users.QuotaLimit+1 {
				if err := notifications.NotifyXeForQuota(rep.GitUserName, rep.GitUserEmail, count); err != nil {
					slog.Error("failed to send quota exceeded notification", "error", err, "email", rep.GitUserEmail)
				} else {
					slog.Info("quota exceeded notification sent", "name", rep.GitUserName, "email", rep.GitUserEmail, "count", count)
				}
			}
		}

		// Store report in S3
		if err := st.Store(ctx, storage.Report{
			OS:               rep.OS,
			Arch:             rep.Arch,
			GoVersion:        rep.GoVersion,
			NumCPU:           rep.NumCPU,
			Hostname:         rep.Hostname,
			UnameAll:         rep.UnameAll,
			GitUserName:      rep.GitUserName,
			GitUserEmail:     rep.GitUserEmail,
			Version:          rep.Version,
			Program:          rep.Program,
			ProgramSHA256:    rep.ProgramSHA256,
			DurationMs:       rep.DurationMs,
			ToolCallCount:    rep.ToolCallCount,
			ToolsUsed:        rep.ToolsUsed,
			MCPServers:       rep.MCPServers,
			MCPToolsUsed:     rep.MCPToolsUsed,
			ModelProviderURL: rep.ModelProviderURL,
			ModelName:        rep.ModelName,
			Shell:            rep.Shell,
			Term:             rep.Term,
			Timezone:         rep.Timezone,
			WorkingDir:       rep.WorkingDir,
			Timestamp:        rep.Timestamp,
		}); err != nil {
			slog.Error("failed to store report", "error", err, "email", rep.GitUserEmail)
			http.Error(w, "Failed to store report", http.StatusInternalServerError)
			return
		}

		slog.Debug("report ingested successfully", "email", rep.GitUserEmail, "count", count)
		w.WriteHeader(http.StatusAccepted)
	})

	// Health check endpoint
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "OK")
	})

	srv := &http.Server{
		Addr:    *httpAddr,
		Handler: http.DefaultServeMux,
	}

	go func() {
		slog.Info("markdownlang-telemetryd started", "httpAddr", *httpAddr, "bucket", st.BucketName())
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	select {
	case <-sigCh:
		slog.Info("shutdown signal received")
	case <-ctx.Done():
		slog.Info("context cancelled")
	}

	// Graceful shutdown
	if err := srv.Shutdown(context.Background()); err != nil {
		log.Fatalf("shutdown error: %v", err)
	}

	slog.Info("markdownlang-telemetryd stopped")
}
