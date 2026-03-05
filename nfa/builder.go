package nfa

import (
	"errors"

	internalgraph "github.com/atahiraj/fsm/internal/graph"
	"github.com/atahiraj/fsm/internal/set"
	"github.com/atahiraj/fsm/internal/validate"
)

// Builder incrementally constructs a validated NFA.
type Builder[State, Input any, StateKey, InputKey comparable] struct {
	stateKey   func(State) StateKey
	inputKey   func(Input) InputKey
	states     *set.Set[StateKey]
	alphabet   *set.Set[InputKey]
	accepting  *set.Set[StateKey]
	stateByKey map[StateKey]State
	inputByKey map[InputKey]Input
	start      State
	hasStart   bool
	table      *internalgraph.Nondeterministic[State, Input, StateKey, InputKey]
	injected   Graph[State, Input]
	hasEdges   bool
}

// NewBuilder constructs an empty builder with injected identity functions.
func NewBuilder[State, Input any, StateKey, InputKey comparable](stateKey func(State) StateKey, inputKey func(Input) InputKey) (*Builder[State, Input, StateKey, InputKey], error) {
	if stateKey == nil {
		return nil, errors.New("nfa builder: state key function is nil")
	}
	if inputKey == nil {
		return nil, errors.New("nfa builder: input key function is nil")
	}
	return &Builder[State, Input, StateKey, InputKey]{
		stateKey:   stateKey,
		inputKey:   inputKey,
		states:     set.New[StateKey](),
		alphabet:   set.New[InputKey](),
		accepting:  set.New[StateKey](),
		stateByKey: make(map[StateKey]State),
		inputByKey: make(map[InputKey]Input),
		table:      internalgraph.NewNondeterministic(stateKey, inputKey),
	}, nil
}

// AddStates inserts states into Q.
func (b *Builder[State, Input, StateKey, InputKey]) AddStates(states ...State) {
	for _, state := range states {
		key := b.stateKey(state)
		b.states.Add(key)
		b.stateByKey[key] = state
		b.table.AddVertex(state)
	}
}

// AddAlphabet inserts inputs into Σ.
func (b *Builder[State, Input, StateKey, InputKey]) AddAlphabet(inputs ...Input) {
	for _, input := range inputs {
		key := b.inputKey(input)
		b.alphabet.Add(key)
		b.inputByKey[key] = input
	}
}

// SetStart sets q₀ and adds it to Q.
func (b *Builder[State, Input, StateKey, InputKey]) SetStart(state State) {
	b.AddStates(state)
	b.start = state
	b.hasStart = true
}

// AddAccepting inserts states into F and Q.
func (b *Builder[State, Input, StateKey, InputKey]) AddAccepting(states ...State) {
	b.AddStates(states...)
	for _, state := range states {
		b.accepting.Add(b.stateKey(state))
	}
}

// WithGraph injects the complete transition relation.
//
// The graph is shared with the built NFA. WithGraph cannot be mixed with
// transitions added through this builder.
func (b *Builder[State, Input, StateKey, InputKey]) WithGraph(graph Graph[State, Input]) error {
	if validate.IsNil(graph) {
		return errors.New("nfa builder: graph is nil")
	}
	if b.hasEdges {
		return ErrMixedGraph
	}
	b.injected = graph
	return nil
}

// AddTransition inserts a transition and its states/input into Q and Σ.
func (b *Builder[State, Input, StateKey, InputKey]) AddTransition(from State, input Input, to State) error {
	if b.injected != nil {
		return ErrMixedGraph
	}
	b.AddStates(from, to)
	b.AddAlphabet(input)
	b.hasEdges = true
	b.table.AddTransition(from, input, to)
	return nil
}

// AddEpsilon inserts an epsilon transition and its states into Q.
func (b *Builder[State, Input, StateKey, InputKey]) AddEpsilon(from, to State) error {
	if b.injected != nil {
		return ErrMixedGraph
	}
	b.AddStates(from, to)
	b.hasEdges = true
	b.table.AddEpsilon(from, to)
	return nil
}

// Build validates the definition and returns an NFA.
func (b *Builder[State, Input, StateKey, InputKey]) Build() (*NFA[State, Input, StateKey, InputKey], error) {
	if !b.hasStart {
		return nil, errors.New("nfa builder: start state is not set")
	}
	graph := b.injected
	if graph == nil {
		graph = b.table.Clone()
	}
	return New(Config[State, Input, StateKey, InputKey]{
		States:    nfaValues(b.states.Slice(), b.stateByKey),
		Alphabet:  nfaValues(b.alphabet.Slice(), b.inputByKey),
		Start:     b.start,
		Accepting: nfaValues(b.accepting.Slice(), b.stateByKey),
		Graph:     graph,
		StateKey:  b.stateKey,
		InputKey:  b.inputKey,
	})
}
