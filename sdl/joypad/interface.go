// Package joypad offers a simpler interface to joystick/gamepad data.
package joypad

// Button constants
const (
	// "face" buttons
	A = iota // A is the primary action button. This should be "okay" in menus.
	B        // B is the secondary action button. This should be "cancel" in menus.
	X
	Y

	// "shoulder" buttons
	L  // L is present on the left shoulder of the controller
	R  // R is present on the right shoulder of the controller
	L2 // L2 is the second left shoulder button
	R2 // R2 is the second right shoulder button

	Start  // Start typically pauses or resumes the game. This should be "okay" in menus.
	Select // Select typically manipulates options.

	L3 // L3 is the left stick click button
	R3 // R3 is the right stick click button

	Capture // Capture takes a screenshot.
	Home    // Home returns the game to the system menu.
)

//go:generate stringer -type=Direction

// Direction is a cardinal direction derived from a dpad position
type Direction int

// Direction constants
const (
	None Direction = iota
	Up
	UpRight
	Right
	DownRight
	Down
	DownLeft
	Left
	UpLeft
)

// Gamepad refers to any device that only has a directional pad and a small set of buttons.
// Gamepad is the base interface.
type Gamepad interface {
	// NumFaceButtons refers to the number of buttons on the "face" of the controller.
	// These buttons are normally used by the right thumb.
	NumFaceButtons() int

	// NumShoulderButtons refers to the number of buttons on the "shoulder" of the controller.
	// These buttons are normally used by either pointer finger.
	NumShoulderButtons() int

	FaceButtons() (a, b, x, y bool)
	ShoulderButtons() (l, r bool)
	PauseButtons() (start, selectB bool)

	Dpad() Direction
}
