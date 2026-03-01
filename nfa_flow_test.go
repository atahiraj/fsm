package fsm

import (
	"context"
	"testing"

	nfapkg "github.com/stnhrsprkwns/fsm/nfa"
	"github.com/stnhrsprkwns/fsm/runner"
)

type nfaState int

func (s nfaState) Key() int { return int(s) }

type nfaSymbol string

func (s nfaSymbol) Key() string { return string(s) }

type nfaByteSymbol byte

func (s nfaByteSymbol) Key() byte { return byte(s) }

type nfaGraph struct {
	sym map[nfaState]map[nfaSymbol][]nfaState
	eps map[nfaState][]nfaState
}

func (g nfaGraph) Delta(from nfaState, sym nfaSymbol) []nfaState {
	if m, ok := g.sym[from]; ok {
		return m[sym]
	}
	return nil
}

func (g nfaGraph) Epsilon(from nfaState) []nfaState {
	return g.eps[from]
}

type nfaCountingDeltaer struct {
	calls int
}

func (d *nfaCountingDeltaer) Delta(s int, sym string) []int {
	d.calls++
	if sym == "a" {
		return []int{1}
	}
	return nil
}

func (d *nfaCountingDeltaer) Epsilon(s int) []int { return nil }

type nfaRecordingTransitionHooks[S any, E any] struct {
	calls int
}

func (o *nfaRecordingTransitionHooks[S, E]) OnTransition(_ S, _ S, _ E) {
	o.calls++
}

func TestNFABuilderBuildNFAAccepts(t *testing.T) {
	b := NFA[nfaState, nfaSymbol, int, string]()
	b.WithStart(0).
		WithAccepting(1).
		WithTransition(0, "a", 1).
		WithEpsilon(1, 1)

	n, err := b.BuildNFA()
	if err != nil {
		t.Fatalf("BuildNFA() error = %v", err)
	}
	if !n.Accepts([]nfaSymbol{"a"}) {
		t.Fatalf("nfa should accept [a]")
	}
}

func TestNFABuilderWithGraphOverride(t *testing.T) {
	g := nfaGraph{
		sym: map[nfaState]map[nfaSymbol][]nfaState{
			0: {"a": {1}},
		},
		eps: map[nfaState][]nfaState{},
	}
	b := NFA[nfaState, nfaSymbol, int, string]()
	b.WithGraph(g).
		WithStart(0).
		WithAccepting(1).
		WithStates(0, 1).
		WithAlphabet("a")

	n, err := b.BuildNFA()
	if err != nil {
		t.Fatalf("BuildNFA() error = %v", err)
	}
	if !n.Accepts([]nfaSymbol{"a"}) {
		t.Fatalf("nfa should accept [a] with graph override")
	}
}

func TestNFABuilderWithNFAOverride(t *testing.T) {
	delta := &nfaCountingDeltaer{}
	override := nfapkg.New(nfapkg.Config[nfaState, nfaSymbol, int, string]{
		States:    []nfaState{0, 1},
		Alphabet:  []nfaSymbol{"a"},
		Start:     0,
		Accepting: []nfaState{1},
		Deltaer:   delta,
	})

	b := NFA[nfaState, nfaSymbol, int, string]().WithNFA(override)
	n, err := b.BuildNFA()
	if err != nil {
		t.Fatalf("BuildNFA() error = %v", err)
	}
	if n != override {
		t.Fatalf("override NFA was not used")
	}
	e, err := b.BuildEngine()
	if err != nil {
		t.Fatalf("BuildEngine() error = %v", err)
	}
	e.Step("a")
	if delta.calls == 0 {
		t.Fatalf("override deltaer was not called")
	}
}

func TestNFABuilderBuildAtomic(t *testing.T) {
	b := NFA[nfaState, nfaSymbol, int, string]()
	b.WithStart(0).
		WithAccepting(1).
		WithTransition(0, "a", 1)

	a, err := b.BuildAtomicNFA()
	if err != nil {
		t.Fatalf("BuildAtomicNFA() error = %v", err)
	}
	if !a.Accepts([]nfaSymbol{"a"}) {
		t.Fatalf("atomic nfa should accept [a]")
	}
}

func TestNFABuilderBuildEngineTransitionHooksOrder(t *testing.T) {
	b := NFA[nfaState, nfaSymbol, int, string]()
	b.WithStart(0).
		WithAccepting(1).
		WithTransition(0, "a", 1).
		WithEpsilon(0, 1)

	var order []string
	b.WithOnTransitionAny(func(from []nfaState, to []nfaState, e nfaSymbol) {
		order = append(order, "step:any")
	})
	b.WithOnTransition([]nfaState{0, 1}, []nfaState{1}, func(e nfaSymbol) {
		order = append(order, "step:[0 1]->[1]")
	})
	b.WithOnExit([]nfaState{0, 1}, func(e nfaSymbol) {
		order = append(order, "exit:[0 1]")
	})
	b.WithOnEnter([]nfaState{1}, func(e nfaSymbol) {
		order = append(order, "enter:[1]")
	})

	e, err := b.BuildEngine()
	if err != nil {
		t.Fatalf("BuildEngine() error = %v", err)
	}
	e.Step("a")
	if len(order) != 4 {
		t.Fatalf("order len = %d, want 4", len(order))
	}
	want := []string{"exit:[0 1]", "step:any", "step:[0 1]->[1]", "enter:[1]"}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("order[%d] = %q, want %q", i, order[i], want[i])
		}
	}
}

