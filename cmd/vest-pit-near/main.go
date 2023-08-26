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
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"tailscale.com/hostinfo"
	"tailscale.com/tsnet"
	"tailscale.com/tsweb"
	"within.website/x/internal"
	"within.website/x/internal/yeet"
	"within.website/x/web/discordwebhook"
)

var (
	checkURL      = flag.String("check-url", "https://am.i.mullvad.net/json", "connection endpoint to check")
	containerNet  = flag.String("container", "wireguard", "container to assume the network stack of")
	dockerImage   = flag.String("docker-image", "ghcr.io/xe/alpine:3.18.2", "docker image to use")
	stateDir      = flag.String("state-dir", "", "where to store state data")
	tsnetHostname = flag.String("tsnet-hostname", "vest-pit-near", "hostname for tsnet")
	webhook       = flag.String("webhook", "", "Discord webhook URL")

	failureCount = expvar.NewInt("vest-pit-near_failure")
)

func main() {
	internal.HandleStartup()

	hostinfo.SetApp("within.website/x/cmd/vest-pit-near")

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
		log.Fatalf("can't listen over tsnet: %v", err)
	}

	http.DefaultServeMux.HandleFunc("/metrics", tsweb.VarzHandler)

	defer srv.Close()
	defer lis.Close()
	log.Fatal(http.Serve(lis, http.DefaultServeMux))
}

func cron() {
	for range time.Tick(5 * time.Minute) {
		func() {
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()

			if err := check(ctx); err != nil {
				slog.Error("can't check status", "err", err)
				defer failureCount.Set(1)

				checkErr := err

				if failureCount.Value() == 0 {
					resp, err := http.DefaultClient.Do(discordwebhook.Send(*webhook, discordwebhook.Webhook{
						Content:   "VPN is down: " + checkErr.Error(),
						Username:  "vest-pit-near",
						AvatarURL: "https://cdn.discordapp.com/attachments/262330174140448768/1093162451341684736/04401-1288759123-flat_color_limited_palette_low_contrast_high_contrast_chromatic_aberration_1girl_hoodie_green_hair_coffee_onsen_green.png",
					}))
					if err != nil {
						slog.Error("can't report VPN is down", "err", err)
						return
					}

					if err := discordwebhook.Validate(resp); err != nil {
						slog.Error("can't validate webhook response", "err", err)
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
	dataStr, err := yeet.Output(ctx, "docker", "run", "--rm", "--net=container:"+*containerNet, *dockerImage, "wget", "-T", "30", "-q", "-O-", *checkURL)
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
