package fsm

import (
	"context"
	"testing"

	"github.com/atahiraj/fsm/dfa"
	"github.com/atahiraj/fsm/runner"
)

type dfaState int

func (s dfaState) Key() int { return int(s) }

type dfaInput string

func (s dfaInput) Key() string { return string(s) }

type dfaByteInput byte

func (s dfaByteInput) Key() byte { return byte(s) }

type dfaEvent struct {
	kind    string
	payload string
}

func (e dfaEvent) Key() string { return e.kind }

type dfaGraph struct {
	next map[dfaState]map[dfaInput]dfaState
}

func (g dfaGraph) Delta(from dfaState, label dfaInput) []dfaState {
	if g.next == nil {
		return nil
	}
	if m, ok := g.next[from]; ok {
		if to, ok := m[label]; ok {
			return []dfaState{to}
		}
	}
	return nil
}

type dfaCountingDeltaer struct {
	calls int
}

func (d *dfaCountingDeltaer) Delta(s int, sym string) (int, bool) {
	d.calls++
	if sym == "a" {
		return 1, true
	}
	return 0, true
}

type dfaRecordingTransitionHooks[S any, E any] struct {
	calls int
}

func (o *dfaRecordingTransitionHooks[S, E]) OnTransition(_ S, _ S, _ E) {
	o.calls++
}

func TestDefaultExecutor(t *testing.T) {
	exec := DefaultExecutor[dfaState, dfaInput]{}
	called := false
	exec.Execute(func(from dfaState, to dfaState, e dfaInput) {
		called = true
	}, 1, 2, "a")
	if !called {
		t.Fatalf("callback not executed")
	}
}

func TestDFABuilderBuildDFAAccepts(t *testing.T) {
	b := DFA[dfaState, dfaInput]()
	b.WithStart(0).
		WithAccepting(2).
		WithTransition(0, "a", 1).
		WithTransition(0, "b", 0).
		WithTransition(1, "a", 1).
		WithTransition(1, "b", 2).
		WithTransition(2, "a", 2).
		WithTransition(2, "b", 2)

	d, err := b.BuildDFA()
	if err != nil {
		t.Fatalf("BuildDFA() error = %v", err)
	}
	if !d.Accepts([]dfaInput{"a", "b"}) {
		t.Fatalf("dfa should accept [a b]")
	}
	if d.Accepts([]dfaInput{"a", "a"}) {
		t.Fatalf("dfa should reject [a a]")
	}
}

func TestDFABuilderWithTransitionHook(t *testing.T) {
	b := DFA[dfaState, dfaInput]()

	var order []string
	b.WithStart(0).
		WithAccepting(1).
		WithTransitionHook(
			0,
			"a",
			1,
			func(e dfaInput) {
				if e != "a" {
					t.Fatalf("unexpected input = %q, want %q", e, "a")
				}
				order = append(order, "first")
			},
			func(e dfaInput) {
				order = append(order, "second")
			},
		)

	e, err := b.BuildEngine()
	if err != nil {
		t.Fatalf("BuildEngine() error = %v", err)
	}
	e.Step("a")

	if len(order) != 2 {
		t.Fatalf("transition hook calls = %d, want 2", len(order))
	}
	if order[0] != "first" || order[1] != "second" {
		t.Fatalf("transition hook order = %v, want [first second]", order)
	}
}

func TestDFABuilderWithTransitionUsesInputKey(t *testing.T) {
	b := DFA[dfaState, dfaEvent]()
	b.WithStart(0).
		WithAccepting(1).
		WithTransition(0, "go", 1)

	var gotPayload string
	b.WithOnTransition(0, 1, func(e dfaEvent) {
		gotPayload = e.payload
	})

	e, err := b.BuildEngine()
	if err != nil {
		t.Fatalf("BuildEngine() error = %v", err)
	}
	e.Step(dfaEvent{kind: "go", payload: "p-123"})

	if gotPayload != "p-123" {
		t.Fatalf("payload = %q, want %q", gotPayload, "p-123")
	}
	if !e.Accepting() {
		t.Fatalf("engine should be accepting after transition")
	}
}

