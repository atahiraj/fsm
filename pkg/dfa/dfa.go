package dfa

import "github.com/stnhrsprkwns/fsm/pkg/set"

// Deltaer abstracts the DFA primitive transition function δ.
//
// δ : Q × Σ → Q
type Deltaer[State comparable, Symbol comparable] interface {
	Delta(state State, symbol Symbol) (State, bool)
}

// DeltaFunc adapts a plain function to a Deltaer.
type DeltaFunc[State comparable, Symbol comparable] func(state State, symbol Symbol) (State, bool)

func (f DeltaFunc[State, Symbol]) Delta(state State, symbol Symbol) (State, bool) {
	return f(state, symbol)
}

// DFA models a deterministic finite automaton.
//
// The field Deltaer holds the primitive δ.
type DFA[State comparable, Symbol comparable] struct {
	states    set.Set[State]         // Q, all states.
	alphabet  set.Set[Symbol]        // Σ, all symbols.
	deltaer   Deltaer[State, Symbol] // δ, primitive transition.
	start     State                  // q₀, start state.
	accepting set.Set[State]         // F, accepting states.
}

// Config holds the data needed to construct a DFA (Q, Σ, δ, q₀, F).
type Config[State comparable, Symbol comparable] struct {
	// States is Q, the set of all states.
	States []State
	// Alphabet is Σ, the input alphabet.
	Alphabet []Symbol
	// Start is q₀, the start state.
	Start State
	// Accepting is F, the accepting states.
	Accepting []State
	// Deltaer provides δ, the primitive transition function.
	Deltaer Deltaer[State, Symbol]
}

// New constructs a DFA from (Q, Σ, δ, q₀, F).
func New[State comparable, Symbol comparable](cfg Config[State, Symbol]) *DFA[State, Symbol] {
	return &DFA[State, Symbol]{
		states:    *set.New(cfg.States...),
		alphabet:  *set.New(cfg.Alphabet...),
		deltaer:   cfg.Deltaer,
		start:     cfg.Start,
		accepting: *set.New(cfg.Accepting...),
	}
}

// Delta applies δ(s, a).
func (d *DFA[State, Symbol]) Delta(s State, a Symbol) (State, bool) {
	var zero State
	newState, ok := d.deltaer.Delta(s, a)
	if !ok || !d.states.Has(newState) {
		return zero, false
	}
	return newState, true
}

// DeltaStar applies δ repeatedly over a word (sequence of symbols).
func (d *DFA[State, Symbol]) DeltaStar(s State, word []Symbol) (State, bool) {
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
func (d *DFA[State, Symbol]) IsAccepting(s State) bool {
	return d.accepting.Has(s)
}

// Accepts reports whether the DFA accepts the given word.
func (d *DFA[State, Symbol]) Accepts(word []Symbol) bool {
	end, ok := d.DeltaStar(d.start, word)
	return ok && d.IsAccepting(end)
}

// States returns Q, the set of all states.
func (d *DFA[State, Symbol]) States() []State {
	return d.states.Clone().Slice()
}

// Alphabet returns Σ, the input alphabet.
func (d *DFA[State, Symbol]) Alphabet() []Symbol {
	return d.alphabet.Clone().Slice()
}

// Start returns q₀, the start state.
func (d *DFA[State, Symbol]) Start() State {
	return d.start
}

// Accepting returns F, the set of accepting states.
func (d *DFA[State, Symbol]) Accepting() []State {
	return d.accepting.Clone().Slice()
}

// SetDeltaer sets δ, the primitive transition function.
func (d *DFA[State, Symbol]) SetDeltaer(deltaer Deltaer[State, Symbol]) {
	d.deltaer = deltaer
}

// SetStart sets q₀, the start state.
func (d *DFA[State, Symbol]) SetStart(state State) {
	d.start = state
}

// AddStates inserts states into Q.
func (d *DFA[State, Symbol]) AddStates(states ...State) {
	d.states.Add(states...)
}

// AddAlphabet inserts symbols into Σ.
func (d *DFA[State, Symbol]) AddAlphabet(symbols ...Symbol) {
	d.alphabet.Add(symbols...)
}

// AddAccepting inserts states into F.
func (d *DFA[State, Symbol]) AddAccepting(states ...State) {
	d.accepting.Add(states...)
}

// RemoveAccepting removes states from F.
func (d *DFA[State, Symbol]) RemoveAccepting(states ...State) {
	d.accepting.Remove(states...)
}

// HasState reports whether state ∈ Q.
func (d *DFA[State, Symbol]) HasState(state State) bool {
	return d.states.Has(state)
}

// HasSymbol reports whether symbol ∈ Σ.
func (d *DFA[State, Symbol]) HasSymbol(symbol Symbol) bool {
	return d.alphabet.Has(symbol)
}
