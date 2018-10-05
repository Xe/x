// Package failure implements the Phi Accrual Failure Detector
/*

This package implements the paper "The φ Accrual Failure Detector" (2004)
(available at http://hdl.handle.net/10119/4784).

To use the failure detection algorithm, you need a heartbeat loop that will
call Ping() at regular intervals.  At any point, you can call Phi() which will
report how suspicious it is that a heartbeat has not been heard since the last
time Ping() was called.

*/
package failure

import (
	"math"
	"sync"
	"time"

	"github.com/dgryski/go-onlinestats"
)

// Detector is a failure detector
type Detector struct {
	w          *onlinestats.Windowed
	last       time.Time
	minSamples int
	mu         sync.Mutex
}

// New returns a new failure detector that considers the last windowSize
// samples, and ensures there are at least minSamples in the window before
// returning an answer
func New(windowSize, minSamples int) *Detector {

	d := &Detector{
		w:          onlinestats.NewWindowed(windowSize),
		minSamples: minSamples,
	}

	return d
}

// Ping registers a heart-beat at time now
func (d *Detector) Ping(now time.Time) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if !d.last.IsZero() {
		d.w.Push(now.Sub(d.last).Seconds())
	}
	d.last = now
}

// Phi calculates the suspicion level at time 'now' that the remote end has failed
func (d *Detector) Phi(now time.Time) float64 {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.w.Len() < d.minSamples {
		return 0
	}

	t := now.Sub(d.last).Seconds()
	pLater := 1 - cdf(d.w.Mean(), d.w.Stddev(), t)
	phi := -math.Log10(pLater)

	return phi
}

// cdf is the cumulative distribution function of a normally distributed random
// variable with the given mean and standard deviation
func cdf(mean, stddev, x float64) float64 {
	return 0.5 + 0.5*math.Erf((x-mean)/(stddev*math.Sqrt2))
}