func TestDFABuilderWithGraphOverride(t *testing.T) {
	g := dfaGraph{next: map[dfaState]map[dfaInput]dfaState{
		0: {"a": 1},
	}}
	b := DFA[dfaState, dfaInput]()
	b.WithGraph(g).
		WithStart(0).
		WithAccepting(1).
		WithStates(0, 1).
		WithAlphabet("a")

	d, err := b.BuildDFA()
	if err != nil {
		t.Fatalf("BuildDFA() error = %v", err)
	}
	if !d.Accepts([]dfaInput{"a"}) {
		t.Fatalf("dfa should accept [a] with graph override")
	}
}

func TestDFABuilderWithDFAOverride(t *testing.T) {
	delta := &dfaCountingDeltaer{}
	override := dfa.New(dfa.Config[dfaState, dfaInput, int, string]{
		States:    []dfaState{0, 1},
		Alphabet:  []dfaInput{"a"},
		Start:     0,
		Accepting: []dfaState{1},
		Deltaer:   delta,
	})

	b := DFA[dfaState, dfaInput]().WithDFA(override)
	d, err := b.BuildDFA()
	if err != nil {
		t.Fatalf("BuildDFA() error = %v", err)
	}
	if d != override {
		t.Fatalf("override DFA was not used")
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

func TestDFABuilderBuildAtomic(t *testing.T) {
	b := DFA[dfaState, dfaInput]()
	b.WithStart(0).
		WithAccepting(1).
		WithTransition(0, "a", 1)

	a, err := b.BuildAtomicDFA()
	if err != nil {
		t.Fatalf("BuildAtomicDFA() error = %v", err)
	}
	if !a.Accepts([]dfaInput{"a"}) {
		t.Fatalf("atomic dfa should accept [a]")
	}
}

func TestDFABuilderBuildEngineTransitionHooksOrder(t *testing.T) {
	b := DFA[dfaState, dfaInput]()
	b.WithStart(0).
		WithAccepting(1).
		WithTransition(0, "a", 1).
		WithTransition(1, "a", 1).
		WithTransition(1, "b", 0)

	var order []string
	b.WithOnTransitionAny(func(from dfaState, to dfaState, e dfaInput) {
		order = append(order, "step:any")
	})
	b.WithOnTransition(0, 1, func(e dfaInput) {
		order = append(order, "step:0->1")
	})
	b.WithOnEnterAny(func(to dfaState, e dfaInput) {
		order = append(order, "enter:any")
	})
	b.WithOnExitAny(func(from dfaState, e dfaInput) {
		order = append(order, "exit:any")
	})
	b.WithOnExit(0, func(e dfaInput) {
		order = append(order, "exit:0")
	})
	b.WithOnEnter(1, func(e dfaInput) {
		order = append(order, "enter:1")
	})

	e, err := b.BuildEngine()
	if err != nil {
		t.Fatalf("BuildEngine() error = %v", err)
	}
	e.Step("a")

	wantOrder := []string{"step:any", "step:0->1", "exit:any", "exit:0", "enter:any", "enter:1"}
	if len(order) != len(wantOrder) {
		t.Fatalf("order len = %d, want %d", len(order), len(wantOrder))
	}
	for i := range wantOrder {
		if order[i] != wantOrder[i] {
			t.Fatalf("order[%d] = %q, want %q", i, order[i], wantOrder[i])
		}
	}
}

func TestDFABuilderEnterExitCallbacksIgnoreSelfTransition(t *testing.T) {
	b := DFA[dfaState, dfaInput]()
	b.WithStart(0).
		WithAccepting(0).
		WithTransition(0, "a", 0)

	var enterAnyCalls int
	var enterCalls int
	var exitAnyCalls int
	var exitCalls int

	b.WithOnEnterAny(func(to dfaState, e dfaInput) {
		enterAnyCalls++
	})
	b.WithOnEnter(0, func(e dfaInput) {
		enterCalls++
	})
	b.WithOnExitAny(func(from dfaState, e dfaInput) {
		exitAnyCalls++
	})
	b.WithOnExit(0, func(e dfaInput) {
		exitCalls++
	})

	e, err := b.BuildEngine()
	if err != nil {
		t.Fatalf("BuildEngine() error = %v", err)
	}
	e.Step("a")

	if enterAnyCalls != 0 {
		t.Fatalf("onEnterAny calls = %d, want 0", enterAnyCalls)
	}
	if enterCalls != 0 {
		t.Fatalf("onEnter calls = %d, want 0", enterCalls)
	}
	if exitAnyCalls != 0 {
		t.Fatalf("onExitAny calls = %d, want 0", exitAnyCalls)
	}
	if exitCalls != 0 {
		t.Fatalf("onExit calls = %d, want 0", exitCalls)
	}
}

func TestDFABuilderEnterExitCallbacksIncludeSelfTransitionWhenEnabled(t *testing.T) {
	b := DFA[dfaState, dfaInput]()
	b.WithStart(0).
		WithAccepting(0).
		WithTransition(0, "a", 0).
		WithSelfTransitionCallbacks()

	var enterAnyCalls int
	var enterCalls int
	var exitAnyCalls int
	var exitCalls int

	b.WithOnEnterAny(func(to dfaState, e dfaInput) {
		enterAnyCalls++
	})
	b.WithOnEnter(0, func(e dfaInput) {
		enterCalls++
	})
	b.WithOnExitAny(func(from dfaState, e dfaInput) {
		exitAnyCalls++
	})
	b.WithOnExit(0, func(e dfaInput) {
		exitCalls++
	})

	e, err := b.BuildEngine()
	if err != nil {
		t.Fatalf("BuildEngine() error = %v", err)
	}
	e.Step("a")

	if enterAnyCalls != 1 {
		t.Fatalf("onEnterAny calls = %d, want 1", enterAnyCalls)
	}
	if enterCalls != 1 {
		t.Fatalf("onEnter calls = %d, want 1", enterCalls)
	}
	if exitAnyCalls != 1 {
		t.Fatalf("onExitAny calls = %d, want 1", exitAnyCalls)
	}
	if exitCalls != 1 {
		t.Fatalf("onExit calls = %d, want 1", exitCalls)
	}
}

func TestDFABuilderWithTransitionHooksOverride(t *testing.T) {
	b := DFA[dfaState, dfaInput]()
	b.WithStart(0).
		WithAccepting(1).
		WithTransition(0, "a", 1)

	hooks := &dfaRecordingTransitionHooks[dfaState, dfaInput]{}
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

func TestDFABuilderBuildAtomicEngine(t *testing.T) {
	b := DFA[dfaState, dfaInput]()
	b.WithStart(0).
		WithAccepting(1).
		WithTransition(0, "a", 1)

	var calls int
	b.WithOnEnterAny(func(to dfaState, e dfaInput) {
		calls++
	})

	e, err := b.BuildAtomicEngine()
	if err != nil {
		t.Fatalf("BuildAtomicEngine() error = %v", err)
	}
	e.Step("a")
	if calls != 1 {
		t.Fatalf("onEnterAny calls = %d, want 1", calls)
	}
}

func TestDFABuilderBuildRunner(t *testing.T) {
	b := DFA[dfaState, dfaInput]()
	b.WithStart(0).
		WithAccepting(1).
		WithTransition(0, "a", 1)

	var calls int
	b.WithOnTransitionAny(func(from dfaState, to dfaState, e dfaInput) {
		calls++
	})

	r, err := b.BuildRunner(1)
	if err != nil {
		t.Fatalf("BuildRunner() error = %v", err)
	}

	done := make(chan error, 1)
	go func() { done <- r.Run(context.Background()) }()

	events := r.Events()
	events <- runner.Event[dfaInput]{Event: "a"}
	close(events)

	if err := <-done; err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
	if calls != 1 {
		t.Fatalf("onStepAny calls = %d, want 1", calls)
	}
}

func TestDFABuilderBuildEngineErrorMissingExecutor(t *testing.T) {
	b := DFA[dfaState, dfaInput]()
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

func TestDFABuilderTransitionErrorPropagates(t *testing.T) {
	b := DFA[dfaState, dfaInput]()
	b.WithStart(0).
		WithAccepting(1).
		WithTransition(0, "a", 1).
		WithTransition(0, "a", 0)

	if _, err := b.BuildDFA(); err == nil {
		t.Fatalf("expected error from nondeterministic transition")
	}
}

func TestFacadeDFAFlow(t *testing.T) {
	b := DFA[dfaState, dfaByteInput]()
	b.WithStart(0).
		WithAccepting(1).
		WithTransition(0, 'a', 1)

	e, err := b.BuildEngine()
	if err != nil {
		t.Fatalf("BuildEngine() error = %v", err)
	}
	e.Step('a')
	if !e.Accepting() {
		t.Fatalf("expected accepting state")
	}
}
