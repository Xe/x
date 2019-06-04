package main

import (
	"errors"

	"within.website/x/markov"
	"within.website/x/web/switchcounter"
)

// ilo li ilo pi toki sona.
type ilo struct {
	cfg   lipuSona
	sw    switchcounter.API
	chain *markov.Chain
	words []Word
}

var (
	ErrJanLawaAla = errors.New("ilo-kesi: sina jan lawa ala")
)

func (i ilo) janLawaAnuSeme(authorID string) bool {
	for _, jan := range i.cfg.JanLawa {
		if authorID == jan {
			return true
		}
	}

	return false
}

type reply struct {
	msg string
}
