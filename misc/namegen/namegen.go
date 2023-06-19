// Package namegen generates a random name with one of several strategies.
package namegen

import (
	"math/rand"
	"time"

	"cirello.io/goherokuname"
	"within.website/x/misc/namegen/elfs"
	"within.website/x/misc/namegen/tarot"
)

// Generator is a name generation function.
type Generator func() string

// AddGenerator adds a generator to the list
func AddGenerator(g Generator) {
	strats = append(strats, g)
}

func init() {
	rand.Seed(time.Now().UnixNano())

	AddGenerator(elfs.Next)
	AddGenerator(tarot.Next)
	AddGenerator(goherokuname.HaikunateHex)
}

var strats []Generator

// Next gives you the next name in the series.
func Next() string {
	gen := rand.Intn(len(strats))

	return strats[gen]()
}
