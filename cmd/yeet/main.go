package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"

	"github.com/dop251/goja"
	"golang.org/x/exp/slog"
	"within.website/x/internal"
	"within.website/x/internal/appsluggr"
	"within.website/x/internal/kahless"
	"within.website/x/internal/yeet"
	"within.website/x/writer"
)

var (
	fname  = flag.String("fname", "yeetfile.js", "filename for the yeetfile")
	flyctl = flag.String("flyctl-path", flyctlPath(), "path to flyctl binary")
)

func flyctlPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "flyctl"
	}

	homedirPath := filepath.Join(home, ".fly", "bin", "fly")

	if _, err := os.Stat(homedirPath); err != nil {
		return "flyctl"
	}

	return homedirPath
}

func runcmd(cmdName string, args ...string) string {
	ctx := context.Background()

	result, err := yeet.Output(ctx, cmdName, args...)
	if err != nil {
		panic(err)
	}

	return result
}

func gittag() string {
	ctx := context.Background()

	tag, err := yeet.GitTag(ctx)
	if err != nil {
		panic(err)
	}

	return tag
}

func dockerload(fname string) {
	if fname == "" {
		fname = "./result"
	}
	yeet.DockerLoadResult(context.Background(), fname)
}

func dockertag(org, repo, image string) string {
	return yeet.DockerTag(context.Background(), org, repo, image)
}

func dockerbuild(tag string, args ...string) {
	yeet.DockerBuild(context.Background(), yeet.WD, tag, args...)
}

func dockerpush(image string) {
	yeet.DockerPush(context.Background(), image)
}

func flydeploy() {
	runcmd(*flyctl, "deploy", "--now")
}

func nixbuild(target string) {
	runcmd("nix", "build", target)
}

func slugbuild(bin string, extraFiles map[string]string) {
	appsluggr.Must(bin, fmt.Sprintf("%s-%s.tar.gz", bin, yeet.DateTag), extraFiles)
	os.Remove(bin)
}

func slugpush(bin string) string {
	fname := fmt.Sprintf("%s-%s.tar.gz", bin, yeet.DateTag)
	fin, err := os.Open(fname)
	if err != nil {
		panic(err)
	}
	defer fin.Close()
	pubURL, err := kahless.CopySlug(fname, fin)
	if err != nil {
		panic(err)
	}

	os.Remove(fname)

	return pubURL
}

func buildNixExpr(literals []string, exprs ...any) string {
	/*
		function nix(strings, ...expressions) {
		    let result = "";
		    expressions.forEach((value, i) => {
			let formattedValue = `(builtins.fromJSON ${JSON.stringify(JSON.stringify(value))});`;
			result += `${strings[i]} ${formattedValue}`;
		    });

		    result += strings[strings.length - 1]

		    return result;
		}
	*/

	result := ""
	for i, value := range exprs {
		formattedValue, _ := json.Marshal(value)
		formattedValue = []byte(fmt.Sprintf(`(builtins.fromJSON %s)`, strconv.Quote(string(formattedValue))))
		result += literals[i] + string(formattedValue)
	}

	result += literals[len(literals)-1]

	return result
}

func evalNixExpr(literals []string, exprs ...any) any {
	expr := buildNixExpr(literals, exprs...)
	data := []byte(runcmd("nix", "eval", "--json", "--expr", expr))
	var result any
	if err := json.Unmarshal(data, &result); err != nil {
		panic(err)
	}

	return result
}

func main() {
	internal.HandleStartup()

	vm := goja.New()

	defer func() {
		if r := recover(); r != nil {
			slog.Error("error in JS", "err", r)
			debug.PrintStack()
		}
	}()

	data, err := os.ReadFile(*fname)
	if err != nil {
		log.Fatal(err)
	}

	lg := log.New(writer.LineSplitting(writer.PrefixWriter("[yeet] ", os.Stdout)), "", 0)

	vm.Set("docker", map[string]any{
		"build": dockerbuild,
		"load":  dockerload,
		"push":  dockerpush,
		"tag":   dockertag,
	})

	vm.Set("file", map[string]any{
		"write": func(fname, data string) {
			if err := os.WriteFile(fname, []byte(data), 0660); err != nil {
				panic(err)
			}
		},
	})

	vm.Set("fly", map[string]any{
		"deploy": flydeploy,
	})

	vm.Set("go", map[string]any{
		"build": func() { runcmd("go", "build") },
	})

	vm.Set("git", map[string]any{
		"tag": gittag,
	})

	vm.Set("log", map[string]any{
		"info":    lg.Println,
		"println": fmt.Println,
	})

	vm.Set("nix", map[string]any{
		"build":   nixbuild,
		"hashURL": func(fileURL string) string { return runcmd("nix-prefetch-url", fileURL) },
		"expr":    buildNixExpr,
		"eval":    evalNixExpr,
	})

	vm.Set("slug", map[string]any{
		"build": slugbuild,
		"push":  slugpush,
	})

	vm.Set("yeet", map[string]any{
		"cwd":     yeet.WD,
		"datetag": yeet.DateTag,
		"runcmd":  runcmd,
		"setenv":  os.Setenv,
		"goos":    runtime.GOOS,
		"goarch":  runtime.GOARCH,
	})

	if _, err := vm.RunScript(*fname, string(data)); err != nil {
		log.Fatal(err)
	}
}
