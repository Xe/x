// Package s6 allows Go programs to signal readiness to the s6[1] suite of system
// supervision tools. This should be run in func main.
//
// [1]: http://skarnet.org/software/s6/index.html
package s6

import (
	"errors"
	"flag"
	"os"
)

var (
	ErrCantFindNotificationFD = errors.New("s6: can't find notification file descriptor")
	notificationFD            = flag.Int("notification-fd", 0, "notification file descriptor")
)

// Signal signals readiness to s6.
//
// See: http://skarnet.org/software/s6/notifywhenup.html
func Signal() error {
	var err error

	// If this is unset, we probably don't care about notifying s6.
	if *notificationFD == 0 {
		return nil
	}

	fout := os.NewFile(uintptr(*notificationFD), "s6-notification")
	if fout == nil {
		return ErrCantFindNotificationFD
	}
	defer fout.Close()

	_, err = fout.Write([]byte("\n"))
	return err
}
