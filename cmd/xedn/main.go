// Command quiche is a little cute cache server for my B2 bucket.
package main

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/gob"
	"expvar"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/golang/groupcache"
	"github.com/sebest/xff"
	"tailscale.com/tsnet"
	"tailscale.com/tsweb"
	"within.website/ln"
	"within.website/ln/ex"
	"within.website/ln/opname"
	"within.website/x/internal"
	"within.website/x/web"
)

var (
	b2Backend = flag.String("b2-backend", "https://f001.backblazeb2.com", "Backblaze B2 base URL")
	addr      = flag.String("addr", ":8080", "server address")

	//go:embed index.html
	indexHTML []byte
)

const cacheSize = 128 * 1024 * 1024 // 128 mebibytes

type CacheData struct {
	Headers http.Header
	Body    []byte
}

var Group = groupcache.NewGroup("b2-bucket", cacheSize, groupcache.GetterFunc(
	func(ctx groupcache.Context, key string, dest groupcache.Sink) error {
		ln.Log(context.Background(), ln.F{"key": key})

		fileHits.Add(key, 1)

		resp, err := http.Get(*b2Backend + key)
		if err != nil {
			return fmt.Errorf("can't fetch from b2: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return web.NewError(http.StatusOK, resp)
		}

		etag := fmt.Sprintf("%q", resp.Header.Get("x-bz-content-sha1"))
		resp.Header.Set("ETag", etag)
		etagLock.Lock()
		etags[key] = etag
		etagLock.Unlock()

		// cache control headers
		resp.Header.Set("Cache-Control", "max-age:604800")
		resp.Header.Set("Expires", time.Now().Add(604800 * time.Second).Format(http.TimeFormat))

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("can't read from b2: %v", err)
		}

		result := &CacheData{
			Headers: resp.Header,
			Body:    body,
		}

		var buf bytes.Buffer
		err = gob.NewEncoder(&buf).Encode(result)
		if err != nil {
			return err
		}

		dest.SetBytes(buf.Bytes())

		return nil
	},
))

var (
	cacheGets = expvar.NewInt("cache_gets")
	cacheHits = expvar.NewInt("cache_hits")
	cacheErrors = expvar.NewInt("cache_errors")
	cacheLoads = expvar.NewInt("cache_loads")

	etagMatches = expvar.NewInt("etag_matches")

	fileHits = expvar.NewMap("file_hits")
	referers = expvar.NewMap("referers")

	etags map[string]string
	etagLock sync.RWMutex
)

func init() {
	etags = map[string]string{}
}

func refreshMetrics () {
	t := time.NewTicker(10 * time.Second)
	defer t.Stop()

	for range t.C {
		cacheGets.Set(Group.Stats.Gets.Get())
		cacheHits.Set(Group.Stats.CacheHits.Get())
		cacheErrors.Set(int64(Group.Stats.LocalLoadErrs))
		cacheLoads.Set(int64(Group.Stats.LocalLoads))
	}
}

func main() {
	internal.HandleStartup()
	ctx := opname.With(context.Background(), "startup")

	go refreshMetrics()

	go func () {
		srv := &tsnet.Server{
			Hostname: "xedn-" + os.Getenv("FLY_REGION"),
			Logf: log.New(io.Discard, "", 0).Printf,
			AuthKey:   os.Getenv("TS_AUTHKEY"),
		}

		lis, err := srv.Listen("tcp", ":80")
		if err != nil {
			ln.FatalErr(ctx, err, ln.Action("tsnet listening"))
		}

		http.DefaultServeMux.HandleFunc("/debug/varz", tsweb.VarzHandler)

		defer srv.Close()
		defer lis.Close()
		ln.FatalErr(opname.With(ctx, "metrics-tsnet"), http.Serve(lis, ex.HTTPLog(http.DefaultServeMux)))
	} ()

	xffMW, err := xff.Default()
	if err != nil {
		ln.FatalErr(ctx, err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/.within/metrics", tsweb.VarzHandler)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Length", strconv.Itoa(len(indexHTML)))
		w.WriteHeader(http.StatusOK)
		w.Write(indexHTML)
	})

	mux.HandleFunc("/file/christine-static/", func(w http.ResponseWriter, r *http.Request) {
		etagLock.RLock()
		etag, ok := etags[r.URL.Path]
		etagLock.RUnlock()

		if r.Header.Get("If-None-Match") == etag && ok {
			etagMatches.Add(1)
			w.WriteHeader(http.StatusNotModified)
			return
		}

		referers.Add(r.Header.Get("Referer"), 1)

		var b []byte
		err := Group.Get(nil, r.URL.Path, groupcache.AllocatingByteSliceSink(&b))
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		var result CacheData
		err = gob.NewDecoder(bytes.NewBuffer(b)).Decode(&result)
		if err != nil {
			ln.Error(r.Context(), err)
			http.Error(w, "internal cache error", http.StatusInternalServerError)
			return
		}

		for k, vs := range result.Headers {
			for _, v := range vs {
				w.Header().Add(k, v)
			}
		}
		w.WriteHeader(http.StatusOK)
		w.Write(result.Body)
	})

	ln.Log(context.Background(), ln.F{"addr": *addr})
	http.ListenAndServe(*addr, xffMW.Handler(ex.HTTPLog(mux)))
}