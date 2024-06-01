// Program azurda is a fake s3 server implementation. All objects are generated on the fly from Stable Diffusion.
//
// This is intended to be used as a "shadow bucket" endpoint with Tigris so that Tigris can "fall through" to Azurda if the object is not found in the real bucket.
package main

import (
	"bytes"
	"context"
	"embed"
	"flag"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/sync/errgroup"
	"within.website/x/internal"
	"within.website/x/web/stablediffusion"
)

var (
	accessKey    = flag.String("access-key", "", "Access key for the client to use")
	secretKey    = flag.String("secret-key", "", "Secret key for the client to use")
	bucketName   = flag.String("bucket-name", "fallthrough", "The bucket name to expect from Tigris")
	bind         = flag.String("bind", ":8085", "address to bind to")
	internalBind = flag.String("internal-bind", ":8086", "address to bind internal services (metrics, etc) to")
	sdServerURL  = flag.String("stablediffusion-server-url", "http://xe-automatic1111.internal:8080", "URL for the Stable Diffusion API used with the default client")

	isHexRegex = regexp.MustCompile(`[a-fA-F0-9]+$`)

	authErrors = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "azurda_auth_errors",
		Help: "Number of auth errors encountered while serving requests.",
	}, []string{"kind"})

	requestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "azurda_request_duration_seconds",
		Help:    "The duration of requests in seconds.",
		Buckets: []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10},
	}, []string{"method"})

	stableDiffusionHits = promauto.NewCounter(prometheus.CounterOpts{
		Name: "azurda_stable_diffusion_hits",
		Help: "Number of hits to the stable diffusion endpoint.",
	})

	stableDiffusionCreationErrors = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "azurda_stable_diffusion_creation_errors",
		Help: "Number of errors encountered while creating a stable diffusion image.",
	})

	//go:embed static
	static embed.FS
)

func main() {
	internal.HandleStartup()

	stablediffusion.Default.APIServer = *sdServerURL

	slog.Info("starting azurda",
		"bind", *bind,
		"internalBind", *internalBind,
		"bucket", *bucketName,
		"accessKey", *accessKey,
		"hasSecretKey", *secretKey != "",
		"stableDiffusionURL", stablediffusion.Default.APIServer,
	)

	if *accessKey == "" {
		fmt.Println("access-key is required")
		os.Exit(2)
	}

	if *secretKey == "" {
		fmt.Println("secret-key is required")
		os.Exit(2)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/{$}", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFileFS(w, r, static, "static/index.html")
	})
	mux.Handle("/static/", http.FileServerFS(static))
	mux.HandleFunc("GET fallthrough.azurda.within.website/{hash}", ServeStableDiffusion)

	http.Handle("/metrics", promhttp.Handler())

	g, _ := errgroup.WithContext(context.Background())

	g.Go(func() error {
		slog.Info("starting internal server", "bind", *internalBind)
		return http.ListenAndServe(*internalBind, nil)
	})

	g.Go(func() error {
		slog.Info("starting server", "bind", *bind)
		return http.ListenAndServe(*bind, SpewMiddleware(mux))
	})

	if err := g.Wait(); err != nil {
		slog.Error("error doing work", "err", err)
		os.Exit(1)
	}
}

func SpewMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info("got request", "method", r.Method, "url", r.URL.String(), "headers", r.Header)
		next.ServeHTTP(w, r)
	})
}

func ServeStableDiffusion(w http.ResponseWriter, r *http.Request) {
	hash := r.PathValue("hash")

	if !isHexRegex.MatchString(hash) {
		http.Error(w, "the input must be a hexadecimal string", http.StatusBadRequest)
		return
	}

	prompt, seed := hallucinatePrompt(hash)

	imgs, err := stablediffusion.Default.Generate(r.Context(), stablediffusion.SimpleImageRequest{
		Prompt:         "headshot, portrait, masterpiece, best quality, " + prompt,
		NegativePrompt: "person in distance, worst quality, low quality, medium quality, deleted, lowres, comic, bad anatomy, bad hands, text, error, missing fingers, extra digit, fewer digits, cropped, jpeg artifacts, signature, watermark, username, blurry",
		Seed:           seed,
		SamplerName:    "DPM++ 2M Karras",
		BatchSize:      1,
		NIter:          1,
		Steps:          20,
		CfgScale:       7,
		Width:          256,
		Height:         256,
		SNoise:         1,

		OverrideSettingsRestoreAfterwards: true,
	})
	if err != nil {
		stableDiffusionCreationErrors.Add(1)
		http.Error(w, "Not found", http.StatusNotFound)
		slog.Error("can't fabricate image", "err", err)
		return
	}

	stableDiffusionHits.Add(1)

	img, _, err := image.Decode(bytes.NewBuffer(imgs.Images[0]))
	if err != nil {
		stableDiffusionCreationErrors.Add(1)
		http.Error(w, "can't decode image", http.StatusInternalServerError)
		slog.Error("can't decode image", "err", err)
		return
	}

	buf := &bytes.Buffer{}

	if err := jpeg.Encode(buf, img, &jpeg.Options{Quality: 75}); err != nil {
		stableDiffusionCreationErrors.Add(1)
		http.Error(w, "can't encode image", http.StatusInternalServerError)
		slog.Error("can't encode image", "err", err)
		return
	}

	imgs.Images[0] = buf.Bytes()

	w.Header().Set("content-type", "image/jpeg")
	w.Header().Set("content-length", fmt.Sprint(len(imgs.Images[0])))
	w.Header().Set("expires", time.Now().Add(30*24*time.Hour).Format(http.TimeFormat))
	w.Header().Set("Cache-Control", "max-age:2630000") // one month
	w.WriteHeader(http.StatusOK)
	w.Write(imgs.Images[0])
}

