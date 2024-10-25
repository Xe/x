// Command quiche is a little cute cache server for my B2 bucket.
package main

import (
	_ "embed"
	"encoding/json"
	"expvar"
	"flag"
	"fmt"
	"image/png"
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
	"within.website/x/internal"
	"within.website/x/internal/xesite"
	"within.website/x/web/fly/flymachines"
	"within.website/x/web/stablediffusion"
)

var (
	b2Backend             = flag.String("b2-backend", "f001.backblazeb2.com", "Backblaze B2 base host")
	addr                  = flag.String("addr", ":8080", "server bind address")
	metricsAddr           = flag.String("metrics-addr", ":8081", "metrics bind address")
	dir                   = flag.String("dir", envOr("XEDN_STATE", "./var"), "where XeDN should store cached data")
	staticDir             = flag.String("static-dir", envOr("XEDN_STATIC", "./static"), "where XeDN should look for static assets")
	stableDiffusionServer = flag.String("stable-diffusion-server", "http://xe-automatic1111.internal:8080", "where XeDN should request Stable Diffusion images from (Automatic1111)")

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

	etags    map[string]string
	etagLock sync.RWMutex
)

func init() {
	etags = map[string]string{}
}

func main() {
	internal.HandleStartup()

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

	sd := &StableDiffusion{
		db:     db,
		client: &stablediffusion.Client{HTTP: http.DefaultClient, APIServer: *stableDiffusionServer},
		group:  &singleflight.Group{},
	}

	iu := &ImageUploader{
		fmc: flymachines.New(*flyAPIToken, &http.Client{}),
	}

	os.MkdirAll(filepath.Join(*dir, "xesite"), 0700)
	zs, err := xesite.NewZipServer(filepath.Join(*dir, "xesite", "latest.zip"), *dir)
	if err != nil {
		log.Fatal(err)
	}

	xffMW, err := xff.Default()
	if err != nil {
		log.Fatal(err)
	}

	os.MkdirAll(*dir, 0700)

	{
		mux := http.NewServeMux()

		mux.HandleFunc("/xedn/optimize", iu.CreateImage)

		mux.HandleFunc("/", http.FileServer(http.Dir(filepath.Join(*dir, "uploud"))).ServeHTTP)

		go http.ListenAndServe(*metricsAddr, mux)
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
	cdn.HandleFunc("/cgi-cdn/wtf", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello, here is what I know about you:")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "HTTP headers from your client:")
		fmt.Fprintln(w)

		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		enc.Encode(r.Header)

		fmt.Fprintln(w)
		fmt.Fprintf(w, "I am XeDN %s (instance ID %s) running %s\n", os.Getenv("FLY_REGION"), os.Getenv("FLY_ALLOC_ID"), os.Args[0])
		fmt.Fprintln(w)
	})

	cdn.HandleFunc("/cgi-cdn/wtf/json", func(w http.ResponseWriter, r *http.Request) {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")

		enc.Encode(map[string]any{
			"headers": r.Header,
			"your_ip": r.Header.Get("Fly-Client-Ip"),
			"xedn": map[string]any{
				"region":   os.Getenv("FLY_REGION"),
				"instance": os.Getenv("FLY_ALLOC_ID"),
				"binary":   os.Args[0],
			},
		})
	})

	hdlr := func(w http.ResponseWriter, r *http.Request) {
		etagLock.RLock()
		etag, ok := etags[r.URL.Path]
		etagLock.RUnlock()

		if r.Header.Get("If-None-Match") == etag && ok {
			etagMatches.Add(1)
			w.WriteHeader(http.StatusNotModified)
			return
		}

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
	h = XeDNAnnotation(h)

	slog.Info("starting up", "addr", *addr)
	log.Fatal(http.ListenAndServe(*addr, h))
}

func XeDNAnnotation(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("XeDN", "true")
		next.ServeHTTP(w, r)
	})
}

func FlyRegionAnnotation(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Fly-Region", os.Getenv("FLY_REGION"))
		next.ServeHTTP(w, r)
	})
}
