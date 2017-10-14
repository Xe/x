package main

import (
	"bufio"
	"flag"
	"fmt"
	"net/http"
	"os"
	"runtime/pprof"

	"github.com/Xe/x/tools/glue/libs/gluaexpect"
	"github.com/Xe/x/tools/glue/libs/gluasimplebox"
	"github.com/ailncode/gluaxmlpath"
	"github.com/cjoudrey/gluahttp"
	"github.com/cjoudrey/gluaurl"
	"github.com/kohkimakimoto/gluaenv"
	"github.com/kohkimakimoto/gluafs"
	"github.com/kohkimakimoto/gluaquestion"
	"github.com/kohkimakimoto/gluassh"
	"github.com/kohkimakimoto/gluatemplate"
	"github.com/kohkimakimoto/gluayaml"
	"github.com/otm/gluaflag"
	"github.com/otm/gluash"
	"github.com/yuin/gluare"
	"github.com/yuin/gopher-lua"
	"github.com/yuin/gopher-lua/parse"
	json "layeh.com/gopher-json"
)

func main() {
	os.Exit(mainAux())
}

func mainAux() int {
	var opt_e, opt_l, opt_p string
	var opt_i, opt_v, opt_dt, opt_dc bool
	var opt_m int
	flag.StringVar(&opt_e, "e", "", "")
	flag.StringVar(&opt_l, "l", "", "")
	flag.StringVar(&opt_p, "p", "", "")
	flag.IntVar(&opt_m, "mx", 0, "")
	flag.BoolVar(&opt_i, "i", false, "")
	flag.BoolVar(&opt_v, "v", false, "")
	flag.BoolVar(&opt_dt, "dt", false, "")
	flag.BoolVar(&opt_dc, "dc", false, "")
	flag.Usage = func() {
		fmt.Println(`Usage: glue [options] [script [args]].
Available options are:
  -e stat  execute string 'stat'
  -l name  require library 'name'
  -mx MB   memory limit(default: unlimited)
  -dt      dump AST trees
  -dc      dump VM codes
  -i       enter interactive mode after executing 'script'
  -p file  write cpu profiles to the file
  -v       show version information
`)
	}
	flag.Parse()
	if len(opt_p) != 0 {
		f, err := os.Create(opt_p)
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}
	if len(opt_e) == 0 && !opt_i && !opt_v && flag.NArg() == 0 {
		opt_i = true
	}

	status := 0

	L := lua.NewState()
	defer L.Close()
	if opt_m > 0 {
		L.SetMx(opt_m)
	}

	preload(L)

	if opt_v || opt_i {
		fmt.Println(lua.PackageCopyRight)
	}

	if len(opt_l) > 0 {
		if err := L.DoFile(opt_l); err != nil {
			fmt.Println(err.Error())
		}
	}

	if nargs := flag.NArg(); nargs > 0 {
		script := flag.Arg(0)
		argtb := L.NewTable()
		for i := 1; i < nargs; i++ {
			L.RawSet(argtb, lua.LNumber(i), lua.LString(flag.Arg(i)))
		}
		L.SetGlobal("arg", argtb)
		if opt_dt || opt_dc {
			file, err := os.Open(script)
			if err != nil {
				fmt.Println(err.Error())
				return 1
			}
			chunk, err2 := parse.Parse(file, script)
			if err2 != nil {
				fmt.Println(err2.Error())
				return 1
			}
			if opt_dt {
				fmt.Println(parse.Dump(chunk))
			}
			if opt_dc {
				proto, err3 := lua.Compile(chunk, script)
				if err3 != nil {
					fmt.Println(err3.Error())
					return 1
				}
				fmt.Println(proto.String())
			}
		}

		if err := L.DoFile(script); err != nil {
			fmt.Println(err.Error())
			status = 1
		}
	}

	if len(opt_e) > 0 {
		if err := L.DoString(opt_e); err != nil {
			fmt.Println(err.Error())
			status = 1
		}
	}

	if opt_i {
		doREPL(L)
	}
	return status
}

func preload(L *lua.LState) {
	L.PreloadModule("re", gluare.Loader)
	L.PreloadModule("sh", gluash.Loader)
	L.PreloadModule("fs", gluafs.Loader)
	L.PreloadModule("env", gluaenv.Loader)
	L.PreloadModule("yaml", gluayaml.Loader)
	L.PreloadModule("question", gluaquestion.Loader)
	L.PreloadModule("ssh", gluassh.Loader)
	L.PreloadModule("http", gluahttp.NewHttpModule(&http.Client{}).Loader)
	L.PreloadModule("flag", gluaflag.Loader)
	L.PreloadModule("template", gluatemplate.Loader)
	L.PreloadModule("url", gluaurl.Loader)
	gluaexpect.Preload(L)
	gluasimplebox.Preload(L)
	gluaxmlpath.Preload(L)
	json.Preload(L)
}

// do read/eval/print/loop
func doREPL(L *lua.LState) {
	reader := bufio.NewReader(os.Stdin)
	for {
		if str, err := loadline(reader, L); err == nil {
			if err := L.DoString(str); err != nil {
				fmt.Println(err)
			}
		} else { // error on loadline
			fmt.Println(err)
			return
		}
	}
}

func incomplete(err error) bool {
	if lerr, ok := err.(*lua.ApiError); ok {
		if perr, ok := lerr.Cause.(*parse.Error); ok {
			return perr.Pos.Line == parse.EOF
		}
	}
	return false
}

func loadline(reader *bufio.Reader, L *lua.LState) (string, error) {
	fmt.Print("> ")
	if line, err := reader.ReadString('\n'); err == nil {
		if _, err := L.LoadString("return " + line); err == nil { // try add return <...> then compile
			return line, nil
		} else {
			return multiline(line, reader, L)
		}
	} else {
		return "", err
	}
}

func multiline(ml string, reader *bufio.Reader, L *lua.LState) (string, error) {
	for {
		if _, err := L.LoadString(ml); err == nil { // try compile
			return ml, nil
		} else if !incomplete(err) { // syntax error , but not EOF
			return ml, nil
		} else {
			fmt.Print(">> ")
			if line, err := reader.ReadString('\n'); err == nil {
				ml = ml + "\n" + line
			} else {
				return "", err
			}
		}
	}
}
