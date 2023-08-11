package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

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

func main() {
	internal.HandleStartup()

	vm := goja.New()

	defer func() {
		if r := recover(); r != nil {
			slog.Error("error in JS", "err", r)
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
		"info": lg.Println,
	})

	vm.Set("nix", map[string]any{
		"build":   nixbuild,
		"hashURL": func(fileURL string) string { return runcmd("nix-prefetch-url", fileURL) },
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
