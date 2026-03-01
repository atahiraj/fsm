package engine

import (
	"sync"
)

// AtomicEngine is a thread-safe wrapper around Engine.
type AtomicEngine[S any, I any] struct {
	mu sync.Mutex
	e  *Engine[S, I]
}

// NewAtomic wraps an Engine with a mutex for concurrent access.
func NewAtomic[S any, I any](e *Engine[S, I]) *AtomicEngine[S, I] {
	return &AtomicEngine[S, I]{e: e}
}

// Reset returns the Engine to the FSM's start configuration.
func (a *AtomicEngine[S, I]) Reset() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.e.Reset()
}

// Cur returns the current configuration.
func (a *AtomicEngine[S, I]) Cur() S {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.e.Cur()
}

// Accepting reports whether the current configuration is accepting.
func (a *AtomicEngine[S, I]) Accepting() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.e.Accepting()
}

// Step attempts to advance the machine by one input.
func (a *AtomicEngine[S, I]) Step(input I) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.e.Step(input)
}

// TryStep advances the machine by one input and reports whether a transition existed.
func (a *AtomicEngine[S, I]) TryStep(input I) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.e.TryStep(input)
}

// CanStep reports whether a transition exists for the current configuration and input.
func (a *AtomicEngine[S, I]) CanStep(input I) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.e.CanStep(input)
}

// PeekStep reports the next configuration for a input without mutating engine state.
func (a *AtomicEngine[S, I]) PeekStep(input I) (S, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.e.PeekStep(input)
}
