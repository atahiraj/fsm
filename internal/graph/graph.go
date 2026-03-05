// Package graph contains the automata's default transition tables.
package graph

import (
	"errors"

	"github.com/atahiraj/fsm/internal/set"
)

// ErrTransitionExists reports a conflicting deterministic transition.
var ErrTransitionExists = errors.New("transition already exists")

// Deterministic is an insertion-ordered deterministic transition table.
type Deterministic[S, I any, SK, IK comparable] struct {
	stateKey func(S) SK
	inputKey func(I) IK
	vertices map[SK]S
	order    *set.Set[SK]
	next     map[SK]map[IK]S
}

// NewDeterministic constructs an empty deterministic transition table.
func NewDeterministic[S, I any, SK, IK comparable](stateKey func(S) SK, inputKey func(I) IK) *Deterministic[S, I, SK, IK] {
	return &Deterministic[S, I, SK, IK]{
		stateKey: stateKey,
		inputKey: inputKey,
		vertices: make(map[SK]S),
		order:    set.New[SK](),
		next:     make(map[SK]map[IK]S),
	}
}

// AddVertex records a vertex, including isolated vertices.
func (g *Deterministic[S, I, SK, IK]) AddVertex(state S) {
	key := g.stateKey(state)
	g.order.Add(key)
	g.vertices[key] = state
}

// AddTransition adds a deterministic transition.
func (g *Deterministic[S, I, SK, IK]) AddTransition(from S, input I, to S) error {
	fromKey := g.stateKey(from)
	inputKey := g.inputKey(input)
	toKey := g.stateKey(to)
	g.AddVertex(from)
	g.AddVertex(to)
	byInput := g.next[fromKey]
	if byInput == nil {
		byInput = make(map[IK]S)
		g.next[fromKey] = byInput
	}
	if existing, exists := byInput[inputKey]; exists {
		if g.stateKey(existing) == toKey {
			return nil
		}
		return ErrTransitionExists
	}
	byInput[inputKey] = to
	return nil
}

// RemoveTransition removes a transition.
func (g *Deterministic[S, I, SK, IK]) RemoveTransition(from S, input I) bool {
	byInput := g.next[g.stateKey(from)]
	if byInput == nil {
		return false
	}
	inputKey := g.inputKey(input)
	if _, exists := byInput[inputKey]; !exists {
		return false
	}
	delete(byInput, inputKey)
	return true
}

// Delta returns the destination for a state and input.
func (g *Deterministic[S, I, SK, IK]) Delta(from S, input I) (S, bool) {
	var zero S
	if g == nil {
		return zero, false
	}
	byInput := g.next[g.stateKey(from)]
	if byInput == nil {
		return zero, false
	}
	to, exists := byInput[g.inputKey(input)]
	return to, exists
}

// Vertices returns vertices in insertion order.
func (g *Deterministic[S, I, SK, IK]) Vertices() []S {
	if g == nil {
		return nil
	}
	out := make([]S, 0, g.order.Len())
	for _, key := range g.order.Slice() {
		out = append(out, g.vertices[key])
	}
	return out
}

// Clone creates a deep copy, including isolated vertices.
func (g *Deterministic[S, I, SK, IK]) Clone() *Deterministic[S, I, SK, IK] {
	clone := NewDeterministic(g.stateKey, g.inputKey)
	for _, key := range g.order.Slice() {
		clone.AddVertex(g.vertices[key])
	}
	for from, byInput := range g.next {
		clone.next[from] = make(map[IK]S, len(byInput))
		for input, to := range byInput {
			clone.next[from][input] = to
		}
	}
	return clone
}

// Nondeterministic is an insertion-ordered NFA transition table.
type Nondeterministic[S, I any, SK, IK comparable] struct {
	stateKey func(S) SK
	inputKey func(I) IK
	vertices map[SK]S
	order    *set.Set[SK]
	next     map[SK]map[IK]*set.Set[SK]
	epsilon  map[SK]*set.Set[SK]
}

// NewNondeterministic constructs an empty nondeterministic transition table.
func NewNondeterministic[S, I any, SK, IK comparable](stateKey func(S) SK, inputKey func(I) IK) *Nondeterministic[S, I, SK, IK] {
	return &Nondeterministic[S, I, SK, IK]{
		stateKey: stateKey,
		inputKey: inputKey,
		vertices: make(map[SK]S),
		order:    set.New[SK](),
		next:     make(map[SK]map[IK]*set.Set[SK]),
		epsilon:  make(map[SK]*set.Set[SK]),
	}
}

