package nfa

import (
	"errors"

	"github.com/stnhrsprkwns/fsm/graph"
	"github.com/stnhrsprkwns/fsm/internal/set"
	"github.com/stnhrsprkwns/fsm/key"
)

type keyGraphLike[StateKey comparable, InputKey comparable] interface {
	Delta(from StateKey, label Label[InputKey]) []StateKey
}

type graphLike[State any, Input any] interface {
	Delta(from State, input Input) []State
	Epsilon(from State) []State
}

// BuildAtomic returns a thread-safe NFA backed by the builder's graph.
func (b *Builder[State, Input, StateKey, InputKey]) BuildAtomic() (*AtomicNFA[State, Input, StateKey, InputKey], error) {
	n, err := b.Build()
	if err != nil {
		return nil, err
	}
	return NewAtomic(n), nil
}

type graphDeltaer[StateKey comparable, InputKey comparable] struct {
	g keyGraphLike[StateKey, InputKey]
}

func (d graphDeltaer[StateKey, InputKey]) Delta(state StateKey, input InputKey) []StateKey {
	return d.g.Delta(state, Sym(input))
}

func (d graphDeltaer[StateKey, InputKey]) Epsilon(state StateKey) []StateKey {
	return d.g.Delta(state, Epsilon[InputKey]())
}

type valueGraphDeltaer[State key.Keyer[StateKey], Input key.Keyer[InputKey], StateKey comparable, InputKey comparable] struct {
	g          graphLike[State, Input]
	stateByKey map[StateKey]State
	symByKey   map[InputKey]Input
}

func (d valueGraphDeltaer[State, Input, StateKey, InputKey]) Delta(state StateKey, input InputKey) []StateKey {
	from, ok := d.stateByKey[state]
	if !ok {
		return nil
	}
	label, ok := d.symByKey[input]
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

func (d valueGraphDeltaer[State, Input, StateKey, InputKey]) Epsilon(state StateKey) []StateKey {
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

// Label represents an input input key or ε for graph-backed NFAs.
type Label[InputKey comparable] struct {
	Input     InputKey
	IsEpsilon bool
}

// Sym constructs a label for a concrete input key.
func Sym[InputKey comparable](input InputKey) Label[InputKey] {
	return Label[InputKey]{Input: input}
}

// Epsilon constructs a label for ε.
func Epsilon[InputKey comparable]() Label[InputKey] {
	return Label[InputKey]{IsEpsilon: true}
}

// Builder constructs an NFA from (Q, Σ, δ, q₀, F) using a graph backend.
type Builder[State key.Keyer[StateKey], Input key.Keyer[InputKey], StateKey comparable, InputKey comparable] struct {
	g         *graph.Graph[StateKey, Label[InputKey]]
	gLike     graphLike[State, Input]
	states    set.Set[StateKey] // Q
	alphabet  set.Set[InputKey] // Σ
	start     State             // q₀
	accepting set.Set[StateKey] // F

	stateByKey map[StateKey]State
	symByKey   map[InputKey]Input
}

// NewBuilder creates an empty NFA builder.
func NewBuilder[State key.Keyer[StateKey], Input key.Keyer[InputKey], StateKey comparable, InputKey comparable]() *Builder[State, Input, StateKey, InputKey] {
	return &Builder[State, Input, StateKey, InputKey]{
		g:          graph.New[StateKey, Label[InputKey]](),
		stateByKey: make(map[StateKey]State),
		symByKey:   make(map[InputKey]Input),
	}
}

// SetStart sets q₀, the start state.
func (b *Builder[State, Input, StateKey, InputKey]) SetStart(state State) {
	b.start = state
	key := state.Key()
	b.stateByKey[key] = state
	b.states.Add(key)
}

// AddStates inserts states into Q.
func (b *Builder[State, Input, StateKey, InputKey]) AddStates(states ...State) {
	for _, state := range states {
		key := state.Key()
		b.stateByKey[key] = state
		b.states.Add(key)
	}
}

// AddAlphabet inserts inputs into Σ.
func (b *Builder[State, Input, StateKey, InputKey]) AddAlphabet(inputs ...Input) {
	for _, input := range inputs {
		key := input.Key()
		b.symByKey[key] = input
		b.alphabet.Add(key)
	}
}

// AddAccepting inserts states into F.
func (b *Builder[State, Input, StateKey, InputKey]) AddAccepting(states ...State) {
	for _, state := range states {
		key := state.Key()
		b.stateByKey[key] = state
		b.accepting.Add(key)
		b.states.Add(key)
	}
}

// WithGraph replaces the builder's graph for Build and shares it.
// Preconditions: g is non-nil.
func (b *Builder[State, Input, StateKey, InputKey]) WithGraph(g graphLike[State, Input]) *Builder[State, Input, StateKey, InputKey] {
	b.gLike = g
	return b
}

// Transition inserts (from, a, to) into δ.
func (b *Builder[State, Input, StateKey, InputKey]) Transition(from State, input Input, to State) {
	fromKey := from.Key()
	toKey := to.Key()
	symKey := input.Key()

	b.stateByKey[fromKey] = from
	b.stateByKey[toKey] = to
	b.symByKey[symKey] = input

	b.states.Add(fromKey, toKey)
	b.alphabet.Add(symKey)
	b.g.AddEdge(fromKey, toKey, Sym(symKey))
}

// Epsilon inserts (from, ε, to) into δ.
func (b *Builder[State, Input, StateKey, InputKey]) Epsilon(from State, to State) {
	fromKey := from.Key()
	toKey := to.Key()

	b.stateByKey[fromKey] = from
	b.stateByKey[toKey] = to
	b.states.Add(fromKey, toKey)
	b.g.AddEdge(fromKey, toKey, Epsilon[InputKey]())
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
func (b *Builder[State, Input, StateKey, InputKey]) Build() (*NFA[State, Input, StateKey, InputKey], error) {
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

	var delta Deltaer[StateKey, InputKey] = graphDeltaer[StateKey, InputKey]{g: b.g}
	if b.gLike != nil {
		delta = valueGraphDeltaer[State, Input, StateKey, InputKey]{
			g:          b.gLike,
			stateByKey: b.stateByKey,
			symByKey:   b.symByKey,
		}
	}

	return New(Config[State, Input, StateKey, InputKey]{
		States:    valuesFromKeys(b.states.Clone().Slice(), b.stateByKey),
		Alphabet:  valuesFromKeys(b.alphabet.Clone().Slice(), b.symByKey),
		Start:     b.start,
		Accepting: valuesFromKeys(b.accepting.Clone().Slice(), b.stateByKey),
		Deltaer:   delta,
	}), nil
}
