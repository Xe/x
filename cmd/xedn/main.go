// Command quiche is a little cute cache server for my B2 bucket.
package main

import (
	"context"
	"crypto/md5"
	_ "embed"
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
	"strconv"
	"sync"
	"time"

	"github.com/rs/cors"
	"github.com/sebest/xff"
	"go.etcd.io/bbolt"
	"tailscale.com/tsnet"
	"tailscale.com/tsweb"
	"within.website/ln"
	"within.website/ln/ex"
	"within.website/ln/opname"
	"within.website/x/internal"
	"within.website/x/web"
)

var (
	b2Backend = flag.String("b2-backend", "f001.backblazeb2.com", "Backblaze B2 base host")
	addr      = flag.String("addr", ":8080", "server address")
	dir       = flag.String("dir", os.Getenv("XEDN_STATE"), "where XeDN should store cached data")

	//go:embed index.html
	indexHTML []byte
)

type Cache struct {
	ActualHost string
	Client     *http.Client
	DB         *bbolt.DB
}

func Hash(data string) string {
	output := md5.Sum([]byte(data))
	return fmt.Sprintf("%x", output)
}

func (dc *Cache) ListFiles(w http.ResponseWriter, r *http.Request) {
	var result []string

	err := dc.DB.View(func(tx *bbolt.Tx) error {
		return tx.ForEach(func(name []byte, b *bbolt.Bucket) error {
			result = append(result, string(name))
			return nil
		})
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(result)
}

func (dc *Cache) Purge(w http.ResponseWriter, r *http.Request) {
	var files []string

	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&files); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := dc.DB.Update(func(tx *bbolt.Tx) error {
		for _, fname := range files {
			bkt := tx.Bucket([]byte(fname))
			if bkt == nil {
				continue
			}

			if err := tx.DeleteBucket([]byte(fname)); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
}

func (dc *Cache) Save(dir string, resp *http.Response) error {
	return dc.DB.Update(func(tx *bbolt.Tx) error {
		bkt, err := tx.CreateBucketIfNotExists([]byte(dir))
		if err != nil {
			return err
		}

		etag := fmt.Sprintf("%q", resp.Header.Get("x-bz-content-sha1"))
		resp.Header.Set("ETag", etag)
		etagLock.Lock()
		etags[dir] = etag
		etagLock.Unlock()

		data, err := json.Marshal(resp.Header)
		if err != nil {
			return err
		}

		if err := bkt.Put([]byte("header"), data); err != nil {
			return err
		}

		data, err = io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		if err := bkt.Put([]byte("body"), data); err != nil {
			return err
		}

		diesAt := time.Now().Add(604800 * time.Second).Format(http.TimeFormat)

		if err := bkt.Put([]byte("diesAt"), []byte(diesAt)); err != nil {
			return err
		}

		// cache control headers
		resp.Header.Set("Cache-Control", "max-age:604800")
		resp.Header.Set("Expires", diesAt)

		return nil
	})
}

var ErrNotCached = errors.New("data is not cached")

func (dc *Cache) Load(dir string, w http.ResponseWriter) error {
	return dc.DB.View(func(tx *bbolt.Tx) error {
		bkt := tx.Bucket([]byte(dir))
		if bkt == nil {
			return ErrNotCached
		}

		diesAtBytes := bkt.Get([]byte("diesAt"))
		if diesAtBytes == nil {
			return ErrNotCached
		}

		t, err := time.Parse(http.TimeFormat, string(diesAtBytes))
		if err != nil {
			return err
		}

		if t.Before(time.Now()) {
			tx.DeleteBucket([]byte(dir))
			return ErrNotCached
		}

		h := http.Header{}

		data := bkt.Get([]byte("header"))
		if data == nil {
			return ErrNotCached
		}
		if err := json.Unmarshal(data, &h); err != nil {
			return err
		}

		data = bkt.Get([]byte("body"))
		if data == nil {
			return ErrNotCached
		}

		for k, vs := range h {
			for _, v := range vs {
				w.Header().Add(k, v)
			}
		}

		w.Write(data)
		cacheHits.Add(1)
		fileHits.Add(dir, 1)

		return nil
	})
}

func (dc *Cache) GetFile(w http.ResponseWriter, r *http.Request) error {
	dir := filepath.Join(r.URL.Path)

	err := dc.Load(dir, w)
	if err != nil {
		if err == ErrNotCached {
			r.URL.Host = dc.ActualHost
			r.URL.Scheme = "https"
			resp, err := dc.Client.Get(r.URL.String())
			if err != nil {
				cacheErrors.Add(1)
				return err
			}

			if resp.StatusCode != http.StatusOK {
				cacheErrors.Add(1)
				return web.NewError(http.StatusOK, resp)
			}

			err = dc.Save(dir, resp)
			if err != nil {
				cacheErrors.Add(1)
				return err
			}

			cacheLoads.Add(1)
		} else {
			cacheErrors.Add(1)
			return err
		}
	}

	return dc.Load(dir, w)
}

func (dc *Cache) ServeHTTP(w http.ResponseWriter, r *http.Request) {
}

var (
	cacheHits   = expvar.NewInt("cache_hits")
	cacheErrors = expvar.NewInt("cache_errors")
	cacheLoads  = expvar.NewInt("cache_loads")

	etagMatches = expvar.NewInt("etag_matches")

	referers = expvar.NewMap("referers")
	fileHits = expvar.NewMap("filehits")

	etags    map[string]string
	etagLock sync.RWMutex
)

func init() {
	etags = map[string]string{}
}

func main() {
	internal.HandleStartup()
	ctx := opname.With(context.Background(), "startup")

	os.MkdirAll(filepath.Join(*dir, "tsnet"), 0700)

	db, err := bbolt.Open(filepath.Join(*dir, "data"), 0600, &bbolt.Options{})
	if err != nil {
		ln.FatalErr(ctx, err)
	}

	dc := &Cache{
		ActualHost: *b2Backend,
		Client:     &http.Client{},
		DB:         db,
	}

	go func() {
		srv := &tsnet.Server{
			Hostname: "xedn-" + os.Getenv("FLY_REGION"),
			Logf:     log.New(io.Discard, "", 0).Printf,
			AuthKey:  os.Getenv("TS_AUTHKEY"),
			Dir:      filepath.Join(*dir, "tsnet"),
		}

		lis, err := srv.Listen("tcp", ":80")
		if err != nil {
			ln.FatalErr(ctx, err, ln.Action("tsnet listening"))
		}

		http.DefaultServeMux.HandleFunc("/debug/varz", tsweb.VarzHandler)
		http.DefaultServeMux.HandleFunc("/xedn/files", dc.ListFiles)
		http.DefaultServeMux.HandleFunc("/xedn/purge", dc.Purge)

		defer srv.Close()
		defer lis.Close()
		ln.FatalErr(opname.With(ctx, "metrics-tsnet"), http.Serve(lis, ex.HTTPLog(http.DefaultServeMux)))
	}()

	xffMW, err := xff.Default()
	if err != nil {
		ln.FatalErr(ctx, err)
	}

	os.MkdirAll(*dir, 0700)

	mux := http.NewServeMux()
	mux.HandleFunc("/.within/metrics", tsweb.VarzHandler)
	mux.Handle("/.within/metrics/json", expvar.Handler())

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Length", strconv.Itoa(len(indexHTML)))
		w.WriteHeader(http.StatusOK)
		w.Write(indexHTML)
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

		referers.Add(r.Header.Get("Referer"), 1)

		if err := dc.GetFile(w, r); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	mux.HandleFunc("/file/christine-static/", hdlr)
	mux.HandleFunc("/file/xeserv-akko/", hdlr)

	ln.Log(context.Background(), ln.F{"addr": *addr})
	http.ListenAndServe(*addr, cors.Default().Handler(xffMW.Handler(ex.HTTPLog(mux))))
}
