package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"al.essio.dev/pkg/shellescape"
	"github.com/dop251/goja"
	yeetinternal "within.website/x/cmd/yeet/internal"
	"within.website/x/cmd/yeet/internal/mkdeb"
	"within.website/x/cmd/yeet/internal/mkrpm"
	"within.website/x/cmd/yeet/internal/mktarball"
	"within.website/x/cmd/yeet/internal/pkgmeta"
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

	slog.Debug("running command", "cmd", cmdName, "args", args)

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

func dockerbuild(tag string, args ...string) {
	yeet.DockerBuild(context.Background(), yeet.WD, tag, args...)
}

func dockerpush(image string) {
	yeet.DockerPush(context.Background(), image)
}

func flydeploy() {
	runcmd(*flyctl, "deploy", "--now")
}

func slugbuild(bin string, extraFiles map[string]string) string {
	result := fmt.Sprintf("%s-%s.tar.gz", bin, yeet.DateTag)
	appsluggr.Must(bin, result, extraFiles)
	os.Remove(bin)
	return result
}

func slugpush(fname string) string {
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

func buildShellCommand(literals []string, exprs ...any) string {
	var sb strings.Builder
	for i, value := range exprs {
		sb.WriteString(literals[i])
		sb.WriteString(shellescape.Quote(fmt.Sprint(value)))
	}

	sb.WriteString(literals[len(literals)-1])

	return sb.String()
}

func runShellCommand(literals []string, exprs ...any) string {
	shPath, err := exec.LookPath("sh")
	if err != nil {
		panic(err)
	}

	cmd := buildShellCommand(literals, exprs...)

	slog.Debug("running command", "cmd", cmd)
	output, err := yeet.Output(context.Background(), shPath, "-c", cmd)
	if err != nil {
		panic(err)
	}

	return output
}

func hostname() string {
	result, err := os.Hostname()
	if err != nil {
		panic(err)
	}
	return result
}

func main() {
	internal.HandleStartup()

	vm := goja.New()
	vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))

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

	vm.Set("$", runShellCommand)

	vm.Set("deb", map[string]any{
		"build": func(p pkgmeta.Package) string {
			foutpath, err := mkdeb.Build(p)
			if err != nil {
				panic(err)
			}
			return foutpath
		},
	})

	vm.Set("docker", map[string]any{
		"build": dockerbuild,
		"load":  dockerload,
		"push":  dockerpush,
	})

	vm.Set("file", map[string]any{
		"install": func(src, dst string) {
			if err := mktarball.Copy(src, dst); err != nil {
				panic(err)
			}
		},
	})

	vm.Set("fly", map[string]any{
		"deploy": flydeploy,
	})

	vm.Set("git", map[string]any{
		"repoRoot": func() string {
			return runcmd("git", "rev-parse", "--show-toplevel")
		},
		"tag": yeetinternal.GitVersion,
	})

	vm.Set("go", map[string]any{
		"build": func(args ...string) {
			args = append([]string{"build"}, args...)
			runcmd("go", args...)
		},
		"install": func() { runcmd("go", "install") },
	})

	vm.Set("log", map[string]any{
		"info":    lg.Println,
		"println": fmt.Println,
	})

	vm.Set("rpm", map[string]any{
		"build": func(p pkgmeta.Package) string {
			foutpath, err := mkrpm.Build(p)
			if err != nil {
				panic(err)
			}
			return foutpath
		},
	})

	vm.Set("slug", map[string]any{
		"build": slugbuild,
		"push":  slugpush,
	})

	vm.Set("tarball", map[string]any{
		"build": func(p pkgmeta.Package) string {
			foutpath, err := mktarball.Build(p)
			if err != nil {
				panic(err)
			}
			return foutpath
		},
	})

	vm.Set("yeet", map[string]any{
		"cwd":      yeet.WD,
		"datetag":  yeet.DateTag,
		"hostname": hostname(),
		"runcmd":   runcmd,
		"run":      runcmd,
		"setenv":   os.Setenv,
		"goos":     runtime.GOOS,
		"goarch":   runtime.GOARCH,
	})

	if _, err := vm.RunScript(*fname, string(data)); err != nil {
		log.Fatal(err)
	}
}
