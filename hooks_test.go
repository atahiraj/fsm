package fsm_test

import (
	"reflect"
	"testing"

	"github.com/atahiraj/fsm"
)

func TestHooksRoutingAndRegistrationOrder(t *testing.T) {
	hooks, err := fsm.NewHooks[int, string](func(left, right int) bool { return left == right })
	if err != nil {
		t.Fatalf("NewHooks() error = %v", err)
	}
	var got []string
	add := func(name string) fsm.Hook[int, string] {
		return func(_ fsm.Transition[int, string]) { got = append(got, name) }
	}
	registrations := []error{
		hooks.OnBefore(add("before")),
		hooks.OnExitAny(add("exit:any")),
		hooks.OnExit(1, add("exit:1")),
		hooks.OnTransitionAny(add("transition:any:first")),
		hooks.OnTransition(1, 2, add("transition:1->2")),
		hooks.OnIf(fsm.PhaseTransition, func(transition fsm.Transition[int, string]) bool {
			return transition.Input == "go"
		}, add("transition:predicate")),
		hooks.OnTransitionAny(add("transition:any:last")),
		hooks.OnEnter(2, add("enter:2")),
		hooks.OnEnterAny(add("enter:any")),
		hooks.OnAfter(add("after")),
	}
	for _, err := range registrations {
		if err != nil {
			t.Fatalf("registration error = %v", err)
		}
	}

	machine := &testMachine{initial: 1, next: map[int]map[string]int{1: {"go": 2}}}
	engine, err := fsm.New(machine, fsm.WithLifecycle[int, string](hooks))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	engine.Step("go")

	want := []string{
		"before",
		"exit:any",
		"exit:1",
		"transition:any:first",
		"transition:1->2",
		"transition:predicate",
		"transition:any:last",
		"enter:2",
		"enter:any",
		"after",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("callbacks = %v, want %v", got, want)
	}
}

func TestHooksValidation(t *testing.T) {
	if _, err := fsm.NewHooks[int, string](nil); err == nil {
		t.Fatal("NewHooks(nil) error = nil")
	}
	hooks, err := fsm.NewHooks[int, string](func(left, right int) bool { return left == right })
	if err != nil {
		t.Fatalf("NewHooks() error = %v", err)
	}
	if err := hooks.OnBefore(nil); err == nil {
		t.Fatal("OnBefore(nil) error = nil")
	}
	if err := hooks.OnIf(fsm.PhaseBefore, nil, func(fsm.Transition[int, string]) {}); err == nil {
		t.Fatal("OnIf(nil predicate) error = nil")
	}
	if err := hooks.OnIf(fsm.Phase(255), func(fsm.Transition[int, string]) bool { return true }, func(fsm.Transition[int, string]) {}); err == nil {
		t.Fatal("OnIf(invalid phase) error = nil")
	}
}
