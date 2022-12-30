package h

import (
	"strings"

	"github.com/eaburns/peggy/peg"
)

// RemoveSpace removes whitespace-only nodes.
func RemoveSpace(n *peg.Node) { removeSpace(n) }

func removeSpace(n *peg.Node) bool {
	if whitespace(n.Text) {
		return false
	}
	if len(n.Kids) == 0 {
		return true
	}
	var kids []*peg.Node
	for _, k := range n.Kids {
		if removeSpace(k) {
			kids = append(kids, k)
		}
	}
	n.Kids = kids
	return len(n.Kids) > 0
}

// SpaceChars is the string of all whitespace characters.
const SpaceChars = "\x20"

func whitespace(s string) bool {
	for _, r := range s {
		if !strings.ContainsRune(SpaceChars, r) {
			return false
		}
	}
	return true
}

// CollapseLists collapses chains of single-kid nodes.
func CollapseLists(n *peg.Node) {
	if collapseLists(n) == 1 {
		n.Kids = n.Kids[0].Kids
	}
}

func collapseLists(n *peg.Node) int {
	var kids []*peg.Node
	for _, k := range n.Kids {
		if gk := collapseLists(k); gk == 1 {
			kids = append(kids, k.Kids[0])
		} else {
			kids = append(kids, k)
		}
	}
	n.Kids = kids
	return len(n.Kids)
}
