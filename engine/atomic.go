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

// Step attempts to advance the machine by one symbol.
func (a *AtomicEngine[S, I]) Step(symbol I) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.e.Step(symbol)
}

// TryStep advances the machine by one symbol and reports whether a transition existed.
func (a *AtomicEngine[S, I]) TryStep(symbol I) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.e.TryStep(symbol)
}

// CanStep reports whether a transition exists for the current configuration and symbol.
func (a *AtomicEngine[S, I]) CanStep(symbol I) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.e.CanStep(symbol)
}

// PeekStep reports the next configuration for a symbol without mutating engine state.
func (a *AtomicEngine[S, I]) PeekStep(symbol I) (S, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.e.PeekStep(symbol)
}
