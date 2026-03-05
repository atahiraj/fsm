package fsm

import (
	"errors"

	"github.com/atahiraj/fsm/internal/validate"
)

// Machine is the contract required by Engine.
//
// Transition and Equal must be observationally pure: Peek and CanStep may call
// them without committing engine state.
type Machine[State, Input any] interface {
	Initial() State
	Transition(State, Input) (State, bool)
	IsAccepting(State) bool
	Equal(State, State) bool
}

// Transition describes one successful machine transition.
type Transition[State, Input any] struct {
	From    State
	To      State
	Input   Input
	Changed bool
}

// Phase identifies a transition lifecycle phase.
type Phase uint8

const (
	PhaseBefore Phase = iota
	PhaseExit
	PhaseTransition
	PhaseEnter
	PhaseAfter
)

// Lifecycle receives synchronous transition notifications.
type Lifecycle[State, Input any] interface {
	Handle(Phase, Transition[State, Input])
}

// LifecycleFunc adapts a function to Lifecycle.
type LifecycleFunc[State, Input any] func(Phase, Transition[State, Input])

// Handle calls f.
func (f LifecycleFunc[State, Input]) Handle(phase Phase, transition Transition[State, Input]) {
	f(phase, transition)
}

type engineConfig[State, Input any] struct {
	lifecycles []Lifecycle[State, Input]
}

// Option configures an Engine.
type Option[State, Input any] func(*engineConfig[State, Input]) error

// WithLifecycle appends lifecycle handlers in dispatch order.
func WithLifecycle[State, Input any](lifecycles ...Lifecycle[State, Input]) Option[State, Input] {
	return func(config *engineConfig[State, Input]) error {
		for _, lifecycle := range lifecycles {
			if validate.IsNil(lifecycle) {
				return errors.New("fsm: lifecycle is nil")
			}
			config.lifecycles = append(config.lifecycles, lifecycle)
		}
		return nil
	}
}

// Engine executes a Machine. It is not safe for concurrent use.
type Engine[State, Input any] struct {
	machine    Machine[State, Input]
	lifecycles []Lifecycle[State, Input]
	current    State
}

// New constructs an Engine at the machine's initial state.
func New[State, Input any](machine Machine[State, Input], options ...Option[State, Input]) (*Engine[State, Input], error) {
	if validate.IsNil(machine) {
		return nil, errors.New("fsm: machine is nil")
	}
	config := engineConfig[State, Input]{}
	for _, option := range options {
		if option == nil {
			return nil, errors.New("fsm: option is nil")
		}
		if err := option(&config); err != nil {
			return nil, err
		}
	}
	return &Engine[State, Input]{
		machine:    machine,
		lifecycles: append([]Lifecycle[State, Input](nil), config.lifecycles...),
		current:    machine.Initial(),
	}, nil
}

// Current returns the engine's current state.
func (e *Engine[State, Input]) Current() State { return e.current }

// Reset restores the machine's current initial state without emitting hooks.
func (e *Engine[State, Input]) Reset() { e.current = e.machine.Initial() }

// IsAccepting reports whether the current state is accepting.
func (e *Engine[State, Input]) IsAccepting() bool {
	return e.machine.IsAccepting(e.current)
}

// Peek computes a transition without committing it or notifying lifecycles.
func (e *Engine[State, Input]) Peek(input Input) (Transition[State, Input], bool) {
	next, ok := e.machine.Transition(e.current, input)
	if !ok {
		return Transition[State, Input]{}, false
	}
	return Transition[State, Input]{
		From:    e.current,
		To:      next,
		Input:   input,
		Changed: !e.machine.Equal(e.current, next),
	}, true
}

// CanStep reports whether input has a transition from the current state.
func (e *Engine[State, Input]) CanStep(input Input) bool {
	_, ok := e.Peek(input)
	return ok
}

func (e *Engine[State, Input]) notify(phase Phase, transition Transition[State, Input]) {
	for _, lifecycle := range e.lifecycles {
		lifecycle.Handle(phase, transition)
	}
}

// Step commits one transition and reports it.
//
// A changed transition dispatches Before, Exit, commits the state, then
// dispatches Transition, Enter, and After. A self-transition omits Exit and
// Enter. A missing transition changes nothing and emits no lifecycle phases.
func (e *Engine[State, Input]) Step(input Input) (Transition[State, Input], bool) {
	transition, ok := e.Peek(input)
	if !ok {
		return Transition[State, Input]{}, false
	}

	e.notify(PhaseBefore, transition)
	if transition.Changed {
		e.notify(PhaseExit, transition)
	}

	e.current = transition.To
	e.notify(PhaseTransition, transition)
	if transition.Changed {
		e.notify(PhaseEnter, transition)
	}
	e.notify(PhaseAfter, transition)
	return transition, true
}
