package nfa

import (
	"errors"
	"fmt"

	"github.com/atahiraj/fsm/internal/set"
	"github.com/atahiraj/fsm/internal/validate"
)

var (
	// ErrReadOnlyGraph reports that a graph does not support mutation.
	ErrReadOnlyGraph = errors.New("nfa: graph is read-only")
	// ErrMixedGraph reports that builder-owned and injected transitions were mixed.
	ErrMixedGraph = errors.New("nfa: cannot mix an injected graph with builder transitions")
)

// Graph supplies nondeterministic and epsilon transitions over domain values.
type Graph[State, Input any] interface {
	Delta(State, Input) []State
	Epsilon(State) []State
}

// MutableGraph is a Graph that supports transition edits.
type MutableGraph[State, Input any] interface {
	Graph[State, Input]
	AddTransition(State, Input, State) bool
	RemoveTransition(State, Input, State) bool
	AddEpsilon(State, State) bool
	RemoveEpsilon(State, State) bool
}

// GraphFuncs adapts functions to Graph.
type GraphFuncs[State, Input any] struct {
	DeltaFunc   func(State, Input) []State
	EpsilonFunc func(State) []State
}

// Delta calls DeltaFunc, treating nil as no transition.
func (g GraphFuncs[State, Input]) Delta(state State, input Input) []State {
	if g.DeltaFunc == nil {
		return nil
	}
	return g.DeltaFunc(state, input)
}

// Epsilon calls EpsilonFunc, treating nil as no transition.
func (g GraphFuncs[State, Input]) Epsilon(state State) []State {
	if g.EpsilonFunc == nil {
		return nil
	}
	return g.EpsilonFunc(state)
}

// Config describes a nondeterministic finite automaton (Q, Σ, δ, q₀, F).
type Config[State, Input any, StateKey, InputKey comparable] struct {
	States    []State
	Alphabet  []Input
	Start     State
	Accepting []State
	Graph     Graph[State, Input]
	StateKey  func(State) StateKey
	InputKey  func(Input) InputKey
}

