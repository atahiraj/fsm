package observer

import "testing"

type execRecorder struct{ calls int }

func (e *execRecorder) Execute(f func(from int, sp struct{}, to int, e2 string, ep struct{}), from int, sp struct{}, to int, e2 string, ep struct{}) {
	e.calls++
	f(from, sp, to, e2, ep)
}

func TestBuildObserver(t *testing.T) {
	exec := &execRecorder{}

	b := New[int, string, struct{}, struct{}]()
	b.WithExecutor(exec)
	var seen []string
	b.OnExitAny(func(from int, _ struct{}, to int, _ string, _ struct{}) { seen = append(seen, "exit:any") })
	b.OnEnterAny(func(from int, _ struct{}, to int, _ string, _ struct{}) { seen = append(seen, "enter:any") })
	b.OnExit(1, func(from int, _ struct{}, to int, _ string, _ struct{}) { seen = append(seen, "exit:1") })
	b.OnEnter(2, func(from int, _ struct{}, to int, _ string, _ struct{}) { seen = append(seen, "enter:2") })

	o, err := b.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	o.OnStep(1, struct{}{}, 2, "e", struct{}{})

	if exec.calls != 4 {
		t.Fatalf("executor calls = %d, want 4", exec.calls)
	}
	want := []string{"exit:any", "exit:1", "enter:any", "enter:2"}
	if len(seen) != len(want) {
		t.Fatalf("seen len = %d, want %d", len(seen), len(want))
	}
	for i := range want {
		if seen[i] != want[i] {
			t.Fatalf("seen[%d] = %q, want %q", i, seen[i], want[i])
		}
	}
}
