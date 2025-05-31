package recording

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var (
	ErrMismatchWrite = errors.New("recording: did not write the same number of bytes that were read")
)

// Recording ...
type Recording struct {
	ctx      context.Context
	url      string
	fname    string
	tmpDir   string
	cancel   context.CancelFunc
	started  time.Time
	restarts int

	Debug bool
	Err   error
}

// New creates a new Recording of the given URL to the given filename for output.
func New(url, destFname string) (*Recording, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Hour)

	tmpDir, err := os.MkdirTemp("", "aura-*")
	if err != nil {
		return nil, err
	}

	r := &Recording{
		ctx:     ctx,
		url:     url,
		fname:   destFname,
		cancel:  cancel,
		started: time.Now(),
		tmpDir:  tmpDir,
	}

	return r, nil
}

// Cancel stops the recording.
func (r *Recording) Cancel() {
	r.cancel()
}

// Done returns the done channel of the recording.
func (r *Recording) Done() <-chan struct{} {
	return r.ctx.Done()
}

// OutputFilename gets the output filename originally passed into New.
func (r *Recording) OutputFilename() string {
	return r.fname
}

// StartTime gets start time
func (r *Recording) StartTime() time.Time {
	return r.started
}

// Start blockingly starts the recording and returns the error if one is encountered while streaming.
// This should be stopped in another goroutine.
func (r *Recording) Start() error {
	sr, err := exec.LookPath("streamripper")
	if err != nil {
		return err
	}

	fname := filepath.Join(r.tmpDir, "temp.mp3")

	cmd := exec.CommandContext(r.ctx, sr, r.url, "-A", "-a", fname)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	slog.Info("starting streamripper", "cmd", cmd.Args)

	err = cmd.Start()
	if err != nil {
		return err
	}

	// Automatically kill recordings after eight hours
	go func() {
		t := time.NewTicker(8 * time.Hour)
		defer t.Stop()

		log.Println("got here")

		for {
			select {
			case <-r.ctx.Done():
				return
			case <-t.C:
				log.Printf("Automatically killing recording after 8 hours...")
				r.Cancel()
			}
		}
	}()

	go func() {
		defer r.Cancel()
		err := cmd.Wait()
		if err != nil {
			log.Println(err)
		}
	}()

	defer r.cancel()

	for {
		time.Sleep(250 * time.Millisecond)

		select {
		case <-r.ctx.Done():
			return Move(fname, r.fname)
		default:
		}
	}
}

func Move(source, destination string) error {
	err := os.Rename(source, destination)
	if err != nil && strings.Contains(err.Error(), "invalid cross-device link") {
		return moveCrossDevice(source, destination)
	}
	return err
}

func moveCrossDevice(source, destination string) error {
	src, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("open %s: %w", source, err)
	}
	defer src.Close()

	dst, err := os.Create(destination)
	if err != nil {
		return fmt.Errorf("create %s: %w", destination, err)
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	if err != nil {
		return fmt.Errorf("copy: %w", err)
	}

	return nil
}
