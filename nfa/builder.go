package nfa

import (
	"errors"

	"github.com/stnhrsprkwns/fsm/graph"
	"github.com/stnhrsprkwns/fsm/internal/set"
	"github.com/stnhrsprkwns/fsm/key"
)

type keyGraphLike[StateKey comparable, SymbolKey comparable] interface {
	Delta(from StateKey, label Label[SymbolKey]) []StateKey
}

type graphLike[State any, Symbol any] interface {
	Delta(from State, symbol Symbol) []State
	Epsilon(from State) []State
}

// BuildAtomic returns a thread-safe NFA backed by the builder's graph.
func (b *Builder[State, Symbol, StateKey, SymbolKey]) BuildAtomic() (*AtomicNFA[State, Symbol, StateKey, SymbolKey], error) {
	n, err := b.Build()
	if err != nil {
		return nil, err
	}
	return NewAtomic(n), nil
}

type graphDeltaer[StateKey comparable, SymbolKey comparable] struct {
	g keyGraphLike[StateKey, SymbolKey]
}

func (d graphDeltaer[StateKey, SymbolKey]) Delta(state StateKey, symbol SymbolKey) []StateKey {
	return d.g.Delta(state, Sym(symbol))
}

func (d graphDeltaer[StateKey, SymbolKey]) Epsilon(state StateKey) []StateKey {
	return d.g.Delta(state, Epsilon[SymbolKey]())
}

type valueGraphDeltaer[State key.Keyer[StateKey], Symbol key.Keyer[SymbolKey], StateKey comparable, SymbolKey comparable] struct {
	g          graphLike[State, Symbol]
	stateByKey map[StateKey]State
	symByKey   map[SymbolKey]Symbol
}

func (d valueGraphDeltaer[State, Symbol, StateKey, SymbolKey]) Delta(state StateKey, symbol SymbolKey) []StateKey {
	from, ok := d.stateByKey[state]
	if !ok {
		return nil
	}
	label, ok := d.symByKey[symbol]
	if !ok {
		return nil
	}
	next := d.g.Delta(from, label)
	out := make([]StateKey, 0, len(next))
	for _, to := range next {
		out = append(out, to.Key())
	}
	return out
}

func (d valueGraphDeltaer[State, Symbol, StateKey, SymbolKey]) Epsilon(state StateKey) []StateKey {
	from, ok := d.stateByKey[state]
	if !ok {
		return nil
	}
	next := d.g.Epsilon(from)
	out := make([]StateKey, 0, len(next))
	for _, to := range next {
		out = append(out, to.Key())
	}
	return out
}

// Label represents an input symbol key or ε for graph-backed NFAs.
type Label[SymbolKey comparable] struct {
	Symbol    SymbolKey
	IsEpsilon bool
}

// Sym constructs a label for a concrete symbol key.
func Sym[SymbolKey comparable](symbol SymbolKey) Label[SymbolKey] {
	return Label[SymbolKey]{Symbol: symbol}
}

// Epsilon constructs a label for ε.
func Epsilon[SymbolKey comparable]() Label[SymbolKey] {
	return Label[SymbolKey]{IsEpsilon: true}
}

// Builder constructs an NFA from (Q, Σ, δ, q₀, F) using a graph backend.
type Builder[State key.Keyer[StateKey], Symbol key.Keyer[SymbolKey], StateKey comparable, SymbolKey comparable] struct {
	g         *graph.Graph[StateKey, Label[SymbolKey]]
	gLike     graphLike[State, Symbol]
	states    set.Set[StateKey]  // Q
	alphabet  set.Set[SymbolKey] // Σ
	start     State              // q₀
	accepting set.Set[StateKey]  // F

	stateByKey map[StateKey]State
	symByKey   map[SymbolKey]Symbol
}

// NewBuilder creates an empty NFA builder.
func NewBuilder[State key.Keyer[StateKey], Symbol key.Keyer[SymbolKey], StateKey comparable, SymbolKey comparable]() *Builder[State, Symbol, StateKey, SymbolKey] {
	return &Builder[State, Symbol, StateKey, SymbolKey]{
		g:          graph.New[StateKey, Label[SymbolKey]](),
		stateByKey: make(map[StateKey]State),
		symByKey:   make(map[SymbolKey]Symbol),
	}
}

