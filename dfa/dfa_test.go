package dfa_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/atahiraj/fsm"
	"github.com/atahiraj/fsm/dfa"
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
	next map[int]map[string]state
}

func newMutableGraph() *mutableGraph {
	return &mutableGraph{next: make(map[int]map[string]state)}
}

func (g *mutableGraph) Delta(from state, input event) (state, bool) {
	to, ok := g.next[from.ID][input.Kind]
	return to, ok
}

func (g *mutableGraph) AddTransition(from state, input event, to state) error {
	byInput := g.next[from.ID]
	if byInput == nil {
		byInput = make(map[string]state)
		g.next[from.ID] = byInput
	}
	if existing, ok := byInput[input.Kind]; ok {
		if existing.ID == to.ID {
			return nil
		}
		return dfa.ErrTransitionExists
	}
	byInput[input.Kind] = to
	return nil
}

func (g *mutableGraph) RemoveTransition(from state, input event) bool {
	byInput := g.next[from.ID]
	if _, ok := byInput[input.Kind]; !ok {
		return false
	}
	delete(byInput, input.Kind)
	return true
}

var _ dfa.MutableGraph[state, event] = (*mutableGraph)(nil)
var _ fsm.Machine[state, event] = (*dfa.DFA[state, event, int, string])(nil)

func buildDFA(t *testing.T) *dfa.DFA[state, event, int, string] {
	t.Helper()
	builder, err := dfa.NewBuilder(stateKey, inputKey)
	if err != nil {
		t.Fatalf("NewBuilder() error = %v", err)
	}
	builder.SetStart(state{ID: 0})
	builder.AddAccepting(state{ID: 2})
	for _, transition := range []struct {
		from  int
		input string
		to    int
	}{
		{0, "a", 1},
		{1, "b", 2},
		{2, "a", 2},
	} {
		if err := builder.AddTransition(
			state{ID: transition.from},
			event{Kind: transition.input},
			state{ID: transition.to},
		); err != nil {
			t.Fatalf("AddTransition() error = %v", err)
		}
	}
	machine, err := builder.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	return machine
}

func TestDFAAcceptsAndUsesInputIdentity(t *testing.T) {
	machine := buildDFA(t)
	word := []event{{Kind: "a", Payload: "first"}, {Kind: "b", Payload: "second"}}
	if !machine.Accepts(word) {
		t.Fatal("DFA should accept [a b]")
	}
	if machine.Accepts([]event{{Kind: "a"}, {Kind: "a"}}) {
		t.Fatal("DFA should reject [a a]")
	}
	if _, ok := machine.Delta(state{ID: 0}, event{Kind: "missing"}); ok {
		t.Fatal("missing transition exists")
	}
}

