// Package breaker is a minimal circuit breaker: N consecutive failures open it for a cool-down.
package breaker

import (
	"sync"
	"time"
)

type Breaker struct {
	mu        sync.Mutex
	failures  int
	threshold int
	openUntil time.Time
	cooldown  time.Duration
}

func New(threshold int, cooldown time.Duration) *Breaker {
	return &Breaker{threshold: threshold, cooldown: cooldown}
}

func (b *Breaker) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return time.Now().After(b.openUntil)
}

func (b *Breaker) Success() {
	b.mu.Lock()
	b.failures = 0
	b.mu.Unlock()
}

func (b *Breaker) Failure() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.failures++
	if b.failures >= b.threshold {
		b.openUntil = time.Now().Add(b.cooldown)
		b.failures = 0
	}
}

func (b *Breaker) Open() bool { return !b.Allow() }
