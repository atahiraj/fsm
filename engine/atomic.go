package engine

import (
	"sync"
)

// AtomicEngine is a thread-safe wrapper around Engine.
type AtomicEngine[S any, E any, SP any, EP any] struct {
	mu sync.Mutex
	e  *Engine[S, E, SP, EP]
}

// NewAtomic wraps an Engine with a mutex for concurrent access.
func NewAtomic[S any, E any, SP any, EP any](e *Engine[S, E, SP, EP]) *AtomicEngine[S, E, SP, EP] {
	return &AtomicEngine[S, E, SP, EP]{e: e}
}

// Reset returns the Engine to the FSM's start configuration.
func (a *AtomicEngine[S, E, SP, EP]) Reset(p SP) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.e.Reset(p)
}

// Cur returns the current configuration.
func (a *AtomicEngine[S, E, SP, EP]) Cur() S {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.e.Cur()
}

// Accepting reports whether the current configuration is accepting.
func (a *AtomicEngine[S, E, SP, EP]) Accepting() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.e.Accepting()
}

// Step advances the machine by one symbol and notifies the observer.
func (a *AtomicEngine[S, E, SP, EP]) Step(event E, payload EP) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.e.Step(event, payload)
}
