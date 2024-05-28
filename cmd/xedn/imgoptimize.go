package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	"github.com/gen2brain/avif"
	_ "github.com/gen2brain/heic"
	_ "github.com/gen2brain/jpegxl"
	"github.com/gen2brain/webp"
	"go.etcd.io/bbolt"
	"golang.org/x/sync/singleflight"
	"tailscale.com/metrics"
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
			if err := webp.Encode(&result, dstImg, webp.Options{Quality: 95}); err != nil {
				return nil, err
			}
		case "avif":
			if err := avif.Encode(&result, dstImg, avif.Options{Quality: 95, Speed: 7}); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("i don't know how to render to %s yet, sorry", format)
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