// AddVertex records a vertex, including isolated vertices.
func (g *Nondeterministic[S, I, SK, IK]) AddVertex(state S) {
	key := g.stateKey(state)
	g.order.Add(key)
	g.vertices[key] = state
}

// AddTransition adds an NFA transition and reports whether it was new.
func (g *Nondeterministic[S, I, SK, IK]) AddTransition(from S, input I, to S) bool {
	fromKey := g.stateKey(from)
	inputKey := g.inputKey(input)
	toKey := g.stateKey(to)
	g.AddVertex(from)
	g.AddVertex(to)
	byInput := g.next[fromKey]
	if byInput == nil {
		byInput = make(map[IK]*set.Set[SK])
		g.next[fromKey] = byInput
	}
	tos := byInput[inputKey]
	if tos == nil {
		tos = set.New[SK]()
		byInput[inputKey] = tos
	}
	return tos.Add(toKey)
}

// RemoveTransition removes one NFA transition.
func (g *Nondeterministic[S, I, SK, IK]) RemoveTransition(from S, input I, to S) bool {
	byInput := g.next[g.stateKey(from)]
	if byInput == nil {
		return false
	}
	tos := byInput[g.inputKey(input)]
	if tos == nil {
		return false
	}
	return tos.Remove(g.stateKey(to))
}

// AddEpsilon adds an epsilon transition and reports whether it was new.
func (g *Nondeterministic[S, I, SK, IK]) AddEpsilon(from S, to S) bool {
	fromKey := g.stateKey(from)
	toKey := g.stateKey(to)
	g.AddVertex(from)
	g.AddVertex(to)
	tos := g.epsilon[fromKey]
	if tos == nil {
		tos = set.New[SK]()
		g.epsilon[fromKey] = tos
	}
	return tos.Add(toKey)
}

// RemoveEpsilon removes one epsilon transition.
func (g *Nondeterministic[S, I, SK, IK]) RemoveEpsilon(from S, to S) bool {
	tos := g.epsilon[g.stateKey(from)]
	if tos == nil {
		return false
	}
	return tos.Remove(g.stateKey(to))
}

func (g *Nondeterministic[S, I, SK, IK]) states(keys *set.Set[SK]) []S {
	if keys == nil {
		return nil
	}
	out := make([]S, 0, keys.Len())
	for _, key := range keys.Slice() {
		if state, exists := g.vertices[key]; exists {
			out = append(out, state)
		}
	}
	return out
}

// Delta returns destinations in insertion order.
func (g *Nondeterministic[S, I, SK, IK]) Delta(from S, input I) []S {
	if g == nil {
		return nil
	}
	byInput := g.next[g.stateKey(from)]
	if byInput == nil {
		return nil
	}
	return g.states(byInput[g.inputKey(input)])
}

// Epsilon returns epsilon destinations in insertion order.
func (g *Nondeterministic[S, I, SK, IK]) Epsilon(from S) []S {
	if g == nil {
		return nil
	}
	return g.states(g.epsilon[g.stateKey(from)])
}

// Vertices returns vertices in insertion order.
func (g *Nondeterministic[S, I, SK, IK]) Vertices() []S {
	if g == nil {
		return nil
	}
	out := make([]S, 0, g.order.Len())
	for _, key := range g.order.Slice() {
		out = append(out, g.vertices[key])
	}
	return out
}

// Clone creates a deep copy, including isolated vertices.
func (g *Nondeterministic[S, I, SK, IK]) Clone() *Nondeterministic[S, I, SK, IK] {
	clone := NewNondeterministic(g.stateKey, g.inputKey)
	for _, key := range g.order.Slice() {
		clone.AddVertex(g.vertices[key])
	}
	for from, byInput := range g.next {
		clone.next[from] = make(map[IK]*set.Set[SK], len(byInput))
		for input, tos := range byInput {
			clone.next[from][input] = tos.Clone()
		}
	}
	for from, tos := range g.epsilon {
		clone.epsilon[from] = tos.Clone()
	}
	return clone
}
