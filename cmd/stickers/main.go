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
	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"within.website/x/htmx"
	"within.website/x/internal"
	"within.website/x/xess"
)

//go:generate go run github.com/a-h/templ/cmd/templ@latest generate

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

	cfg, err := awsConfig.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatal(err)
	}

	s3c := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	presigner := Presigner{s3.NewPresignClient(s3c)}
	_ = presigner

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
			_, err := s3c.HeadObject(r.Context(), &s3.HeadObjectInput{
				Bucket: aws.String(*bucketName),
				Key:    aws.String(key),
			})
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

		req, err := presigner.GetObject(r.Context(), key, 3600)
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

		http.Redirect(w, r, req.URL, http.StatusTemporaryRedirect)
	})

	slog.Info("listening", "bind", *bind)
	log.Fatal(http.ListenAndServe(*bind, mux))
}

// Presigner encapsulates the Amazon Simple Storage Service (Amazon S3) presign actions
// used in the examples.
// It contains PresignClient, a client that is used to presign requests to Amazon S3.
// Presigned requests contain temporary credentials and can be made from any HTTP client.
type Presigner struct {
	PresignClient *s3.PresignClient
}

// GetObject makes a presigned request that can be used to get an object from a bucket.
// The presigned request is valid for the specified number of seconds.
func (presigner Presigner) GetObject(
	ctx context.Context,
	objectKey string,
	lifetimeSecs int64,
) (*v4.PresignedHTTPRequest, error) {
	request, err := presigner.PresignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(*bucketName),
		Key:    aws.String(objectKey),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = time.Duration(lifetimeSecs * int64(time.Second))
	})
	if err != nil {
		log.Printf("Couldn't get a presigned request to get %v:%v. Here's why: %v\n",
			bucketName, objectKey, err)
	}
	return request, err
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
