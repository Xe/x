package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/chai2010/webp"
	"github.com/disintegration/imaging"
	"github.com/google/uuid"
	"go.etcd.io/bbolt"
	"golang.org/x/sync/singleflight"
	"tailscale.com/metrics"
	"within.website/x/internal/avif"
)

type OptimizedImageServer struct {
	DB     *bbolt.DB
	Cache  *Cache
	PNGEnc *png.Encoder
	group  *singleflight.Group
}

var (
	OISFileConversions = metrics.LabelMap{Label: "format"}
	OISFileHits        = metrics.LabelMap{Label: "path"}

	b2KeyID     = flag.String("b2-key-id", "", "Backblaze B2 application key ID")
	b2KeySecret = flag.String("b2-application-key", "", "Backblaze B2 application secret")

	avifQuality      = flag.Int("avif-quality", 24, "AVIF quality (higher is worse quality)")
	avifEncoderSpeed = flag.Int("avif-encoder-speed", 0, "AVIF encoder speed (higher is faster)")

	jpegQuality = flag.Int("jpeg-quality", 85, "JPEG quality (lower means lower file size)")

	webpQuality = flag.Int("webp-quality", 25, "WEBP quality (higher is worse quality)")
)

func (ois *OptimizedImageServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// /sticker/character/mood/width
	acc := strings.Split(r.Header.Get("Accept"), ",")
	var format = "png"
	for _, acceptFormat := range acc {
		_, theirFormat, ok := strings.Cut(acceptFormat, "image/")
		if !ok {
			continue
		}

		switch theirFormat {
		case "avif", "webp", "png":
			format = theirFormat
		}
	}

	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) != 5 {
		http.Error(w, "usage: /sticker/:character/:mood/:width", http.StatusBadRequest)
		return
	}

	character := pathParts[2]
	mood := pathParts[3]
	widthStr := pathParts[4]

	width, err := strconv.Atoi(widthStr)
	if err != nil {
		http.Error(w, "width must be an int", http.StatusBadRequest)
		return
	}

	if width > 256 {
		http.Error(w, "width must be less than 257", http.StatusBadRequest)
		return
	}

	data, err := ois.ResizeTo(width, character, mood, format)
	if err != nil {
		slog.Error("can't convert image", "err", err)
		http.Error(w, "can't convert image", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/"+format)
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Header().Set("Cache-Control", "max-age:604800")
	w.Header().Set("Expires", time.Now().Add(604800*time.Second).Format(http.TimeFormat))
	w.Header().Set("Etag", fmt.Sprintf(`W/"%s"`, Hash(r.URL.Path)))
	w.WriteHeader(http.StatusOK)
	w.Write(data)
	OISFileHits.Add(r.URL.Path, 1)
}

func (ois *OptimizedImageServer) ResizeTo(widthPixels int, character, mood, format string) ([]byte, error) {
	if widthPixels <= 0 {
		return nil, errors.New("widthPixels must be greater than 0")
	}

	var result bytes.Buffer
	boltPath := []byte(filepath.Join(character, mood, strconv.Itoa(widthPixels), format))

	err := ois.DB.Update(func(tx *bbolt.Tx) error {
		bkt, err := tx.CreateBucketIfNotExists([]byte("sticker_cache"))
		if err != nil {
			return err
		}

		(&result).Write(bkt.Get(boltPath))

		return nil
	})
	if err != nil {
		return nil, err
	}

	if result.Len() != 0 /* because found in boltdb */ {
		return result.Bytes(), nil
	}

	data, err, _ := ois.group.Do(string(boltPath), func() (interface{}, error) {
		// /file/christine-static/stickers/aoi/yawn.png
		path := fmt.Sprintf("/file/christine-static/stickers/%s/%s.png", character, mood)
		data, err := ois.Cache.LoadBytesOrFetch(path)
		if err != nil {
			return nil, fmt.Errorf("can't fetch: %w", err)
		}

		os.WriteFile("foo.png", data, 0666)

		img, _, err := image.Decode(bytes.NewBuffer(data))
		if err != nil {
			return nil, fmt.Errorf("can't decode image: %w", err)
		}

		dstImg := imaging.Resize(img, widthPixels, 0, imaging.Lanczos)

		switch format {
		case "png":
			if err := ois.PNGEnc.Encode(&result, dstImg); err != nil {
				return nil, err
			}
		case "webp":
			if err := webp.Encode(&result, dstImg, &webp.Options{Quality: 95}); err != nil {
				return nil, err
			}
		case "avif":
			if err := avif.Encode(&result, dstImg, &avif.Options{Quality: 48, Speed: avif.MaxSpeed}); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("I don't know how to render to %s yet, sorry", format)
		}

		OISFileConversions.Add(format, 1)

		err = ois.DB.Update(func(tx *bbolt.Tx) error {
			bkt, err := tx.CreateBucketIfNotExists([]byte("sticker_cache"))
			if err != nil {
				return err
			}

			if err := bkt.Put(boltPath, result.Bytes()); err != nil {
				return err
			}

			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("can't write to database: %w", err)
		}

		return result.Bytes(), nil
	})
	if err != nil {
		return nil, err
	}

	return data.([]byte), nil
}

func (ois *OptimizedImageServer) ListFiles(w http.ResponseWriter, r *http.Request) {
	var data []string

	err := ois.DB.Update(func(tx *bbolt.Tx) error {
		bkt, err := tx.CreateBucketIfNotExists([]byte("sticker_cache"))
		if err != nil {
			return err
		}

		err = bkt.ForEach(func(key, _ []byte) error {
			data = append(data, string(key))
			return nil
		})
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(data)
}

func (ois *OptimizedImageServer) Purge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "must POST", http.StatusMethodNotAllowed)
		return
	}

	var data []string

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "must be JSON", http.StatusBadRequest)
		return
	}

	err := ois.DB.Update(func(tx *bbolt.Tx) error {
		bkt, err := tx.CreateBucketIfNotExists([]byte("sticker_cache"))
		if err != nil {
			return err
		}

		for _, key := range data {
			if err := bkt.Delete([]byte(key)); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

type ImageUploader struct {
	s3 *s3.Client
}

func (iu *ImageUploader) CreateImage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := uuid.New().String()

	// Ten megabytes should be enough for anybody
	reader := http.MaxBytesReader(w, r.Body, 1024*1024*10)

	buf := bytes.NewBuffer(make([]byte, r.ContentLength))

	if _, err := io.Copy(buf, reader); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		slog.Error("cannot copy image to buffer", "err", err)
		return
	}

	img, _, err := image.Decode(buf)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		slog.Error("cannot decode image", "err", err)
		return
	}

	directory, err := os.MkdirTemp(*staticDir, "uploud")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		slog.Error("cannot create temp directory", "err", err)
		return
	}
	defer os.RemoveAll(directory)

	if err := doAVIF(img, filepath.Join(directory, "image.avif")); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		slog.Error("cannot encode AVIF", "err", err)
		return
	}

	if err := doWEBP(img, filepath.Join(directory, "image.webp")); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		slog.Error("cannot encode WEBP", "err", err)
		return
	}

	if err := doJPEG(img, filepath.Join(directory, "image.jpg")); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		slog.Error("cannot encode JPEG", "err", err)
		return
	}

	files, err := os.ReadDir(directory)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		slog.Error("cannot read directory", "err", err)
		return
	}

	s3c := mkS3Client()

	for _, finfo := range files {
		log.Printf("uploading %s", finfo.Name())
		fin, err := os.Open(filepath.Join(directory, finfo.Name()))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			slog.Error("cannot read file", "err", err)
			return
		}
		defer fin.Close()

		_, err = s3c.PutObject(ctx, &s3.PutObjectInput{
			Body:        fin,
			Bucket:      b2Backend,
			Key:         aws.String("xedn/dynamic/" + id + "/" + finfo.Name()),
			ContentType: aws.String(mimeTypes[filepath.Ext(finfo.Name())]),
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			slog.Error("cannot upload file", "err", err)
			return
		}
	}

	json.NewEncoder(w).Encode(map[string]string{
		"avif": "https://cdn.xeiaso.net/file/christine-static/xedn/dynamic/" + id + "/image.avif",
		"webp": "https://cdn.xeiaso.net/file/christine-static/xedn/dynamic/" + id + "/image.webp",
		"jpeg": "https://cdn.xeiaso.net/file/christine-static/xedn/dynamic/" + id + "/image.jpg",
	})
}

