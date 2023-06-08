// Package cardio enables Go programs to speed up and slow down based on demand.
package cardio

import (
	"context"
	"expvar"
	"sync"
	"time"

	"within.website/ln/opname"
)

// Heartbeat is a function that creates a "heartbeat" channel that you can influence to go faster
// or slower. This is intended to model the behavior of the human heart's ability to slow down and
// speed up based on physical activity.
//
// The basic usage is something like this:
//
//	heartbeat, slower, faster := cardio.Heartbeat(ctx, time.Minute, time.Millisecond)
//
// The min and max arguments control the minimum and maximum heart rate. This returns three things:
//
// - The heartbeat channel that your event loop will poll on
// - A function to influence the heartbeat to slow down (beacuse there isn't work to do)
// - A function to influence the heartbeat to speed up (because there is work to do)
//
// Your event loop should look something like this:
//
//	for range heartbeat {
//	    // do something
//	    if noWork {
//	        slower()
//	    } else {
//	        faster()
//	    }
//	}
//
// This will let you have a dynamically adjusting heartbeat for when your sick, twisted desires
// demand it.
//
// If you are using ln's opname facility (https://pkg.go.dev/within.website/ln/opname), then an
// expvar gauge will be created that will contain the current heartbeat. This allows you to
// monitor and alert on this value changing erratically.
func Heartbeat(ctx context.Context, min, max time.Duration) (<-chan struct{}, func(), func()) {
	heartbeat := make(chan struct{}, 1) // output channel
	currDelay := (max + min) / 2        // start at half speed
	var currDelayLock sync.Mutex

	var counter *expvar.Int
	var tachycardiaCounter *expvar.Int

	if name, ok := opname.Get(ctx); ok {
		counter = expvar.NewInt("gauge_heartbeat_" + name)
		tachycardiaCounter = expvar.NewInt("gauge_heartbeat_backoff_" + name)
	}

	slower := func() {
		currDelayLock.Lock()
		currDelay = currDelay / 2
		if currDelay < min {
			currDelay = min
		}
		currDelayLock.Unlock()
	}

	faster := func() {
		currDelayLock.Lock()
		currDelay = currDelay * 2
		if currDelay > max {
			currDelay = max
		}
		currDelayLock.Unlock()
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				close(heartbeat)
				return
			default:
				currDelayLock.Lock()
				toSleep := currDelay
				currDelayLock.Unlock()
				time.Sleep(toSleep)
				
				if counter != nil {
					counter.Set(int64(toSleep))
				}
				
				select {
				case heartbeat <- struct{}{}:
				default:
					tachycardiaCounter.Add(1)
					slower() // back off if the channel is full
				}
			}
		}
	}()

	return heartbeat, slower, faster
}