// NFA is a mutable nondeterministic finite automaton.
//
// It retains an injected graph by reference and is not safe for concurrent use.
type NFA[State, Input any, StateKey, InputKey comparable] struct {
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

// New validates config and constructs an NFA.
func New[State, Input any, StateKey, InputKey comparable](config Config[State, Input, StateKey, InputKey]) (*NFA[State, Input, StateKey, InputKey], error) {
	if config.StateKey == nil {
		return nil, errors.New("nfa: state key function is nil")
	}
	if config.InputKey == nil {
		return nil, errors.New("nfa: input key function is nil")
	}
	if validate.IsNil(config.Graph) {
		return nil, errors.New("nfa: graph is nil")
	}
	if len(config.States) == 0 {
		return nil, errors.New("nfa: state set is empty")
	}

	n := &NFA[State, Input, StateKey, InputKey]{
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
		n.AddState(state)
	}
	for _, input := range config.Alphabet {
		n.AddInput(input)
	}
	if err := n.SetStart(config.Start); err != nil {
		return nil, fmt.Errorf("nfa: invalid start state: %w", err)
	}
	for _, state := range config.Accepting {
		if err := n.AddAccepting(state); err != nil {
			return nil, fmt.Errorf("nfa: invalid accepting state: %w", err)
		}
	}
	return n, nil
}

// Start returns q₀.
func (n *NFA[State, Input, StateKey, InputKey]) Start() State {
	return n.stateByKey[n.startKey]
}

// StartSet returns {q₀}.
func (n *NFA[State, Input, StateKey, InputKey]) StartSet() []State {
	return []State{n.Start()}
}

// Initial returns the epsilon closure of q₀ and implements fsm.Machine.
func (n *NFA[State, Input, StateKey, InputKey]) Initial() []State {
	return n.EpsilonClosure(n.Start())
}

func (n *NFA[State, Input, StateKey, InputKey]) keySet(states []State) *set.Set[StateKey] {
	keys := set.New[StateKey]()
	for _, state := range states {
		keys.Add(n.stateKey(state))
	}
	return keys
}

// Equal reports whether two configurations contain the same state identities.
func (n *NFA[State, Input, StateKey, InputKey]) Equal(left, right []State) bool {
	return n.keySet(left).Equals(n.keySet(right))
}

func (n *NFA[State, Input, StateKey, InputKey]) canonical(keys *set.Set[StateKey]) []State {
	if keys == nil || keys.IsEmpty() {
		return nil
	}
	out := make([]State, 0, keys.Len())
	for _, key := range n.states.Slice() {
		if !keys.Has(key) {
			continue
		}
		if state, ok := n.stateByKey[key]; ok {
			out = append(out, state)
		}
	}
	return out
}

// Delta applies δ to one state and input.
func (n *NFA[State, Input, StateKey, InputKey]) Delta(state State, input Input) []State {
	if !n.HasState(state) || !n.HasInput(input) {
		return nil
	}
	keys := set.New[StateKey]()
	for _, next := range n.graph.Delta(state, input) {
		key := n.stateKey(next)
		if n.states.Has(key) {
			keys.Add(key)
		}
	}
	return n.canonical(keys)
}

// DeltaSet applies δ to a configuration and input.
func (n *NFA[State, Input, StateKey, InputKey]) DeltaSet(states []State, input Input) []State {
	keys := set.New[StateKey]()
	for _, state := range states {
		for _, next := range n.Delta(state, input) {
			keys.Add(n.stateKey(next))
		}
	}
	return n.canonical(keys)
}

// DeltaStar applies δ* to a state and word, including epsilon closure.
func (n *NFA[State, Input, StateKey, InputKey]) DeltaStar(state State, word []Input) []State {
	return n.DeltaStarSet([]State{state}, word)
}

// DeltaStarSet applies δ* to a configuration and word.
func (n *NFA[State, Input, StateKey, InputKey]) DeltaStarSet(states []State, word []Input) []State {
	current := n.EpsilonClosureSet(states)
	for _, input := range word {
		current = n.EpsilonClosureSet(n.DeltaSet(current, input))
	}
	return current
}

// Epsilon returns direct epsilon destinations from state.
func (n *NFA[State, Input, StateKey, InputKey]) Epsilon(state State) []State {
	if !n.HasState(state) {
		return nil
	}
	keys := set.New[StateKey]()
	for _, next := range n.graph.Epsilon(state) {
		key := n.stateKey(next)
		if n.states.Has(key) {
			keys.Add(key)
		}
	}
	return n.canonical(keys)
}

// EpsilonSet returns direct epsilon destinations from a configuration.
func (n *NFA[State, Input, StateKey, InputKey]) EpsilonSet(states []State) []State {
	keys := set.New[StateKey]()
	for _, state := range states {
		for _, next := range n.Epsilon(state) {
			keys.Add(n.stateKey(next))
		}
	}
	return n.canonical(keys)
}

// EpsilonClosure returns the epsilon closure of state.
func (n *NFA[State, Input, StateKey, InputKey]) EpsilonClosure(state State) []State {
	return n.EpsilonClosureSet([]State{state})
}

// EpsilonClosureSet returns the epsilon closure of a configuration.
func (n *NFA[State, Input, StateKey, InputKey]) EpsilonClosureSet(states []State) []State {
	seen := set.New[StateKey]()
	stack := make([]State, 0, len(states))
	for _, state := range states {
		key := n.stateKey(state)
		if !n.states.Has(key) || !seen.Add(key) {
			continue
		}
		stack = append(stack, n.stateByKey[key])
	}
	for len(stack) > 0 {
		state := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for _, next := range n.Epsilon(state) {
			key := n.stateKey(next)
			if seen.Add(key) {
				stack = append(stack, next)
			}
		}
	}
	return n.canonical(seen)
}

// Transition advances a configuration and implements fsm.Machine.
func (n *NFA[State, Input, StateKey, InputKey]) Transition(states []State, input Input) ([]State, bool) {
	current := n.EpsilonClosureSet(states)
	next := n.DeltaSet(current, input)
	if len(next) == 0 {
		return nil, false
	}
	return n.EpsilonClosureSet(next), true
}

// Accepts reports whether word reaches an accepting configuration.
func (n *NFA[State, Input, StateKey, InputKey]) Accepts(word []Input) bool {
	return n.IsAccepting(n.DeltaStarSet(n.StartSet(), word))
}

// IsAccepting reports whether a configuration intersects F.
func (n *NFA[State, Input, StateKey, InputKey]) IsAccepting(states []State) bool {
	for _, state := range states {
		if n.accepting.Has(n.stateKey(state)) {
			return true
		}
	}
	return false
}

func nfaValues[K comparable, V any](keys []K, byKey map[K]V) []V {
	out := make([]V, 0, len(keys))
	for _, key := range keys {
		if value, ok := byKey[key]; ok {
			out = append(out, value)
		}
	}
	return out
}

// States returns Q in insertion order.
func (n *NFA[State, Input, StateKey, InputKey]) States() []State {
	return nfaValues(n.states.Slice(), n.stateByKey)
}

// Alphabet returns Σ in insertion order.
func (n *NFA[State, Input, StateKey, InputKey]) Alphabet() []Input {
	return nfaValues(n.alphabet.Slice(), n.inputByKey)
}

// Accepting returns F in insertion order.
func (n *NFA[State, Input, StateKey, InputKey]) Accepting() []State {
	return nfaValues(n.accepting.Slice(), n.stateByKey)
}

// Graph returns the currently injected transition graph.
func (n *NFA[State, Input, StateKey, InputKey]) Graph() Graph[State, Input] {
	return n.graph
}

// HasState reports whether state belongs to Q.
func (n *NFA[State, Input, StateKey, InputKey]) HasState(state State) bool {
	return n.states.Has(n.stateKey(state))
}

// HasInput reports whether input belongs to Σ.
func (n *NFA[State, Input, StateKey, InputKey]) HasInput(input Input) bool {
	return n.alphabet.Has(n.inputKey(input))
}

// AddState inserts or replaces a state value by identity.
func (n *NFA[State, Input, StateKey, InputKey]) AddState(state State) bool {
	key := n.stateKey(state)
	added := n.states.Add(key)
	n.stateByKey[key] = state
	return added
}

// RemoveState removes a non-start state and its accepting status.
func (n *NFA[State, Input, StateKey, InputKey]) RemoveState(state State) (bool, error) {
	key := n.stateKey(state)
	if key == n.startKey {
		return false, errors.New("nfa: cannot remove the start state")
	}
	if !n.states.Remove(key) {
		return false, nil
	}
	n.accepting.Remove(key)
	delete(n.stateByKey, key)
	return true, nil
}

// AddInput inserts or replaces an input value by identity.
func (n *NFA[State, Input, StateKey, InputKey]) AddInput(input Input) bool {
	key := n.inputKey(input)
	added := n.alphabet.Add(key)
	n.inputByKey[key] = input
	return added
}

// RemoveInput removes an input from Σ.
func (n *NFA[State, Input, StateKey, InputKey]) RemoveInput(input Input) bool {
	key := n.inputKey(input)
	if !n.alphabet.Remove(key) {
		return false
	}
	delete(n.inputByKey, key)
	return true
}

// SetStart changes q₀. State must already belong to Q.
func (n *NFA[State, Input, StateKey, InputKey]) SetStart(state State) error {
	key := n.stateKey(state)
	if !n.states.Has(key) {
		return errors.New("state is not in Q")
	}
	n.startKey = key
	return nil
}

// AddAccepting inserts state into F. State must already belong to Q.
func (n *NFA[State, Input, StateKey, InputKey]) AddAccepting(state State) error {
	key := n.stateKey(state)
	if !n.states.Has(key) {
		return errors.New("state is not in Q")
	}
	n.accepting.Add(key)
	return nil
}

// RemoveAccepting removes state from F.
func (n *NFA[State, Input, StateKey, InputKey]) RemoveAccepting(state State) bool {
	return n.accepting.Remove(n.stateKey(state))
}

// SetGraph replaces δ and retains graph by reference.
func (n *NFA[State, Input, StateKey, InputKey]) SetGraph(graph Graph[State, Input]) error {
	if validate.IsNil(graph) {
		return errors.New("nfa: graph is nil")
	}
	n.graph = graph
	return nil
}

func (n *NFA[State, Input, StateKey, InputKey]) validateTransition(from State, input *Input, to State) error {
	if !n.HasState(from) {
		return errors.New("nfa: transition source is not in Q")
	}
	if !n.HasState(to) {
		return errors.New("nfa: transition destination is not in Q")
	}
	if input != nil && !n.HasInput(*input) {
		return errors.New("nfa: transition input is not in Σ")
	}
	return nil
}

func (n *NFA[State, Input, StateKey, InputKey]) mutableGraph() (MutableGraph[State, Input], error) {
	graph, ok := n.graph.(MutableGraph[State, Input])
	if !ok {
		return nil, ErrReadOnlyGraph
	}
	return graph, nil
}

// AddTransition adds a transition when the graph is mutable.
func (n *NFA[State, Input, StateKey, InputKey]) AddTransition(from State, input Input, to State) (bool, error) {
	if err := n.validateTransition(from, &input, to); err != nil {
		return false, err
	}
	graph, err := n.mutableGraph()
	if err != nil {
		return false, err
	}
	return graph.AddTransition(from, input, to), nil
}

// RemoveTransition removes a transition when the graph is mutable.
func (n *NFA[State, Input, StateKey, InputKey]) RemoveTransition(from State, input Input, to State) (bool, error) {
	if err := n.validateTransition(from, &input, to); err != nil {
		return false, err
	}
	graph, err := n.mutableGraph()
	if err != nil {
		return false, err
	}
	return graph.RemoveTransition(from, input, to), nil
}

// AddEpsilon adds an epsilon transition when the graph is mutable.
func (n *NFA[State, Input, StateKey, InputKey]) AddEpsilon(from, to State) (bool, error) {
	if err := n.validateTransition(from, nil, to); err != nil {
		return false, err
	}
	graph, err := n.mutableGraph()
	if err != nil {
		return false, err
	}
	return graph.AddEpsilon(from, to), nil
}

// RemoveEpsilon removes an epsilon transition when the graph is mutable.
func (n *NFA[State, Input, StateKey, InputKey]) RemoveEpsilon(from, to State) (bool, error) {
	if err := n.validateTransition(from, nil, to); err != nil {
		return false, err
	}
	graph, err := n.mutableGraph()
	if err != nil {
		return false, err
	}
	return graph.RemoveEpsilon(from, to), nil
}
