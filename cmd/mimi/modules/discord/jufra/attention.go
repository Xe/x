package jufra

import (
	"math"
	"time"

	"golang.org/x/exp/rand"
)

type AttentionAttenuator struct {
	lastMessageTime time.Time
	probability     float64
}

func NewAttentionAttenuator() *AttentionAttenuator {
	return &AttentionAttenuator{
		probability: 1.0,
	}
}

func (a *AttentionAttenuator) Poke() {
	a.probability = 1.0
	a.lastMessageTime = time.Now()
}

func (a *AttentionAttenuator) Reset() {
	a.probability = 0.0
}

func (a *AttentionAttenuator) Update() {
	elapsed := time.Since(a.lastMessageTime)

	a.probability *= (1 - 0.01*elapsed.Minutes())

	if a.probability < 0.0 {
		a.probability = 0.0
	}
}

func (a *AttentionAttenuator) Attention() bool {
	return math.Abs(rand.Float64()) < a.probability
}

func (a *AttentionAttenuator) GetProbability() float64 {
	return a.probability
}
