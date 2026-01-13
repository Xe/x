package python

import (
	"bytes"
	"context"
	_ "embed"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

var (
	//go:embed python.wasm
	Binary []byte

	r    wazero.Runtime
	code wazero.CompiledModule
)

func init() {
	ctx := context.Background()
	r = wazero.NewRuntime(ctx)

	wasi_snapshot_preview1.MustInstantiate(ctx, r)

	var err error
	code, err = r.CompileModule(ctx, Binary)
	if err != nil {
		panic(err)
	}
}

type Result struct {
	Stdout        string
	Stderr        string
	PlatformError string
}

// mainPyPath is the path where main.py is placed in the filesystem.
const mainPyPath = "/.within.website.main.py"

func Run(ctx context.Context, fsys fs.FS, userCode string) (*Result, error) {
	fout := &bytes.Buffer{}
	ferr := &bytes.Buffer{}
	fin := &bytes.Buffer{}

	// If fsys is nil, use an empty filesystem.
	if fsys == nil {
		fsys = emptyFS{}
	}

	// Create a temporary directory and write main.py there.
	tmpDir, err := os.MkdirTemp("", "python-wasm-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	// Write main.py at the special path.
	// The path is /.within.website.main.py (no directory separator).
	if err := os.WriteFile(filepath.Join(tmpDir, ".within.website.main.py"), []byte(userCode), 0644); err != nil {
		return nil, err
	}

	// Mount the temporary directory at root.
	fsConfig := wazero.NewFSConfig().
		WithDirMount(tmpDir, "/")

	config := wazero.NewModuleConfig().
		// stdio
		WithStdout(fout).
		WithStderr(ferr).
		WithStdin(fin).
		// argv
		WithArgs("python", mainPyPath).
		WithName("python").
		// fs / system
		WithFSConfig(fsConfig).
		WithSysNanosleep().
		WithSysNanotime().
		WithSysWalltime()

	mod, err := r.InstantiateModule(ctx, code, config)
	if err != nil {
		result := &Result{
			Stdout:        fout.String(),
			Stderr:        ferr.String(),
			PlatformError: err.Error(),
		}
		return result, err
	}

	defer mod.Close(ctx)

	return &Result{
		Stdout: fout.String(),
		Stderr: ferr.String(),
	}, nil
}

// emptyFS is an empty filesystem that returns ENOENT for all paths.
type emptyFS struct{}

func (emptyFS) Open(name string) (fs.File, error) {
	// Return a more specific error for better debugging.
	return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
}

// globFS implements fs.GlobFS for better compatibility.
func (emptyFS) Glob(pattern string) ([]string, error) {
	return nil, nil
}

// readDirFS implements fs.ReadDirFS for directory listing.
func (emptyFS) ReadDir(name string) ([]fs.DirEntry, error) {
	return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrNotExist}
}

// subFS is a wrapper to make emptyFS implement fs.SubFS.
type subFS struct {
	fs.FS
}

func (s subFS) Sub(dir string) (fs.FS, error) {
	if dir == "." {
		return s, nil
	}
	return nil, &fs.PathError{Op: "sub", Path: dir, Err: fs.ErrNotExist}
}