var mimeTypes = map[string]string{
	".avif": "image/avif",
	".webp": "image/webp",
	".jpg":  "image/jpeg",
	".png":  "image/png",
	".wasm": "application/wasm",
	".css":  "text/css",
}

func mkS3Client() *s3.Client {
	s3Config := aws.Config{
		Credentials:  credentials.NewStaticCredentialsProvider(*b2KeyID, *b2KeySecret, ""),
		BaseEndpoint: aws.String("https://s3.us-west-001.backblazeb2.com"),
		Region:       "us-west-001",
	}
	s3Client := s3.NewFromConfig(s3Config, (func(o *s3.Options) {
		o.UsePathStyle = true
	}))
	return s3Client
}

func doAVIF(src image.Image, dstPath string) error {
	dst, err := os.Create(dstPath)
	if err != nil {
		log.Fatalf("Can't create destination file: %v", err)
	}
	defer dst.Close()

	err = avif.Encode(dst, src, &avif.Options{
		Threads: runtime.GOMAXPROCS(0),
		Speed:   *avifEncoderSpeed,
		Quality: *avifQuality,
	})
	if err != nil {
		return err
	}

	log.Printf("Encoded AVIF at %s", dstPath)

	return nil
}

func doWEBP(src image.Image, dstPath string) error {
	fout, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer fout.Close()

	err = webp.Encode(fout, src, &webp.Options{Quality: float32(*webpQuality)})
	if err != nil {
		return err
	}

	log.Printf("Encoded WEBP at %s", dstPath)

	return nil
}

func doJPEG(src image.Image, dstPath string) error {
	fout, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer fout.Close()

	if err := jpeg.Encode(fout, src, &jpeg.Options{Quality: *jpegQuality}); err != nil {
		return err
	}

	log.Printf("Encoded JPEG at %s", dstPath)

	return nil
}
