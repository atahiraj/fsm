package engine

import "testing"

type testDFALike struct {
	start       int
	accepting   map[int]bool
	transitions map[[2]int]int
}

func (d testDFALike) Start() int { return d.start }

func (d testDFALike) Delta(state int, input byte) (int, bool) {
	next, ok := d.transitions[[2]int{state, int(input)}]
	return next, ok
}

func (d testDFALike) IsAccepting(state int) bool { return d.accepting[state] }

type recordedStep struct {
	from int
	to   int
	sym  byte
}

type recordingTransitionHooks struct {
	steps []recordedStep
}

func (o *recordingTransitionHooks) OnTransition(from int, to int, e byte) {
	o.steps = append(o.steps, recordedStep{from: from, to: to, sym: e})
}

func TestEngineTryStepCanStepAndPeekStepWithDFA(t *testing.T) {
	d := testDFALike{
		start: 0,
		accepting: map[int]bool{
			1: true,
		},
		transitions: map[[2]int]int{
			{0, int('a')}: 1,
			{1, int('a')}: 1,
		},
	}
	hooks := &recordingTransitionHooks{}

	e, err := DFA[int, byte](d).WithTransitionHooks(hooks).Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	e.Reset()

	if got := e.CanStep('b'); got {
		t.Fatalf("CanStep('b') = %v, want false", got)
	}
	if next, ok := e.PeekStep('b'); ok || next != 0 {
		t.Fatalf("PeekStep('b') = (%d, %v), want (0, false)", next, ok)
	}
	if got := e.Cur(); got != 0 {
		t.Fatalf("Cur() after missing probes = %d, want 0", got)
	}
	if len(hooks.steps) != 0 {
		t.Fatalf("transition hooks steps after missing probes = %d, want 0", len(hooks.steps))
	}

	if handled := e.TryStep('b'); handled {
		t.Fatalf("TryStep('b') handled = %v, want false", handled)
	}
	if got := e.Cur(); got != 0 {
		t.Fatalf("Cur() after missing transition = %d, want 0", got)
	}
	if len(hooks.steps) != 0 {
		t.Fatalf("transition hooks steps after missing transition = %d, want 0", len(hooks.steps))
	}

	if got := e.CanStep('a'); !got {
		t.Fatalf("CanStep('a') = %v, want true", got)
	}
	if next, ok := e.PeekStep('a'); !ok || next != 1 {
		t.Fatalf("PeekStep('a') = (%d, %v), want (1, true)", next, ok)
	}
	if got := e.Cur(); got != 0 {
		t.Fatalf("Cur() after PeekStep('a') = %d, want 0", got)
	}
	if len(hooks.steps) != 0 {
		t.Fatalf("transition hooks steps after PeekStep('a') = %d, want 0", len(hooks.steps))
	}

	if handled := e.TryStep('a'); !handled {
		t.Fatalf("TryStep('a') handled = %v, want true", handled)
	}
	if got := e.Cur(); got != 1 {
		t.Fatalf("Cur() after existing transition = %d, want 1", got)
	}
	if len(hooks.steps) != 1 {
		t.Fatalf("transition hooks steps after existing transition = %d, want 1", len(hooks.steps))
	}
	if got := hooks.steps[0]; got.from != 0 || got.to != 1 || got.sym != 'a' {
		t.Fatalf("transition hooks step = %+v, want from=0 to=1 sym='a'", got)
	}
}

func TestEngineStepSkipsMissingTransition(t *testing.T) {
	d := testDFALike{
		start:       0,
		accepting:   map[int]bool{},
		transitions: map[[2]int]int{{0, int('a')}: 1},
	}
	hooks := &recordingTransitionHooks{}

	e, err := DFA[int, byte](d).WithTransitionHooks(hooks).Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	e.Reset()

	e.Step('b')

	if got := e.Cur(); got != 0 {
		t.Fatalf("Cur() after Step('b') = %d, want 0", got)
	}
	if len(hooks.steps) != 0 {
		t.Fatalf("transition hooks steps after Step('b') = %d, want 0", len(hooks.steps))
	}
}

type noopTransitionHooks2 struct{}

func (noopTransitionHooks2) OnTransition(int, int, byte) {}

func TestAtomicEngineTryStepCanStepAndPeekStep(t *testing.T) {
	d := testDFALike{
		start:       0,
		accepting:   map[int]bool{},
		transitions: map[[2]int]int{{0, int('a')}: 1},
	}
	e, err := DFA[int, byte](d).WithTransitionHooks(noopTransitionHooks2{}).Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	a := NewAtomic(e)
	a.Reset()

	if got := a.CanStep('b'); got {
		t.Fatalf("CanStep('b') = %v, want false", got)
	}
	if next, ok := a.PeekStep('b'); ok || next != 0 {
		t.Fatalf("PeekStep('b') = (%d, %v), want (0, false)", next, ok)
	}
	if got := a.Cur(); got != 0 {
		t.Fatalf("Cur() after missing probes = %d, want 0", got)
	}
	if handled := a.TryStep('b'); handled {
		t.Fatalf("TryStep('b') handled = %v, want false", handled)
	}
	if got := a.Cur(); got != 0 {
		t.Fatalf("Cur() after missing transition = %d, want 0", got)
	}

	if got := a.CanStep('a'); !got {
		t.Fatalf("CanStep('a') = %v, want true", got)
	}
	if next, ok := a.PeekStep('a'); !ok || next != 1 {
		t.Fatalf("PeekStep('a') = (%d, %v), want (1, true)", next, ok)
	}
	if got := a.Cur(); got != 0 {
		t.Fatalf("Cur() after PeekStep('a') = %d, want 0", got)
	}
	if handled := a.TryStep('a'); !handled {
		t.Fatalf("TryStep('a') handled = %v, want true", handled)
	}
	if got := a.Cur(); got != 1 {
		t.Fatalf("Cur() after transition = %d, want 1", got)
	}
}
