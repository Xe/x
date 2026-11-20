//go:build linux && cgo

package main

import (
	"context"
	"encoding/binary"
	"errors"
	"os"
	"reflect"
	"testing"
	"time"
)

func TestKeyboardEventFrames(t *testing.T) {
	t.Parallel()
	rd, wr, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer rd.Close()
	defer wr.Close()
	kb := &inputDevice{file: wr}
	for _, pressed := range []bool{true, false} {
		if err := kb.emit(keys[0], pressed); err != nil {
			t.Fatal(err)
		}
		buf := make([]byte, 128)
		n, err := rd.Read(buf)
		if err != nil {
			t.Fatal(err)
		}
		// Linux input_event is 16 or 24 bytes, depending on the native ABI.
		if n != 32 && n != 48 {
			t.Fatalf("frame size = %d, want 32 or 48", n)
		}
		size := n / 2
		for i := range 2 {
			payload := buf[(i+1)*size-8 : (i+1)*size]
			gotType := binary.NativeEndian.Uint16(payload[:2])
			gotCode := binary.NativeEndian.Uint16(payload[2:4])
			gotValue := binary.NativeEndian.Uint32(payload[4:])
			var wantType, wantCode uint16
			var wantValue uint32
			if i == 0 {
				wantType, wantCode = 1, keys[0] // EV_KEY
				if pressed {
					wantValue = 1
				}
			}
			if gotType != wantType || gotCode != wantCode || gotValue != wantValue {
				t.Errorf("pressed=%v event=%d: got (%d, %d, %d), want (%d, %d, %d)", pressed, i, gotType, gotCode, gotValue, wantType, wantCode, wantValue)
			}
		}
	}
}

func TestStickEventFrames(t *testing.T) {
	t.Parallel()
	rd, wr, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer rd.Close()
	defer wr.Close()
	dev := &inputDevice{file: wr}
	for _, tt := range []struct {
		name     string
		position position
	}{
		{name: "deflected", position: position{12345, -23456}},
		{name: "centered"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := dev.emitStick(tt.position); err != nil {
				t.Fatal(err)
			}
			buf := make([]byte, 128)
			n, err := rd.Read(buf)
			if err != nil {
				t.Fatal(err)
			}
			if n != 48 && n != 72 {
				t.Fatalf("frame size = %d, want 48 or 72", n)
			}
			size := n / 3
			for i, want := range []struct {
				eventType, code uint16
				value           int32
			}{
				{3, 0, tt.position.x}, // EV_ABS, ABS_X
				{3, 1, tt.position.y}, // EV_ABS, ABS_Y
				{0, 0, 0},             // EV_SYN, SYN_REPORT
			} {
				payload := buf[(i+1)*size-8 : (i+1)*size]
				if got := binary.NativeEndian.Uint16(payload[:2]); got != want.eventType {
					t.Errorf("event %d type = %d, want %d", i, got, want.eventType)
				}
				if got := binary.NativeEndian.Uint16(payload[2:4]); got != want.code {
					t.Errorf("event %d code = %d, want %d", i, got, want.code)
				}
				if got := int32(binary.NativeEndian.Uint32(payload[4:])); got != want.value {
					t.Errorf("event %d value = %d, want %d", i, got, want.value)
				}
			}
		})
	}
}

