package attention

import (
	"math"
	"time"

	rand "math/rand/v2"
)

type Attenuator struct {
	lastMessageTime time.Time
	probability     float64
}

func NewAttentionAttenuator() *Attenuator {
	return &Attenuator{
		probability: 1.0,
	}
}

func (a *Attenuator) Poke() {
	a.probability = 1.0
	a.lastMessageTime = time.Now()
}

func (a *Attenuator) Reset() {
	a.probability = 0.0
}

func (a *Attenuator) Update() {
	elapsed := time.Since(a.lastMessageTime)

	a.probability *= (1 - 0.01*elapsed.Minutes())

	if a.probability < 0.0 {
		a.probability = 0.0
	}
}

func (a *Attenuator) Attention() bool {
	return math.Abs(rand.Float64()) < a.probability
}

func (a *Attenuator) GetProbability() float64 {
	return a.probability
}
