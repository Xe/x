package main

import (
	"context"
	"encoding/json"
	"errors"
	"expvar"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"tailscale.com/tsnet"
	"tailscale.com/tsweb"
	"within.website/ln"
	"within.website/ln/ex"
	"within.website/ln/opname"
	"within.website/x/internal"
	"within.website/x/internal/yeet"
	"within.website/x/web/discordwebhook"
)

var (
	checkURL      = flag.String("check-url", "https://am.i.mullvad.net/json", "connection endpoint to check")
	containerNet  = flag.String("container", "wireguard", "container to assume the network stack of")
	dockerImage   = flag.String("docker-image", "ghcr.io/xe/alpine:3.17.3", "docker image to use")
	stateDir      = flag.String("state-dir", "", "where to store state data")
	tsnetHostname = flag.String("tsnet-hostname", "vest-pit-near", "hostname for tsnet")
	webhook       = flag.String("webhook", "", "Discord webhook URL")

	failureCount = expvar.NewInt("vest-pit-near_failure")
)

func main() {
	internal.HandleStartup()

	ctx := context.Background()

	os.MkdirAll(filepath.Join(*stateDir, "tsnet"), 0700)

	srv := &tsnet.Server{
		Hostname: *tsnetHostname,
		Logf:     log.New(io.Discard, "", 0).Printf,
		AuthKey:  os.Getenv("TS_AUTHKEY"),
		Dir:      filepath.Join(*stateDir, "tsnet"),
	}

	go cron()

	lis, err := srv.Listen("tcp", ":80")
	if err != nil {
		ln.FatalErr(ctx, err, ln.Action("tsnet listening"))
	}

	http.DefaultServeMux.HandleFunc("/metrics", tsweb.VarzHandler)

	defer srv.Close()
	defer lis.Close()
	ln.FatalErr(opname.With(ctx, "metrics-tsnet"), http.Serve(lis, ex.HTTPLog(http.DefaultServeMux)))
}

func cron() {
	for range time.Tick(5 * time.Minute) {
		func() {
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()

			if err := check(ctx); err != nil {
				ln.Error(ctx, err)
				defer failureCount.Set(1)

				checkErr := err

				if failureCount.Value() == 0 {
					resp, err := http.DefaultClient.Do(discordwebhook.Send(*webhook, discordwebhook.Webhook{
						Content: "VPN is down: " + checkErr.Error(),
					}))
					if err != nil {
						ln.Error(ctx, err)
						return
					}

					if err := discordwebhook.Validate(resp); err != nil {
						ln.Error(ctx, err)
						return
					}
				}

				return
			}

			failureCount.Set(0)
		}()
	}
}

func check(ctx context.Context) error {
	dataStr, err := yeet.Output(ctx, "docker", "run", "--rm", "-it", *dockerImage, "wget", "-T", "30", "-q", "-O-", *checkURL)
	if err != nil {
		return fmt.Errorf("vest-pit-near: can't run docker command: %w", err)
	}

	var mi MullvadInfo
	if err := json.Unmarshal([]byte(dataStr), &mi); err != nil {
		return fmt.Errorf("vest-pit-near: can't parse data as JSON: %w: %s", err, dataStr)
	}

	if !mi.MullvadExitIP {
		return errors.New("vest-pit-near: not using a Mullvad exit node")
	}

	return err
}

type MullvadInfo struct {
	IP          string      `json:"ip"`
	Country     string      `json:"country"`
	City        string      `json:"city"`
	Longitude   float64     `json:"longitude"`
	Latitude    float64     `json:"latitude"`
	Blacklisted Blacklisted `json:"blacklisted"`

	// Mullvad exit node information
	MullvadExitIP         bool    `json:"mullvad_exit_ip"`
	MullvadExitIPHostname *string `json:"mullvad_exit_ip_hostname"`
	MullvadServerType     *string `json:"mullvad_server_type"`
	Organization          *string `json:"organization"`
}

type Results struct {
	Name        string `json:"name"`
	Link        string `json:"link"`
	Blacklisted bool   `json:"blacklisted"`
}

type Blacklisted struct {
	Blacklisted bool      `json:"blacklisted"`
	Results     []Results `json:"results"`
}
