package main

import (
	"context"
	"flag"
	"io/fs"
	"math/rand"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"within.website/ln"
	"within.website/ln/opname"
	"within.website/x/internal"
)

var (
	r    wazero.Runtime
	code wazero.CompiledModule

	binary = flag.String("wasm-binary", "./bin.wasm", "binary to run against every line of input from connections")
)

func main() {
	internal.HandleStartup()
	ctx := opname.With(context.Background(), "tensei")

	data, err := os.ReadFile(*binary)
	if err != nil {
		ln.FatalErr(ctx, err)
	}

	r = wazero.NewRuntime(ctx)

	wasi_snapshot_preview1.MustInstantiate(ctx, r)

	code, err = r.CompileModule(ctx, data)
	if err != nil {
		ln.FatalErr(ctx, err)
	}

	name := strconv.Itoa(rand.Int())
	config := wazero.NewModuleConfig().WithStdout(os.Stdout).WithStdin(os.Stdin).WithStderr(os.Stderr).WithArgs("aiyou").WithName(name).WithFS(ConnFS{})

	mod, err := r.InstantiateModule(ctx, code, config)
	if err != nil {
		ln.Error(ctx, err)
		return
	}
	defer mod.Close(ctx)
}

type ConnFS struct{}

func (ConnFS) Open(name string) (fs.File, error) {
	conn, err := net.Dial("tcp", name)
	if err != nil {
		return nil, err
	}

	return ConnFile{Conn: conn}, nil
}

type ConnFile struct {
	net.Conn
}

func (c ConnFile) Stat() (fs.FileInfo, error) {
	return ConnFileInfo{c.Conn}, nil
}

type ConnFileInfo struct {
	conn net.Conn
}

func (c ConnFileInfo) Name() string       { return c.conn.RemoteAddr().String() } // base name of the file
func (c ConnFileInfo) Size() int64        { return 0 }                            // length in bytes for regular files; system-dependent for others
func (c ConnFileInfo) Mode() fs.FileMode  { return 0 }                            // file mode bits
func (c ConnFileInfo) ModTime() time.Time { return time.Now() }                   // modification time
func (c ConnFileInfo) IsDir() bool        { return false }                        // abbreviation for Mode().IsDir()
func (c ConnFileInfo) Sys() any           { return c.conn }                       // underlying data source (can return nil)
