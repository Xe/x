// Command quiche is a little cute cache server for my B2 bucket.
package main

import (
	_ "embed"
	"expvar"
	"flag"
	"image/png"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"github.com/sebest/xff"
	"go.etcd.io/bbolt"
	"golang.org/x/sync/singleflight"
	"tailscale.com/hostinfo"
	"tailscale.com/metrics"
	"tailscale.com/tsnet"
	"tailscale.com/tsweb"
	"within.website/x/cmd/xedn/internal/xesite"
	"within.website/x/internal"
	"within.website/x/web/stablediffusion"
)

var (
	b2Backend             = flag.String("b2-backend", "f001.backblazeb2.com", "Backblaze B2 base host")
	addr                  = flag.String("addr", ":8080", "server bind address")
	metricsAddr           = flag.String("metrics-addr", ":8081", "metrics bind address")
	dir                   = flag.String("dir", envOr("XEDN_STATE", "./var"), "where XeDN should store cached data")
	staticDir             = flag.String("static-dir", envOr("XEDN_STATIC", "./static"), "where XeDN should look for static assets")
	stableDiffusionServer = flag.String("stable-diffusion-server", "http://xe-automatic1111.internal:8080", "where XeDN should request Stable Diffusion images from (Automatic1111)")
	tailscaleVerbose      = flag.Bool("tailscale-verbose", false, "enable verbose tailscale logging")

	//go:embed index.html
	indexHTML []byte
)

func envOr(name, def string) string {
	if val, ok := os.LookupEnv(name); ok {
		return val
	}

	return def
}

var (
	cacheHits   = expvar.NewInt("counter_xedn_cache_hits")
	cacheErrors = expvar.NewInt("counter_xedn_cache_errors")
	cacheLoads  = expvar.NewInt("counter_xedn_cache_loads")

	etagMatches = expvar.NewInt("counter_xedn_etag_matches")

	referers      = metrics.LabelMap{Label: "url"}
	fileHits      = metrics.LabelMap{Label: "path"}
	fileDeaths    = metrics.LabelMap{Label: "path"}
	fileMimeTypes = metrics.LabelMap{Label: "type"}

	etags    map[string]string
	etagLock sync.RWMutex
)

func init() {
	etags = map[string]string{}

	expvar.Publish("gauge_xedn_referers", &referers)
	expvar.Publish("gauge_xedn_file_hits", &fileHits)
	expvar.Publish("gauge_xedn_file_deaths", &fileDeaths)
	expvar.Publish("gauge_xedn_file_mime_type", &fileMimeTypes)
	expvar.Publish("gauge_xedn_ois_file_conversions", &OISFileConversions)
	expvar.Publish("gauge_xedn_ois_file_hits", &OISFileHits)
}

