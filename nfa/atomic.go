package nfa

import (
	"sync"

	"github.com/atahiraj/fsm/key"
)

// AtomicNFA is a thread-safe wrapper around NFA.
// Use NewAtomic to construct a non-nil wrapper.
type AtomicNFA[State key.Keyer[StateKey], Input key.Keyer[InputKey], StateKey comparable, InputKey comparable] struct {
	mu sync.RWMutex
	n  *NFA[State, Input, StateKey, InputKey]
}

// NewAtomic wraps an NFA with a mutex for concurrent access.
func NewAtomic[State key.Keyer[StateKey], Input key.Keyer[InputKey], StateKey comparable, InputKey comparable](n *NFA[State, Input, StateKey, InputKey]) *AtomicNFA[State, Input, StateKey, InputKey] {
	return &AtomicNFA[State, Input, StateKey, InputKey]{n: n}
}

// Delta applies δ to a state on a single input.
func (a *AtomicNFA[State, Input, StateKey, InputKey]) Delta(s State, sym Input) []State {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.n.Delta(s, sym)
}

// DeltaStar applies δ repeatedly over a word. It implements δ*.
func (a *AtomicNFA[State, Input, StateKey, InputKey]) DeltaStar(s State, w []Input) []State {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.n.DeltaStar(s, w)
}

// DeltaSet applies δ to a set of states on a input.
func (a *AtomicNFA[State, Input, StateKey, InputKey]) DeltaSet(states []State, sym Input) []State {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.n.DeltaSet(states, sym)
}

// DeltaStarSet applies δ* to a set of states over a word.
func (a *AtomicNFA[State, Input, StateKey, InputKey]) DeltaStarSet(states []State, w []Input) []State {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.n.DeltaStarSet(states, w)
}

// Epsilon applies δ to a state on ε.
func (a *AtomicNFA[State, Input, StateKey, InputKey]) Epsilon(s State) []State {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.n.Epsilon(s)
}

// EpsilonClosure returns ε-closure(s).
func (a *AtomicNFA[State, Input, StateKey, InputKey]) EpsilonClosure(s State) []State {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.n.EpsilonClosure(s)
}

// EpsilonSet applies δ to a set of states on ε.
func (a *AtomicNFA[State, Input, StateKey, InputKey]) EpsilonSet(states []State) []State {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.n.EpsilonSet(states)
}

// EpsilonClosureSet returns ε-closure(Q).
func (a *AtomicNFA[State, Input, StateKey, InputKey]) EpsilonClosureSet(states []State) []State {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.n.EpsilonClosureSet(states)
}

// Start returns q₀, the start state.
func (a *AtomicNFA[State, Input, StateKey, InputKey]) Start() State {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.n.Start()
}

// StartSet returns {q₀}, the singleton set containing the start state.
func (a *AtomicNFA[State, Input, StateKey, InputKey]) StartSet() []State {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.n.StartSet()
}

// States returns Q, the set of all states.
func (a *AtomicNFA[State, Input, StateKey, InputKey]) States() []State {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.n.States()
}

// Alphabet returns Σ, the input alphabet.
func (a *AtomicNFA[State, Input, StateKey, InputKey]) Alphabet() []Input {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.n.Alphabet()
}

// Accepting returns F, the set of accepting states.
func (a *AtomicNFA[State, Input, StateKey, InputKey]) Accepting() []State {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.n.Accepting()
}

// Accepts reports whether the NFA accepts the given word.
func (a *AtomicNFA[State, Input, StateKey, InputKey]) Accepts(w []Input) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.n.Accepts(w)
}

// HasState reports whether state ∈ Q.
func (a *AtomicNFA[State, Input, StateKey, InputKey]) HasState(s State) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.n.HasState(s)
}

// HasInput reports whether input ∈ Σ.
func (a *AtomicNFA[State, Input, StateKey, InputKey]) HasInput(sym Input) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.n.HasInput(sym)
}

// IsAccepting reports whether any state in s is an accepting state.
func (a *AtomicNFA[State, Input, StateKey, InputKey]) IsAccepting(states []State) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.n.IsAccepting(states)
}

// SetStart sets q₀, the start state.
func (a *AtomicNFA[State, Input, StateKey, InputKey]) SetStart(s State) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.n.SetStart(s)
}

// AddStates inserts states into Q.
func (a *AtomicNFA[State, Input, StateKey, InputKey]) AddStates(states ...State) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.n.AddStates(states...)
}

// AddAlphabet inserts inputs into Σ.
func (a *AtomicNFA[State, Input, StateKey, InputKey]) AddAlphabet(inputs ...Input) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.n.AddAlphabet(inputs...)
}

// AddAccepting inserts states into F.
func (a *AtomicNFA[State, Input, StateKey, InputKey]) AddAccepting(states ...State) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.n.AddAccepting(states...)
}

// RemoveAccepting removes states from F.
func (a *AtomicNFA[State, Input, StateKey, InputKey]) RemoveAccepting(states ...State) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.n.RemoveAccepting(states...)
}

// SetDeltaer sets δ and ε.
func (a *AtomicNFA[State, Input, StateKey, InputKey]) SetDeltaer(deltaer Deltaer[StateKey, InputKey]) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.n.SetDeltaer(deltaer)
}
