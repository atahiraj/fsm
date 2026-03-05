# fsm

`fsm` is a small, synchronous finite-state-machine runtime for Go, with
deterministic and nondeterministic finite automata as composable definitions.

The module has three public packages:

- `github.com/atahiraj/fsm` executes any value satisfying `fsm.Machine` and
  dispatches lifecycle notifications.
- `github.com/atahiraj/fsm/dfa` builds deterministic finite automata.
- `github.com/atahiraj/fsm/nfa` builds nondeterministic finite automata with
  epsilon transitions.

## Example

```go
type State string

type Event struct {
	Kind    string
	Payload any
}

builder, err := dfa.NewBuilder(
	func(state State) string { return string(state) },
	func(event Event) string { return event.Kind },
)
if err != nil {
	return err
}

builder.SetStart(State("idle"))
builder.AddAccepting(State("done"))
if err := builder.AddTransition(
	State("idle"),
	Event{Kind: "finish"},
	State("done"),
); err != nil {
	return err
}

machine, err := builder.Build()
if err != nil {
	return err
}

hooks, err := fsm.NewHooks[State, Event](machine.Equal)
if err != nil {
	return err
}
if err := hooks.OnEnter(State("done"), func(t fsm.Transition[State, Event]) {
	fmt.Printf("entered %s with payload %v\n", t.To, t.Input.Payload)
}); err != nil {
	return err
}

engine, err := fsm.New(machine, fsm.WithLifecycle[State, Event](hooks))
if err != nil {
	return err
}
engine.Step(Event{Kind: "finish", Payload: 42})
```

## Injection and ownership

The runtime depends only on `fsm.Machine`. Lifecycle behavior is supplied via
`fsm.Lifecycle`, and `fsm.Hooks` is an optional default router.

The DFA and NFA packages accept domain-value graph interfaces. Graphs supplied
through `Config.Graph`, `SetGraph`, or `Builder.WithGraph` remain owned by the
caller and are not copied. Builder-owned graphs are snapshotted by `Build`.

The engine, automata, default graphs, and hook router are deliberately not
synchronized. Callers that share them between goroutines must provide their
own synchronization.

This module requires Go 1.27.1 or newer.
