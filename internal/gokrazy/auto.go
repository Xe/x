// Package gokrazy makes it easier to adapt tools to https://gokrazy.org.
package gokrazy

import "time"

// WaitForClock returns once the system clock appears to have been
// set. Assumes that the system boots with a clock value of January 1,
// 1970 UTC (UNIX epoch), as is the case on the Raspberry Pi 3.
func WaitForClock() {
	epochPlus1Year := time.Unix(60*60*24*365, 0)
	for {
		if time.Now().After(epochPlus1Year) {
			return
		}
		// Sleeps for 1 real second, regardless of wall-clock time.
		// See https://github.com/golang/proposal/blob/master/design/12914-monotonic.md
		time.Sleep(1 * time.Second)
	}
}

func init() {
	WaitForClock()
}
