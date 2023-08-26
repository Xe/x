package dcc

import (
	"context"
	"encoding/binary"
	"io"
	"log/slog"
	"net"
	"time"
)

// Progress contains the progression of the
// download handled by the DCC client socket
type Progress struct {
	Speed           float64
	Percentage      float64
	CurrentFileSize float64
	FileSize        float64
}

func (p Progress) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Float64("speed", p.Speed),
		slog.Float64("percentage", p.Percentage),
		slog.Float64("curr", p.CurrentFileSize),
		slog.Float64("total", p.FileSize),
	)
}

// DCC creates a new socket client instance where
// it'll download the DCC transaction into the
// specified io.Writer destination
type DCC struct {
	// important properties
	address string
	size    int

	// output channels used for the Run and the receiver methods()
	// to avoid parameter passing
	progressc chan Progress
	done      chan error

	// internal DCC socket connection
	conn net.Conn

	// assigned context passed from the Run() method
	ctx context.Context

	// destination writer
	writer io.Writer
}

// NewDCC creates a new DCC instance.
// the host, port are needed for the socket client connection
// the size is required so the download progress is calculated
// the writer is required to store the transaction fragments into
// the specified io.Writer
func NewDCC(
	address string,
	size int,
	writer io.Writer,
) *DCC {
	return &DCC{
		address:   address,
		size:      size,
		progressc: make(chan Progress, 1),
		done:      make(chan error, 1),
		writer:    writer,
	}
}

func (d *DCC) progress(written float64, speed *float64) time.Time {
	d.progressc <- Progress{
		Speed:           written - *speed,
		Percentage:      (written / float64(d.size)) * 100,
		CurrentFileSize: written,
		FileSize:        float64(d.size),
	}

	*speed = float64(written)

	return time.Now()
}

func (d *DCC) receive() {
	defer func() { // close channels
		close(d.done)

		// close the connection afterwards..
		d.conn.Close()
	}()

	var (
		written int
		speed   float64
		buf     = make([]byte, 30720)
		reader  = io.LimitReader(d.conn, int64(d.size))
		ticker  = time.NewTicker(time.Second)
	)

	defer ticker.Stop()

D:
	for {
		select {
		case <-d.ctx.Done():
			d.done <- nil // send empty to notify the watchers that we're done
			return        // terminated..
		case <-ticker.C:
			d.progress(float64(written), &speed)
			// notify the other side about the state of the connection
			writtenNetworkOrder := uint32(written)
			if err := binary.Write(d.conn, binary.BigEndian, writtenNetworkOrder); err != nil {
				if err == io.EOF {
					err = nil
				}

				d.progress(float64(written), &speed)
				d.done <- err

				return
			}
		default:
			n, err := reader.Read(buf)

			if err != nil {
				if err == io.EOF { // empty out the error
					err = nil
				}

				d.progress(float64(written), &speed)
				d.done <- err

				return
			}

			if n > 0 {
				_, err = d.writer.Write(buf[0:n])

				if err != nil {
					d.done <- err
					return
				} else if written >= d.size { // finished
					break D
				}

				written += n
			}
		}
	}
}

// Run established the connection with the DCC TCP socket
// and returns two channels, where one is used for the download progress
// and the other is used to return exceptions during our transaction.
// A context is required, where you have the ability to cancel and timeout
// a download.
// One should check the second value for the progress/error channels when
// receiving data as if the channels are closed, it means that the transaction
// is finished or got interrupted.
func (d *DCC) Run(ctx context.Context) (
	progressc <-chan Progress,
	done <-chan error,
) {
	// assign the output to the struct properties
	progressc = d.progressc
	done = d.done

	// assign the passed context
	d.ctx = ctx

	dialer := &net.Dialer{Resolver: net.DefaultResolver}
	conn, err := dialer.DialContext(
		d.ctx, "tcp", d.address,
	)

	if err != nil {
		d.done <- err
		return
	}

	// setup the connection for the receiver
	d.conn = conn

	go d.receive()

	return
}
