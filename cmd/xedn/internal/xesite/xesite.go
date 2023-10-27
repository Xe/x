package xesite

import (
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	compressionGZIP = 0x69
)

func init() {
	zip.RegisterCompressor(compressionGZIP, func(w io.Writer) (io.WriteCloser, error) {
		return gzip.NewWriterLevel(w, gzip.BestCompression)
	})
	zip.RegisterDecompressor(compressionGZIP, func(r io.Reader) io.ReadCloser {
		rdr, err := gzip.NewReader(r)
		if err != nil {
			slog.Error("can't read from gzip stream", "err", err)
			panic(err)
		}
		return rdr
	})
}

type ZipServer struct {
	dir  string
	lock sync.RWMutex
	zip  *zip.ReadCloser
}

func NewZipServer(zipPath, dir string) (*ZipServer, error) {
	result := &ZipServer{dir: dir}

	if _, err := os.Stat(zipPath); !os.IsNotExist(err) {
		file, err := zip.OpenReader(zipPath)
		if err != nil {
			return nil, err
		}

		result.zip = file
	}

	return result, nil
}

func (zs *ZipServer) UploadNewZip(w http.ResponseWriter, r *http.Request) {
	fname := fmt.Sprintf("xesite-%s.zip", time.Now().Format("2006-01-02T15-04-05"))
	fpath := filepath.Join(zs.dir, "xesite", fname)
	fout, err := os.Create(fpath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		slog.Error("can't create file", "err", err, "fpath", fpath)
		return
	}
	defer fout.Close()

	if _, err := io.Copy(fout, r.Body); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		slog.Error("can't write file", "err", err, "fpath", fpath)
		return
	}

	if err := fout.Close(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		slog.Error("can't close file", "err", err, "fpath", fpath)
		return
	}

	if err := zs.Update(fpath); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		slog.Error("can't update zip", "err", err, "fpath", fpath)
		return
	}
}

func (zs *ZipServer) Update(fname string) error {
	zs.lock.Lock()
	defer zs.lock.Unlock()

	old := zs.zip

	file, err := zip.OpenReader(fname)
	if err != nil {
		return err
	}

	zs.zip = file

	old.Close()
	return nil
}

func (zs *ZipServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	zs.lock.RLock()
	defer zs.lock.RUnlock()

	if zs.zip == nil {
		http.Error(w, "no zip file", http.StatusNotFound)
		return
	}

	http.FileServer(http.FS(zs.zip)).ServeHTTP(w, r)
}
