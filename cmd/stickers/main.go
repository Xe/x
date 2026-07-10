package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/a-h/templ"
	simplestorage "github.com/tigrisdata/storage-go/simplestorage"
	"within.website/x/htmx"
	"within.website/x/internal"
	"within.website/x/xess"
)

//go:generate go tool templ generate

var (
	bind       = flag.String("bind", ":3923", "TCP address to bind to")
	bucketName = flag.String("bucket-name", "xedn", "Bucket to pull from")
	folderName = flag.String("folder-name", "stickers", "Folder name for stickers")

	//go:embed data/*
	static embed.FS

	cache     = map[string]struct{}{}
	cacheLock = sync.RWMutex{}
)

func main() {
	internal.HandleStartup()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sc, err := simplestorage.New(ctx,
		simplestorage.WithBucket(*bucketName),
		simplestorage.WithFlyEndpoint(),
	)
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	htmx.Mount(mux)
	xess.Mount(mux)

	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		character := r.FormValue("character")
		mood := r.FormValue("mood")

		templ.Handler(
			xess.Simple("Sticker Tester", index(character, mood)),
		).ServeHTTP(w, r)
	})

	mux.HandleFunc("GET /sticker/{name}/{mood}", func(w http.ResponseWriter, r *http.Request) {
		acc := strings.Split(r.Header.Get("Accept"), ",")

		name := r.PathValue("name")
		mood := r.PathValue("mood")

		format := "png"
		for _, acceptFormat := range acc {
			_, theirFormat, ok := strings.Cut(acceptFormat, "image/")
			if !ok {
				continue
			}

			switch theirFormat {
			case "avif", "webp":
				format = theirFormat
			}
		}

		key := fmt.Sprintf("%s/%s/%s.%s", *folderName, name, mood, format)

		if !inCache(name, mood) {
			_, err := sc.Head(r.Context(), key)
			if err != nil {
				slog.Error("can't head key", "format", format, "bucket", *bucketName, "key", key, "err", err)

				st, err := fs.Stat(static, "data/not_found."+format)
				if err != nil {
					http.Error(w, "internal stat error, sorry", http.StatusInternalServerError)
					return
				}

				fin, err := static.Open("data/not_found." + format)
				if err != nil {
					http.Error(w, "internal fopen error, sorry", http.StatusInternalServerError)
					return
				}
				defer fin.Close()

				w.Header().Add("Content-Length", fmt.Sprint(st.Size()))
				w.Header().Add("Content-Type", "image/"+format)
				w.WriteHeader(http.StatusNotFound)

				io.Copy(w, fin)

				return
			}

			setCache(name, mood)
		}

		url, err := sc.PresignURL(r.Context(), http.MethodGet, key, time.Hour)
		if err != nil {
			slog.Error("can't presign get for key", "format", format, "bucket", *bucketName, "key", key, "err", err)

			st, err := fs.Stat(static, "data/error."+format)
			if err != nil {
				http.Error(w, "internal stat error, sorry", http.StatusInternalServerError)
				return
			}

			fin, err := static.Open("data/error." + format)
			if err != nil {
				http.Error(w, "internal fopen error, sorry", http.StatusInternalServerError)
				return
			}
			defer fin.Close()

			w.Header().Add("Content-Length", fmt.Sprint(st.Size()))
			w.Header().Add("Content-Type", "image/"+format)
			w.WriteHeader(http.StatusInternalServerError)

			io.Copy(w, fin)

			return
		}

		w.Header().Add("Cache-Control", "max-age=3599")
		w.Header().Add("Expires", time.Now().Add(time.Hour-time.Second).Format(http.TimeFormat))

		brandedURL := strings.ReplaceAll(url, "xedn.fly.storage.tigris.dev", "files.xeiaso.net")

		http.Redirect(w, r, brandedURL, http.StatusTemporaryRedirect)
	})

	slog.Info("listening", "bind", *bind)
	log.Fatal(http.ListenAndServe(*bind, mux))
}

func inCache(character, mood string) bool {
	cacheLock.RLock()
	defer cacheLock.RUnlock()

	_, ok := cache[character+" "+mood]
	return ok
}

func setCache(character, mood string) {
	cacheLock.Lock()
	defer cacheLock.Unlock()

	cache[character+" "+mood] = struct{}{}
}
