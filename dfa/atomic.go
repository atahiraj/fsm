package dfa

import (
	"sync"

	"github.com/stnhrsprkwns/fsm/key"
)

// AtomicDFA is a thread-safe wrapper around DFA.
// Use NewAtomic to construct a non-nil wrapper.
type AtomicDFA[State key.Keyer[StateKey], Symbol key.Keyer[SymbolKey], StateKey comparable, SymbolKey comparable] struct {
	mu sync.RWMutex
	d  *DFA[State, Symbol, StateKey, SymbolKey]
}

// NewAtomic wraps a DFA with a mutex for concurrent access.
func NewAtomic[State key.Keyer[StateKey], Symbol key.Keyer[SymbolKey], StateKey comparable, SymbolKey comparable](d *DFA[State, Symbol, StateKey, SymbolKey]) *AtomicDFA[State, Symbol, StateKey, SymbolKey] {
	return &AtomicDFA[State, Symbol, StateKey, SymbolKey]{d: d}
}

// Delta applies δ(s, a).
func (a *AtomicDFA[State, Symbol, StateKey, SymbolKey]) Delta(s State, sym Symbol) (State, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.d.Delta(s, sym)
}

// DeltaStar applies δ repeatedly over a word.
func (a *AtomicDFA[State, Symbol, StateKey, SymbolKey]) DeltaStar(s State, word []Symbol) (State, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.d.DeltaStar(s, word)
}

// IsAccepting reports whether s ∈ F.
func (a *AtomicDFA[State, Symbol, StateKey, SymbolKey]) IsAccepting(s State) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.d.IsAccepting(s)
}

// Accepts reports whether the DFA accepts the given word.
func (a *AtomicDFA[State, Symbol, StateKey, SymbolKey]) Accepts(word []Symbol) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.d.Accepts(word)
}

// States returns Q, the set of all states.
func (a *AtomicDFA[State, Symbol, StateKey, SymbolKey]) States() []State {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.d.States()
}

// Alphabet returns Σ, the input alphabet.
func (a *AtomicDFA[State, Symbol, StateKey, SymbolKey]) Alphabet() []Symbol {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.d.Alphabet()
}

// Start returns q₀, the start state.
func (a *AtomicDFA[State, Symbol, StateKey, SymbolKey]) Start() State {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.d.Start()
}

// Accepting returns F, the set of accepting states.
func (a *AtomicDFA[State, Symbol, StateKey, SymbolKey]) Accepting() []State {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.d.Accepting()
}

// SetDeltaer sets δ, the primitive transition function.
func (a *AtomicDFA[State, Symbol, StateKey, SymbolKey]) SetDeltaer(deltaer Deltaer[StateKey, SymbolKey]) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.d.SetDeltaer(deltaer)
}

// SetStart sets q₀, the start state.
func (a *AtomicDFA[State, Symbol, StateKey, SymbolKey]) SetStart(state State) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.d.SetStart(state)
}

// AddStates inserts states into Q.
func (a *AtomicDFA[State, Symbol, StateKey, SymbolKey]) AddStates(states ...State) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.d.AddStates(states...)
}

// AddAlphabet inserts symbols into Σ.
func (a *AtomicDFA[State, Symbol, StateKey, SymbolKey]) AddAlphabet(symbols ...Symbol) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.d.AddAlphabet(symbols...)
}

// AddAccepting inserts states into F.
func (a *AtomicDFA[State, Symbol, StateKey, SymbolKey]) AddAccepting(states ...State) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.d.AddAccepting(states...)
}

// RemoveAccepting removes states from F.
func (a *AtomicDFA[State, Symbol, StateKey, SymbolKey]) RemoveAccepting(states ...State) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.d.RemoveAccepting(states...)
}

// HasState reports whether state ∈ Q.
func (a *AtomicDFA[State, Symbol, StateKey, SymbolKey]) HasState(state State) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.d.HasState(state)
}

// HasSymbol reports whether symbol ∈ Σ.
func (a *AtomicDFA[State, Symbol, StateKey, SymbolKey]) HasSymbol(symbol Symbol) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.d.HasSymbol(symbol)
}
