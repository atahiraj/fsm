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

	onExitAny := []func(from int, sp struct{}, e string, ep struct{}){
		func(from int, _ struct{}, _ string, _ struct{}) { rec.add("exit:any", from, 0) },
	}
	onEnterAny := []func(to int, sp struct{}, e string, ep struct{}){
		func(to int, _ struct{}, _ string, _ struct{}) { rec.add("enter:any", 0, to) },
	}
	onStepAny := []func(from int, sp struct{}, to int, e string, ep struct{}){
		func(from int, _ struct{}, to int, _ string, _ struct{}) { rec.add("step:any", from, to) },
	}
	onStep := []transitionCallback[int, string, struct{}, struct{}]{
		{from: 1, to: 2, f: func(from int, _ struct{}, to int, _ string, _ struct{}) { rec.add("step:1->2", from, to) }},
	}

	onExit := []stateCallback[int, string, struct{}, struct{}]{
		{state: 1, f: func(from int, _ struct{}, _ string, _ struct{}) { rec.add("exit:1", from, 0) }},
	}
	onEnter := []stateCallback[int, string, struct{}, struct{}]{
		{state: 2, f: func(to int, _ struct{}, _ string, _ struct{}) { rec.add("enter:2", 0, to) }},
	}

	o := NewObserver[int, string, struct{}, struct{}, int](Config[int, string, struct{}, struct{}, int]{
		Executor:   exec,
		EqualState: func(a, b int) bool { return a == b },
		OnStepAny:  onStepAny,
		OnStep:     onStep,
		OnExitAny:  onExitAny,
		OnEnterAny: onEnterAny,
		OnExit:     onExit,
		OnEnter:    onEnter,
	})
	o.OnStep(1, struct{}{}, 2, "e", struct{}{})

	want := []call{
		{kind: "step:any", from: 1, to: 2},
		{kind: "step:1->2", from: 1, to: 2},
		{kind: "exit:any", from: 1, to: 0},
		{kind: "exit:1", from: 1, to: 0},
		{kind: "enter:any", from: 0, to: 2},
		{kind: "enter:2", from: 0, to: 2},
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
