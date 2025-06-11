package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	pb "within.website/x/gen/within/website/x/xedn/uplodr/v1"
	"within.website/x/web/fly/flymachines"
)

var (
	flyAPIToken        = flag.String("fly-api-token", "", "Fly API token to use")
	flyAppName         = flag.String("fly-app-name", "xedn", "Fly app name to use")
	flyRegion          = flag.String("fly-region", "yyz", "Fly region to use")
	uplodrMachineImage = flag.String("uplodr-machine-image", "registry.fly.io/xedn", "Docker image to use for uplodr Machines")
	uplodrBinary       = flag.String("uplodr-binary", "/bin/uplodr", "Binary to run on uplodr Machines")
	uplodrPort         = flag.String("uplodr-port", "9001", "Port to run uplodr on")
)

type ImageUploader struct {
	fmc *flymachines.Client
}

func (iu *ImageUploader) CreateImage(w http.ResponseWriter, r *http.Request) {
	if r.ContentLength == 0 {
		http.Error(w, "empty body", http.StatusBadRequest)
		return
	}

	ext := r.URL.Query().Get("ext")
	if ext == "" {
		ext = ".png"
		if r.Header.Get("Content-Type") != "" {
			mimeType := r.Header.Get("Content-Type")
			ext = mimeTypes[mimeType]
		}
	}

	fname := r.URL.Query().Get("name")
	if fname == "" {
		fname = uuid.New().String()
	}

	folder := r.URL.Query().Get("folder")
	if folder == "" {
		folder = "xedn/dynamic"
	}

	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Minute)
	defer cancel()

	data, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("cannot read body", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	chonkiness := len(data) + 4096

	slog.Debug("creating machine")
	m, err := iu.fmc.CreateMachine(ctx, *flyAppName, flymachines.CreateMachine{
		Name:   "uplodr-" + uuid.New().String(),
		Region: *flyRegion,
		Config: flymachines.MachineConfig{
			Guest: flymachines.MachineGuest{
				CPUKind:  "performance",
				CPUs:     16,
				MemoryMB: 8192 * 4,
			},
			Image: *uplodrMachineImage,
			Processes: []flymachines.MachineProcess{
				{
					Cmd: []string{*uplodrBinary, "--grpc-addr=:" + *uplodrPort, "--slog-level=debug", fmt.Sprintf("--msg-size=%d", chonkiness)},
				},
			},
			StopConfig: flymachines.MachineStopConfig{
				Timeout: "30s",
				Signal:  "SIGKILL",
			},
			Restart: flymachines.MachineRestart{
				MaxRetries: 0,
				Policy:     flymachines.MachineRestartPolicyNo,
			},
		},
	})
	if err != nil {
		slog.Error("cannot create machine", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		bo := backoff.NewExponentialBackOff()

		err := backoff.Retry(func() error {
			select {
			case <-ctx.Done():
				return backoff.Permanent(ctx.Err())
			default:
			}

			slog.Debug("deleting machine", "machine", m.ID)
			return iu.fmc.DestroyAppMachine(ctx, *flyAppName, m.ID)
		}, bo)
		if err != nil {
			slog.Error("cannot delete machine", "err", err, "machine", m.ID)
		}
	}()

	running := false
	for !running {
		time.Sleep(time.Second)
		mi, err := iu.fmc.GetAppMachine(ctx, *flyAppName, m.ID)
		if err != nil {
			slog.Error("can't get machine state", "id", m.ID, "err", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if mi.State != "started" {
			slog.Debug("not started", "want", "started", "got", mi.State)
			continue
		}
		running = true
	}

	bo := backoff.NewExponentialBackOff()
	addr := net.JoinHostPort(m.PrivateIP, *uplodrPort)

	if err := backoff.Retry(func() error {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			return err
		}
		return conn.Close()
	}, bo); err != nil {
		slog.Error("cannot test dial machine", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	bo.Reset()
	conn, err := backoff.RetryWithData[*grpc.ClientConn](func() (*grpc.ClientConn, error) {
		slog.Debug("dialing machine", "addr", addr)

		conn, err := grpc.DialContext(ctx, addr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithMaxMsgSize(chonkiness),
		)
		if err != nil {
			slog.Error("cannot dial machine", "err", err, "addr", addr)
		}

		return conn, err
	}, bo)
	if err != nil {
		slog.Error("cannot dial machine", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer conn.Close()
	slog.Debug("conn created")

	client := pb.NewImageClient(conn)
	id := uuid.New().String()
	bo.Reset()
	pong, err := backoff.RetryWithData[*pb.Echo](func() (*pb.Echo, error) {
		return client.Ping(ctx, &pb.Echo{Nonce: id})
	}, bo)
	if err != nil {
		slog.Error("cannot ping machine", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if pong.GetNonce() != id {
		slog.Error("invalid nonce", "got", pong.GetNonce(), "want", id)
		http.Error(w, "invalid nonce", http.StatusInternalServerError)
		return
	}

	slog.Info("pong", "msg", pong.GetNonce())

	bo.Reset()
	imageData, err := backoff.RetryWithData[*pb.UploadResp](func() (*pb.UploadResp, error) {
		slog.Debug("uploading image", "fname", fname, "ext", ext)
		resp, err := client.Upload(ctx, &pb.UploadReq{
			FileName: fname + "." + ext,
			Data:     data,
			Folder:   folder,
		},
			grpc.MaxCallRecvMsgSize(chonkiness),
			grpc.MaxCallSendMsgSize(chonkiness),
		)
		if err != nil {
			slog.Error("can't upload image", "err", err)
		}

		return resp, err
	}, bo)
	if err != nil {
		slog.Error("cannot upload image", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	result := map[string]string{}

	for _, vari := range imageData.Variants {
		result[extForMimeType[vari.MimeType]] = vari.Url
	}

	json.NewEncoder(w).Encode(result)
}

var mimeTypes = map[string]string{
	".avif": "image/avif",
	".webp": "image/webp",
	".jpg":  "image/jpeg",
	".png":  "image/png",
	".wasm": "application/wasm",
	".css":  "text/css",
}

var extForMimeType = map[string]string{
	"image/avif":       "avif",
	"image/webp":       "webp",
	"image/jpeg":       "jpg",
	"image/png":        "png",
	"application/wasm": "wasm",
	"text/css":         "css",
}
