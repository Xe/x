package main

import (
	"bufio"
	"bytes"
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"math/rand"
	"net"
	"os"
	"strconv"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"within.website/x/internal"
)

var (
	r    wazero.Runtime
	code wazero.CompiledModule

	binary = flag.String("wasm-binary", "./bin.wasm", "binary to run against every line of input from connections")
	bind   = flag.String("bind", ":1997", "TCP host:port to bind on")
)

func main() {
	internal.HandleStartup()
	ctx := context.Background()

	data, err := os.ReadFile(*binary)
	if err != nil {
		log.Fatal(err)
	}

	r = wazero.NewRuntime(ctx)

	wasi_snapshot_preview1.MustInstantiate(ctx, r)

	code, err = r.CompileModule(ctx, data)
	if err != nil {
		log.Fatal(err)
	}

	server, err := net.Listen("tcp", *bind)
	if err != nil {
		log.Fatal(err)
	}

	slog.Info("listening", "bind", *bind)

	for {
		conn, err := server.Accept()
		if err != nil {
			log.Println("Failed to accept conn.", err)
			continue
		}

		fmt.Println(conn.RemoteAddr().String())

		go func(conn net.Conn) {
			defer func() {
				fmt.Println("disconnect")
				conn.Close()
			}()

			scn := bufio.NewScanner(conn)
			scn.Split(bufio.ScanLines)

			for scn.Scan() {
				fout := &bytes.Buffer{}
				fin := bytes.NewBuffer(scn.Bytes())

				fmt.Println("<", fin.String())

				name := strconv.Itoa(rand.Int())
				config := wazero.NewModuleConfig().WithStdout(fout).WithStdin(fin).WithArgs("mastosan").WithName(name)

				mod, err := r.InstantiateModule(ctx, code, config)
				if err != nil {
					slog.Error("can't instantiate module", "err", err, "remote_host", conn.RemoteAddr().String())
					return
				}
				defer mod.Close(ctx)

				fmt.Print(">", fout.String())

				conn.Write(fout.Bytes())
				conn.Close()
			}
		}(conn)
	}
}
