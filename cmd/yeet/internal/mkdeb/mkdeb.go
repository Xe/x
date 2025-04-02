package mkdeb

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"

	"github.com/goreleaser/nfpm/v2"
	_ "github.com/goreleaser/nfpm/v2/deb"
	"github.com/goreleaser/nfpm/v2/files"
	"within.website/x/cmd/yeet/internal"
	"within.website/x/cmd/yeet/internal/pkgmeta"
)

func Build(p pkgmeta.Package) (foutpath string, err error) {
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

	os.MkdirAll("./var", 0755)
	os.WriteFile(filepath.Join("./var", ".gitignore"), []byte("*\n!.gitignore"), 0644)

	if p.Version == "" {
		p.Version = internal.GitVersion()
	}

	dir, err := os.MkdirTemp("", "yeet-mkdeb")
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

	p.Build(pkgmeta.BuildInput{
		Output:  dir,
		Bin:     filepath.Join(dir, "usr", "bin"),
		Doc:     filepath.Join(dir, "usr", "share", "doc"),
		Etc:     filepath.Join(dir, "etc", p.Name),
		Man:     filepath.Join(dir, "usr", "share", "man"),
		Systemd: filepath.Join(dir, "usr", "lib", "systemd", "system"),
	})

	var contents files.Contents

	for _, d := range p.EmptyDirs {
		if d == "" {
			continue
		}

		contents = append(contents, &files.Content{Type: files.TypeDir, Destination: d})
	}

	for repoPath, osPath := range p.ConfigFiles {
		contents = append(contents, &files.Content{
			Type:        files.TypeConfig,
			Source:      repoPath,
			Destination: osPath,
		})
	}

	for repoPath, rpmPath := range p.Documentation {
		contents = append(contents, &files.Content{
			Type:        files.TypeFile,
			Source:      repoPath,
			Destination: filepath.Join("/usr/share/doc", p.Name, rpmPath),
		})
	}

	for repoPath, rpmPath := range p.Files {
		contents = append(contents, &files.Content{
			Type:        files.TypeFile,
			Source:      repoPath,
			Destination: rpmPath,
		})
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
		return "", fmt.Errorf("mkdeb: can't walk output directory: %w", err)
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
		slog.Debug("using GPG key", "file", *internal.GPGKeyFile, "id", *internal.GPGKeyID)
		info.Overridables.Deb.Signature.KeyFile = *internal.GPGKeyFile
		info.Overridables.Deb.Signature.KeyID = internal.GPGKeyID
		info.Overridables.Deb.Signature.KeyPassphrase = *internal.GPGKeyPassword
	}

	pkg, err := nfpm.Get("deb")
	if err != nil {
		return "", fmt.Errorf("mkdeb: can't get RPM packager: %w", err)
	}

	foutpath = pkg.ConventionalFileName(info)
	fout, err := os.Create(filepath.Join("./var", foutpath))
	if err != nil {
		return "", fmt.Errorf("mkdeb: can't create output file: %w", err)
	}
	defer fout.Close()

	if err := pkg.Package(info, fout); err != nil {
		return "", fmt.Errorf("mkdeb: can't build package: %w", err)
	}

	slog.Info("built package", "name", p.Name, "arch", p.Goarch, "version", p.Version, "path", fout.Name())

	return foutpath, err
}
