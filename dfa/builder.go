package dfa

import (
	"errors"
	"fmt"

	"github.com/stnhrsprkwns/fsm/graph"
	"github.com/stnhrsprkwns/fsm/internal/set"
)

type graphLike[State comparable, Symbol comparable] interface {
	Delta(from State, label Symbol) []State
}

type graphDeltaer[State comparable, Symbol comparable] struct {
	g graphLike[State, Symbol]
}

func (d graphDeltaer[State, Symbol]) Delta(state State, symbol Symbol) (State, bool) {
	next := d.g.Delta(state, symbol)
	if len(next) != 1 {
		panic("expected exactly one next state")
	}
	return next[0], true
}

// Builder constructs a DFA from (Q, Σ, δ, q₀, F) using a graph backend.
type Builder[State comparable, Symbol comparable] struct {
	g         *graph.Graph[State, Symbol] // internal graph built by builder
	gLike     graphLike[State, Symbol]    // provided graphLike by user
	states    set.Set[State]              // Q
	alphabet  set.Set[Symbol]             // Σ
	start     State                       // q₀
	accepting set.Set[State]              // F
}

// NewBuilder creates an empty DFA builder.
func NewBuilder[State comparable, Symbol comparable]() *Builder[State, Symbol] {
	return &Builder[State, Symbol]{
		g: graph.New[State, Symbol](),
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

// Transition inserts (from, a, to) into δ. It rejects nondeterminism.
func (b *Builder[State, Symbol]) Transition(from State, symbol Symbol, to State) error {
	b.states.Add(from, to)
	b.alphabet.Add(symbol)
	existing := b.g.Delta(from, symbol)
	if len(existing) == 0 {
		b.g.AddEdge(from, to, symbol)
		return nil
	}
	if len(existing) == 1 && existing[0] == to {
		return nil
	}
	return fmt.Errorf("dfa: transition already defined for (%v, %v)", from, symbol)
}

// Build returns a DFA with δ backed by the builder's graph.
// Returns an error if the graph is missing.
func (b *Builder[State, Symbol]) Build() (*DFA[State, Symbol], error) {
	if b.gLike == nil && b.g == nil {
		return nil, errors.New("dfa builder: graph is nil")
	}
	start := b.start
	var g graphLike[State, Symbol] = b.g
	if b.gLike != nil {
		g = b.gLike
	}

	return New(Config[State, Symbol]{
		States:    b.states.Clone().Slice(),
		Alphabet:  b.alphabet.Clone().Slice(),
		Start:     start,
		Accepting: b.accepting.Clone().Slice(),
		Deltaer:   graphDeltaer[State, Symbol]{g: g},
	}), nil
}

// BuildAtomic returns a thread-safe DFA backed by the builder's graph.
func (b *Builder[State, Symbol]) BuildAtomic() (*AtomicDFA[State, Symbol], error) {
	d, err := b.Build()
	if err != nil {
		return nil, err
	}
	return NewAtomic(d), nil
}
