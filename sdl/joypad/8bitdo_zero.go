package joypad

import (
	"log"

	"github.com/veandco/go-sdl2/sdl"
)

// from manual testing, set as constants to ease readability.
const (
	eightBitdoZeroA      = 0
	eightBitdoZeroB      = 1
	eightBitdoZeroX      = 3
	eightBitdoZeroY      = 4
	eightBitdoZeroL      = 6
	eightBitdoZeroR      = 7
	eightBitdoZeroStart  = 11
	eightBitdoZeroSelect = 10

	eightBitdoZeroDpadYAxis = 1
	eightBitdoZeroDpadXAxis = 0

	eightBitdoZeroName = "8Bitdo Zero GamePad"
)

// EightBitdoZero adapts an 8bitdo zero controller into the Gamepad type.
type EightBitdoZero struct {
	j *sdl.Joystick
}

func NewEightBitdoZero(j *sdl.Joystick) Gamepad {
	if j.Name() != eightBitdoZeroName {
		return nil
	}

	return &EightBitdoZero{j: j}
}

// NumFaceButtons refers to the number of buttons on the "face" of the controller.
// These buttons are normally used by the right thumb.
func (e *EightBitdoZero) NumFaceButtons() int {
	return 4
}

// NumShoulderButtons refers to the number of buttons on the "shoulder" of the controller.
// These buttons are normally used by either pointer finger.
func (e *EightBitdoZero) NumShoulderButtons() int {
	return 2
}

// FaceButtons returns the values of every face button in the order
// A, B, X, Y.
func (e *EightBitdoZero) FaceButtons() (bool, bool, bool, bool) {
	a := e.j.GetButton(eightBitdoZeroA) == 1
	b := e.j.GetButton(eightBitdoZeroB) == 1
	x := e.j.GetButton(eightBitdoZeroX) == 1
	y := e.j.GetButton(eightBitdoZeroY) == 1

	return a, b, x, y
}

// ShoulderButtons returns the values of the shoulder buttons in the order
// L, R.
func (e *EightBitdoZero) ShoulderButtons() (bool, bool) {
	l := e.j.GetButton(eightBitdoZeroL)
	r := e.j.GetButton(eightBitdoZeroR)

	log.Printf("l: %v, r: %v", l, r)

	return l == 1, r == 1
}

func (e *EightBitdoZero) PauseButtons() (bool, bool) {
	start := e.j.GetButton(eightBitdoZeroStart) == 1
	selectB := e.j.GetButton(eightBitdoZeroSelect) == 1

	return start, selectB
}

func (e *EightBitdoZero) Dpad() Direction {
	x := e.j.GetAxis(eightBitdoZeroDpadXAxis)
	y := e.j.GetAxis(eightBitdoZeroDpadYAxis)

	// Dpad up: negative axis 1
	// Dpad down: positive axis 1
	// Dpad left: negative axis 0
	// Dpad right: positive axis 0

	noUp := x == 0
	noLeft := y == 0
	up := x < 0
	left := y < 0

	switch {
	// simple
	case up && noLeft:
		return Up
	case !up && noLeft:
		return Down
	case left && noUp:
		return Left
	case !left && noUp:
		return Right

	case up && left:
		return UpLeft
	case up && !left:
		return UpRight
	case !up && left:
		return DownLeft
	case !up && !left:
		return DownRight
	}

	return None
}
