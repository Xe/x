package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"within.website/x/internal"
	"within.website/x/web/vastai/vastaicli"
)

var (
	bind          = flag.String("bind", ":3238", "HTTP port to bind to")
	diskSizeGB    = flag.Int("vastai-disk-size-gb", 32, "amount of disk we need from vast.ai")
	dockerImage   = flag.String("docker-image", "reg.xeiaso.net/xeserv/waifuwave:latest", "docker image to start")
	onstartCmd    = flag.String("onstart-cmd", "/opt/comfyui/startup.sh", "onstart command to run in vast.ai")
	vastaiPort    = flag.Int("vastai-port", 8080, "port that the guest will use in vast.ai")
	vastaiFilters = flag.String("vastai-filters", "verified=True cuda_max_good>=12.1 gpu_ram>=24 num_gpus=1 inet_down>=2000", "vast.ai search filters")

	idleTimeout = flag.Duration("idle-timeout", 5*time.Minute, "how long the instance should be considered \"idle\" before it is slain")
)

func main() {
	internal.HandleStartup()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if flag.NArg() != 1 {
		fmt.Println("usage: orodyagzou [flags] <whatever.env>")
		os.Exit(2)
	}

	fname := flag.Arg(0)
	slog.Debug("using env file", "fname", fname)

	env, err := godotenv.Read(fname)
	if err != nil {
		slog.Error("can't read env file", "fname", fname, "err", err)
		os.Exit(1)
	}

	var cfg vastaicli.InstanceConfig

	cfg.DiskSizeGB = *diskSizeGB
	cfg.Environment = env
	cfg.DockerImage = *dockerImage
	cfg.OnStartCMD = *onstartCmd
	cfg.Ports = append(cfg.Ports, *vastaiPort)

	images := &ScaleToZeroProxy{
		cfg: cfg,
	}

	go images.slayLoop(ctx)

	mux := http.NewServeMux()
	mux.Handle("/", images)

	fmt.Printf("http://localhost%s\n", *bind)
	log.Fatal(http.ListenAndServe(*bind, mux))
}

type ScaleToZeroProxy struct {
	cfg vastaicli.InstanceConfig

	// locked fields
	lock        sync.RWMutex
	endpointURL string
	instanceID  int
	ready       bool
	lastUsed    time.Time
}

func (s *ScaleToZeroProxy) slayLoop(ctx context.Context) {
	t := time.NewTicker(time.Second)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Error("context canceled", "err", ctx.Err())
			return
		case <-t.C:
			s.lock.RLock()
			ready := s.ready
			lastUsed := s.lastUsed
			s.lock.RUnlock()

			if !ready {
				continue
			}

			if lastUsed.Add(*idleTimeout).Before(time.Now()) {
				if err := s.slay(ctx); err != nil {
					slog.Error("can't slay instance", "err", err)
				}
			}
		}
	}
}

func (s *ScaleToZeroProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.lock.RLock()
	ready := s.ready
	s.lock.RUnlock()

	if !ready {
		if err := s.mint(r.Context()); err != nil {
			slog.Error("can't mint", "err", err)
			http.Error(w, "can't mint", http.StatusInternalServerError)
			return
		}
	}

	s.lock.RLock()
	endpointURL := s.endpointURL
	s.lock.RUnlock()

	u, err := url.Parse(endpointURL)
	if err != nil {
		slog.Error("can't url parse", "err", err, "url", s.endpointURL)
		http.Error(w, "can't url parse", http.StatusInternalServerError)
		return
	}

	next := httputil.NewSingleHostReverseProxy(u)
	next.ServeHTTP(w, r)

	s.lock.Lock()
	s.lastUsed = time.Now()
	s.lock.Unlock()
}

func (s *ScaleToZeroProxy) mint(ctx context.Context) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	candidates, err := vastaicli.Search(ctx, *vastaiFilters, "dph+")
	if err != nil {
		return err
	}

	candidate := candidates[0]
	slog.Info("found instance", "costDPH", candidate.DphTotal, "gpuName", candidate.GpuName)

	instanceData, err := vastaicli.Mint(ctx, candidate.AskContractID, s.cfg)
	if err != nil {
		return err
	}

	slog.Info("created instance, waiting for things to settle", "id", instanceData.NewContract)

	instance, err := s.delayUntilRunning(ctx, instanceData.NewContract)
	if err != nil {
		return err
	}

	addr, ok := instance.AddrFor(s.cfg.Ports[0])
	if !ok {
		return fmt.Errorf("somehow can't get port %d for instance %d, is humanity dead?", s.cfg.Ports[0], instance.ID)
	}

	s.endpointURL = "http://" + addr + "/"
	s.ready = true
	s.instanceID = instance.ID
	s.lastUsed = time.Now().Add(5 * time.Minute)

	if err := s.delayUntilReady(ctx, s.endpointURL); err != nil {
		return fmt.Errorf("can't do healthcheck: %w", err)
	}

	slog.Info("ready", "endpointURL", s.endpointURL, "instanceID", s.instanceID)

	return nil
}

func (s *ScaleToZeroProxy) slay(ctx context.Context) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if err := vastaicli.Slay(ctx, s.instanceID); err != nil {
		return err
	}

	s.endpointURL = ""
	s.ready = false
	s.lastUsed = time.Now()
	s.instanceID = 0

	slog.Info("instance slayed", "docker_image", s.cfg.DockerImage)

	return nil
}

func (s *ScaleToZeroProxy) delayUntilReady(ctx context.Context, endpointURL string) error {
	type cogHealthCheck struct {
		Status string `json:"status"`
	}

	u, err := url.Parse(endpointURL)
	if err != nil {
		return fmt.Errorf("[unexpected] can't parse endpoint url %q: %w", endpointURL, err)
	}

	u.Path = "/health-check"

	t := time.NewTicker(time.Second)
	defer t.Stop()

	failCount := 0

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
			if failCount >= 100 {
				return fmt.Errorf("healthcheck failed %d times", failCount+1)
			}

			resp, err := http.Get(u.String())
			if err != nil {
				slog.Error("health check failed", "err", err)
				continue
			}

			var status cogHealthCheck
			if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
				return fmt.Errorf("can't parse health check response: %w", err)
			}

			if status.Status == "READY" {
				slog.Info("health check passed")
				return nil
			}
		}
	}
}

func (s *ScaleToZeroProxy) delayUntilRunning(ctx context.Context, instanceID int) (*vastaicli.Instance, error) {
	t := time.NewTicker(10 * time.Second)
	defer t.Stop()

	var instance *vastaicli.Instance
	var err error

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-t.C:
			instance, err = vastaicli.GetInstance(ctx, instanceID)
			if err != nil {
				return nil, err
			}

			slog.Debug("instance is cooking", "curr", instance.ActualStatus, "next", instance.NextState, "status", instance.StatusMsg)

			if instance.ActualStatus == "running" {
				_, ok := instance.AddrFor(s.cfg.Ports[0])
				if !ok {
					slog.Info("no addr", "ports", s.cfg.Ports)
					continue
				}

				return instance, nil
			}
		}
	}
}
