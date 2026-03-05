package dfa

import (
	"errors"
	"fmt"

	internalgraph "github.com/atahiraj/fsm/internal/graph"
	"github.com/atahiraj/fsm/internal/set"
	"github.com/atahiraj/fsm/internal/validate"
)

var (
	// ErrReadOnlyGraph reports that a graph does not support mutation.
	ErrReadOnlyGraph = errors.New("dfa: graph is read-only")
	// ErrTransitionExists reports a conflicting deterministic transition.
	ErrTransitionExists = errors.New("dfa: transition already exists")
	// ErrMixedGraph reports that builder-owned and injected transitions were mixed.
	ErrMixedGraph = errors.New("dfa: cannot mix an injected graph with builder transitions")
)

// Graph supplies the deterministic transition function over domain values.
type Graph[State, Input any] interface {
	Delta(State, Input) (State, bool)
}

// MutableGraph is a Graph that supports transition edits.
type MutableGraph[State, Input any] interface {
	Graph[State, Input]
	AddTransition(State, Input, State) error
	RemoveTransition(State, Input) bool
}

// GraphFunc adapts a function to Graph.
type GraphFunc[State, Input any] func(State, Input) (State, bool)

// Delta calls f.
func (f GraphFunc[State, Input]) Delta(state State, input Input) (State, bool) {
	if f == nil {
		var zero State
		return zero, false
	}
	return f(state, input)
}

// Config describes a deterministic finite automaton (Q, Σ, δ, q₀, F).
type Config[State, Input any, StateKey, InputKey comparable] struct {
	States    []State
	Alphabet  []Input
	Start     State
	Accepting []State
	Graph     Graph[State, Input]
	StateKey  func(State) StateKey
	InputKey  func(Input) InputKey
}

// DFA is a mutable deterministic finite automaton.
//
// It retains an injected graph by reference and is not safe for concurrent use.
type DFA[State, Input any, StateKey, InputKey comparable] struct {
	graph      Graph[State, Input]
	stateKey   func(State) StateKey
	inputKey   func(Input) InputKey
	states     *set.Set[StateKey]
	alphabet   *set.Set[InputKey]
	accepting  *set.Set[StateKey]
	stateByKey map[StateKey]State
	inputByKey map[InputKey]Input
	startKey   StateKey
}

// New validates config and constructs a DFA.
func New[State, Input any, StateKey, InputKey comparable](config Config[State, Input, StateKey, InputKey]) (*DFA[State, Input, StateKey, InputKey], error) {
	if config.StateKey == nil {
		return nil, errors.New("dfa: state key function is nil")
	}
	if config.InputKey == nil {
		return nil, errors.New("dfa: input key function is nil")
	}
	if validate.IsNil(config.Graph) {
		return nil, errors.New("dfa: graph is nil")
	}
	if len(config.States) == 0 {
		return nil, errors.New("dfa: state set is empty")
	}

	d := &DFA[State, Input, StateKey, InputKey]{
		graph:      config.Graph,
		stateKey:   config.StateKey,
		inputKey:   config.InputKey,
		states:     set.New[StateKey](),
		alphabet:   set.New[InputKey](),
		accepting:  set.New[StateKey](),
		stateByKey: make(map[StateKey]State, len(config.States)),
		inputByKey: make(map[InputKey]Input, len(config.Alphabet)),
	}
	for _, state := range config.States {
		d.AddState(state)
	}
	for _, input := range config.Alphabet {
		d.AddInput(input)
	}
	if err := d.SetStart(config.Start); err != nil {
		return nil, fmt.Errorf("dfa: invalid start state: %w", err)
	}
	for _, state := range config.Accepting {
		if err := d.AddAccepting(state); err != nil {
			return nil, fmt.Errorf("dfa: invalid accepting state: %w", err)
		}
	}
	return d, nil
}

// Initial returns q₀ and implements fsm.Machine.
func (d *DFA[State, Input, StateKey, InputKey]) Initial() State {
	return d.stateByKey[d.startKey]
}

// Start returns q₀.
func (d *DFA[State, Input, StateKey, InputKey]) Start() State { return d.Initial() }

// Equal reports whether two state values have the same identity.
func (d *DFA[State, Input, StateKey, InputKey]) Equal(left, right State) bool {
	return d.stateKey(left) == d.stateKey(right)
}

// Delta applies δ to a state and input.
func (d *DFA[State, Input, StateKey, InputKey]) Delta(state State, input Input) (State, bool) {
	var zero State
	if !d.HasState(state) || !d.HasInput(input) {
		return zero, false
	}
	next, ok := d.graph.Delta(state, input)
	if !ok {
		return zero, false
	}
	canonical, ok := d.stateByKey[d.stateKey(next)]
	if !ok {
		return zero, false
	}
	return canonical, true
}

// Transition applies δ and implements fsm.Machine.
func (d *DFA[State, Input, StateKey, InputKey]) Transition(state State, input Input) (State, bool) {
	return d.Delta(state, input)
}

// DeltaStar applies δ repeatedly over a word.
func (d *DFA[State, Input, StateKey, InputKey]) DeltaStar(state State, word []Input) (State, bool) {
	current := state
	for _, input := range word {
		next, ok := d.Delta(current, input)
		if !ok {
			var zero State
			return zero, false
		}
		current = next
	}
	return current, true
}

// Accepts reports whether word ends in an accepting state.
func (d *DFA[State, Input, StateKey, InputKey]) Accepts(word []Input) bool {
	state, ok := d.DeltaStar(d.Initial(), word)
	return ok && d.IsAccepting(state)
}

