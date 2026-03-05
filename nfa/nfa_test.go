package nfa_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/atahiraj/fsm"
	"github.com/atahiraj/fsm/nfa"
)

type state struct {
	ID   int
	Name string
}

type event struct {
	Kind    string
	Payload string
}

func stateKey(value state) int    { return value.ID }
func inputKey(value event) string { return value.Kind }

func states(ids ...int) []state {
	out := make([]state, len(ids))
	for i, id := range ids {
		out[i] = state{ID: id}
	}
	return out
}

type mutableGraph struct {
	next    map[int]map[string]map[int]state
	epsilon map[int]map[int]state
}

func newMutableGraph() *mutableGraph {
	return &mutableGraph{
		next:    make(map[int]map[string]map[int]state),
		epsilon: make(map[int]map[int]state),
	}
}

func mapValues(values map[int]state) []state {
	if len(values) == 0 {
		return nil
	}
	out := make([]state, 0, len(values))
	for _, value := range values {
		out = append(out, value)
	}
	return out
}

func (g *mutableGraph) Delta(from state, input event) []state {
	return mapValues(g.next[from.ID][input.Kind])
}

func (g *mutableGraph) Epsilon(from state) []state {
	return mapValues(g.epsilon[from.ID])
}

func (g *mutableGraph) AddTransition(from state, input event, to state) bool {
	byInput := g.next[from.ID]
	if byInput == nil {
		byInput = make(map[string]map[int]state)
		g.next[from.ID] = byInput
	}
	tos := byInput[input.Kind]
	if tos == nil {
		tos = make(map[int]state)
		byInput[input.Kind] = tos
	}
	_, exists := tos[to.ID]
	tos[to.ID] = to
	return !exists
}

func (g *mutableGraph) RemoveTransition(from state, input event, to state) bool {
	tos := g.next[from.ID][input.Kind]
	if _, exists := tos[to.ID]; !exists {
		return false
	}
	delete(tos, to.ID)
	return true
}

func (g *mutableGraph) AddEpsilon(from, to state) bool {
	tos := g.epsilon[from.ID]
	if tos == nil {
		tos = make(map[int]state)
		g.epsilon[from.ID] = tos
	}
	_, exists := tos[to.ID]
	tos[to.ID] = to
	return !exists
}

func (g *mutableGraph) RemoveEpsilon(from, to state) bool {
	tos := g.epsilon[from.ID]
	if _, exists := tos[to.ID]; !exists {
		return false
	}
	delete(tos, to.ID)
	return true
}

var _ nfa.MutableGraph[state, event] = (*mutableGraph)(nil)
var _ fsm.Machine[[]state, event] = (*nfa.NFA[state, event, int, string])(nil)

func buildNFA(t *testing.T) *nfa.NFA[state, event, int, string] {
	t.Helper()
	builder, err := nfa.NewBuilder(stateKey, inputKey)
	if err != nil {
		t.Fatalf("NewBuilder() error = %v", err)
	}
	builder.SetStart(state{ID: 0})
	builder.AddAccepting(state{ID: 3})
	if err := builder.AddEpsilon(state{ID: 0}, state{ID: 1}); err != nil {
		t.Fatal(err)
	}
	if err := builder.AddTransition(state{ID: 1}, event{Kind: "a"}, state{ID: 1}); err != nil {
		t.Fatal(err)
	}
	if err := builder.AddTransition(state{ID: 1}, event{Kind: "b"}, state{ID: 2}); err != nil {
		t.Fatal(err)
	}
	if err := builder.AddEpsilon(state{ID: 2}, state{ID: 3}); err != nil {
		t.Fatal(err)
	}
	machine, err := builder.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	return machine
}

func TestNFAAcceptsWithEpsilonAndUsesInputIdentity(t *testing.T) {
	machine := buildNFA(t)
	if !machine.Accepts([]event{{Kind: "a", Payload: "one"}, {Kind: "b", Payload: "two"}}) {
		t.Fatal("NFA should accept [a b]")
	}
	if machine.Accepts([]event{{Kind: "a"}}) {
		t.Fatal("NFA should reject [a]")
	}
	if got, want := machine.Initial(), states(0, 1); !reflect.DeepEqual(got, want) {
		t.Fatalf("Initial() = %#v, want %#v", got, want)
	}
}

func TestNFAConfigurationsAreStableAndSetEqual(t *testing.T) {
	builder, err := nfa.NewBuilder(stateKey, inputKey)
	if err != nil {
		t.Fatal(err)
	}
	builder.AddStates(state{ID: 2}, state{ID: 0}, state{ID: 1})
	builder.SetStart(state{ID: 0})
	if err := builder.AddEpsilon(state{ID: 0}, state{ID: 1}); err != nil {
		t.Fatal(err)
	}
	if err := builder.AddEpsilon(state{ID: 0}, state{ID: 2}); err != nil {
		t.Fatal(err)
	}
	machine, err := builder.Build()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := machine.Initial(), states(2, 0, 1); !reflect.DeepEqual(got, want) {
		t.Fatalf("Initial() = %#v, want %#v", got, want)
	}
	if !machine.Equal(states(0, 1, 1, 2), states(2, 0, 1)) {
		t.Fatal("Equal() should ignore order and duplicates")
	}
}