func main() {
	internal.HandleStartup()

	hostinfo.SetApp("within.website/x/cmd/xedn")

	os.MkdirAll(filepath.Join(*dir, "tsnet"), 0700)

	db, err := bbolt.Open(filepath.Join(*dir, "data"), 0600, &bbolt.Options{})
	if err != nil {
		log.Fatal(err)
	}

	dc := &Cache{
		ActualHost: *b2Backend,
		Client:     &http.Client{},
		DB:         db,
		cacheGroup: &singleflight.Group{},
	}

	go dc.CronPurgeDead()

	ois := &OptimizedImageServer{
		DB:     db,
		Cache:  dc,
		PNGEnc: &png.Encoder{CompressionLevel: png.BestCompression},
		group:  &singleflight.Group{},
	}

	srv := &tsnet.Server{
		Hostname: "xedn-" + os.Getenv("FLY_REGION"),
		Logf:     log.New(io.Discard, "", 0).Printf,
		AuthKey:  os.Getenv("TS_AUTHKEY"),
		Dir:      filepath.Join(*dir, "tsnet"),
	}

	if *tailscaleVerbose {
		srv.Logf = log.Printf
	}

	srv.Start()

	sd := &StableDiffusion{
		db:     db,
		client: &stablediffusion.Client{HTTP: http.DefaultClient, APIServer: *stableDiffusionServer},
		group:  &singleflight.Group{},
	}

	iu := &ImageUploader{
		s3: mkS3Client(),
	}

	os.MkdirAll(filepath.Join(*dir, "xesite"), 0700)
	zs, err := xesite.NewZipServer(filepath.Join(*dir, "xesite", "latest.zip"), *dir)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		lis, err := srv.Listen("tcp", ":80")
		if err != nil {
			log.Fatalf("can't listen on tsnet: %v", err)
		}

		http.DefaultServeMux.HandleFunc("/debug/varz", tsweb.VarzHandler)
		http.DefaultServeMux.HandleFunc("/xedn/files", dc.ListFiles)
		http.DefaultServeMux.HandleFunc("/xedn/purge", dc.Purge)
		http.DefaultServeMux.HandleFunc("/xesite/generations", zs.ListGenerations)
		http.DefaultServeMux.HandleFunc("/xesite/nuke", zs.NukeGeneration)
		http.DefaultServeMux.HandleFunc("/xesite/upload", zs.UploadNewZip)
		http.DefaultServeMux.HandleFunc("/sticker/files", ois.ListFiles)
		http.DefaultServeMux.HandleFunc("/sticker/purge", ois.Purge)

		defer srv.Close()
		defer lis.Close()
		log.Fatal(http.Serve(lis, http.DefaultServeMux))
	}()

	xffMW, err := xff.Default()
	if err != nil {
		log.Fatal(err)
	}

	os.MkdirAll(*dir, 0700)

	{
		mux := http.NewServeMux()

		mux.HandleFunc("/metrics", tsweb.VarzHandler)
		mux.HandleFunc("/xedn/optimize", iu.CreateImage)

		mux.HandleFunc("/", http.FileServer(http.Dir(filepath.Join(*dir, "uploud"))).ServeHTTP)

		go http.ListenAndServe(*metricsAddr, tsweb.StdHandler(tsweb.ReturnHandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
			mux.ServeHTTP(w, r)
			return nil
		}), tsweb.HandlerOptions{
			Logf: log.Printf,
		}))
	}

	cdn := http.NewServeMux()

	cdn.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Length", strconv.Itoa(len(indexHTML)))
		w.WriteHeader(http.StatusOK)
		w.Write(indexHTML)
	})

	cdn.Handle("/sticker/", ois)
	cdn.Handle("/avatar/", sd)
	cdn.Handle("/static/", http.FileServer(http.Dir(*staticDir)))

	hdlr := func(w http.ResponseWriter, r *http.Request) {
		etagLock.RLock()
		etag, ok := etags[r.URL.Path]
		etagLock.RUnlock()

		if r.Header.Get("If-None-Match") == etag && ok {
			etagMatches.Add(1)
			w.WriteHeader(http.StatusNotModified)
			return
		}

		referers.Get(r.Header.Get("Referer")).Add(1)
		fileMimeTypes.Get(r.Header.Get("Content-Type")).Add(1)

		if err := dc.GetFile(w, r); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	cdn.HandleFunc("/file/christine-static/", hdlr)
	cdn.HandleFunc("/file/xeserv-akko/", hdlr)

	topLevel := mux.NewRouter()

	topLevel.Host("cdn.christine.website").Handler(cdn)
	topLevel.Host("cdn.xeiaso.net").Handler(cdn)
	topLevel.Host("xedn.fly.dev").Handler(cdn)
	topLevel.Host("pneuma.shark-harmonic.ts.net").Handler(cdn)
	topLevel.Host("xelaso.net").Handler(zs)

	topLevel.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "wait, what, how did you do that?", http.StatusBadRequest)
	})

	var h http.Handler = topLevel
	h = xffMW.Handler(h)
	h = cors.Default().Handler(h)
	h = FlyRegionAnnotation(h)

	slog.Info("starting up", "addr", *addr)
	http.ListenAndServe(*addr, h)
}

func FlyRegionAnnotation(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Fly-Region", os.Getenv("FLY_REGION"))
		next.ServeHTTP(w, r)
	})
}