// func AWSValidationMiddleware(accessKey, secretKey string, next http.Handler) http.Handler {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		//slog.Debug("incoming request", "method", r.Method, "path", r.URL.Path, "headers", r.Header)
// 		if r.Header.Get("Authorization") == "" {
// 			http.Error(w, "missing Authorization header", http.StatusUnauthorized)
// 			authErrors.WithLabelValues("missing").Inc()
// 			return
// 		}
//
// 		//slog.Debug("auth header", "header", r.Header.Get("Authorization"))
//
// 		sp := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
// 		if len(sp) != 2 {
// 			http.Error(w, "malformed Authorization header", http.StatusUnauthorized)
// 			slog.Error("malformed auth header")
// 			authErrors.WithLabelValues("malformed").Inc()
// 			return
// 		}
//
// 		if sp[0] != "AWS4-HMAC-SHA256" {
// 			http.Error(w, "unsupported authorization type", http.StatusUnauthorized)
// 			slog.Error("unsupported auth type", "type", sp[0])
// 			authErrors.WithLabelValues("unsupported").Inc()
// 			return
// 		}
//
// 		authPartSlice := strings.SplitN(sp[1], ", ", 3)
// 		slog.Debug("auth parts", "parts", authPartSlice)
// 		if len(authPartSlice) != 3 {
// 			http.Error(w, "malformed Authorization header auth parts", http.StatusUnauthorized)
// 			authErrors.WithLabelValues("malformed_authparts").Inc()
// 			return
// 		}
//
// 		authParts := map[string]string{}
// 		for _, part := range authPartSlice {
// 			sp := strings.SplitN(part, "=", 2)
// 			if len(sp) != 2 {
// 				http.Error(w, "malformed Authorization header auth part", http.StatusUnauthorized)
// 				slog.Debug("malformed auth part", "part", part)
// 				authErrors.WithLabelValues("malformed_authpart").Inc()
// 				return
// 			}
//
// 			authParts[strings.ToLower(sp[0])] = strings.Trim(sp[1], "\"")
// 		}
//
// 		if authParts["credential"] == "" {
// 			http.Error(w, "missing credential in Authorization header", http.StatusUnauthorized)
// 			slog.Debug("missing credential in auth header")
// 			authErrors.WithLabelValues("missing_credential").Inc()
// 			return
// 		}
//
// 		if authParts["signature"] == "" {
// 			http.Error(w, "missing signature in Authorization header", http.StatusUnauthorized)
// 			slog.Debug("missing signature in auth header")
// 			authErrors.WithLabelValues("missing_signature").Inc()
// 			return
// 		}
//
// 		if authParts["signedheaders"] == "" {
// 			http.Error(w, "missing signedheaders in Authorization header", http.StatusUnauthorized)
// 			slog.Debug("missing signedheaders in auth header")
// 			authErrors.WithLabelValues("missing_signedheaders").Inc()
// 			return
// 		}
//
// 		if !strings.Contains(authParts["credential"], accessKey) {
// 			http.Error(w, "access key mismatch", http.StatusUnauthorized)
// 			authErrors.WithLabelValues("access_key_mismatch").Inc()
// 			return
// 		}
//
// 		req := r.Clone(r.Context())
// 		req.Header.Del("Authorization")
//
// 		req = awsauth.Sign4(req, awsauth.Credentials{
// 			AccessKeyID:     accessKey,
// 			SecretAccessKey: secretKey,
// 		})
//
// 		fmt.Println("Theirs: ", r.Header.Get("Authorization"))
// 		fmt.Println("Ours:   ", req.Header.Get("Authorization"))
//
// 		if req.Header.Get("Authorization") != r.Header.Get("Authorization") {
// 			http.Error(w, "failed to sign request", http.StatusUnauthorized)
// 			return
// 		}
//
// 		next.ServeHTTP(w, r)
// 	})
// }
