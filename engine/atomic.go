package engine

import (
	"sync"
)

// AtomicEngine is a thread-safe wrapper around Engine.
type AtomicEngine[S any, I any, SP any, IP any] struct {
	mu sync.Mutex
	e  *Engine[S, I, SP, IP]
}

// NewAtomic wraps an Engine with a mutex for concurrent access.
func NewAtomic[S any, I any, SP any, IP any](e *Engine[S, I, SP, IP]) *AtomicEngine[S, I, SP, IP] {
	return &AtomicEngine[S, I, SP, IP]{e: e}
}

// Reset returns the Engine to the FSM's start configuration.
func (a *AtomicEngine[S, I, SP, IP]) Reset(p SP) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.e.Reset(p)
}

// Cur returns the current configuration.
func (a *AtomicEngine[S, I, SP, IP]) Cur() S {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.e.Cur()
}

// Accepting reports whether the current configuration is accepting.
func (a *AtomicEngine[S, I, SP, IP]) Accepting() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.e.Accepting()
}

// Step attempts to advance the machine by one symbol.
func (a *AtomicEngine[S, I, SP, IP]) Step(symbol I, payload IP) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.e.Step(symbol, payload)
}

// TryStep advances the machine by one symbol and reports whether a transition existed.
func (a *AtomicEngine[S, I, SP, IP]) TryStep(symbol I, payload IP) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.e.TryStep(symbol, payload)
}

// CanStep reports whether a transition exists for the current configuration and symbol.
func (a *AtomicEngine[S, I, SP, IP]) CanStep(symbol I) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.e.CanStep(symbol)
}

// PeekStep reports the next configuration for a symbol without mutating engine state.
func (a *AtomicEngine[S, I, SP, IP]) PeekStep(symbol I) (S, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.e.PeekStep(symbol)
}
