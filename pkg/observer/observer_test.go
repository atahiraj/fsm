package observer

import "testing"

type call struct {
	kind string
	from int
	to   int
}

type recorder struct{ calls []call }

func (r *recorder) add(kind string, from, to int) {
	r.calls = append(r.calls, call{kind: kind, from: from, to: to})
}

type recordingExecutor struct{ r *recorder }

func (e *recordingExecutor) Execute(f func(from int, sp struct{}, to int, evt string, ep struct{}), from int, sp struct{}, to int, evt string, ep struct{}) {
	f(from, sp, to, evt, ep)
}

func TestObserverDispatchOrder(t *testing.T) {
	rec := &recorder{}
	exec := &recordingExecutor{r: rec}

	onExitAny := []func(from int, sp struct{}, to int, e string, ep struct{}){
		func(from int, _ struct{}, to int, _ string, _ struct{}) { rec.add("exit:any", from, to) },
	}
	onEnterAny := []func(from int, sp struct{}, to int, e string, ep struct{}){
		func(from int, _ struct{}, to int, _ string, _ struct{}) { rec.add("enter:any", from, to) },
	}

	onExit := map[int][]func(from int, sp struct{}, to int, e string, ep struct{}){
		1: {func(from int, _ struct{}, to int, _ string, _ struct{}) { rec.add("exit:1", from, to) }},
	}
	onEnter := map[int][]func(from int, sp struct{}, to int, e string, ep struct{}){
		2: {func(from int, _ struct{}, to int, _ string, _ struct{}) { rec.add("enter:2", from, to) }},
	}

	o := NewObserver[int, string, struct{}, struct{}](Config[int, string, struct{}, struct{}]{
		Executor:   exec,
		OnExitAny:  onExitAny,
		OnEnterAny: onEnterAny,
		OnExit:     onExit,
		OnEnter:    onEnter,
	})
	o.OnStep(1, struct{}{}, 2, "e", struct{}{})

	want := []call{
		{kind: "exit:any", from: 1, to: 2},
		{kind: "exit:1", from: 1, to: 2},
		{kind: "enter:any", from: 1, to: 2},
		{kind: "enter:2", from: 1, to: 2},
	}
	if len(rec.calls) != len(want) {
		t.Fatalf("got %d calls, want %d", len(rec.calls), len(want))
	}
	for i, w := range want {
		if rec.calls[i] != w {
			t.Fatalf("call %d = %+v, want %+v", i, rec.calls[i], w)
		}
	}
}