func TestDFAStableEnumerationAndCanonicalValues(t *testing.T) {
	graph := dfa.GraphFunc[state, event](func(state, event) (state, bool) { return state{}, false })
	machine, err := dfa.New(dfa.Config[state, event, int, string]{
		States: []state{
			{ID: 2, Name: "two"},
			{ID: 1, Name: "old"},
			{ID: 1, Name: "new"},
			{ID: 3, Name: "three"},
		},
		Alphabet:  []event{{Kind: "b"}, {Kind: "a"}, {Kind: "b", Payload: "new"}},
		Start:     state{ID: 2},
		Accepting: []state{{ID: 3}, {ID: 1}},
		Graph:     graph,
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

func TestDFAInjectedGraphIsSharedAndMutable(t *testing.T) {
	graph := newMutableGraph()
	if err := graph.AddTransition(state{ID: 0}, event{Kind: "go"}, state{ID: 1}); err != nil {
		t.Fatal(err)
	}
	machine, err := dfa.New(dfa.Config[state, event, int, string]{
		States:    states(0, 1, 2),
		Alphabet:  []event{{Kind: "go"}, {Kind: "next"}},
		Start:     state{ID: 0},
		Accepting: []state{{ID: 2}},
		Graph:     graph,
		StateKey:  stateKey,
		InputKey:  inputKey,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := graph.AddTransition(state{ID: 1}, event{Kind: "next"}, state{ID: 2}); err != nil {
		t.Fatal(err)
	}
	if !machine.Accepts([]event{{Kind: "go"}, {Kind: "next"}}) {
		t.Fatal("external graph mutation was not visible")
	}
	removed, err := machine.RemoveTransition(state{ID: 1}, event{Kind: "next"})
	if err != nil || !removed {
		t.Fatalf("RemoveTransition() = (%v, %v)", removed, err)
	}
	if _, ok := graph.Delta(state{ID: 1}, event{Kind: "next"}); ok {
		t.Fatal("machine mutation was not visible in injected graph")
	}
}

func TestDFAReadOnlyGraphAndStrictMutations(t *testing.T) {
	graph := dfa.GraphFunc[state, event](func(from state, input event) (state, bool) {
		if from.ID == 0 && input.Kind == "go" {
			return state{ID: 1}, true
		}
		return state{}, false
	})
	machine, err := dfa.New(dfa.Config[state, event, int, string]{
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
	if err := machine.AddTransition(state{ID: 0}, event{Kind: "go"}, state{ID: 1}); !errors.Is(err, dfa.ErrReadOnlyGraph) {
		t.Fatalf("AddTransition() error = %v", err)
	}
	if err := machine.AddTransition(state{ID: 0}, event{Kind: "unknown"}, state{ID: 1}); err == nil {
		t.Fatal("unknown input mutation error = nil")
	}
	if err := machine.SetStart(state{ID: 9}); err == nil {
		t.Fatal("SetStart(unknown) error = nil")
	}
	if err := machine.AddAccepting(state{ID: 9}); err == nil {
		t.Fatal("AddAccepting(unknown) error = nil")
	}
}

func TestDFABuilderRejectsGraphMixingInEitherOrder(t *testing.T) {
	graph := newMutableGraph()

	first, err := dfa.NewBuilder(stateKey, inputKey)
	if err != nil {
		t.Fatal(err)
	}
	if err := first.AddTransition(state{ID: 0}, event{Kind: "go"}, state{ID: 1}); err != nil {
		t.Fatal(err)
	}
	if err := first.WithGraph(graph); !errors.Is(err, dfa.ErrMixedGraph) {
		t.Fatalf("WithGraph() error = %v", err)
	}

	second, err := dfa.NewBuilder(stateKey, inputKey)
	if err != nil {
		t.Fatal(err)
	}
	if err := second.WithGraph(graph); err != nil {
		t.Fatal(err)
	}
	if err := second.AddTransition(state{ID: 0}, event{Kind: "go"}, state{ID: 1}); !errors.Is(err, dfa.ErrMixedGraph) {
		t.Fatalf("AddTransition() error = %v", err)
	}
}

func TestDFABuilderSnapshotsOwnedGraph(t *testing.T) {
	builder, err := dfa.NewBuilder(stateKey, inputKey)
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
	if err := builder.AddTransition(state{ID: 1}, event{Kind: "b"}, state{ID: 2}); err != nil {
		t.Fatal(err)
	}
	second, err := builder.Build()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := first.Delta(state{ID: 1}, event{Kind: "b"}); ok {
		t.Fatal("first build changed with builder")
	}
	if next, ok := second.Delta(state{ID: 1}, event{Kind: "b"}); !ok || next.ID != 2 {
		t.Fatalf("second Delta() = (%+v, %v)", next, ok)
	}
}

func TestDFADefaultGraphRemainsMutableAfterBuild(t *testing.T) {
	machine := buildDFA(t)
	machine.AddState(state{ID: 3})
	machine.AddInput(event{Kind: "c"})
	if err := machine.AddTransition(state{ID: 2}, event{Kind: "c"}, state{ID: 3}); err != nil {
		t.Fatalf("AddTransition() error = %v", err)
	}
	if next, ok := machine.Delta(state{ID: 2}, event{Kind: "c"}); !ok || next.ID != 3 {
		t.Fatalf("Delta() = (%+v, %v)", next, ok)
	}
}

func TestDFAConflictingTransition(t *testing.T) {
	builder, err := dfa.NewBuilder(stateKey, inputKey)
	if err != nil {
		t.Fatal(err)
	}
	if err := builder.AddTransition(state{ID: 0}, event{Kind: "go"}, state{ID: 1}); err != nil {
		t.Fatal(err)
	}
	if err := builder.AddTransition(state{ID: 0}, event{Kind: "go"}, state{ID: 1}); err != nil {
		t.Fatalf("identical transition error = %v", err)
	}
	if err := builder.AddTransition(state{ID: 0}, event{Kind: "go"}, state{ID: 2}); !errors.Is(err, dfa.ErrTransitionExists) {
		t.Fatalf("conflicting transition error = %v", err)
	}
}

func TestDFAConfigValidation(t *testing.T) {
	validGraph := dfa.GraphFunc[state, event](func(state, event) (state, bool) { return state{}, false })
	base := dfa.Config[state, event, int, string]{
		States:   states(0),
		Start:    state{ID: 0},
		Graph:    validGraph,
		StateKey: stateKey,
		InputKey: inputKey,
	}
	tests := []struct {
		name   string
		config dfa.Config[state, event, int, string]
	}{
		{name: "empty states", config: func() dfa.Config[state, event, int, string] { c := base; c.States = nil; return c }()},
		{name: "unknown start", config: func() dfa.Config[state, event, int, string] { c := base; c.Start = state{ID: 1}; return c }()},
		{name: "unknown accepting", config: func() dfa.Config[state, event, int, string] { c := base; c.Accepting = states(1); return c }()},
		{name: "nil graph", config: func() dfa.Config[state, event, int, string] { c := base; c.Graph = nil; return c }()},
		{name: "typed nil graph", config: func() dfa.Config[state, event, int, string] {
			c := base
			c.Graph = dfa.GraphFunc[state, event](nil)
			return c
		}()},
		{name: "nil state key", config: func() dfa.Config[state, event, int, string] { c := base; c.StateKey = nil; return c }()},
		{name: "nil input key", config: func() dfa.Config[state, event, int, string] { c := base; c.InputKey = nil; return c }()},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := dfa.New(test.config); err == nil {
				t.Fatal("New() error = nil")
			}
		})
	}
}

func TestDFABuilderValidation(t *testing.T) {
	if _, err := dfa.NewBuilder((func(state) int)(nil), inputKey); err == nil {
		t.Fatal("nil state key error = nil")
	}
	if _, err := dfa.NewBuilder(stateKey, (func(event) string)(nil)); err == nil {
		t.Fatal("nil input key error = nil")
	}
	builder, err := dfa.NewBuilder(stateKey, inputKey)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := builder.Build(); err == nil {
		t.Fatal("Build() without start error = nil")
	}
}