// SetStart sets q₀, the start state.
func (b *Builder[State, Symbol, StateKey, SymbolKey]) SetStart(state State) {
	b.start = state
	key := state.Key()
	b.stateByKey[key] = state
	b.states.Add(key)
}

// AddStates inserts states into Q.
func (b *Builder[State, Symbol, StateKey, SymbolKey]) AddStates(states ...State) {
	for _, state := range states {
		key := state.Key()
		b.stateByKey[key] = state
		b.states.Add(key)
	}
}

// AddAlphabet inserts symbols into Σ.
func (b *Builder[State, Symbol, StateKey, SymbolKey]) AddAlphabet(symbols ...Symbol) {
	for _, symbol := range symbols {
		key := symbol.Key()
		b.symByKey[key] = symbol
		b.alphabet.Add(key)
	}
}

// AddAccepting inserts states into F.
func (b *Builder[State, Symbol, StateKey, SymbolKey]) AddAccepting(states ...State) {
	for _, state := range states {
		key := state.Key()
		b.stateByKey[key] = state
		b.accepting.Add(key)
		b.states.Add(key)
	}
}

// WithGraph replaces the builder's graph for Build and shares it.
// Preconditions: g is non-nil.
func (b *Builder[State, Symbol, StateKey, SymbolKey]) WithGraph(g graphLike[State, Symbol]) *Builder[State, Symbol, StateKey, SymbolKey] {
	b.gLike = g
	return b
}

// Transition inserts (from, a, to) into δ.
func (b *Builder[State, Symbol, StateKey, SymbolKey]) Transition(from State, symbol Symbol, to State) {
	fromKey := from.Key()
	toKey := to.Key()
	symKey := symbol.Key()

	b.stateByKey[fromKey] = from
	b.stateByKey[toKey] = to
	b.symByKey[symKey] = symbol

	b.states.Add(fromKey, toKey)
	b.alphabet.Add(symKey)
	b.g.AddEdge(fromKey, toKey, Sym(symKey))
}

// Epsilon inserts (from, ε, to) into δ.
func (b *Builder[State, Symbol, StateKey, SymbolKey]) Epsilon(from State, to State) {
	fromKey := from.Key()
	toKey := to.Key()

	b.stateByKey[fromKey] = from
	b.stateByKey[toKey] = to
	b.states.Add(fromKey, toKey)
	b.g.AddEdge(fromKey, toKey, Epsilon[SymbolKey]())
}

func valuesFromKeys[K comparable, V any](keys []K, byKey map[K]V) []V {
	out := make([]V, 0, len(keys))
	for _, key := range keys {
		if value, ok := byKey[key]; ok {
			out = append(out, value)
		}
	}
	return out
}

// Build returns an NFA with δ backed by the builder's graph.
// Returns an error if the graph is missing.
func (b *Builder[State, Symbol, StateKey, SymbolKey]) Build() (*NFA[State, Symbol, StateKey, SymbolKey], error) {
	if b.gLike == nil && b.g == nil {
		return nil, errors.New("nfa builder: graph is nil")
	}
	if b.gLike != nil {
		if b.states.Len() == 0 {
			return nil, errors.New("nfa builder: WithGraph requires states; add Q with AddStates")
		}
		if b.alphabet.Len() == 0 {
			return nil, errors.New("nfa builder: WithGraph requires alphabet; add Σ with AddAlphabet")
		}
	}

	var delta Deltaer[StateKey, SymbolKey] = graphDeltaer[StateKey, SymbolKey]{g: b.g}
	if b.gLike != nil {
		delta = valueGraphDeltaer[State, Symbol, StateKey, SymbolKey]{
			g:          b.gLike,
			stateByKey: b.stateByKey,
			symByKey:   b.symByKey,
		}
	}

	return New(Config[State, Symbol, StateKey, SymbolKey]{
		States:    valuesFromKeys(b.states.Clone().Slice(), b.stateByKey),
		Alphabet:  valuesFromKeys(b.alphabet.Clone().Slice(), b.symByKey),
		Start:     b.start,
		Accepting: valuesFromKeys(b.accepting.Clone().Slice(), b.stateByKey),
		Deltaer:   delta,
	}), nil
}
