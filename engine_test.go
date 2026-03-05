package fsm_test

import (
	"reflect"
	"testing"

	"github.com/atahiraj/fsm"
)

type testMachine struct {
	initial int
	next    map[int]map[string]int
}

func (m *testMachine) Initial() int { return m.initial }

func (m *testMachine) Transition(state int, input string) (int, bool) {
	next, ok := m.next[state][input]
	return next, ok
}

func (m *testMachine) IsAccepting(state int) bool { return state == 2 }
func (m *testMachine) Equal(left, right int) bool { return left == right }

var _ fsm.Machine[int, string] = (*testMachine)(nil)

func TestEngineLifecycleOrderAndCommitBoundary(t *testing.T) {
	machine := &testMachine{
		initial: 1,
		next:    map[int]map[string]int{1: {"go": 2}},
	}
	var engine *fsm.Engine[int, string]
	var phases []fsm.Phase
	var current []int
	lifecycle := fsm.LifecycleFunc[int, string](func(phase fsm.Phase, transition fsm.Transition[int, string]) {
		phases = append(phases, phase)
		current = append(current, engine.Current())
		if transition.From != 1 || transition.To != 2 || transition.Input != "go" || !transition.Changed {
			t.Fatalf("unexpected transition: %+v", transition)
		}
	})

	var err error
	engine, err = fsm.New(machine, fsm.WithLifecycle[int, string](lifecycle))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	transition, ok := engine.Step("go")
	if !ok || transition.To != 2 {
		t.Fatalf("Step() = (%+v, %v)", transition, ok)
	}

	wantPhases := []fsm.Phase{
		fsm.PhaseBefore,
		fsm.PhaseExit,
		fsm.PhaseTransition,
		fsm.PhaseEnter,
		fsm.PhaseAfter,
	}
	if !reflect.DeepEqual(phases, wantPhases) {
		t.Fatalf("phases = %v, want %v", phases, wantPhases)
	}
	if want := []int{1, 1, 2, 2, 2}; !reflect.DeepEqual(current, want) {
		t.Fatalf("current during phases = %v, want %v", current, want)
	}
	if !engine.IsAccepting() {
		t.Fatal("engine should be accepting")
	}
}

func TestEngineSelfAndMissingTransitions(t *testing.T) {
	machine := &testMachine{
		initial: 1,
		next:    map[int]map[string]int{1: {"stay": 1}},
	}
	var phases []fsm.Phase
	engine, err := fsm.New(machine, fsm.WithLifecycle[int, string](
		fsm.LifecycleFunc[int, string](func(phase fsm.Phase, _ fsm.Transition[int, string]) {
			phases = append(phases, phase)
		}),
	))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	transition, ok := engine.Step("stay")
	if !ok || transition.Changed {
		t.Fatalf("self Step() = (%+v, %v)", transition, ok)
	}
	want := []fsm.Phase{fsm.PhaseBefore, fsm.PhaseTransition, fsm.PhaseAfter}
	if !reflect.DeepEqual(phases, want) {
		t.Fatalf("self phases = %v, want %v", phases, want)
	}

	phases = nil
	if transition, ok := engine.Step("missing"); ok || !reflect.DeepEqual(transition, fsm.Transition[int, string]{}) {
		t.Fatalf("missing Step() = (%+v, %v)", transition, ok)
	}
	if len(phases) != 0 || engine.Current() != 1 {
		t.Fatalf("missing transition changed engine: phases=%v current=%d", phases, engine.Current())
	}
}

func TestEnginePeekCanStepAndReset(t *testing.T) {
	machine := &testMachine{
		initial: 1,
		next:    map[int]map[string]int{1: {"go": 2}},
	}
	engine, err := fsm.New[int, string](machine)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	peeked, ok := engine.Peek("go")
	if !ok || peeked.From != 1 || peeked.To != 2 || engine.Current() != 1 {
		t.Fatalf("Peek() = (%+v, %v), current=%d", peeked, ok, engine.Current())
	}
	if !engine.CanStep("go") || engine.CanStep("missing") {
		t.Fatal("unexpected CanStep result")
	}
	engine.Step("go")
	machine.initial = 2
	engine.Reset()
	if engine.Current() != 2 {
		t.Fatalf("Current() = %d, want 2", engine.Current())
	}
}

func TestEnginePanicCommitSemantics(t *testing.T) {
	machine := &testMachine{
		initial: 1,
		next:    map[int]map[string]int{1: {"go": 2}},
	}

	for _, test := range []struct {
		name        string
		panicPhase  fsm.Phase
		wantCurrent int
	}{
		{name: "before commit", panicPhase: fsm.PhaseExit, wantCurrent: 1},
		{name: "after commit", panicPhase: fsm.PhaseEnter, wantCurrent: 2},
	} {
		t.Run(test.name, func(t *testing.T) {
			engine, err := fsm.New(machine, fsm.WithLifecycle[int, string](
				fsm.LifecycleFunc[int, string](func(phase fsm.Phase, _ fsm.Transition[int, string]) {
					if phase == test.panicPhase {
						panic("hook panic")
					}
				}),
			))
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}
			func() {
				defer func() {
					if recover() == nil {
						t.Fatal("Step() did not propagate panic")
					}
				}()
				engine.Step("go")
			}()
			if engine.Current() != test.wantCurrent {
				t.Fatalf("Current() = %d, want %d", engine.Current(), test.wantCurrent)
			}
		})
	}
}

func TestNewValidation(t *testing.T) {
	if _, err := fsm.New[int, string](nil); err == nil {
		t.Fatal("New(nil) error = nil")
	}
	var typedNilMachine *testMachine
	if _, err := fsm.New[int, string](typedNilMachine); err == nil {
		t.Fatal("New(typed nil) error = nil")
	}
	machine := &testMachine{}
	if _, err := fsm.New(machine, fsm.WithLifecycle[int, string](nil)); err == nil {
		t.Fatal("nil lifecycle error = nil")
	}
	var typedNilLifecycle *fsm.Hooks[int, string]
	if _, err := fsm.New(machine, fsm.WithLifecycle[int, string](typedNilLifecycle)); err == nil {
		t.Fatal("typed nil lifecycle error = nil")
	}
	if _, err := fsm.New(machine, fsm.Option[int, string](nil)); err == nil {
		t.Fatal("nil option error = nil")
	}
}
