package nfa

import (
	"errors"

	"github.com/stnhrsprkwns/fsm/graph"
	"github.com/stnhrsprkwns/fsm/internal/set"
)

// BuildAtomic returns a thread-safe NFA backed by the builder's graph.
func (b *Builder[State, Symbol]) BuildAtomic() (*AtomicNFA[State, Symbol], error) {
	n, err := b.Build()
	if err != nil {
		return nil, err
	}
	return NewAtomic(n), nil
}

type graphDeltaer[State comparable, Symbol comparable] struct {
	g graphLike[State, Symbol]
}

func (d graphDeltaer[State, Symbol]) Delta(state State, symbol Symbol) []State {
	return d.g.Delta(state, Sym(symbol))
}

func (d graphDeltaer[State, Symbol]) Epsilon(state State) []State {
	return d.g.Delta(state, Epsilon[Symbol]())
}

type graphLike[State comparable, Symbol comparable] interface {
	Delta(from State, label Label[Symbol]) []State
}

// Label represents an input symbol or ε for graph-backed NFAs.
type Label[Symbol comparable] struct {
	Symbol    Symbol
	IsEpsilon bool
}

// Sym constructs a label for a concrete symbol.
func Sym[Symbol comparable](symbol Symbol) Label[Symbol] {
	return Label[Symbol]{Symbol: symbol}
}

// Epsilon constructs a label for ε.
func Epsilon[Symbol comparable]() Label[Symbol] {
	return Label[Symbol]{IsEpsilon: true}
}

// Builder constructs an NFA from (Q, Σ, δ, q₀, F) using a graph backend.
type Builder[State comparable, Symbol comparable] struct {
	g         *graph.Graph[State, Label[Symbol]]
	gLike     graphLike[State, Symbol]
	states    set.Set[State]  // Q
	alphabet  set.Set[Symbol] // Σ
	start     State           // q₀
	accepting set.Set[State]  // F
}

// NewBuilder creates an empty NFA builder.
func NewBuilder[State comparable, Symbol comparable]() *Builder[State, Symbol] {
	return &Builder[State, Symbol]{
		g: graph.New[State, Label[Symbol]](),
	}
}

// SetStart sets q₀, the start state.
func (b *Builder[State, Symbol]) SetStart(state State) {
	b.start = state
	b.states.Add(state)
}

// AddStates inserts states into Q.
func (b *Builder[State, Symbol]) AddStates(states ...State) {
	b.states.Add(states...)
}

// AddAlphabet inserts symbols into Σ.
func (b *Builder[State, Symbol]) AddAlphabet(symbols ...Symbol) {
	b.alphabet.Add(symbols...)
}

// AddAccepting inserts states into F.
func (b *Builder[State, Symbol]) AddAccepting(states ...State) {
	b.accepting.Add(states...)
	b.states.Add(states...)
}

// WithGraph replaces the builder's graph for Build and shares it.
// Preconditions: g is non-nil.
func (b *Builder[State, Symbol]) WithGraph(g graphLike[State, Symbol]) *Builder[State, Symbol] {
	b.gLike = g
	return b
}

// Transition inserts (from, a, to) into δ.
func (b *Builder[State, Symbol]) Transition(from State, symbol Symbol, to State) {
	b.states.Add(from, to)
	b.alphabet.Add(symbol)
	b.g.AddEdge(from, to, Sym(symbol))
}

// Epsilon inserts (from, ε, to) into δ.
func (b *Builder[State, Symbol]) Epsilon(from State, to State) {
	b.states.Add(from, to)
	b.g.AddEdge(from, to, Epsilon[Symbol]())
}

// Build returns an NFA with δ backed by the builder's graph.
// Returns an error if the graph is missing.
func (b *Builder[State, Symbol]) Build() (*NFA[State, Symbol], error) {
	if b.gLike == nil && b.g == nil {
		return nil, errors.New("nfa builder: graph is nil")
	}
	start := b.start
	var gg graphLike[State, Symbol] = b.g
	if b.gLike != nil {
		gg = b.gLike
	}
	return New(Config[State, Symbol]{
		States:    b.states.Clone().Slice(),
		Alphabet:  b.alphabet.Clone().Slice(),
		Start:     start,
		Accepting: b.accepting.Clone().Slice(),
		Deltaer:   graphDeltaer[State, Symbol]{g: gg},
	}), nil
}
