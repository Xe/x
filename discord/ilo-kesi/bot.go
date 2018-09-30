package main

import (
	"errors"

	"github.com/Xe/x/web/switchcounter"
)

// ilo li ilo pi toki sona.
type ilo struct {
	cfg   lipuSona
	sw    switchcounter.API
	chain *Chain
}

var (
	ErrJanLawaAla = errors.New("ilo-kesi: sina jan lawa ala")
)

func (i ilo) janLawaAnuSeme(authorID string) bool {
	for _, jan := range i.cfg.janLawa {
		if authorID == jan {
			return true
		}
	}

	return false
}

type reply struct {
	msg string
}
