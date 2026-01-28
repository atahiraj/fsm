package atomic

import (
	"sync"

	"github.com/stnhrsprkwns/fsm/pkg/dfa"
)

// DFA is a thread-safe wrapper around dfa.DFA.
// Use New to construct a non-nil DFA.
type DFA[State comparable, Symbol comparable] struct {
	mu sync.RWMutex
	d  *dfa.DFA[State, Symbol]
}

// New wraps a DFA with a mutex for concurrent access.
func New[State comparable, Symbol comparable](d *dfa.DFA[State, Symbol]) *DFA[State, Symbol] {
	return &DFA[State, Symbol]{d: d}
}

// Delta applies δ(s, a).
func (a *DFA[State, Symbol]) Delta(s State, sym Symbol) (State, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.d.Delta(s, sym)
}

// DeltaStar applies δ repeatedly over a word.
func (a *DFA[State, Symbol]) DeltaStar(s State, word []Symbol) (State, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.d.DeltaStar(s, word)
}

// IsAccepting reports whether s ∈ F.
func (a *DFA[State, Symbol]) IsAccepting(s State) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.d.IsAccepting(s)
}

// Accepts reports whether the DFA accepts the given word.
func (a *DFA[State, Symbol]) Accepts(word []Symbol) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.d.Accepts(word)
}

// States returns Q, the set of all states.
func (a *DFA[State, Symbol]) States() []State {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.d.States()
}

// Alphabet returns Σ, the input alphabet.
func (a *DFA[State, Symbol]) Alphabet() []Symbol {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.d.Alphabet()
}

// Start returns q₀, the start state.
func (a *DFA[State, Symbol]) Start() State {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.d.Start()
}

// Accepting returns F, the set of accepting states.
func (a *DFA[State, Symbol]) Accepting() []State {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.d.Accepting()
}

// SetDeltaer sets δ, the primitive transition function.
func (a *DFA[State, Symbol]) SetDeltaer(deltaer dfa.Deltaer[State, Symbol]) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.d.SetDeltaer(deltaer)
}

// SetStart sets q₀, the start state.
func (a *DFA[State, Symbol]) SetStart(state State) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.d.SetStart(state)
}

// AddStates inserts states into Q.
func (a *DFA[State, Symbol]) AddStates(states ...State) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.d.AddStates(states...)
}

// AddAlphabet inserts symbols into Σ.
func (a *DFA[State, Symbol]) AddAlphabet(symbols ...Symbol) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.d.AddAlphabet(symbols...)
}

// AddAccepting inserts states into F.
func (a *DFA[State, Symbol]) AddAccepting(states ...State) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.d.AddAccepting(states...)
}

// RemoveAccepting removes states from F.
func (a *DFA[State, Symbol]) RemoveAccepting(states ...State) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.d.RemoveAccepting(states...)
}

// HasState reports whether state ∈ Q.
func (a *DFA[State, Symbol]) HasState(state State) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.d.HasState(state)
}

// HasSymbol reports whether symbol ∈ Σ.
func (a *DFA[State, Symbol]) HasSymbol(symbol Symbol) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.d.HasSymbol(symbol)
}
