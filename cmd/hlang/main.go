package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"within.website/x/internal"
)

var _ = func() error { log.SetFlags(log.LstdFlags | log.Llongfile); return nil }()

var (
	program      = flag.String("p", "", "h program to compile/run")
	outFname     = flag.String("o", "", "if specified, write the webassembly binary created by -p here")
	watFname     = flag.String("o-wat", "", "if specified, write the uncompiled webassembly created by -p here")
	port         = flag.String("port", "", "HTTP port to listen on")
	sockpath     = flag.String("sockpath", "", "Unix domain socket to listen on")
	writeTao     = flag.Bool("koan", false, "if true, print the h koan and then exit")
	writeVersion = flag.Bool("v", false, "if true, print the version of h and then exit")
)

const koan = `And Jesus said unto the theologians, "Who do you say that I am?"

They replied: "You are the eschatological manifestation of the ground of our
being, the kerygma of which we find the ultimate meaning in our interpersonal
relationships."

And Jesus said "...What?"

Some time passed and one of them spoke "h".

Jesus was enlightened.`

func tao() {
	fmt.Println(koan)
	os.Exit(0)
}

func oneOff() error {
	log.Println("compiling...")
	comp, err := compile(*program)
	if err != nil {
		return err
	}

	log.Println("running...")
	er, err := run(comp.Binary)
	if err != nil {
		return err
	}

	log.Println("success!")

	log.Printf("gas used:\t%d", er.GasUsed)
	log.Printf("exec time:\t%s", er.ExecTime)
	log.Println("output:")
	fmt.Print(er.Output)

	if *outFname != "" {
		err := ioutil.WriteFile(*outFname, comp.Binary, 0666)
		if err != nil {
			return err
		}

		log.Printf("wrote %d bytes to %s", len(comp.Binary), *outFname)
	}

	if *watFname != "" {
		err := ioutil.WriteFile(*watFname, []byte(comp.WebAssemblyText), 0666)
		if err != nil {
			return err
		}

		log.Printf("write %d bytes of source to %s", len(comp.WebAssemblyText), *watFname)
	}

	return nil
}

func main() {
	internal.HandleStartup()

	if *writeVersion {
		dumpVersion()
	}

	if *writeTao {
		tao()
	}

	if *program != "" {
		err := oneOff()
		if err != nil {
			panic(err)
		}

		return
	}

	if *port != "" || *sockpath != "" {
		err := doHTTP()
		if err != nil {
			panic(err)
		}

		return
	}
}

const wasmTemplate = `(module
 (import "h" "h" (func $h (param i32)))
 (func $h_main
       (local i32 i32 i32)
       (local.set 0 (i32.const 10))
       (local.set 1 (i32.const 104))
       (local.set 2 (i32.const 39))
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
 (export "h" (func $h_main))
)`
