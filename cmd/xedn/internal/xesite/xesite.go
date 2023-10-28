package xesite

import (
	"archive/zip"
	"compress/gzip"
	"encoding/json"
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

func (zs *ZipServer) NukeGeneration(w http.ResponseWriter, r *http.Request) {
	gen := r.URL.Query().Get("gen")

	os.Remove(filepath.Join(zs.dir, "xesite", gen))

	fmt.Fprintln(w, "removed", gen)
}

func (zs *ZipServer) ListGenerations(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// list files in zs.dir/xesite
	var files []string

	dirEntries, err := os.ReadDir(filepath.Join(zs.dir, "xesite"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		slog.Error("can't read dir", "err", err, "dir", filepath.Join(zs.dir, "xesite"))
		return
	}

	for _, dirEntry := range dirEntries {
		if dirEntry.IsDir() {
			continue
		}

		if dirEntry.Name() == "latest.zip" {
			continue
		}

		files = append(files, dirEntry.Name())
	}

	json.NewEncoder(w).Encode(files)
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

	os.Link(fpath, filepath.Join(zs.dir, "xesite", "latest.zip"))
	if err := deleteOldestFileInFolder(filepath.Join(zs.dir, "xesite")); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		slog.Error("can't delete oldest file", "err", err)
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

	if old != nil {
		old.Close()
	}

	slog.Info("activated new generation", "fname", fname)
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

func deleteOldestFileInFolder(folderPath string) error {
	// Read the files in the folder
	files, err := os.ReadDir(folderPath)
	if err != nil {
		return err
	}

	if len(files) == 0 {
		slog.Debug("no files in the folder")
		return nil
	}

	if len(files) < 5 {
		slog.Debug("less than 5 files in the folder")
		return nil
	}

	// Initialize variables to keep track of the oldest file
	var oldestFile os.DirEntry
	var oldestTime time.Time

	// Find the oldest file in the folder
	for _, file := range files {
		fileInfo, err := file.Info()
		if err != nil {
			return err
		}
		if oldestFile == nil || fileInfo.ModTime().Before(oldestTime) {
			oldestFile = file
			oldestTime = fileInfo.ModTime()
		}
	}

	// Delete the oldest file
	oldestFilePath := filepath.Join(folderPath, oldestFile.Name())
	err = os.Remove(oldestFilePath)
	if err != nil {
		return err
	}

	slog.Info("deleted oldest generation", "path", oldestFilePath)

	return nil
}
