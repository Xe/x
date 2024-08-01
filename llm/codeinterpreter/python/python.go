package python

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"os"

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
	Stdout string
	Stderr string
}

func Run(ctx context.Context, tmpDir, userCode string) (*Result, error) {
	fout := &bytes.Buffer{}
	ferr := &bytes.Buffer{}
	fin := &bytes.Buffer{}

	os.WriteFile(tmpDir+"/main.py", []byte(userCode), 0644)

	fsConfig := wazero.NewFSConfig().
		WithFSMount(os.DirFS(tmpDir), "/")

	config := wazero.NewModuleConfig().
		// stdio
		WithStdout(fout).
		WithStderr(ferr).
		WithStdin(fin).
		// argv
		WithArgs("python", "/main.py").
		WithName("python").
		// fs / system
		WithFSConfig(fsConfig).
		WithSysNanosleep().
		WithSysNanotime().
		WithSysWalltime()

	mod, err := r.InstantiateModule(ctx, code, config)
	if err != nil {
		fmt.Println(fout.String())
		fmt.Println(ferr.String())
		return nil, err
	}

	defer mod.Close(ctx)

	return &Result{
		Stdout: fout.String(),
		Stderr: ferr.String(),
	}, nil
}
