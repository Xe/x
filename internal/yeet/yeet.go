package yeet

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// current working directory and date:time tag of app boot (useful for tagging slugs)
var (
	WD      string
	DateTag string
)

func init() {
	lwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	WD = lwd
	DateTag = time.Now().Format("200601021504")
}

// ShouldWork explodes if the given command with the given env, working dir and context fails.
func ShouldWork(ctx context.Context, env []string, dir string, cmdName string, args ...string) {
	loc, err := exec.LookPath(cmdName)
	if err != nil {
		panic(err)
	}

	cmd := exec.CommandContext(ctx, loc, args...)
	cmd.Dir = dir
	cmd.Env = env

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	slog.Info("starting process", "pwd", dir, "cmd", loc, "args", args)

	err = cmd.Run()
	if err != nil {
		panic(err)
	}
}

// Output returns the output of a command or an error.
func Output(ctx context.Context, cmd string, args ...string) (string, error) {
	c := exec.CommandContext(ctx, cmd, args...)
	c.Env = os.Environ()
	c.Stderr = os.Stderr
	b, err := c.Output()
	if err != nil {
		return "", errors.Wrapf(err, `failed to run %v %q`, cmd, args)
	}
	return string(b), nil
}

// GitTag returns the curreng git tag.
func GitTag(ctx context.Context) (string, error) {
	s, err := Output(ctx, "git", "describe", "--tags")
	if err != nil {
		ee, ok := errors.Cause(err).(*exec.ExitError)
		if ok && ee.Exited() {
			// probably no git tag
			return "dev", nil
		}
		return "", err
	}

	return strings.TrimSuffix(s, "\n"), nil
}

// DockerTag tags a docker image
func DockerTag(ctx context.Context, org, repo, image string) string {
	tag, err := GitTag(ctx)
	if err != nil {
		panic(err)
	}

	repoTag := fmt.Sprintf("%s/%s:%s", org, repo, tag)

	ShouldWork(ctx, nil, WD, "docker", "tag", image, repoTag)

	return repoTag
}

// DockerBuild builds a docker image with the given working directory and tag.
func DockerBuild(ctx context.Context, dir, tag string, args ...string) {
	args = append([]string{"build", "-t", tag}, args...)
	args = append(args, ".")
	ShouldWork(ctx, nil, dir, "docker", args...)
}

// DockerLoadResult loads a nix-built docker image
func DockerLoadResult(ctx context.Context, at string) {
	c := exec.CommandContext(ctx, "docker", "load")
	c.Env = os.Environ()
	fin, err := os.Open(at)
	if err != nil {
		panic(err)
	}
	defer fin.Close()
	c.Stdin = fin

	if err := c.Run(); err != nil {
		panic(err)
	}
}

// DockerPush pushes a docker image to a given host
func DockerPush(ctx context.Context, image string) {
	ShouldWork(ctx, nil, WD, "docker", "push", image)
}
