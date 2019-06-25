package main

import (
	"flag"
	"fmt"
	"log"

	"within.website/x/internal"
)

var (
	program = flag.String("p", "h", "h program to compile/run")
)

func main() {
	internal.HandleStartup()

	log.Println("compiling...")
	comp, err := compile(*program)
	if err != nil {
		panic(err)
	}

	log.Println("running...")
	er, err := run(*comp)
	if err != nil {
		panic(err)
	}

	log.Println("success!")

	log.Printf("gas used:\t%d", er.GasUsed)
	log.Printf("exec time:\t%s", er.ExecTime)
	log.Println("output:")
	fmt.Println(er.Output)
}

const wasmTemplate = `(module
 (import "h" "h" (func $h (param i32)))
 (func $h_main
       (local i32 i32 i32)
       (local.set 0 (i32.const 10))
       (local.set 1 (i32.const 104))
       (local.set 2 (i32.const 37))
       {{ range . -}}
       {{ if eq . 32 -}}
       (call $h (get_local 0))
       {{ end -}}
       {{ if eq . 104 -}}
       (call $h (get_local 1))
       {{ end -}}
       {{ if eq . 39 -}}
       (call $h (get_local 2))
       {{ end -}}
       {{ end -}}
       (call $h (get_local 0))
 )
 (export "main" (func $h_main))
)`
