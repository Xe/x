package mktarball

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"

	"within.website/x/cmd/yeet/internal"
	"within.website/x/cmd/yeet/internal/pkgmeta"
)

func defaultFname(p pkgmeta.Package) string {
	return fmt.Sprintf("%s-%s-%s-%s", p.Name, p.Version, p.Platform, p.Goarch)
}

func Build(p pkgmeta.Package) (foutpath string, err error) {
	defer func() {
		if r := recover(); r != nil {
			if err, ok := r.(error); ok {
				slog.Error("mkrpm: error while building", "err", err)
			} else {
				err = fmt.Errorf("%v", r)
				slog.Error("mkrpm: error while building", "err", err)
			}
		}
	}()

	os.MkdirAll("./var", 0755)
	os.WriteFile(filepath.Join("./var", ".gitignore"), []byte("*\n!.gitignore"), 0644)

	if p.Version == "" {
		p.Version = internal.GitVersion()
	}
	if p.Platform == "" {
		p.Platform = "linux"
	}

	dir, err := os.MkdirTemp("", "yeet-mktarball")
	if err != nil {
		return "", fmt.Errorf("can't make temporary directory")
	}
	defer os.RemoveAll(dir)

	folderName := defaultFname(p)
	if p.Filename != nil {
		folderName = p.Filename(p)
	}

	pkgDir := filepath.Join(dir, folderName)
	os.MkdirAll(pkgDir, 0755)

	fname := filepath.Join("var", folderName+".tar.gz")
	fout, err := os.Create(fname)
	if err != nil {
		return "", fmt.Errorf("can't make output file: %w", err)
	}
	defer fout.Close()

	gw, err := gzip.NewWriterLevel(fout, 9)
	if err != nil {
		return "", fmt.Errorf("can't make gzip writer: %w", err)
	}
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	cgoEnabled := os.Getenv("CGO_ENABLED")
	defer func() {
		os.Setenv("GOARCH", runtime.GOARCH)
		os.Setenv("GOOS", runtime.GOOS)
		os.Setenv("CGO_ENABLED", cgoEnabled)
	}()
	os.Setenv("GOARCH", p.Goarch)
	os.Setenv("GOOS", p.Platform)
	os.Setenv("CGO_ENABLED", "0")

	bi := pkgmeta.BuildInput{
		Output:  pkgDir,
		Bin:     filepath.Join(pkgDir, "bin"),
		Doc:     filepath.Join(pkgDir, "doc"),
		Etc:     filepath.Join(pkgDir, "run"),
		Man:     filepath.Join(pkgDir, "man"),
		Systemd: filepath.Join(pkgDir, "run"),
	}

	os.MkdirAll(bi.Doc, 0755)
	os.WriteFile(filepath.Join(bi.Doc, "VERSION"), []byte(p.Version+"\n"), 0666)

	p.Build(bi)

	for src, dst := range p.ConfigFiles {
		if err := Copy(src, filepath.Join(bi.Etc, dst)); err != nil {
			return "", fmt.Errorf("can't copy %s to %s: %w", src, dst, err)
		}
	}

	for src, dst := range p.Documentation {
		fname := filepath.Join(bi.Doc, dst)
		if filepath.Base(fname) == "README.md" {
			fname = filepath.Join(pkgDir, "README.md")
		}

		if err := Copy(src, fname); err != nil {
			return "", fmt.Errorf("can't copy %s to %s: %w", src, dst, err)
		}
	}

	root, err := os.OpenRoot(dir)
	if err != nil {
		return "", fmt.Errorf("can't open root FS %s: %w", dir, err)
	}

	if err := tw.AddFS(root.FS()); err != nil {
		return "", fmt.Errorf("can't copy built files to tarball: %w", err)
	}

	slog.Info("built package", "name", p.Name, "arch", p.Goarch, "version", p.Version, "path", fout.Name())

	return fname, nil
}

// Copy copies the contents of the file at srcpath to a regular file
// at dstpath. If the file named by dstpath already exists, it is
// truncated. The function does not copy the file mode, file
// permission bits, or file attributes.
func Copy(srcpath, dstpath string) (err error) {
	r, err := os.Open(srcpath)
	if err != nil {
		return err
	}
	defer r.Close() // ignore error: file was opened read-only.

	st, err := r.Stat()
	if err != nil {
		return err
	}

	os.MkdirAll(filepath.Dir(dstpath), 0755)

	w, err := os.Create(dstpath)
	if err != nil {
		return err
	}

	if err := w.Chmod(st.Mode()); err != nil {
		return err
	}

	defer func() {
		// Report the error, if any, from Close, but do so
		// only if there isn't already an outgoing error.
		if c := w.Close(); err == nil {
			err = c
		}
	}()

	_, err = io.Copy(w, r)
	return err
}