func TestNFAStableEnumeration(t *testing.T) {
	machine, err := nfa.New(nfa.Config[state, event, int, string]{
		States: []state{
			{ID: 2, Name: "two"},
			{ID: 1, Name: "old"},
			{ID: 1, Name: "new"},
			{ID: 3, Name: "three"},
		},
		Alphabet:  []event{{Kind: "b"}, {Kind: "a"}, {Kind: "b", Payload: "new"}},
		Start:     state{ID: 2},
		Accepting: []state{{ID: 3}, {ID: 1}},
		Graph:     nfa.GraphFuncs[state, event]{},
		StateKey:  stateKey,
		InputKey:  inputKey,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if got, want := machine.States(), []state{{ID: 2, Name: "two"}, {ID: 1, Name: "new"}, {ID: 3, Name: "three"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("States() = %#v, want %#v", got, want)
	}
	if got := machine.Alphabet(); len(got) != 2 || got[0].Kind != "b" || got[0].Payload != "new" || got[1].Kind != "a" {
		t.Fatalf("Alphabet() = %#v", got)
	}
	if got, want := machine.Accepting(), []state{{ID: 3, Name: "three"}, {ID: 1, Name: "new"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Accepting() = %#v, want %#v", got, want)
	}
}

func TestNFAInjectedGraphIsSharedAndMutable(t *testing.T) {
	graph := newMutableGraph()
	graph.AddTransition(state{ID: 0}, event{Kind: "go"}, state{ID: 1})
	machine, err := nfa.New(nfa.Config[state, event, int, string]{
		States:    states(0, 1, 2),
		Alphabet:  []event{{Kind: "go"}},
		Start:     state{ID: 0},
		Accepting: states(2),
		Graph:     graph,
		StateKey:  stateKey,
		InputKey:  inputKey,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	graph.AddEpsilon(state{ID: 1}, state{ID: 2})
	if !machine.Accepts([]event{{Kind: "go"}}) {
		t.Fatal("external graph mutation was not visible")
	}
	removed, err := machine.RemoveEpsilon(state{ID: 1}, state{ID: 2})
	if err != nil || !removed {
		t.Fatalf("RemoveEpsilon() = (%v, %v)", removed, err)
	}
	if machine.Accepts([]event{{Kind: "go"}}) {
		t.Fatal("machine mutation was not visible in injected graph")
	}
}

func TestNFAEngineAndPredicateHooks(t *testing.T) {
	machine := buildNFA(t)
	hooks, err := fsm.NewHooks[[]state, event](machine.Equal)
	if err != nil {
		t.Fatal(err)
	}
	contains := func(configuration []state, target int) bool {
		for _, current := range configuration {
			if current.ID == target {
				return true
			}
		}
		return false
	}
	var exitedOne bool
	if err := hooks.OnIf(fsm.PhaseExit, func(transition fsm.Transition[[]state, event]) bool {
		return contains(transition.From, 1) && !contains(transition.To, 1)
	}, func(fsm.Transition[[]state, event]) {
		exitedOne = true
	}); err != nil {
		t.Fatal(err)
	}
	engine, err := fsm.New(machine, fsm.WithLifecycle[[]state, event](hooks))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := engine.Step(event{Kind: "b"}); !ok {
		t.Fatal("Step() found no transition")
	}
	if !exitedOne || !engine.IsAccepting() {
		t.Fatalf("exitedOne=%v accepting=%v", exitedOne, engine.IsAccepting())
	}
}

func TestNFAReadOnlyGraphAndStrictMutations(t *testing.T) {
	graph := nfa.GraphFuncs[state, event]{
		DeltaFunc: func(from state, input event) []state {
			if from.ID == 0 && input.Kind == "go" {
				return states(1)
			}
			return nil
		},
	}
	machine, err := nfa.New(nfa.Config[state, event, int, string]{
		States:   states(0, 1),
		Alphabet: []event{{Kind: "go"}},
		Start:    state{ID: 0},
		Graph:    graph,
		StateKey: stateKey,
		InputKey: inputKey,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if _, err := machine.AddTransition(state{ID: 0}, event{Kind: "go"}, state{ID: 1}); !errors.Is(err, nfa.ErrReadOnlyGraph) {
		t.Fatalf("AddTransition() error = %v", err)
	}
	if _, err := machine.AddEpsilon(state{ID: 0}, state{ID: 1}); !errors.Is(err, nfa.ErrReadOnlyGraph) {
		t.Fatalf("AddEpsilon() error = %v", err)
	}
	if _, err := machine.AddTransition(state{ID: 0}, event{Kind: "missing"}, state{ID: 1}); err == nil {
		t.Fatal("unknown input mutation error = nil")
	}
	if err := machine.SetStart(state{ID: 9}); err == nil {
		t.Fatal("SetStart(unknown) error = nil")
	}
}

func TestNFABuilderRejectsGraphMixingInEitherOrder(t *testing.T) {
	graph := newMutableGraph()

	first, err := nfa.NewBuilder(stateKey, inputKey)
	if err != nil {
		t.Fatal(err)
	}
	if err := first.AddEpsilon(state{ID: 0}, state{ID: 1}); err != nil {
		t.Fatal(err)
	}
	if err := first.WithGraph(graph); !errors.Is(err, nfa.ErrMixedGraph) {
		t.Fatalf("WithGraph() error = %v", err)
	}

	second, err := nfa.NewBuilder(stateKey, inputKey)
	if err != nil {
		t.Fatal(err)
	}
	if err := second.WithGraph(graph); err != nil {
		t.Fatal(err)
	}
	if err := second.AddTransition(state{ID: 0}, event{Kind: "go"}, state{ID: 1}); !errors.Is(err, nfa.ErrMixedGraph) {
		t.Fatalf("AddTransition() error = %v", err)
	}
}

func TestNFABuilderSnapshotsOwnedGraph(t *testing.T) {
	builder, err := nfa.NewBuilder(stateKey, inputKey)
	if err != nil {
		t.Fatal(err)
	}
	builder.SetStart(state{ID: 0})
	if err := builder.AddTransition(state{ID: 0}, event{Kind: "a"}, state{ID: 1}); err != nil {
		t.Fatal(err)
	}
	first, err := builder.Build()
	if err != nil {
		t.Fatal(err)
	}
	if err := builder.AddEpsilon(state{ID: 1}, state{ID: 2}); err != nil {
		t.Fatal(err)
	}
	second, err := builder.Build()
	if err != nil {
		t.Fatal(err)
	}
	if got := first.Epsilon(state{ID: 1}); len(got) != 0 {
		t.Fatalf("first Epsilon() = %#v", got)
	}
	if got, want := second.Epsilon(state{ID: 1}), states(2); !reflect.DeepEqual(got, want) {
		t.Fatalf("second Epsilon() = %#v, want %#v", got, want)
	}
}

func TestNFADefaultGraphRemainsMutableAfterBuild(t *testing.T) {
	machine := buildNFA(t)
	machine.AddState(state{ID: 4})
	machine.AddInput(event{Kind: "c"})
	added, err := machine.AddTransition(state{ID: 3}, event{Kind: "c"}, state{ID: 4})
	if err != nil || !added {
		t.Fatalf("AddTransition() = (%v, %v)", added, err)
	}
	if got, want := machine.Delta(state{ID: 3}, event{Kind: "c"}), states(4); !reflect.DeepEqual(got, want) {
		t.Fatalf("Delta() = %#v, want %#v", got, want)
	}
}

func TestNFAConfigValidation(t *testing.T) {
	validGraph := nfa.GraphFuncs[state, event]{}
	base := nfa.Config[state, event, int, string]{
		States:   states(0),
		Start:    state{ID: 0},
		Graph:    validGraph,
		StateKey: stateKey,
		InputKey: inputKey,
	}
	tests := []struct {
		name   string
		config nfa.Config[state, event, int, string]
	}{
		{name: "empty states", config: func() nfa.Config[state, event, int, string] { c := base; c.States = nil; return c }()},
		{name: "unknown start", config: func() nfa.Config[state, event, int, string] { c := base; c.Start = state{ID: 1}; return c }()},
		{name: "unknown accepting", config: func() nfa.Config[state, event, int, string] { c := base; c.Accepting = states(1); return c }()},
		{name: "nil graph", config: func() nfa.Config[state, event, int, string] { c := base; c.Graph = nil; return c }()},
		{name: "typed nil graph", config: func() nfa.Config[state, event, int, string] { c := base; c.Graph = (*mutableGraph)(nil); return c }()},
		{name: "nil state key", config: func() nfa.Config[state, event, int, string] { c := base; c.StateKey = nil; return c }()},
		{name: "nil input key", config: func() nfa.Config[state, event, int, string] { c := base; c.InputKey = nil; return c }()},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := nfa.New(test.config); err == nil {
				t.Fatal("New() error = nil")
			}
		})
	}
}

func TestNFABuilderValidation(t *testing.T) {
	if _, err := nfa.NewBuilder((func(state) int)(nil), inputKey); err == nil {
		t.Fatal("nil state key error = nil")
	}
	if _, err := nfa.NewBuilder(stateKey, (func(event) string)(nil)); err == nil {
		t.Fatal("nil input key error = nil")
	}
	builder, err := nfa.NewBuilder(stateKey, inputKey)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := builder.Build(); err == nil {
		t.Fatal("Build() without start error = nil")
	}
}
