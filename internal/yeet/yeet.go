package yeet

import (
	"context"
	"fmt"
	"log"
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
		log.Fatal(err)
	}

	WD = lwd
	DateTag = time.Now().Format("010220061504")
}

// ShouldWork explodes if the given command with the given env, working dir and context fails.
func ShouldWork(ctx context.Context, env []string, dir string, cmdName string, args ...string) {
	loc, err := exec.LookPath(cmdName)
	if err != nil {
		log.Fatal(err)
	}

	cmd := exec.CommandContext(ctx, loc, args...)
	cmd.Dir = dir
	cmd.Env = env

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Printf("starting process, env: %v, pwd: %s, cmd: %s, args: %v", env, dir, loc, args)
	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

/// Output returns the output of a command or an error.
func output(ctx context.Context, cmd string, args ...string) (string, error) {
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
	s, err := output(ctx, "git", "describe", "--tags")
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
func DockerTag(ctx context.Context, org, repo, image string) (string, error) {
	tag, err := GitTag(ctx)
	if err != nil {
		return "", err
	}

	repoTag := fmt.Sprintf("%s/%s:%s", org, repo, tag)

	ShouldWork(ctx, nil, WD, "docker", "tag", image, repoTag)

	return repoTag, nil
}