// IsAccepting reports whether state belongs to F.
func (d *DFA[State, Input, StateKey, InputKey]) IsAccepting(state State) bool {
	return d.accepting.Has(d.stateKey(state))
}

func values[K comparable, V any](keys []K, byKey map[K]V) []V {
	out := make([]V, 0, len(keys))
	for _, key := range keys {
		if value, ok := byKey[key]; ok {
			out = append(out, value)
		}
	}
	return out
}

// States returns Q in insertion order.
func (d *DFA[State, Input, StateKey, InputKey]) States() []State {
	return values(d.states.Slice(), d.stateByKey)
}

// Alphabet returns Σ in insertion order.
func (d *DFA[State, Input, StateKey, InputKey]) Alphabet() []Input {
	return values(d.alphabet.Slice(), d.inputByKey)
}

// Accepting returns F in insertion order.
func (d *DFA[State, Input, StateKey, InputKey]) Accepting() []State {
	return values(d.accepting.Slice(), d.stateByKey)
}

// Graph returns the currently injected transition graph.
func (d *DFA[State, Input, StateKey, InputKey]) Graph() Graph[State, Input] {
	return d.graph
}

// HasState reports whether state belongs to Q.
func (d *DFA[State, Input, StateKey, InputKey]) HasState(state State) bool {
	return d.states.Has(d.stateKey(state))
}

// HasInput reports whether input belongs to Σ.
func (d *DFA[State, Input, StateKey, InputKey]) HasInput(input Input) bool {
	return d.alphabet.Has(d.inputKey(input))
}

// AddState inserts or replaces a state value by identity.
func (d *DFA[State, Input, StateKey, InputKey]) AddState(state State) bool {
	key := d.stateKey(state)
	added := d.states.Add(key)
	d.stateByKey[key] = state
	return added
}

// RemoveState removes a non-start state and its accepting status.
func (d *DFA[State, Input, StateKey, InputKey]) RemoveState(state State) (bool, error) {
	key := d.stateKey(state)
	if key == d.startKey {
		return false, errors.New("dfa: cannot remove the start state")
	}
	if !d.states.Remove(key) {
		return false, nil
	}
	d.accepting.Remove(key)
	delete(d.stateByKey, key)
	return true, nil
}

// AddInput inserts or replaces an input value by identity.
func (d *DFA[State, Input, StateKey, InputKey]) AddInput(input Input) bool {
	key := d.inputKey(input)
	added := d.alphabet.Add(key)
	d.inputByKey[key] = input
	return added
}

// RemoveInput removes an input from Σ.
func (d *DFA[State, Input, StateKey, InputKey]) RemoveInput(input Input) bool {
	key := d.inputKey(input)
	if !d.alphabet.Remove(key) {
		return false
	}
	delete(d.inputByKey, key)
	return true
}

// SetStart changes q₀. State must already belong to Q.
func (d *DFA[State, Input, StateKey, InputKey]) SetStart(state State) error {
	key := d.stateKey(state)
	if !d.states.Has(key) {
		return errors.New("state is not in Q")
	}
	d.startKey = key
	return nil
}

// AddAccepting inserts state into F. State must already belong to Q.
func (d *DFA[State, Input, StateKey, InputKey]) AddAccepting(state State) error {
	key := d.stateKey(state)
	if !d.states.Has(key) {
		return errors.New("state is not in Q")
	}
	d.accepting.Add(key)
	return nil
}

// RemoveAccepting removes state from F.
func (d *DFA[State, Input, StateKey, InputKey]) RemoveAccepting(state State) bool {
	return d.accepting.Remove(d.stateKey(state))
}

// SetGraph replaces δ and retains graph by reference.
func (d *DFA[State, Input, StateKey, InputKey]) SetGraph(graph Graph[State, Input]) error {
	if validate.IsNil(graph) {
		return errors.New("dfa: graph is nil")
	}
	d.graph = graph
	return nil
}

func (d *DFA[State, Input, StateKey, InputKey]) validateTransition(from State, input Input, to *State) error {
	if !d.HasState(from) {
		return errors.New("dfa: transition source is not in Q")
	}
	if !d.HasInput(input) {
		return errors.New("dfa: transition input is not in Σ")
	}
	if to != nil && !d.HasState(*to) {
		return errors.New("dfa: transition destination is not in Q")
	}
	return nil
}

// AddTransition adds a transition when the graph is mutable.
func (d *DFA[State, Input, StateKey, InputKey]) AddTransition(from State, input Input, to State) error {
	if err := d.validateTransition(from, input, &to); err != nil {
		return err
	}
	graph, ok := d.graph.(MutableGraph[State, Input])
	if !ok {
		return ErrReadOnlyGraph
	}
	if err := graph.AddTransition(from, input, to); err != nil {
		if errors.Is(err, ErrTransitionExists) || errors.Is(err, internalgraph.ErrTransitionExists) {
			return fmt.Errorf("%w: (%v, %v)", ErrTransitionExists, from, input)
		}
		return err
	}
	return nil
}

// RemoveTransition removes a transition when the graph is mutable.
func (d *DFA[State, Input, StateKey, InputKey]) RemoveTransition(from State, input Input) (bool, error) {
	if err := d.validateTransition(from, input, nil); err != nil {
		return false, err
	}
	graph, ok := d.graph.(MutableGraph[State, Input])
	if !ok {
		return false, ErrReadOnlyGraph
	}
	return graph.RemoveTransition(from, input), nil
}
