package fsm

import (
	"context"
	"testing"

	nfapkg "github.com/stnhrsprkwns/fsm/nfa"
	"github.com/stnhrsprkwns/fsm/runner"
)

type nfaGraph struct {
	sym map[int]map[string][]int
	eps map[int][]int
}

func (g nfaGraph) Delta(from int, sym string) []int {
	if m, ok := g.sym[from]; ok {
		return m[sym]
	}
	return nil
}

func (g nfaGraph) Epsilon(from int) []int {
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

type nfaRecordingObserver[S any, E any, SP any, EP any] struct {
	calls int
}

func (o *nfaRecordingObserver[S, E, SP, EP]) OnStep(_ S, _ SP, _ S, _ E, _ EP) {
	o.calls++
}

func TestNFABuilderBuildNFAAccepts(t *testing.T) {
	b := NFA[int, string, struct{}, struct{}]()
	b.WithStart(0).
		WithAccepting(1).
		WithTransition(0, "a", 1).
		WithEpsilon(1, 1)

	n, err := b.BuildNFA()
	if err != nil {
		t.Fatalf("BuildNFA() error = %v", err)
	}
	if !n.Accepts([]string{"a"}) {
		t.Fatalf("nfa should accept [a]")
	}
}

func TestNFABuilderWithGraphOverride(t *testing.T) {
	g := nfaGraph{
		sym: map[int]map[string][]int{
			0: {"a": {1}},
		},
		eps: map[int][]int{},
	}
	b := NFA[int, string, struct{}, struct{}]()
	b.WithGraph(g).
		WithStart(0).
		WithAccepting(1).
		WithStates(0, 1).
		WithAlphabet("a")

	n, err := b.BuildNFA()
	if err != nil {
		t.Fatalf("BuildNFA() error = %v", err)
	}
	if !n.Accepts([]string{"a"}) {
		t.Fatalf("nfa should accept [a] with graph override")
	}
}

func TestNFABuilderWithNFAOverride(t *testing.T) {
	delta := &nfaCountingDeltaer{}
	override := nfapkg.New(nfapkg.Config[int, string]{
		States:    []int{0, 1},
		Alphabet:  []string{"a"},
		Start:     0,
		Accepting: []int{1},
		Deltaer:   delta,
	})

	b := NFA[int, string, struct{}, struct{}]().WithNFA(override)
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
	e.Step("a", struct{}{})
	if delta.calls == 0 {
		t.Fatalf("override deltaer was not called")
	}
}

func TestNFABuilderBuildAtomic(t *testing.T) {
	b := NFA[int, string, struct{}, struct{}]()
	b.WithStart(0).
		WithAccepting(1).
		WithTransition(0, "a", 1)

	a, err := b.BuildAtomicNFA()
	if err != nil {
		t.Fatalf("BuildAtomicNFA() error = %v", err)
	}
	if !a.Accepts([]string{"a"}) {
		t.Fatalf("atomic nfa should accept [a]")
	}
}

func TestNFABuilderBuildEngineObserverOrder(t *testing.T) {
	b := NFA[int, string, struct{}, struct{}]()
	b.WithStart(0).
		WithAccepting(1).
		WithTransition(0, "a", 1).
		WithEpsilon(0, 1)

	var order []string
	b.WithOnStepAny(func(from []int, _ struct{}, to []int, _ string, _ struct{}) {
		order = append(order, "step:any")
	})
	b.WithOnStep([]int{0, 1}, []int{1}, func(from []int, _ struct{}, to []int, _ string, _ struct{}) {
		order = append(order, "step:[0 1]->[1]")
	})
	b.WithOnEnterAny(func(to []int, _ struct{}, _ string, _ struct{}) {
		order = append(order, "enter:any")
	})
	b.WithOnExitAny(func(from []int, _ struct{}, _ string, _ struct{}) {
		order = append(order, "exit:any")
	})

	e, err := b.BuildEngine()
	if err != nil {
		t.Fatalf("BuildEngine() error = %v", err)
	}
	e.Step("a", struct{}{})
	if len(order) != 4 {
		t.Fatalf("order len = %d, want 4", len(order))
	}
	want := []string{"step:any", "step:[0 1]->[1]", "exit:any", "enter:any"}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("order[%d] = %q, want %q", i, order[i], want[i])
		}
	}
}

func TestNFABuilderWithObserverOverride(t *testing.T) {
	b := NFA[int, string, struct{}, struct{}]()
	b.WithStart(0).
		WithAccepting(1).
		WithTransition(0, "a", 1)

	obs := &nfaRecordingObserver[[]int, string, struct{}, struct{}]{}
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

func TestNFABuilderBuildAtomicEngine(t *testing.T) {
	b := NFA[int, string, struct{}, struct{}]()
	b.WithStart(0).
		WithAccepting(1).
		WithTransition(0, "a", 1)

	var calls int
	b.WithOnEnterAny(func(to []int, _ struct{}, _ string, _ struct{}) {
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

func TestNFABuilderBuildRunner(t *testing.T) {
	b := NFA[int, string, struct{}, struct{}]()
	b.WithStart(0).
		WithAccepting(1).
		WithTransition(0, "a", 1)

	var calls int
	b.WithOnStepAny(func(from []int, _ struct{}, to []int, _ string, _ struct{}) {
		calls++
	})

	r, err := b.BuildRunner(1)
	if err != nil {
		t.Fatalf("BuildRunner() error = %v", err)
	}

	done := make(chan error, 1)
	go func() { done <- r.Run(context.Background()) }()

	events := r.Events()
	events <- runner.Event[string, struct{}]{Event: "a", Payload: struct{}{}}
	close(events)

	if err := <-done; err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
	if calls != 1 {
		t.Fatalf("onStepAny calls = %d, want 1", calls)
	}
}

func TestNFABuilderBuildEngineErrorMissingExecutor(t *testing.T) {
	b := NFA[int, string, struct{}, struct{}]()
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
	b := NewNFABuilder[int, byte, struct{}, struct{}]()
	b.WithStart(0).
		WithAccepting(1).
		WithTransition(0, 'a', 1)

	n, err := b.BuildNFA()
	if err != nil {
		t.Fatalf("BuildNFA() error = %v", err)
	}
	if !n.Accepts([]byte{'a'}) {
		t.Fatalf("expected acceptance")
	}
}
