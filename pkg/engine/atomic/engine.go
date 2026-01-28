package atomic

import (
	"sync"

	"github.com/stnhrsprkwns/fsm/pkg/engine"
)

// Engine is a thread-safe wrapper around engine.Engine.
type Engine[S any, E any, SP any, EP any] struct {
	mu sync.Mutex
	e  *engine.Engine[S, E, SP, EP]
}

// New wraps an Engine with a mutex for concurrent access.
func New[S any, E any, SP any, EP any](e *engine.Engine[S, E, SP, EP]) *Engine[S, E, SP, EP] {
	return &Engine[S, E, SP, EP]{e: e}
}

// Reset returns the Engine to the FSM's start configuration.
func (a *Engine[S, E, SP, EP]) Reset(p SP) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.e.Reset(p)
}

// Cur returns the current configuration.
func (a *Engine[S, E, SP, EP]) Cur() S {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.e.Cur()
}

// Accepting reports whether the current configuration is accepting.
func (a *Engine[S, E, SP, EP]) Accepting() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.e.Accepting()
}

// Step advances the machine by one symbol and notifies the observer.
func (a *Engine[S, E, SP, EP]) Step(event E, payload EP) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.e.Step(event, payload)
}
