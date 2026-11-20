# random-frst

`random-frst` creates a Linux virtual gamepad through `/dev/uinput`.
It moves the left stick to random positions inside a circle.
The gamepad advertises face buttons for device recognition but never presses them.

Each action moves smoothly from center to a random position over 25 ms.
It holds that position for a random 50–150 ms, then returns smoothly over 25 ms.
The program pauses for a random 2–5 seconds between actions.
Positions follow a uniform distribution over the disk, up to full deflection by default.
Some positions can fall inside the dead zone of a game.

The first action occurs after a five-second countdown.
The countdown gives applications time to discover the device.
Movement uses eased interpolation with approximately one update per millisecond.
Linux scheduling and game polling can affect the timing and visible motion.

## Build

The build requires Linux, a C compiler, and Linux userspace headers (`linux/uinput.h`).
The C wrapper uses the native kernel structures and ioctl constants.

From the repository root, run:

```sh
CGO_ENABLED=1 go build -o /tmp/random-frst ./cmd/random-frst
```

## Device access

Load the kernel module:

```sh
sudo modprobe uinput
```

Create a dedicated group:

```sh
sudo groupadd -f uinput
sudo usermod -aG uinput "$(id -un)"
```

Create `/etc/udev/rules.d/99-random-frst.rules` with this content:

```udev
SUBSYSTEM=="misc", KERNEL=="uinput", GROUP="uinput", MODE="0660"
```

Reload the rules and apply them to the device:

```sh
sudo udevadm control --reload-rules
sudo udevadm trigger --subsystem-match=misc --sysname-match=uinput
sudo udevadm settle
```

Log out of KDE, then log in again to apply the group membership.
Group members can create other virtual input devices through `/dev/uinput`.

## Run

For a one-minute run after the countdown, use:

```sh
/tmp/random-frst -duration=1m
```

Select the virtual controller in your game.
Controller input follows the input policy of the game, including its background-input behavior.
Some games require the controller to exist before they start.
For these games, use a longer countdown, such as `-countdown=30s`.

The device name is `Generic USB Controller`.
Some games or SDL versions can require a controller mapping for this custom device.
The program exposes the left-stick axes as `ABS_X` and `ABS_Y`, with a range of -32767 to 32767.
The center position is `(0, 0)`.

For a continuous run with different timing, use:

```sh
/tmp/random-frst -min-delay=3s -max-delay=8s -min-hold=50ms -max-hold=150ms -move-time=25ms -strength=0.8
```

Use `-help` for all flags.
`-countdown` must be at least one second.
`-duration=0` runs until interrupted.
`-strength` limits the maximum radius as a fraction of full deflection.

For the original random F, R, S, and T keyboard input, use:

```sh
/tmp/random-frst -mode=keyboard -duration=1m
```

KDE sends keyboard events to the focused window.
The keyboard layout and active modifiers determine the resulting characters.
`-min-hold` and `-max-hold` also control keyboard hold times.
For a fixed hold, set both flags to the same duration.
The previous `-hold` flag is replaced by these two flags.

To stop the program from another terminal, run:

```sh
pkill -TERM -x random-frst
```

Ctrl-C also stops the program when its terminal has focus.
On SIGINT, SIGTERM, or duration expiry, the program immediately centers the stick before it destroys the device.
Keyboard mode releases the current key instead.
The program stops on an input error and attempts to center the stick or release the key.
It reports any input or cleanup error.

## Verification

Run the automated tests:

```sh
go test -race ./cmd/random-frst
```

The tests cover event framing, interpolation, cancellation, and cleanup after a write failure.
They do not require `/dev/uinput`.

For a device check, run `sudo evtest` while the program runs.
Select `Generic USB Controller` to inspect the axis events and returns to center.
For an end-to-end check, use the controller configuration screen of your game during a finite run.
