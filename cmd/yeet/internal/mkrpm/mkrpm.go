package mkrpm

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"

	"github.com/goreleaser/nfpm/v2"
	"github.com/goreleaser/nfpm/v2/files"
	_ "github.com/goreleaser/nfpm/v2/rpm"
	"within.website/x/cmd/yeet/internal"
)

type Package struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Description string   `json:"description"`
	Homepage    string   `json:"homepage"`
	Group       string   `json:"group"`
	License     string   `json:"license"`
	Goarch      string   `json:"goarch"`
	Replaces    []string `json:"replaces"`
	Depends     []string `json:"depends"`
	Recommends  []string `json:"recommends"`

	EmptyDirs   []string          `json:"emptyDirs"`   // rpm destination path
	ConfigFiles map[string]string `json:"configFiles"` // repo-relative source path, rpm destination path

	Build func(out string) `json:"build"`
}

func Build(p Package) (foutpath string, err error) {
	defer func() {
		if r := recover(); r != nil {
			switch r.(type) {
			case error:
				err = r.(error)
				slog.Error("mkrpm: error while building", "err", err)
			default:
				err = fmt.Errorf("mkrpm: error while building: %v", r)
				slog.Error("mkrpm: error while building", "err", err)
			}
		}
	}()

	if p.Version == "" {
		p.Version = internal.GitVersion()
	}

	dir, err := os.MkdirTemp("", "yeet-mkrpm")
	if err != nil {
		return "", fmt.Errorf("mkrpm: can't make temporary directory")
	}
	defer os.RemoveAll(dir)
	os.MkdirAll(dir, 0755)

	defer func() {
		os.Setenv("GOARCH", runtime.GOARCH)
		os.Setenv("GOOS", runtime.GOOS)
	}()
	os.Setenv("GOARCH", p.Goarch)
	os.Setenv("GOOS", "linux")

	p.Build(dir)

	var contents files.Contents

	for _, d := range p.EmptyDirs {
		if d == "" {
			continue
		}

		contents = append(contents, &files.Content{Type: files.TypeDir, Destination: d})
	}

	for repoPath, rpmPath := range p.ConfigFiles {
		contents = append(contents, &files.Content{Type: files.TypeConfig, Source: repoPath, Destination: rpmPath})
	}

	if err := filepath.Walk(dir, func(path string, stat os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if stat.IsDir() {
			return nil
		}

		contents = append(contents, &files.Content{Type: files.TypeFile, Source: path, Destination: path[len(dir)+1:]})

		return nil
	}); err != nil {
		return "", fmt.Errorf("mkrpm: can't walk output directory: %w", err)
	}

	info := nfpm.WithDefaults(&nfpm.Info{
		Name:        p.Name,
		Version:     p.Version,
		Arch:        p.Goarch,
		Platform:    "linux",
		Description: p.Description,
		Maintainer:  fmt.Sprintf("%s <%s>", *internal.UserName, *internal.UserEmail),
		Homepage:    p.Homepage,
		License:     p.License,
		Overridables: nfpm.Overridables{
			Contents:   contents,
			Depends:    p.Depends,
			Recommends: p.Recommends,
			Replaces:   p.Replaces,
			Conflicts:  p.Replaces,
		},
	})

	info.Overridables.RPM.Group = p.Group

	if *internal.GPGKeyID != "" {
		slog.Debug("using GPG key", "file", *internal.GPGKeyFile, "id", *internal.GPGKeyID, "password", *internal.GPGKeyPassword)
		info.Overridables.RPM.Signature.KeyFile = *internal.GPGKeyFile
		info.Overridables.RPM.Signature.KeyID = *&internal.GPGKeyID
		info.Overridables.RPM.Signature.KeyPassphrase = *internal.GPGKeyPassword
	}

	pkg, err := nfpm.Get("rpm")
	if err != nil {
		return "", fmt.Errorf("mkrpm: can't get RPM packager: %w", err)
	}

	foutpath = pkg.ConventionalFileName(info)
	fout, err := os.Create(foutpath)
	if err != nil {
		return "", fmt.Errorf("mkrpm: can't create output file: %w", err)
	}
	defer fout.Close()

	if err := pkg.Package(info, fout); err != nil {
		return "", fmt.Errorf("mkrpm: can't build package: %w", err)
	}

	slog.Debug("built package", "name", p.Name, "version", p.Version, "path", foutpath)

	return foutpath, err
}
