package atomic

import (
	"sync"

	"github.com/stnhrsprkwns/fsm/pkg/nfa"
	"github.com/stnhrsprkwns/fsm/pkg/set"
)

// NFA is a thread-safe wrapper around nfa.NFA.
// Use New to construct a non-nil NFA.
type NFA[State comparable, Symbol comparable] struct {
	mu sync.RWMutex
	n  *nfa.NFA[State, Symbol]
}

// New wraps an NFA with a mutex for concurrent access.
func New[State comparable, Symbol comparable](n *nfa.NFA[State, Symbol]) *NFA[State, Symbol] {
	return &NFA[State, Symbol]{n: n}
}

// Delta applies δ to a state on a single symbol.
func (a *NFA[State, Symbol]) Delta(s State, sym Symbol) []State {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.n.Delta(s, sym)
}

// DeltaStar applies δ repeatedly over a word. It implements δ*.
func (a *NFA[State, Symbol]) DeltaStar(s State, w []Symbol) []State {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.n.DeltaStar(s, w)
}

// DeltaSet applies δ to a set of states on a symbol.
func (a *NFA[State, Symbol]) DeltaSet(Q *set.Set[State], sym Symbol) *set.Set[State] {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.n.DeltaSet(Q, sym)
}

// DeltaStarSet applies δ* to a set of states over a word.
func (a *NFA[State, Symbol]) DeltaStarSet(Q *set.Set[State], w []Symbol) *set.Set[State] {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.n.DeltaStarSet(Q, w)
}

// Epsilon applies δ to a state on ε.
func (a *NFA[State, Symbol]) Epsilon(s State) []State {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.n.Epsilon(s)
}

// EpsilonClosure returns ε-closure(s).
func (a *NFA[State, Symbol]) EpsilonClosure(s State) []State {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.n.EpsilonClosure(s)
}

// EpsilonSet applies δ to a set of states on ε.
func (a *NFA[State, Symbol]) EpsilonSet(Q *set.Set[State]) *set.Set[State] {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.n.EpsilonSet(Q)
}

// EpsilonClosureSet returns ε-closure(Q).
func (a *NFA[State, Symbol]) EpsilonClosureSet(Q *set.Set[State]) *set.Set[State] {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.n.EpsilonClosureSet(Q)
}

// Start returns q₀, the start state.
func (a *NFA[State, Symbol]) Start() State {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.n.Start()
}

// StartSet returns {q₀}, the singleton set containing the start state.
func (a *NFA[State, Symbol]) StartSet() *set.Set[State] {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.n.StartSet()
}

// States returns Q, the set of all states.
func (a *NFA[State, Symbol]) States() []State {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.n.States()
}

// Alphabet returns Σ, the input alphabet.
func (a *NFA[State, Symbol]) Alphabet() []Symbol {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.n.Alphabet()
}

// Accepting returns F, the set of accepting states.
func (a *NFA[State, Symbol]) Accepting() []State {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.n.Accepting()
}

// Accepts reports whether the NFA accepts the given word.
func (a *NFA[State, Symbol]) Accepts(w []Symbol) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.n.Accepts(w)
}

// HasState reports whether state ∈ Q.
func (a *NFA[State, Symbol]) HasState(s State) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.n.HasState(s)
}

// HasSymbol reports whether symbol ∈ Σ.
func (a *NFA[State, Symbol]) HasSymbol(sym Symbol) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.n.HasSymbol(sym)
}

// IsAccepting reports whether any state in s is an accepting state.
func (a *NFA[State, Symbol]) IsAccepting(s *set.Set[State]) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.n.IsAccepting(s)
}

// SetStart sets q₀, the start state.
func (a *NFA[State, Symbol]) SetStart(s State) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.n.SetStart(s)
}

// AddStates inserts states into Q.
func (a *NFA[State, Symbol]) AddStates(states ...State) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.n.AddStates(states...)
}

// AddAlphabet inserts symbols into Σ.
func (a *NFA[State, Symbol]) AddAlphabet(symbols ...Symbol) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.n.AddAlphabet(symbols...)
}

// AddAccepting inserts states into F.
func (a *NFA[State, Symbol]) AddAccepting(states ...State) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.n.AddAccepting(states...)
}

// RemoveAccepting removes states from F.
func (a *NFA[State, Symbol]) RemoveAccepting(states ...State) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.n.RemoveAccepting(states...)
}

// SetDeltaer sets δ and ε.
func (a *NFA[State, Symbol]) SetDeltaer(deltaer nfa.Deltaer[State, Symbol]) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.n.SetDeltaer(deltaer)
}
