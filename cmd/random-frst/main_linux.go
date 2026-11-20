//go:build linux && cgo

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"math"
	"math/rand/v2"
	"os"
	"os/signal"
	"syscall"
	"time"

	"within.website/x/internal"
)

var (
	devicePath = flag.String("device", "/dev/uinput", "uinput device path")
	mode       = flag.String("mode", "joystick", "input mode: joystick or keyboard")
	minDelay   = flag.Duration("min-delay", 2*time.Second, "minimum pause between actions")
	maxDelay   = flag.Duration("max-delay", 5*time.Second, "maximum pause between actions")
	minHold    = flag.Duration("min-hold", 50*time.Millisecond, "minimum hold duration")
	maxHold    = flag.Duration("max-hold", 150*time.Millisecond, "maximum hold duration")
	moveTime   = flag.Duration("move-time", 25*time.Millisecond, "duration of each outward and return stick movement")
	strength   = flag.Float64("strength", 1, "maximum stick deflection as a fraction of full range (0 < strength <= 1)")
	countdown  = flag.Duration("countdown", 5*time.Second, "delay before the first action (at least 1s for device discovery)")
	duration   = flag.Duration("duration", 0, "run duration after countdown (0 runs until interrupted)")
)

type config struct {
	mode                                                                string
	minDelay, maxDelay, minHold, maxHold, moveTime, countdown, duration time.Duration
	strength                                                            float64
}

func (c config) validate() error {
	if c.minDelay <= 0 || c.maxDelay < c.minDelay {
		return errors.New("delays must satisfy 0 < min-delay <= max-delay")
	}
	if c.minHold <= 0 || c.maxHold < c.minHold {
		return errors.New("hold durations must satisfy 0 < min-hold <= max-hold")
	}
	if c.mode != "joystick" && c.mode != "keyboard" {
		return errors.New("mode must be joystick or keyboard")
	}
	if c.moveTime <= 0 {
		return errors.New("move-time must be positive")
	}
	if math.IsNaN(c.strength) || c.strength <= 0 || c.strength > 1 {
		return errors.New("strength must satisfy 0 < strength <= 1")
	}
	if c.countdown < time.Second {
		return errors.New("countdown must be at least 1s for device discovery")
	}
	if c.duration < 0 {
		return errors.New("duration must not be negative")
	}
	return nil
}

func main() {
	internal.HandleStartup()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, config{
		mode: *mode, minDelay: *minDelay, maxDelay: *maxDelay,
		minHold: *minHold, maxHold: *maxHold, moveTime: *moveTime,
		countdown: *countdown, duration: *duration, strength: *strength,
	}); err != nil {
		slog.Error("random-frst stopped", "err", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, cfg config) (err error) {
	if err := cfg.validate(); err != nil {
		return err
	}
	kb, err := openDevice(*devicePath, cfg.mode)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, kb.close()) }()

	slog.Info("prepare the target application", "countdown", cfg.countdown, "mode", cfg.mode)
	if err := wait(ctx, cfg.countdown); err != nil {
		return nil
	}
	if cfg.duration > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cfg.duration)
		defer cancel()
	}
	slog.Info("sending random input", "mode", cfg.mode, "min-delay", cfg.minDelay, "max-delay", cfg.maxDelay)
	if cfg.mode == "joystick" {
		return stickLoop(ctx, cfg, kb.emitStick)
	}
	return pressLoop(ctx, cfg, kb.emit)
}

type emitter func(key uint16, pressed bool) error

func pressLoop(ctx context.Context, cfg config, emit emitter) error {
	for ctx.Err() == nil {
		key := keys[rand.IntN(len(keys))]
		if err := tap(ctx, emit, key, randomDuration(cfg.minHold, cfg.maxHold)); err != nil {
			return err
		}
		if err := wait(ctx, randomDuration(cfg.minDelay, cfg.maxDelay)); err != nil {
			return nil
		}
	}
	return nil
}

func randomDuration(min, max time.Duration) time.Duration {
	if min == max {
		return min
	}
	return min + time.Duration(rand.Int64N(int64(max-min)))
}

const axisMax = 32767

type position struct{ x, y int32 }
type stickEmitter func(position) error

func randomPosition(strength float64) position {
	angle := rand.Float64() * 2 * math.Pi
	// The square root gives a uniform distribution over the disk's area.
	radius := math.Sqrt(rand.Float64()) * strength * axisMax
	return position{int32(math.Round(math.Cos(angle) * radius)), int32(math.Round(math.Sin(angle) * radius))}
}

func stickLoop(ctx context.Context, cfg config, emit stickEmitter) error {
	for ctx.Err() == nil {
		if err := moveStick(ctx, emit, randomPosition(cfg.strength), cfg.moveTime, randomDuration(cfg.minHold, cfg.maxHold)); err != nil {
			return err
		}
		if err := wait(ctx, randomDuration(cfg.minDelay, cfg.maxDelay)); err != nil {
			return nil
		}
	}
	return nil
}

func moveStick(ctx context.Context, emit stickEmitter, target position, travel, hold time.Duration) (err error) {
	// Always center, including after cancellation or a partially delivered write.
	defer func() { err = errors.Join(err, emit(position{})) }()
	if err := ramp(ctx, emit, position{}, target, travel); err != nil {
		return err
	}
	if err := wait(ctx, hold); err != nil {
		return nil
	}
	return ramp(ctx, emit, target, position{}, travel)
}

func interpolate(from, to position, fraction float64) position {
	// Smoothstep eases into and out of motion without overshooting the target.
	fraction = fraction * fraction * (3 - 2*fraction)
	return position{
		x: int32(math.Round(float64(from.x) + float64(to.x-from.x)*fraction)),
		y: int32(math.Round(float64(from.y) + float64(to.y-from.y)*fraction)),
	}
}

func ramp(ctx context.Context, emit stickEmitter, from, to position, duration time.Duration) error {
	start := time.Now()
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	end := time.NewTimer(duration)
	defer end.Stop()
	for {
		if ctx.Err() != nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return nil
		case <-end.C:
			return emit(to)
		case <-ticker.C:
			fraction := math.Min(1, float64(time.Since(start))/float64(duration))
			if err := emit(interpolate(from, to, fraction)); err != nil {
				return err
			}
		}
	}
}

func tap(ctx context.Context, emit emitter, key uint16, hold time.Duration) error {
	pressErr := emit(key, true)
	if pressErr == nil {
		// Cancellation shortens the hold, but must never skip the release.
		_ = wait(ctx, hold)
	}
	// A failed press write can still have delivered the key-down event.
	releaseErr := emit(key, false)
	if err := errors.Join(pressErr, releaseErr); err != nil {
		return fmt.Errorf("tap key %d: %w", key, err)
	}
	return nil
}

func wait(ctx context.Context, d time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return ctx.Err()
	}
}