func TestMoveStickCenters(t *testing.T) {
	t.Parallel()
	errWrite := errors.New("write failed")
	for _, tt := range []struct {
		name           string
		cancelOnFirst  bool
		cancelOnTarget bool
		cancelOnReturn bool
		writeErr       error
		centerErr      error
	}{
		{name: "normal"},
		{name: "cancel outward", cancelOnFirst: true},
		{name: "cancel hold", cancelOnTarget: true},
		{name: "cancel return", cancelOnReturn: true},
		{name: "failed outward write", writeErr: errWrite},
		{name: "failed center write", cancelOnFirst: true, centerErr: errWrite},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			target := position{axisMax, -axisMax / 2}
			var positions []position
			reached := false
			err := moveStick(ctx, func(p position) error {
				positions = append(positions, p)
				if p == target {
					reached = true
				}
				if tt.cancelOnFirst || (tt.cancelOnTarget && reached) || (tt.cancelOnReturn && reached && p != target) {
					cancel()
				}
				if p == (position{}) {
					return tt.centerErr
				}
				return tt.writeErr
			}, target, 3*time.Millisecond, time.Millisecond)
			wantErr := tt.writeErr
			if tt.centerErr != nil {
				wantErr = tt.centerErr
			}
			if !errors.Is(err, wantErr) {
				t.Errorf("error = %v, want %v", err, wantErr)
			}
			if len(positions) == 0 || positions[len(positions)-1] != (position{}) {
				t.Fatalf("stick did not finish centered: %v", positions)
			}
			if tt.name == "normal" && !reached {
				t.Error("stick never reached target")
			}
		})
	}
}

func TestInterpolate(t *testing.T) {
	t.Parallel()
	from, to := position{0, 1000}, position{1000, -1000}
	for _, tt := range []struct {
		name     string
		fraction float64
		want     position
	}{
		{name: "start", fraction: 0, want: from},
		{name: "ease outward", fraction: 0.25, want: position{156, 688}},
		{name: "midpoint", fraction: 0.5, want: position{500, 0}},
		{name: "end", fraction: 1, want: to},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := interpolate(from, to, tt.fraction); got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTapReleasesKey(t *testing.T) {
	t.Parallel()
	errWrite := errors.New("write failed")
	for _, tt := range []struct {
		name          string
		cancelOnPress bool
		pressErr      error
		releaseErr    error
	}{
		{name: "normal"},
		{name: "canceled during hold", cancelOnPress: true},
		{name: "failed press", pressErr: errWrite},
		{name: "failed release", releaseErr: errWrite},
		{name: "canceled with failed release", cancelOnPress: true, releaseErr: errWrite},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var states []bool
			emit := func(key uint16, pressed bool) error {
				if key != keys[0] {
					t.Errorf("key = %d, want %d", key, keys[0])
				}
				states = append(states, pressed)
				if pressed {
					if tt.cancelOnPress {
						cancel()
					}
					return tt.pressErr
				}
				return tt.releaseErr
			}
			hold := time.Duration(0)
			if tt.cancelOnPress {
				hold = time.Hour
			}
			err := tap(ctx, emit, keys[0], hold)
			wantErr := tt.pressErr
			if tt.releaseErr != nil {
				wantErr = tt.releaseErr
			}
			if !errors.Is(err, wantErr) {
				t.Errorf("error = %v, want %v", err, wantErr)
			}
			if !reflect.DeepEqual(states, []bool{true, false}) {
				t.Errorf("key states = %v, want [true false]", states)
			}
		})
	}
}

func TestPressLoopStops(t *testing.T) {
	t.Parallel()
	errWrite := errors.New("write failed")
	for _, tt := range []struct {
		name            string
		cancelOnPress   bool
		cancelOnRelease bool
		writeErr        error
	}{
		{name: "cancellation during press", cancelOnPress: true},
		{name: "cancellation before interval", cancelOnRelease: true},
		{name: "write failure", writeErr: errWrite},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var states []bool
			err := pressLoop(ctx, config{minDelay: time.Hour, maxDelay: time.Hour}, func(key uint16, pressed bool) error {
				valid := false
				for _, allowed := range keys {
					valid = valid || key == allowed
				}
				if !valid {
					t.Errorf("unexpected key %d", key)
				}
				states = append(states, pressed)
				if (pressed && tt.cancelOnPress) || (!pressed && tt.cancelOnRelease) {
					cancel()
				}
				return tt.writeErr
			})
			if !errors.Is(err, tt.writeErr) {
				t.Errorf("error = %v, want %v", err, tt.writeErr)
			}
			if !reflect.DeepEqual(states, []bool{true, false}) {
				t.Errorf("key states = %v, want [true false]", states)
			}
		})
	}
}
