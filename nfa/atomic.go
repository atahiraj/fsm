package nfa

import (
	"sync"

	"github.com/stnhrsprkwns/fsm/key"
)

// AtomicNFA is a thread-safe wrapper around NFA.
// Use NewAtomic to construct a non-nil wrapper.
type AtomicNFA[State key.Keyer[StateKey], Symbol key.Keyer[SymbolKey], StateKey comparable, SymbolKey comparable] struct {
	mu sync.RWMutex
	n  *NFA[State, Symbol, StateKey, SymbolKey]
}

// NewAtomic wraps an NFA with a mutex for concurrent access.
func NewAtomic[State key.Keyer[StateKey], Symbol key.Keyer[SymbolKey], StateKey comparable, SymbolKey comparable](n *NFA[State, Symbol, StateKey, SymbolKey]) *AtomicNFA[State, Symbol, StateKey, SymbolKey] {
	return &AtomicNFA[State, Symbol, StateKey, SymbolKey]{n: n}
}

// Delta applies δ to a state on a single symbol.
func (a *AtomicNFA[State, Symbol, StateKey, SymbolKey]) Delta(s State, sym Symbol) []State {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.n.Delta(s, sym)
}

// DeltaStar applies δ repeatedly over a word. It implements δ*.
func (a *AtomicNFA[State, Symbol, StateKey, SymbolKey]) DeltaStar(s State, w []Symbol) []State {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.n.DeltaStar(s, w)
}

// DeltaSet applies δ to a set of states on a symbol.
func (a *AtomicNFA[State, Symbol, StateKey, SymbolKey]) DeltaSet(states []State, sym Symbol) []State {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.n.DeltaSet(states, sym)
}

// DeltaStarSet applies δ* to a set of states over a word.
func (a *AtomicNFA[State, Symbol, StateKey, SymbolKey]) DeltaStarSet(states []State, w []Symbol) []State {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.n.DeltaStarSet(states, w)
}

// Epsilon applies δ to a state on ε.
func (a *AtomicNFA[State, Symbol, StateKey, SymbolKey]) Epsilon(s State) []State {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.n.Epsilon(s)
}

// EpsilonClosure returns ε-closure(s).
func (a *AtomicNFA[State, Symbol, StateKey, SymbolKey]) EpsilonClosure(s State) []State {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.n.EpsilonClosure(s)
}

// EpsilonSet applies δ to a set of states on ε.
func (a *AtomicNFA[State, Symbol, StateKey, SymbolKey]) EpsilonSet(states []State) []State {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.n.EpsilonSet(states)
}

// EpsilonClosureSet returns ε-closure(Q).
func (a *AtomicNFA[State, Symbol, StateKey, SymbolKey]) EpsilonClosureSet(states []State) []State {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.n.EpsilonClosureSet(states)
}

// Start returns q₀, the start state.
func (a *AtomicNFA[State, Symbol, StateKey, SymbolKey]) Start() State {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.n.Start()
}

// StartSet returns {q₀}, the singleton set containing the start state.
func (a *AtomicNFA[State, Symbol, StateKey, SymbolKey]) StartSet() []State {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.n.StartSet()
}

// States returns Q, the set of all states.
func (a *AtomicNFA[State, Symbol, StateKey, SymbolKey]) States() []State {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.n.States()
}

// Alphabet returns Σ, the input alphabet.
func (a *AtomicNFA[State, Symbol, StateKey, SymbolKey]) Alphabet() []Symbol {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.n.Alphabet()
}

// Accepting returns F, the set of accepting states.
func (a *AtomicNFA[State, Symbol, StateKey, SymbolKey]) Accepting() []State {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.n.Accepting()
}

// Accepts reports whether the NFA accepts the given word.
func (a *AtomicNFA[State, Symbol, StateKey, SymbolKey]) Accepts(w []Symbol) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.n.Accepts(w)
}

// HasState reports whether state ∈ Q.
func (a *AtomicNFA[State, Symbol, StateKey, SymbolKey]) HasState(s State) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.n.HasState(s)
}

// HasSymbol reports whether symbol ∈ Σ.
func (a *AtomicNFA[State, Symbol, StateKey, SymbolKey]) HasSymbol(sym Symbol) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.n.HasSymbol(sym)
}

// IsAccepting reports whether any state in s is an accepting state.
func (a *AtomicNFA[State, Symbol, StateKey, SymbolKey]) IsAccepting(states []State) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.n.IsAccepting(states)
}

// SetStart sets q₀, the start state.
func (a *AtomicNFA[State, Symbol, StateKey, SymbolKey]) SetStart(s State) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.n.SetStart(s)
}

// AddStates inserts states into Q.
func (a *AtomicNFA[State, Symbol, StateKey, SymbolKey]) AddStates(states ...State) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.n.AddStates(states...)
}

// AddAlphabet inserts symbols into Σ.
func (a *AtomicNFA[State, Symbol, StateKey, SymbolKey]) AddAlphabet(symbols ...Symbol) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.n.AddAlphabet(symbols...)
}

// AddAccepting inserts states into F.
func (a *AtomicNFA[State, Symbol, StateKey, SymbolKey]) AddAccepting(states ...State) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.n.AddAccepting(states...)
}

// RemoveAccepting removes states from F.
func (a *AtomicNFA[State, Symbol, StateKey, SymbolKey]) RemoveAccepting(states ...State) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.n.RemoveAccepting(states...)
}

// SetDeltaer sets δ and ε.
func (a *AtomicNFA[State, Symbol, StateKey, SymbolKey]) SetDeltaer(deltaer Deltaer[StateKey, SymbolKey]) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.n.SetDeltaer(deltaer)
}
