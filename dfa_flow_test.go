package fsm

import (
	"testing"

	"github.com/stnhrsprkwns/fsm/dfa"
)

type dfaGraph struct {
	next map[int]map[string]int
}

func (g dfaGraph) Delta(from int, label string) []int {
	if g.next == nil {
		return nil
	}
	if m, ok := g.next[from]; ok {
		if to, ok := m[label]; ok {
			return []int{to}
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

type dfaRecordingObserver[S any, E any, SP any, EP any] struct {
	calls int
}

func (o *dfaRecordingObserver[S, E, SP, EP]) OnStep(_ S, _ SP, _ S, _ E, _ EP) {
	o.calls++
}

func TestDefaultExecutor(t *testing.T) {
	exec := DefaultExecutor[int, string, struct{}, struct{}]{}
	called := false
	exec.Execute(func(from int, _ struct{}, to int, _ string, _ struct{}) {
		called = true
	}, 1, struct{}{}, 2, "a", struct{}{})
	if !called {
		t.Fatalf("callback not executed")
	}
}

func TestDFABuilderBuildDFAAccepts(t *testing.T) {
	b := DFA[int, string, struct{}, struct{}]()
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
	if !d.Accepts([]string{"a", "b"}) {
		t.Fatalf("dfa should accept [a b]")
	}
	if d.Accepts([]string{"a", "a"}) {
		t.Fatalf("dfa should reject [a a]")
	}
}

func TestDFABuilderWithGraphOverride(t *testing.T) {
	g := dfaGraph{next: map[int]map[string]int{
		0: {"a": 1},
	}}
	b := DFA[int, string, struct{}, struct{}]()
	b.WithGraph(g).
		WithStart(0).
		WithAccepting(1).
		WithStates(0, 1).
		WithAlphabet("a")

	d, err := b.BuildDFA()
	if err != nil {
		t.Fatalf("BuildDFA() error = %v", err)
	}
	if !d.Accepts([]string{"a"}) {
		t.Fatalf("dfa should accept [a] with graph override")
	}
}

func TestDFABuilderWithDFAOverride(t *testing.T) {
	delta := &dfaCountingDeltaer{}
	override := dfa.New(dfa.Config[int, string]{
		States:    []int{0, 1},
		Alphabet:  []string{"a"},
		Start:     0,
		Accepting: []int{1},
		Deltaer:   delta,
	})

	b := DFA[int, string, struct{}, struct{}]().WithDFA(override)
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
	e.Step("a", struct{}{})
	if delta.calls == 0 {
		t.Fatalf("override deltaer was not called")
	}
}

func TestDFABuilderBuildAtomic(t *testing.T) {
	b := DFA[int, string, struct{}, struct{}]()
	b.WithStart(0).
		WithAccepting(1).
		WithTransition(0, "a", 1)

	a, err := b.BuildAtomicDFA()
	if err != nil {
		t.Fatalf("BuildAtomicDFA() error = %v", err)
	}
	if !a.Accepts([]string{"a"}) {
		t.Fatalf("atomic dfa should accept [a]")
	}
}

func TestDFABuilderBuildEngineObserverOrder(t *testing.T) {
	b := DFA[int, string, struct{}, struct{}]()
	b.WithStart(0).
		WithAccepting(1).
		WithTransition(0, "a", 1).
		WithTransition(1, "a", 1).
		WithTransition(1, "b", 0)

	var order []string
	b.WithOnStepAny(func(from int, _ struct{}, to int, _ string, _ struct{}) {
		order = append(order, "step:any")
	})
	b.WithOnStep(0, 1, func(from int, _ struct{}, to int, _ string, _ struct{}) {
		order = append(order, "step:0->1")
	})
	b.WithOnEnterAny(func(to int, _ struct{}, _ string, _ struct{}) {
		order = append(order, "enter:any")
	})
	b.WithOnExitAny(func(from int, _ struct{}, _ string, _ struct{}) {
		order = append(order, "exit:any")
	})
	b.WithOnExit(0, func(from int, _ struct{}, _ string, _ struct{}) {
		order = append(order, "exit:0")
	})
	b.WithOnEnter(1, func(to int, _ struct{}, _ string, _ struct{}) {
		order = append(order, "enter:1")
	})

	e, err := b.BuildEngine()
	if err != nil {
		t.Fatalf("BuildEngine() error = %v", err)
	}
	e.Step("a", struct{}{})

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

func TestDFABuilderWithObserverOverride(t *testing.T) {
	b := DFA[int, string, struct{}, struct{}]()
	b.WithStart(0).
		WithAccepting(1).
		WithTransition(0, "a", 1)

	obs := &dfaRecordingObserver[int, string, struct{}, struct{}]{}
	b.WithObserver(obs)

	e, err := b.BuildEngine()
	if err != nil {
		t.Fatalf("BuildEngine() error = %v", err)
	}
	e.Step("a", struct{}{})
	if obs.calls != 1 {
		t.Fatalf("observer calls = %d, want 1", obs.calls)
	}
}

func TestDFABuilderBuildAtomicEngine(t *testing.T) {
	b := DFA[int, string, struct{}, struct{}]()
	b.WithStart(0).
		WithAccepting(1).
		WithTransition(0, "a", 1)

	var calls int
	b.WithOnEnterAny(func(to int, _ struct{}, _ string, _ struct{}) {
		calls++
	})

	e, err := b.BuildAtomicEngine()
	if err != nil {
		t.Fatalf("BuildAtomicEngine() error = %v", err)
	}
	e.Step("a", struct{}{})
	if calls != 1 {
		t.Fatalf("onEnterAny calls = %d, want 1", calls)
	}
}

func TestDFABuilderBuildEngineErrorMissingExecutor(t *testing.T) {
	b := DFA[int, string, struct{}, struct{}]()
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
	b := DFA[int, string, struct{}, struct{}]()
	b.WithStart(0).
		WithAccepting(1).
		WithTransition(0, "a", 1).
		WithTransition(0, "a", 0)

	if _, err := b.BuildDFA(); err == nil {
		t.Fatalf("expected error from nondeterministic transition")
	}
}

func TestFacadeDFAFlow(t *testing.T) {
	b := DFA[int, byte, struct{}, struct{}]()
	b.WithStart(0).
		WithAccepting(1).
		WithTransition(0, 'a', 1)

	e, err := b.BuildEngine()
	if err != nil {
		t.Fatalf("BuildEngine() error = %v", err)
	}
	e.Step('a', struct{}{})
	if !e.Accepting() {
		t.Fatalf("expected accepting state")
	}
}
