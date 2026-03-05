package fsm_test

import (
	"fmt"

	"github.com/atahiraj/fsm"
	"github.com/atahiraj/fsm/dfa"
)

func Example() {
	type state string
	type event struct {
		Kind string
	}

	builder, err := dfa.NewBuilder(
		func(value state) string { return string(value) },
		func(value event) string { return value.Kind },
	)
	if err != nil {
		panic(err)
	}
	builder.SetStart(state("idle"))
	builder.AddAccepting(state("done"))
	if err := builder.AddTransition(state("idle"), event{Kind: "finish"}, state("done")); err != nil {
		panic(err)
	}
	machine, err := builder.Build()
	if err != nil {
		panic(err)
	}

	hooks, err := fsm.NewHooks[state, event](machine.Equal)
	if err != nil {
		panic(err)
	}
	if err := hooks.OnEnter(state("done"), func(transition fsm.Transition[state, event]) {
		fmt.Printf("entered %s via %s\n", transition.To, transition.Input.Kind)
	}); err != nil {
		panic(err)
	}

	engine, err := fsm.New(machine, fsm.WithLifecycle[state, event](hooks))
	if err != nil {
		panic(err)
	}
	engine.Step(event{Kind: "finish"})
	fmt.Println(engine.IsAccepting())

	// Output:
	// entered done via finish
	// true
}
