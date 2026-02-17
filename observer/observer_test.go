package observer

import "testing"

type stepCall struct {
	name string
	from int
	to   int
	evt  string
}

type recordingExecutor struct {
	calls int
}

func (e *recordingExecutor) Execute(f func(from int, sp struct{}, to int, evt string, ep struct{}), from int, sp struct{}, to int, evt string, ep struct{}) {
	e.calls++
	f(from, sp, to, evt, ep)
}

func TestNewObserverInitializesEmptyCallbacks(t *testing.T) {
	exec := &recordingExecutor{}
	o := NewObserver[int, string, struct{}, struct{}, int](exec)

	if o == nil {
		t.Fatalf("observer is nil")
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

func TestOnStepDispatchesCallbacksInOrder(t *testing.T) {
	var got []stepCall
	o := NewObserver[int, string, struct{}, struct{}, int](
		DefaultExecutor[int, string, struct{}, struct{}]{},
		func(from int, _ struct{}, to int, evt string, _ struct{}) {
			got = append(got, stepCall{name: "first", from: from, to: to, evt: evt})
		},
		func(from int, _ struct{}, to int, evt string, _ struct{}) {
			got = append(got, stepCall{name: "second", from: from, to: to, evt: evt})
		},
	)

	o.OnStep(1, struct{}{}, 2, "tick", struct{}{})

	want := []stepCall{
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
	exec := DefaultExecutor[int, string, struct{}, struct{}]{}
	called := false

	exec.Execute(func(from int, _ struct{}, to int, evt string, _ struct{}) {
		if from != 1 || to != 2 || evt != "go" {
			t.Fatalf("unexpected args: from=%d to=%d evt=%q", from, to, evt)
		}
		called = true
	}, 1, struct{}{}, 2, "go", struct{}{})

	if !called {
		t.Fatalf("callback not executed")
	}
}
