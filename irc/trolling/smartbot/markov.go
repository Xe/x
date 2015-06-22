package main

// This Markov chain code is taken from the "Generating arbitrary text"
// codewalk: http://golang.org/doc/codewalk/markov/
//
// Minor modifications have been made to make it easier to integrate
// with a webserver and to save/load state

import (
	"encoding/gob"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"sync"
)

// Prefix is a Markov chain prefix of one or more words.
type Prefix []string

// String returns the Prefix as a string (for use as a map key).
func (p Prefix) String() string {
	return strings.Join(p, " ")
}

// Shift removes the first word from the Prefix and appends the given word.
func (p Prefix) Shift(word string) {
	copy(p, p[1:])
	p[len(p)-1] = word
}

// Chain contains a map ("chain") of prefixes to a list of suffixes.
// A prefix is a string of prefixLen words joined with spaces.
// A suffix is a single word. A prefix can have multiple suffixes.
type Chain struct {
	Chain     map[string][]string
	prefixLen int
	mu        sync.Mutex
}

// NewChain returns a new Chain with prefixes of prefixLen words.
func NewChain(prefixLen int) *Chain {
	return &Chain{
		Chain:     make(map[string][]string),
		prefixLen: prefixLen,
	}
}

// Write parses the bytes into prefixes and suffixes that are stored in Chain.
func (c *Chain) Write(in string) (int, error) {
	sr := strings.NewReader(in)
	p := make(Prefix, c.prefixLen)
	for {
		var s string
		if _, err := fmt.Fscan(sr, &s); err != nil {
			break
		}
		key := p.String()
		c.mu.Lock()
		c.Chain[key] = append(c.Chain[key], s)
		c.mu.Unlock()
		p.Shift(s)
	}
	return len(in), nil
}

// Generate returns a string of at most n words generated from Chain.
func (c *Chain) Generate(n int) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	p := make(Prefix, c.prefixLen)
	var words []string
	for i := 0; i < n; i++ {
		choices := c.Chain[p.String()]
		if len(choices) == 0 {
			break
		}
		next := choices[rand.Intn(len(choices))]
		words = append(words, next)
		p.Shift(next)
	}
	return strings.Join(words, " ")
}

// Save the chain to a file
func (c *Chain) Save(fileName string) error {
	// Open the file for writing
	fo, err := os.Create(fileName)
	if err != nil {
		return err
	}
	// close fo on exit and check for its returned error
	defer func() {
		if err := fo.Close(); err != nil {
			panic(err)
		}
	}()

	// Create an encoder and dump to it
	c.mu.Lock()
	defer c.mu.Unlock()

	enc := gob.NewEncoder(fo)
	err = enc.Encode(c)
	if err != nil {
		return err
	}

	return nil
}

// Load the chain from a file
func (c *Chain) Load(fileName string) error {
	// Open the file for reading
	fi, err := os.Open(fileName)
	if err != nil {
		return err
	}
	// close fi on exit and check for its returned error
	defer func() {
		if err := fi.Close(); err != nil {
			panic(err)
		}
	}()

	// Create a decoder and read from it
	c.mu.Lock()
	defer c.mu.Unlock()

	dec := gob.NewDecoder(fi)
	err = dec.Decode(c)
	if err != nil {
		return err
	}

	return nil
}
