//go:build linux && cgo

package main

/*
#include <errno.h>
#include <linux/uinput.h>
#include <string.h>
#include <sys/ioctl.h>
#include <unistd.h>

static int create_device(int fd, int joystick) {
	if (ioctl(fd, UI_SET_EVBIT, EV_KEY) < 0) return -1;
	const int keys[] = { KEY_F, KEY_R, KEY_S, KEY_T };
	const int buttons[] = { BTN_SOUTH, BTN_EAST, BTN_WEST, BTN_NORTH };
	for (unsigned int i = 0; i < sizeof(keys) / sizeof(keys[0]); i++) {
		if (ioctl(fd, UI_SET_KEYBIT, joystick ? buttons[i] : keys[i]) < 0) return -1;
	}
	if (joystick) {
		if (ioctl(fd, UI_SET_EVBIT, EV_ABS) < 0) return -1;
		const int axes[] = { ABS_X, ABS_Y };
		for (unsigned int i = 0; i < sizeof(axes) / sizeof(axes[0]); i++) {
			struct uinput_abs_setup axis = {
				.code = axes[i],
				.absinfo = { .minimum = -32767, .maximum = 32767, .flat = 4096 },
			};
			if (ioctl(fd, UI_ABS_SETUP, &axis) < 0) return -1;
		}
	}
	struct uinput_setup setup = {
		.id = { .bustype = BUS_VIRTUAL },
		.name = "random-frst",
	};
	if (joystick) strcpy(setup.name, "Generic USB Controller");
	if (ioctl(fd, UI_DEV_SETUP, &setup) < 0) return -1;
	return ioctl(fd, UI_DEV_CREATE);
}

static int write_events(int fd, const struct input_event *events, size_t size) {
	ssize_t n;
	do {
		n = write(fd, events, size);
	} while (n < 0 && errno == EINTR);
	if (n < 0) return -1;
	if ((size_t)n != size) {
		errno = EIO;
		return -1;
	}
	return 0;
}

static int emit_key(int fd, unsigned short key, int value) {
	struct input_event events[2] = {
		{ .type = EV_KEY, .code = key, .value = value },
		{ .type = EV_SYN, .code = SYN_REPORT, .value = 0 },
	};
	return write_events(fd, events, sizeof(events));
}

static int emit_stick(int fd, int x, int y) {
	struct input_event events[3] = {
		{ .type = EV_ABS, .code = ABS_X, .value = x },
		{ .type = EV_ABS, .code = ABS_Y, .value = y },
		{ .type = EV_SYN, .code = SYN_REPORT, .value = 0 },
	};
	return write_events(fd, events, sizeof(events));
}

static int destroy_keyboard(int fd) {
	return ioctl(fd, UI_DEV_DESTROY);
}
*/
import "C"

import (
	"errors"
	"fmt"
	"os"
)

// Use the system headers for ioctl numbers and input_event layout on each architecture.
var keys = [...]uint16{C.KEY_F, C.KEY_R, C.KEY_S, C.KEY_T}

type inputDevice struct {
	file *os.File
}

func openDevice(path, mode string) (*inputDevice, error) {
	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return nil, fmt.Errorf("open uinput (load the uinput module and grant device access): %w", err)
	}
	joystick := C.int(0)
	if mode == "joystick" {
		joystick = 1
	}
	if result, err := C.create_device(C.int(f.Fd()), joystick); result < 0 {
		return nil, errors.Join(fmt.Errorf("create virtual input device: %w", err), f.Close())
	}
	return &inputDevice{file: f}, nil
}

func (kb *inputDevice) emitStick(p position) error {
	if result, err := C.emit_stick(C.int(kb.file.Fd()), C.int(p.x), C.int(p.y)); result < 0 {
		return fmt.Errorf("write stick position: %w", err)
	}
	return nil
}

func (kb *inputDevice) emit(key uint16, pressed bool) error {
	value := C.int(0)
	if pressed {
		value = 1
	}
	if result, err := C.emit_key(C.int(kb.file.Fd()), C.ushort(key), value); result < 0 {
		return fmt.Errorf("write keyboard event: %w", err)
	}
	return nil
}

func (kb *inputDevice) close() error {
	var destroyErr error
	if result, err := C.destroy_keyboard(C.int(kb.file.Fd())); result < 0 {
		destroyErr = fmt.Errorf("destroy virtual input device: %w", err)
	}
	return errors.Join(destroyErr, kb.file.Close())
}
