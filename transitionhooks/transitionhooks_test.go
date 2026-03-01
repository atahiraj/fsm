package transitionhooks

import "testing"

type transitionCall struct {
	name string
	from int
	to   int
	evt  string
}

type recordingExecutor struct {
	calls int
}

func (e *recordingExecutor) Execute(f func(from int, to int, evt string), from int, to int, evt string) {
	e.calls++
	f(from, to, evt)
}

func TestNewInitializesEmptyCallbacks(t *testing.T) {
	exec := &recordingExecutor{}
	o := New[int, string](exec)

	if o == nil {
		t.Fatalf("transition hooks are nil")
	}
	if o.executor != exec {
		t.Fatalf("executor was not stored")
	}
	if o.callbacks == nil {
		t.Fatalf("callbacks should be initialized to an empty slice")
	}
	if len(o.callbacks) != 0 {
		t.Fatalf("callbacks len = %d, want 0", len(o.callbacks))
	}
}

func TestOnTransitionDispatchesCallbacksInOrder(t *testing.T) {
	var got []transitionCall
	o := New[int, string](
		DefaultExecutor[int, string]{},
		func(from int, to int, evt string) {
			got = append(got, transitionCall{name: "first", from: from, to: to, evt: evt})
		},
		func(from int, to int, evt string) {
			got = append(got, transitionCall{name: "second", from: from, to: to, evt: evt})
		},
	)

	o.OnTransition(1, 2, "tick")

	want := []transitionCall{
		{name: "first", from: 1, to: 2, evt: "tick"},
		{name: "second", from: 1, to: 2, evt: "tick"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d calls, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("call %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestDefaultExecutorExecuteCallsCallback(t *testing.T) {
	exec := DefaultExecutor[int, string]{}
	called := false

	exec.Execute(func(from int, to int, evt string) {
		if from != 1 || to != 2 || evt != "go" {
			t.Fatalf("unexpected args: from=%d to=%d evt=%q", from, to, evt)
		}
		called = true
	}, 1, 2, "go")

	if !called {
		t.Fatalf("callback not executed")
	}
}
