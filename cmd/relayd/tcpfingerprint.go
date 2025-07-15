package main

import (
	"fmt"
	"strings"
)

type JA4T struct {
	Window      uint32
	Options     []uint8
	MSS         uint16
	WindowScale uint8
}

func (j JA4T) String() string {
	var sb strings.Builder

	// Start with the window size
	fmt.Fprintf(&sb, "%d", j.Window)

	// Append each option
	for i, opt := range j.Options {
		if i == 0 {
			fmt.Fprint(&sb, "_")
		} else {
			fmt.Fprint(&sb, "-")
		}
		fmt.Fprintf(&sb, "%d", opt)
	}

	// Append MSS and WindowScale
	fmt.Fprintf(&sb, "_%d_%d", j.MSS, j.WindowScale)

	return sb.String()
}
