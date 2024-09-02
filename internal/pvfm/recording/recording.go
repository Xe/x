package recording

import (
	"context"
	"errors"
	"log"
	"os"
	"os/exec"
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
	cancel   context.CancelFunc
	started  time.Time
	restarts int

	Debug bool
	Err   error
}

// New creates a new Recording of the given URL to the given filename for output.
func New(url, fname string) (*Recording, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Hour)

	r := &Recording{
		ctx:     ctx,
		url:     url,
		fname:   fname,
		cancel:  cancel,
		started: time.Now(),
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

	cmd := exec.CommandContext(r.ctx, sr, r.url, "-A", "-a", r.fname)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	log.Printf("%s: %v", cmd.Path, cmd.Args)

	err = cmd.Start()
	if err != nil {
		return err
	}

	// Automatically kill recordings after four hours
	go func() {
		t := time.NewTicker(4 * time.Hour)
		defer t.Stop()

		log.Println("got here")

		for {
			select {
			case <-r.ctx.Done():
				return
			case <-t.C:
				log.Printf("Automatically killing recording after 4 hours...")
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
			return nil
		default:
		}
	}
}
