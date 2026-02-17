package dfa

import (
	"github.com/stnhrsprkwns/fsm/internal/set"
	"github.com/stnhrsprkwns/fsm/key"
)

// Deltaer abstracts the DFA primitive transition function δ.
//
// δ : Q × Σ → Q
//
// Q and Σ are represented by comparable keys.
type Deltaer[StateKey comparable, SymbolKey comparable] interface {
	Delta(state StateKey, symbol SymbolKey) (StateKey, bool)
}

// DeltaFunc adapts a plain function to a Deltaer.
type DeltaFunc[StateKey comparable, SymbolKey comparable] func(state StateKey, symbol SymbolKey) (StateKey, bool)

func (f DeltaFunc[StateKey, SymbolKey]) Delta(state StateKey, symbol SymbolKey) (StateKey, bool) {
	return f(state, symbol)
}

// DFA models a deterministic finite automaton.
//
// The field Deltaer holds the primitive δ over keys.
type DFA[State key.Keyer[StateKey], Symbol key.Keyer[SymbolKey], StateKey comparable, SymbolKey comparable] struct {
	states     set.Set[StateKey]            // Q, all state keys.
	alphabet   set.Set[SymbolKey]           // Σ, all symbol keys.
	deltaer    Deltaer[StateKey, SymbolKey] // δ, primitive transition over keys.
	start      State                        // q₀, start state value.
	accepting  set.Set[StateKey]            // F, accepting state keys.
	stateByKey map[StateKey]State
	symByKey   map[SymbolKey]Symbol
}

// Config holds the data needed to construct a DFA (Q, Σ, δ, q₀, F).
type Config[State key.Keyer[StateKey], Symbol key.Keyer[SymbolKey], StateKey comparable, SymbolKey comparable] struct {
	// States is Q, the set of all states.
	States []State
	// Alphabet is Σ, the input alphabet.
	Alphabet []Symbol
	// Start is q₀, the start state.
	Start State
	// Accepting is F, the accepting states.
	Accepting []State
	// Deltaer provides δ, the primitive transition function over keys.
	Deltaer Deltaer[StateKey, SymbolKey]
}

// New constructs a DFA from (Q, Σ, δ, q₀, F).
func New[State key.Keyer[StateKey], Symbol key.Keyer[SymbolKey], StateKey comparable, SymbolKey comparable](cfg Config[State, Symbol, StateKey, SymbolKey]) *DFA[State, Symbol, StateKey, SymbolKey] {
	d := &DFA[State, Symbol, StateKey, SymbolKey]{
		deltaer:    cfg.Deltaer,
		start:      cfg.Start,
		stateByKey: make(map[StateKey]State, len(cfg.States)),
		symByKey:   make(map[SymbolKey]Symbol, len(cfg.Alphabet)),
	}
	d.AddStates(cfg.States...)
	d.AddAlphabet(cfg.Alphabet...)
	d.AddAccepting(cfg.Accepting...)
	return d
}

// Delta applies δ(s, a).
func (d *DFA[State, Symbol, StateKey, SymbolKey]) Delta(s State, a Symbol) (State, bool) {
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

// DeltaStar applies δ repeatedly over a word (sequence of symbols).
func (d *DFA[State, Symbol, StateKey, SymbolKey]) DeltaStar(s State, word []Symbol) (State, bool) {
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
func (d *DFA[State, Symbol, StateKey, SymbolKey]) IsAccepting(s State) bool {
	return d.accepting.Has(s.Key())
}

// Accepts reports whether the DFA accepts the given word.
func (d *DFA[State, Symbol, StateKey, SymbolKey]) Accepts(word []Symbol) bool {
	end, ok := d.DeltaStar(d.start, word)
	return ok && d.IsAccepting(end)
}

// States returns Q, the set of all states.
func (d *DFA[State, Symbol, StateKey, SymbolKey]) States() []State {
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
func (d *DFA[State, Symbol, StateKey, SymbolKey]) Alphabet() []Symbol {
	keys := d.alphabet.Clone().Slice()
	out := make([]Symbol, 0, len(keys))
	for _, key := range keys {
		if sym, ok := d.symByKey[key]; ok {
			out = append(out, sym)
		}
	}
	return out
}

// Start returns q₀, the start state.
func (d *DFA[State, Symbol, StateKey, SymbolKey]) Start() State {
	return d.start
}

// Accepting returns F, the set of accepting states.
func (d *DFA[State, Symbol, StateKey, SymbolKey]) Accepting() []State {
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
func (d *DFA[State, Symbol, StateKey, SymbolKey]) SetDeltaer(deltaer Deltaer[StateKey, SymbolKey]) {
	d.deltaer = deltaer
}

// SetStart sets q₀, the start state.
func (d *DFA[State, Symbol, StateKey, SymbolKey]) SetStart(state State) {
	d.start = state
}

// AddStates inserts states into Q.
func (d *DFA[State, Symbol, StateKey, SymbolKey]) AddStates(states ...State) {
	for _, state := range states {
		key := state.Key()
		d.states.Add(key)
		d.stateByKey[key] = state
	}
}

// AddAlphabet inserts symbols into Σ.
func (d *DFA[State, Symbol, StateKey, SymbolKey]) AddAlphabet(symbols ...Symbol) {
	for _, sym := range symbols {
		key := sym.Key()
		d.alphabet.Add(key)
		d.symByKey[key] = sym
	}
}

// AddAccepting inserts states into F.
func (d *DFA[State, Symbol, StateKey, SymbolKey]) AddAccepting(states ...State) {
	for _, state := range states {
		key := state.Key()
		d.accepting.Add(key)
		d.stateByKey[key] = state
	}
}

// RemoveAccepting removes states from F.
func (d *DFA[State, Symbol, StateKey, SymbolKey]) RemoveAccepting(states ...State) {
	keys := make([]StateKey, 0, len(states))
	for _, state := range states {
		keys = append(keys, state.Key())
	}
	d.accepting.Remove(keys...)
}

// HasState reports whether state ∈ Q.
func (d *DFA[State, Symbol, StateKey, SymbolKey]) HasState(state State) bool {
	return d.states.Has(state.Key())
}

// HasSymbol reports whether symbol ∈ Σ.
func (d *DFA[State, Symbol, StateKey, SymbolKey]) HasSymbol(symbol Symbol) bool {
	return d.alphabet.Has(symbol.Key())
}
