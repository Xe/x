package main

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"time"

	"go.etcd.io/bbolt"
	"golang.org/x/sync/singleflight"
	"within.website/x/web"
)

type Cache struct {
	ActualHost string
	Client     *http.Client
	DB         *bbolt.DB
	cacheGroup *singleflight.Group
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
		slog.Error("can't read files to be purged", "err", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	slog.Info("purging files", "files", files)

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
		slog.Error("can't purge files", "err", err, "files", files)
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

		diesAt := time.Now().AddDate(0, 0, 7).Format(http.TimeFormat)

		if err := bkt.Put([]byte("diesAt"), []byte(diesAt)); err != nil {
			return err
		}

		// cache control headers
		resp.Header.Set("Cache-Control", "max-age:604800") // one week
		resp.Header.Set("Expires", diesAt)

		return nil
	})
}

var ErrNotCached = errors.New("data is not cached")

func (dc *Cache) Load(dir string, w io.Writer) error {
	return dc.DB.Update(func(tx *bbolt.Tx) error {
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

		now := time.Now()

		if t.Before(now) {
			tx.DeleteBucket([]byte(dir))
			fileDeaths.Get(dir).Add(1)
			return ErrNotCached
		}

		if err := bkt.Put([]byte("diesAt"), []byte(now.AddDate(0, 0, 7).Format(http.TimeFormat))); err != nil {
			return err
		}

		h := http.Header{}

		data := bkt.Get([]byte("header"))
		if data == nil {
			return ErrNotCached
		}
		if err := json.Unmarshal(data, &h); err != nil {
			return err
		}

		if h.Get("Content-Type") == "" && filepath.Ext(dir) == ".svg" {
			h.Set("Content-Type", "image/svg+xml")
		}

		data = bkt.Get([]byte("body"))
		if data == nil {
			return ErrNotCached
		}

		if rw, ok := w.(http.ResponseWriter); ok {
			for k, vs := range h {
				for _, v := range vs {
					rw.Header().Add(k, v)
				}
			}
		}

		w.Write(data)
		cacheHits.Add(1)
		fileHits.Add(dir, 1)

		return nil
	})
}

func (dc *Cache) LoadBytesOrFetch(path string) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	err := dc.Load(path, buf)
	if err != nil {
		if err == ErrNotCached {
			_, err, _ := dc.cacheGroup.Do(path, func() (interface{}, error) {
				resp, err := dc.Client.Get(fmt.Sprintf("https://%s%s", dc.ActualHost, path))
				if err != nil {
					cacheErrors.Add(1)
					return nil, err
				}

				if resp.StatusCode != http.StatusOK {
					cacheErrors.Add(1)
					return nil, web.NewError(http.StatusOK, resp)
				}

				err = dc.Save(path, resp)
				if err != nil {
					cacheErrors.Add(1)
					return nil, err
				}

				return nil, nil
			})
			if err != nil {
				return nil, err
			}

			return dc.LoadBytesOrFetch(path)
		}
		return nil, err
	}
	return buf.Bytes(), nil
}

func (dc *Cache) GetFile(w http.ResponseWriter, r *http.Request) error {
	dir := filepath.Join(r.URL.Path)

	err := dc.Load(dir, w)
	if err != nil {
		if err == ErrNotCached {
			_, err, _ := dc.cacheGroup.Do(r.URL.Path, func() (interface{}, error) {
				r.URL.Host = dc.ActualHost
				r.URL.Scheme = "https"
				resp, err := dc.Client.Get(r.URL.String())
				if err != nil {
					cacheErrors.Add(1)
					return nil, err
				}

				if resp.StatusCode != http.StatusOK {
					cacheErrors.Add(1)
					return nil, web.NewError(http.StatusOK, resp)
				}

				err = dc.Save(dir, resp)
				if err != nil {
					cacheErrors.Add(1)
					return nil, err
				}
				cacheLoads.Add(1)
				return nil, nil
			})
			if err != nil {
				return err
			}
		} else {
			cacheErrors.Add(1)
			return err
		}
	}

	return dc.Load(dir, w)
}

func (dc *Cache) CronPurgeDead() {
	lg := slog.Default().With("job", "purgeDead")

	for range time.Tick(30 * time.Minute) {
		lg.Info("starting")

		if err := dc.DB.Update(func(tx *bbolt.Tx) error {
			if err := tx.ForEach(func(name []byte, b *bbolt.Bucket) error {
				if string(name) == "sticker_cache" {
					return nil
				}

				lg := lg.With("path", string(name))
				diesAtBytes := b.Get([]byte("diesAt"))
				if diesAtBytes == nil {
					lg.Error("no diesAt key")
					return nil
				}

				diesAt, err := time.Parse(http.TimeFormat, string(diesAtBytes))
				if err != nil {
					return fmt.Errorf("when parsing diesAt for %s (%q): %w", string(name), string(diesAtBytes), err)
				}

				if diesAt.Before(time.Now()) {
					if err := tx.DeleteBucket(name); err != nil {
						return fmt.Errorf("when trying to delete bucket %s: %w", string(name), err)
					}

					fileDeaths.Add(string(name), 1)
					lg.Info("deleted", "diesAt", diesAt)
				}

				return nil
			}); err != nil {
				return err
			}

			return nil
		}); err != nil {
			lg.Info("can't update database: %v", "err", err)
		}
	}
}
