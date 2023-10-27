package xesite

import (
	"archive/zip"
	"compress/gzip"
	"io"
	"log/slog"
	"net/http"
	"os"
	"sync"
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
	lock sync.RWMutex
	zip  *zip.ReadCloser
}

func NewZipServer(zipPath string) (*ZipServer, error) {
	result := &ZipServer{}

	if _, err := os.Stat(zipPath); !os.IsNotExist(err) {
		file, err := zip.OpenReader(zipPath)
		if err != nil {
			return nil, err
		}

		result.zip = file
	}

	return result, nil
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
