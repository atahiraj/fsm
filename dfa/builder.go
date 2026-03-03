package dfa

import (
	"errors"
	"fmt"

	"github.com/atahiraj/fsm/graph"
	"github.com/atahiraj/fsm/internal/set"
	"github.com/atahiraj/fsm/key"
)

type graphLike[State any, Input any] interface {
	Delta(from State, label Input) []State
}

type keyGraphLike[StateKey comparable, InputKey comparable] interface {
	Delta(from StateKey, label InputKey) []StateKey
}

type graphDeltaer[StateKey comparable, InputKey comparable] struct {
	g keyGraphLike[StateKey, InputKey]
}

func (d graphDeltaer[StateKey, InputKey]) Delta(state StateKey, input InputKey) (StateKey, bool) {
	next := d.g.Delta(state, input)
	if len(next) != 1 {
		panic("expected exactly one next state")
	}
	return next[0], true
}

type valueGraphDeltaer[State key.Keyer[StateKey], Input key.Keyer[InputKey], StateKey comparable, InputKey comparable] struct {
	g          graphLike[State, Input]
	stateByKey map[StateKey]State
	symByKey   map[InputKey]Input
}

func (d valueGraphDeltaer[State, Input, StateKey, InputKey]) Delta(state StateKey, input InputKey) (StateKey, bool) {
	from, ok := d.stateByKey[state]
	if !ok {
		var zero StateKey
		return zero, false
	}
	label, ok := d.symByKey[input]
	if !ok {
		var zero StateKey
		return zero, false
	}
	next := d.g.Delta(from, label)
	if len(next) != 1 {
		panic("expected exactly one next state")
	}
	return next[0].Key(), true
}

// Builder constructs a DFA from (Q, Σ, δ, q₀, F) using a graph backend.
type Builder[State key.Keyer[StateKey], Input key.Keyer[InputKey], StateKey comparable, InputKey comparable] struct {
	g         *graph.Graph[StateKey, InputKey] // internal graph built by builder
	gLike     graphLike[State, Input]          // provided graphLike by user
	states    set.Set[StateKey]                // Q
	alphabet  set.Set[InputKey]                // Σ
	start     State                            // q₀
	accepting set.Set[StateKey]                // F

	stateByKey map[StateKey]State
	symByKey   map[InputKey]Input
}

// NewBuilder creates an empty DFA builder.
func NewBuilder[State key.Keyer[StateKey], Input key.Keyer[InputKey], StateKey comparable, InputKey comparable]() *Builder[State, Input, StateKey, InputKey] {
	return &Builder[State, Input, StateKey, InputKey]{
		g:          graph.New[StateKey, InputKey](),
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

// Transition inserts (from, a, to) into δ. It rejects nondeterminism.
func (b *Builder[State, Input, StateKey, InputKey]) Transition(from State, input Input, to State) error {
	inputKey := input.Key()
	if err := b.TransitionKey(from, inputKey, to); err != nil {
		return err
	}
	b.symByKey[inputKey] = input
	return nil
}

// TransitionKey inserts (from, a, to) into δ using only the input key.
// It rejects nondeterminism.
func (b *Builder[State, Input, StateKey, InputKey]) TransitionKey(from State, inputKey InputKey, to State) error {
	fromKey := from.Key()
	toKey := to.Key()

	b.stateByKey[fromKey] = from
	b.stateByKey[toKey] = to

	b.states.Add(fromKey, toKey)
	b.alphabet.Add(inputKey)

	existing := b.g.Delta(fromKey, inputKey)
	if len(existing) == 0 {
		b.g.AddEdge(fromKey, toKey, inputKey)
		return nil
	}
	if len(existing) == 1 && existing[0] == toKey {
		return nil
	}
	return fmt.Errorf("dfa: transition already defined for (%v, %v)", from, inputKey)
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

// Build returns a DFA with δ backed by the builder's graph.
// Returns an error if the graph is missing.
func (b *Builder[State, Input, StateKey, InputKey]) Build() (*DFA[State, Input, StateKey, InputKey], error) {
	if b.gLike == nil && b.g == nil {
		return nil, errors.New("dfa builder: graph is nil")
	}
	if b.gLike != nil {
		if b.states.Len() == 0 {
			return nil, errors.New("dfa builder: WithGraph requires states; add Q with AddStates")
		}
		if b.alphabet.Len() == 0 {
			return nil, errors.New("dfa builder: WithGraph requires alphabet; add Σ with AddAlphabet")
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

// BuildAtomic returns a thread-safe DFA backed by the builder's graph.
func (b *Builder[State, Input, StateKey, InputKey]) BuildAtomic() (*AtomicDFA[State, Input, StateKey, InputKey], error) {
	d, err := b.Build()
	if err != nil {
		return nil, err
	}
	return NewAtomic(d), nil
}
