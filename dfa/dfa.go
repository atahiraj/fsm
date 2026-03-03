package dfa

import (
	"github.com/atahiraj/fsm/internal/set"
	"github.com/atahiraj/fsm/key"
)

// Deltaer abstracts the DFA primitive transition function δ.
//
// δ : Q × Σ → Q
//
// Q and Σ are represented by comparable keys.
type Deltaer[StateKey comparable, InputKey comparable] interface {
	Delta(state StateKey, input InputKey) (StateKey, bool)
}

// DeltaFunc adapts a plain function to a Deltaer.
type DeltaFunc[StateKey comparable, InputKey comparable] func(state StateKey, input InputKey) (StateKey, bool)

func (f DeltaFunc[StateKey, InputKey]) Delta(state StateKey, input InputKey) (StateKey, bool) {
	return f(state, input)
}

// DFA models a deterministic finite automaton.
//
// The field Deltaer holds the primitive δ over keys.
type DFA[State key.Keyer[StateKey], Input key.Keyer[InputKey], StateKey comparable, InputKey comparable] struct {
	states     set.Set[StateKey]           // Q, all state keys.
	alphabet   set.Set[InputKey]           // Σ, all input keys.
	deltaer    Deltaer[StateKey, InputKey] // δ, primitive transition over keys.
	start      State                       // q₀, start state value.
	accepting  set.Set[StateKey]           // F, accepting state keys.
	stateByKey map[StateKey]State
	symByKey   map[InputKey]Input
}

// Config holds the data needed to construct a DFA (Q, Σ, δ, q₀, F).
type Config[State key.Keyer[StateKey], Input key.Keyer[InputKey], StateKey comparable, InputKey comparable] struct {
	// States is Q, the set of all states.
	States []State
	// Alphabet is Σ, the input alphabet.
	Alphabet []Input
	// Start is q₀, the start state.
	Start State
	// Accepting is F, the accepting states.
	Accepting []State
	// Deltaer provides δ, the primitive transition function over keys.
	Deltaer Deltaer[StateKey, InputKey]
}

// New constructs a DFA from (Q, Σ, δ, q₀, F).
func New[State key.Keyer[StateKey], Input key.Keyer[InputKey], StateKey comparable, InputKey comparable](cfg Config[State, Input, StateKey, InputKey]) *DFA[State, Input, StateKey, InputKey] {
	d := &DFA[State, Input, StateKey, InputKey]{
		deltaer:    cfg.Deltaer,
		start:      cfg.Start,
		stateByKey: make(map[StateKey]State, len(cfg.States)),
		symByKey:   make(map[InputKey]Input, len(cfg.Alphabet)),
	}
	d.AddStates(cfg.States...)
	d.AddAlphabet(cfg.Alphabet...)
	d.AddAccepting(cfg.Accepting...)
	return d
}

// Delta applies δ(s, a).
func (d *DFA[State, Input, StateKey, InputKey]) Delta(s State, a Input) (State, bool) {
	var zero State
	nextKey, ok := d.deltaer.Delta(s.Key(), a.Key())
	if !ok || !d.states.Has(nextKey) {
		return zero, false
	}
	next, ok := d.stateByKey[nextKey]
	if !ok {
		return zero, false
	}
	return next, true
}

// DeltaStar applies δ repeatedly over a word (sequence of inputs).
func (d *DFA[State, Input, StateKey, InputKey]) DeltaStar(s State, word []Input) (State, bool) {
	cur := s
	for _, a := range word {
		next, ok := d.Delta(cur, a)
		if !ok {
			var zero State
			return zero, false
		}
		cur = next
	}
	return cur, true
}

// IsAccepting reports whether s ∈ F.
func (d *DFA[State, Input, StateKey, InputKey]) IsAccepting(s State) bool {
	return d.accepting.Has(s.Key())
}

// Accepts reports whether the DFA accepts the given word.
func (d *DFA[State, Input, StateKey, InputKey]) Accepts(word []Input) bool {
	end, ok := d.DeltaStar(d.start, word)
	return ok && d.IsAccepting(end)
}

// States returns Q, the set of all states.
func (d *DFA[State, Input, StateKey, InputKey]) States() []State {
	keys := d.states.Clone().Slice()
	out := make([]State, 0, len(keys))
	for _, key := range keys {
		if state, ok := d.stateByKey[key]; ok {
			out = append(out, state)
		}
	}
	return out
}

// Alphabet returns Σ, the input alphabet.
func (d *DFA[State, Input, StateKey, InputKey]) Alphabet() []Input {
	keys := d.alphabet.Clone().Slice()
	out := make([]Input, 0, len(keys))
	for _, key := range keys {
		if sym, ok := d.symByKey[key]; ok {
			out = append(out, sym)
		}
	}
	return out
}

// Start returns q₀, the start state.
func (d *DFA[State, Input, StateKey, InputKey]) Start() State {
	return d.start
}

// Accepting returns F, the set of accepting states.
func (d *DFA[State, Input, StateKey, InputKey]) Accepting() []State {
	keys := d.accepting.Clone().Slice()
	out := make([]State, 0, len(keys))
	for _, key := range keys {
		if state, ok := d.stateByKey[key]; ok {
			out = append(out, state)
		}
	}
	return out
}

// SetDeltaer sets δ, the primitive transition function.
func (d *DFA[State, Input, StateKey, InputKey]) SetDeltaer(deltaer Deltaer[StateKey, InputKey]) {
	d.deltaer = deltaer
}

// SetStart sets q₀, the start state.
func (d *DFA[State, Input, StateKey, InputKey]) SetStart(state State) {
	d.start = state
}

// AddStates inserts states into Q.
func (d *DFA[State, Input, StateKey, InputKey]) AddStates(states ...State) {
	for _, state := range states {
		key := state.Key()
		d.states.Add(key)
		d.stateByKey[key] = state
	}
}

// AddAlphabet inserts inputs into Σ.
func (d *DFA[State, Input, StateKey, InputKey]) AddAlphabet(inputs ...Input) {
	for _, sym := range inputs {
		key := sym.Key()
		d.alphabet.Add(key)
		d.symByKey[key] = sym
	}
}

// AddAccepting inserts states into F.
func (d *DFA[State, Input, StateKey, InputKey]) AddAccepting(states ...State) {
	for _, state := range states {
		key := state.Key()
		d.accepting.Add(key)
		d.stateByKey[key] = state
	}
}

// RemoveAccepting removes states from F.
func (d *DFA[State, Input, StateKey, InputKey]) RemoveAccepting(states ...State) {
	keys := make([]StateKey, 0, len(states))
	for _, state := range states {
		keys = append(keys, state.Key())
	}
	d.accepting.Remove(keys...)
}

// HasState reports whether state ∈ Q.
func (d *DFA[State, Input, StateKey, InputKey]) HasState(state State) bool {
	return d.states.Has(state.Key())
}

// HasInput reports whether input ∈ Σ.
func (d *DFA[State, Input, StateKey, InputKey]) HasInput(input Input) bool {
	return d.alphabet.Has(input.Key())
}