func TestNFABuilderEnterExitCallbacksIgnoreSelfTransition(t *testing.T) {
	b := NFA[nfaState, nfaSymbol, int, string]()
	b.WithStart(0).
		WithAccepting(0).
		WithTransition(0, "a", 0)

	var enterCalls int
	var exitCalls int

	b.WithOnEnter([]nfaState{0}, func(e nfaSymbol) {
		enterCalls++
	})
	b.WithOnExit([]nfaState{0}, func(e nfaSymbol) {
		exitCalls++
	})

	e, err := b.BuildEngine()
	if err != nil {
		t.Fatalf("BuildEngine() error = %v", err)
	}
	e.Step("a")

	if enterCalls != 0 {
		t.Fatalf("onEnter calls = %d, want 0", enterCalls)
	}
	if exitCalls != 0 {
		t.Fatalf("onExit calls = %d, want 0", exitCalls)
	}
}

func TestNFABuilderEnterExitCallbacksIncludeSelfTransitionWhenEnabled(t *testing.T) {
	b := NFA[nfaState, nfaSymbol, int, string]()
	b.WithStart(0).
		WithAccepting(0).
		WithTransition(0, "a", 0).
		WithSelfTransitionCallbacks()

	var enterCalls int
	var exitCalls int

	b.WithOnEnter([]nfaState{0}, func(e nfaSymbol) {
		enterCalls++
	})
	b.WithOnExit([]nfaState{0}, func(e nfaSymbol) {
		exitCalls++
	})

	e, err := b.BuildEngine()
	if err != nil {
		t.Fatalf("BuildEngine() error = %v", err)
	}
	e.Step("a")

	if enterCalls != 1 {
		t.Fatalf("onEnter calls = %d, want 1", enterCalls)
	}
	if exitCalls != 1 {
		t.Fatalf("onExit calls = %d, want 1", exitCalls)
	}
}

func TestNFABuilderStateCallbacks(t *testing.T) {
	b := NFA[nfaState, nfaSymbol, int, string]()
	b.WithStart(0).
		WithAccepting(1).
		WithTransition(0, "a", 1)

	var got []string
	b.WithOnExitState(0, func(e nfaSymbol) {
		got = append(got, "exit:0")
	})
	b.WithOnExitState(2, func(e nfaSymbol) {
		got = append(got, "exit:2")
	})
	b.WithOnEnterState(1, func(e nfaSymbol) {
		got = append(got, "enter:1")
	})
	b.WithOnEnterState(3, func(e nfaSymbol) {
		got = append(got, "enter:3")
	})

	e, err := b.BuildEngine()
	if err != nil {
		t.Fatalf("BuildEngine() error = %v", err)
	}
	e.Step("a")

	want := []string{"exit:0", "enter:1"}
	if len(got) != len(want) {
		t.Fatalf("callbacks len = %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("callbacks[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestNFABuilderWithTransitionHooksOverride(t *testing.T) {
	b := NFA[nfaState, nfaSymbol, int, string]()
	b.WithStart(0).
		WithAccepting(1).
		WithTransition(0, "a", 1)

	hooks := &nfaRecordingTransitionHooks[[]nfaState, nfaSymbol]{}
	b.WithTransitionHooks(hooks)

	e, err := b.BuildEngine()
	if err != nil {
		t.Fatalf("BuildEngine() error = %v", err)
	}
	e.Step("a")
	if hooks.calls != 1 {
		t.Fatalf("transition hooks calls = %d, want 1", hooks.calls)
	}
}

func TestNFABuilderBuildAtomicEngine(t *testing.T) {
	b := NFA[nfaState, nfaSymbol, int, string]()
	b.WithStart(0).
		WithAccepting(1).
		WithTransition(0, "a", 1)

	var calls int
	b.WithOnEnter([]nfaState{1}, func(e nfaSymbol) {
		calls++
	})

	e, err := b.BuildAtomicEngine()
	if err != nil {
		t.Fatalf("BuildAtomicEngine() error = %v", err)
	}
	e.Step("a")
	if calls != 1 {
		t.Fatalf("onEnter calls = %d, want 1", calls)
	}
}

func TestNFABuilderBuildRunner(t *testing.T) {
	b := NFA[nfaState, nfaSymbol, int, string]()
	b.WithStart(0).
		WithAccepting(1).
		WithTransition(0, "a", 1)

	var calls int
	b.WithOnTransitionAny(func(from []nfaState, to []nfaState, e nfaSymbol) {
		calls++
	})

	r, err := b.BuildRunner(1)
	if err != nil {
		t.Fatalf("BuildRunner() error = %v", err)
	}

	done := make(chan error, 1)
	go func() { done <- r.Run(context.Background()) }()

	events := r.Events()
	events <- runner.Event[nfaSymbol]{Event: "a"}
	close(events)

	if err := <-done; err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
	if calls != 1 {
		t.Fatalf("onStepAny calls = %d, want 1", calls)
	}
}

func TestNFABuilderBuildEngineErrorMissingExecutor(t *testing.T) {
	b := NFA[nfaState, nfaSymbol, int, string]()
	b.WithStart(0).
		WithAccepting(1).
		WithTransition(0, "a", 1).
		WithExecutor(nil)

	if _, err := b.BuildEngine(); err == nil {
		t.Fatalf("expected error when executor is nil")
	}
	if _, err := b.BuildAtomicEngine(); err == nil {
		t.Fatalf("expected error when executor is nil for atomic engine")
	}
}

func TestFacadeNFABuilder(t *testing.T) {
	b := NewNFABuilder[nfaState, nfaByteSymbol, int, byte]()
	b.WithStart(0).
		WithAccepting(1).
		WithTransition(0, 'a', 1)

	n, err := b.BuildNFA()
	if err != nil {
		t.Fatalf("BuildNFA() error = %v", err)
	}
	if !n.Accepts([]nfaByteSymbol{'a'}) {
		t.Fatalf("expected acceptance")
	}
}
