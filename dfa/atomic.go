package dfa

import (
	"sync"

	"github.com/atahiraj/fsm/key"
)

// AtomicDFA is a thread-safe wrapper around DFA.
// Use NewAtomic to construct a non-nil wrapper.
type AtomicDFA[State key.Keyer[StateKey], Input key.Keyer[InputKey], StateKey comparable, InputKey comparable] struct {
	mu sync.RWMutex
	d  *DFA[State, Input, StateKey, InputKey]
}

// NewAtomic wraps a DFA with a mutex for concurrent access.
func NewAtomic[State key.Keyer[StateKey], Input key.Keyer[InputKey], StateKey comparable, InputKey comparable](d *DFA[State, Input, StateKey, InputKey]) *AtomicDFA[State, Input, StateKey, InputKey] {
	return &AtomicDFA[State, Input, StateKey, InputKey]{d: d}
}

// Delta applies δ(s, a).
func (a *AtomicDFA[State, Input, StateKey, InputKey]) Delta(s State, sym Input) (State, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.d.Delta(s, sym)
}

// DeltaStar applies δ repeatedly over a word.
func (a *AtomicDFA[State, Input, StateKey, InputKey]) DeltaStar(s State, word []Input) (State, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.d.DeltaStar(s, word)
}

// IsAccepting reports whether s ∈ F.
func (a *AtomicDFA[State, Input, StateKey, InputKey]) IsAccepting(s State) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.d.IsAccepting(s)
}

// Accepts reports whether the DFA accepts the given word.
func (a *AtomicDFA[State, Input, StateKey, InputKey]) Accepts(word []Input) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.d.Accepts(word)
}

// States returns Q, the set of all states.
func (a *AtomicDFA[State, Input, StateKey, InputKey]) States() []State {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.d.States()
}

// Alphabet returns Σ, the input alphabet.
func (a *AtomicDFA[State, Input, StateKey, InputKey]) Alphabet() []Input {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.d.Alphabet()
}

// Start returns q₀, the start state.
func (a *AtomicDFA[State, Input, StateKey, InputKey]) Start() State {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.d.Start()
}

// Accepting returns F, the set of accepting states.
func (a *AtomicDFA[State, Input, StateKey, InputKey]) Accepting() []State {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.d.Accepting()
}

// SetDeltaer sets δ, the primitive transition function.
func (a *AtomicDFA[State, Input, StateKey, InputKey]) SetDeltaer(deltaer Deltaer[StateKey, InputKey]) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.d.SetDeltaer(deltaer)
}

// SetStart sets q₀, the start state.
func (a *AtomicDFA[State, Input, StateKey, InputKey]) SetStart(state State) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.d.SetStart(state)
}

// AddStates inserts states into Q.
func (a *AtomicDFA[State, Input, StateKey, InputKey]) AddStates(states ...State) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.d.AddStates(states...)
}

// AddAlphabet inserts inputs into Σ.
func (a *AtomicDFA[State, Input, StateKey, InputKey]) AddAlphabet(inputs ...Input) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.d.AddAlphabet(inputs...)
}

// AddAccepting inserts states into F.
func (a *AtomicDFA[State, Input, StateKey, InputKey]) AddAccepting(states ...State) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.d.AddAccepting(states...)
}

// RemoveAccepting removes states from F.
func (a *AtomicDFA[State, Input, StateKey, InputKey]) RemoveAccepting(states ...State) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.d.RemoveAccepting(states...)
}

// HasState reports whether state ∈ Q.
func (a *AtomicDFA[State, Input, StateKey, InputKey]) HasState(state State) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.d.HasState(state)
}

// HasInput reports whether input ∈ Σ.
func (a *AtomicDFA[State, Input, StateKey, InputKey]) HasInput(input Input) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.d.HasInput(input)
}
