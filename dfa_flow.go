package fsm

import (
	"errors"

	"github.com/atahiraj/fsm/dfa"
	"github.com/atahiraj/fsm/engine"
	"github.com/atahiraj/fsm/key"
	"github.com/atahiraj/fsm/runner"
	"github.com/atahiraj/fsm/transitionhooks"
)

// DFAGraph is a minimal graph interface for DFA builders.
type DFAGraph[State any, Input any] interface {
	Delta(from State, label Input) []State
}

// Executor controls how callbacks are executed.
type Executor[S any, I any] interface {
	Execute(f func(from S, to S, e I), from S, to S, e I)
}

// DefaultExecutor calls the callback directly.
type DefaultExecutor[S any, I any] struct{}

func (DefaultExecutor[S, I]) Execute(
	f func(from S, to S, e I),
	from S,
	to S,
	e I,
) {
	f(from, to, e)
}

// DFA constructs a top-level DFA builder.
func DFA[
	State key.Keyer[StateKey],
	Input key.Keyer[InputKey],
	StateKey comparable,
	InputKey comparable,
]() *DFABuilder[State, Input, StateKey, InputKey] {
	return &DFABuilder[State, Input, StateKey, InputKey]{
		dfa:  dfa.NewBuilder[State, Input](),
		exec: DefaultExecutor[State, Input]{},
	}
}

// NewDFABuilder constructs a low-level DFA builder.
func NewDFABuilder[
	State key.Keyer[StateKey],
	Input key.Keyer[InputKey],
	StateKey comparable,
	InputKey comparable,
]() *DFABuilder[State, Input, StateKey, InputKey] {
	return &DFABuilder[State, Input, StateKey, InputKey]{
		dfa: dfa.NewBuilder[State, Input](),
	}
}

// DFABuilder wires DFA + TransitionHooks + Engine in one fluent flow.
type DFABuilder[
	State key.Keyer[StateKey],
	Input key.Keyer[InputKey],
	StateKey comparable,
	InputKey comparable,
] struct {
	dfa *dfa.Builder[State, Input, StateKey, InputKey]

	exec Executor[State, Input]

	selfTransitionCallbacks bool

	// Dispatch order is fixed by phase:
	// step-any -> step -> exit-any -> exit -> enter-any -> enter
	stepAnyCallbacks  []func(from State, to State, e Input)
	stepCallbacks     []dfaStepCallback[StateKey, Input]
	exitAnyCallbacks  []func(from State, e Input)
	exitCallbacks     []dfaStateCallback[StateKey, Input]
	enterAnyCallbacks []func(to State, e Input)
	enterCallbacks    []dfaStateCallback[StateKey, Input]

	dfaOverride   *dfa.DFA[State, Input, StateKey, InputKey]
	hooksOverride engine.TransitionHooks[State, Input]
	err           error
}

type dfaStepCallback[StateKey comparable, Input any] struct {
	from     StateKey
	to       StateKey
	callback func(e Input)
}

type dfaStateCallback[StateKey comparable, Input any] struct {
	target   StateKey
	callback func(e Input)
}

// WithDFA overrides the DFA built by this builder.
func (b *DFABuilder[State, Input, StateKey, InputKey]) WithDFA(d *dfa.DFA[State, Input, StateKey, InputKey]) *DFABuilder[State, Input, StateKey, InputKey] {
	b.dfaOverride = d
	return b
}

// WithGraph replaces the graph used by the DFA builder.
//
// The graph only provides the transition relation (δ). You must still provide
// the universe (Q, Σ), typically with WithStates and WithAlphabet.
func (b *DFABuilder[State, Input, StateKey, InputKey]) WithGraph(g DFAGraph[State, Input]) *DFABuilder[State, Input, StateKey, InputKey] {
	b.dfa.WithGraph(g)
	return b
}

// WithStart sets q₀.
func (b *DFABuilder[State, Input, StateKey, InputKey]) WithStart(state State) *DFABuilder[State, Input, StateKey, InputKey] {
	b.dfa.SetStart(state)
	return b
}

// WithStates adds states to Q.
func (b *DFABuilder[State, Input, StateKey, InputKey]) WithStates(states ...State) *DFABuilder[State, Input, StateKey, InputKey] {
	b.dfa.AddStates(states...)
	return b
}

// WithAlphabet adds inputs to Σ.
func (b *DFABuilder[State, Input, StateKey, InputKey]) WithAlphabet(inputs ...Input) *DFABuilder[State, Input, StateKey, InputKey] {
	b.dfa.AddAlphabet(inputs...)
	return b
}

// WithAccepting adds states to F.
func (b *DFABuilder[State, Input, StateKey, InputKey]) WithAccepting(states ...State) *DFABuilder[State, Input, StateKey, InputKey] {
	b.dfa.AddAccepting(states...)
	return b
}

// WithTransition inserts (from, a, to) into δ using an input key.
// Preconditions: transition does not introduce nondeterminism.
func (b *DFABuilder[State, Input, StateKey, InputKey]) WithTransition(from State, inputKey InputKey, to State) *DFABuilder[State, Input, StateKey, InputKey] {
	if b.err != nil {
		return b
	}
	if err := b.dfa.TransitionKey(from, inputKey, to); err != nil {
		b.err = err
	}
	return b
}

// WithTransitionHook inserts (from, a, to) into δ and registers transition callbacks for from -> to.
// Callbacks are executed in the same order as provided.
// Preconditions: transition does not introduce nondeterminism.
func (b *DFABuilder[State, Input, StateKey, InputKey]) WithTransitionHook(from State, inputKey InputKey, to State, hooks ...func(e Input)) *DFABuilder[State, Input, StateKey, InputKey] {
	if b.err != nil {
		return b
	}
	if err := b.dfa.TransitionKey(from, inputKey, to); err != nil {
		b.err = err
		return b
	}
	for _, hook := range hooks {
		if hook == nil {
			continue
		}
		b.stepCallbacks = append(b.stepCallbacks, dfaStepCallback[StateKey, Input]{
			from:     from.Key(),
			to:       to.Key(),
			callback: hook,
		})
	}
	return b
}

// WithTransitionHooks overrides the transition hooks used by the Engine build.
func (b *DFABuilder[State, Input, StateKey, InputKey]) WithTransitionHooks(hooks engine.TransitionHooks[State, Input]) *DFABuilder[State, Input, StateKey, InputKey] {
	b.hooksOverride = hooks
	return b
}

// WithExecutor sets the transition hooks executor.
func (b *DFABuilder[State, Input, StateKey, InputKey]) WithExecutor(exec Executor[State, Input]) *DFABuilder[State, Input, StateKey, InputKey] {
	b.exec = exec
	return b
}

// WithSelfTransitionCallbacks enables enter/exit callbacks for self-transitions (from == to).
func (b *DFABuilder[State, Input, StateKey, InputKey]) WithSelfTransitionCallbacks() *DFABuilder[State, Input, StateKey, InputKey] {
	b.selfTransitionCallbacks = true
	return b
}

// WithOnTransitionAny registers a callback for every transition.
func (b *DFABuilder[State, Input, StateKey, InputKey]) WithOnTransitionAny(f func(from State, to State, e Input)) *DFABuilder[State, Input, StateKey, InputKey] {
	b.stepAnyCallbacks = append(b.stepAnyCallbacks, f)
	return b
}

// WithOnTransition registers a callback for a specific transition from -> to.
func (b *DFABuilder[State, Input, StateKey, InputKey]) WithOnTransition(from State, to State, f func(e Input)) *DFABuilder[State, Input, StateKey, InputKey] {
	b.stepCallbacks = append(b.stepCallbacks, dfaStepCallback[StateKey, Input]{
		from:     from.Key(),
		to:       to.Key(),
		callback: f,
	})
	return b
}

// WithOnStepAny registers a callback for every transition.
// Deprecated: use WithOnTransitionAny.
func (b *DFABuilder[State, Input, StateKey, InputKey]) WithOnStepAny(f func(from State, to State, e Input)) *DFABuilder[State, Input, StateKey, InputKey] {
	return b.WithOnTransitionAny(f)
}

// WithOnStep registers a callback for a specific transition from -> to.
// Deprecated: use WithOnTransition.
func (b *DFABuilder[State, Input, StateKey, InputKey]) WithOnStep(from State, to State, f func(e Input)) *DFABuilder[State, Input, StateKey, InputKey] {
	return b.WithOnTransition(from, to, f)
}

// WithOnExit registers a callback when leaving state s.
func (b *DFABuilder[State, Input, StateKey, InputKey]) WithOnExit(s State, f func(e Input)) *DFABuilder[State, Input, StateKey, InputKey] {
	b.exitCallbacks = append(b.exitCallbacks, dfaStateCallback[StateKey, Input]{
		target:   s.Key(),
		callback: f,
	})
	return b
}

// WithOnEnter registers a callback when entering state s.
func (b *DFABuilder[State, Input, StateKey, InputKey]) WithOnEnter(s State, f func(e Input)) *DFABuilder[State, Input, StateKey, InputKey] {
	b.enterCallbacks = append(b.enterCallbacks, dfaStateCallback[StateKey, Input]{
		target:   s.Key(),
		callback: f,
	})
	return b
}

// WithOnExitAny registers a callback when leaving any state.
func (b *DFABuilder[State, Input, StateKey, InputKey]) WithOnExitAny(f func(from State, e Input)) *DFABuilder[State, Input, StateKey, InputKey] {
	b.exitAnyCallbacks = append(b.exitAnyCallbacks, f)
	return b
}

// WithOnEnterAny registers a callback when entering any state.
func (b *DFABuilder[State, Input, StateKey, InputKey]) WithOnEnterAny(f func(to State, e Input)) *DFABuilder[State, Input, StateKey, InputKey] {
	b.enterAnyCallbacks = append(b.enterAnyCallbacks, f)
	return b
}

// BuildDFA returns the DFA built from the configured pieces.
func (b *DFABuilder[State, Input, StateKey, InputKey]) BuildDFA() (*dfa.DFA[State, Input, StateKey, InputKey], error) {
	if b.err != nil {
		return nil, b.err
	}
	if b.dfaOverride != nil {
		return b.dfaOverride, nil
	}
	return b.dfa.Build()
}

// BuildAtomicDFA returns a thread-safe DFA.
func (b *DFABuilder[State, Input, StateKey, InputKey]) BuildAtomicDFA() (*dfa.AtomicDFA[State, Input, StateKey, InputKey], error) {
	d, err := b.BuildDFA()
	if err != nil {
		return nil, err
	}
	return dfa.NewAtomic(d), nil
}

func (b *DFABuilder[State, Input, StateKey, InputKey]) buildTransitionHooks() (engine.TransitionHooks[State, Input], error) {
	if b.exec == nil {
		return nil, errors.New("transition hooks builder: executor is nil")
	}

	stepRegistrations := make([]transitionhooks.Registration[State, Input, StateKey], 0, len(b.stepAnyCallbacks)+len(b.stepCallbacks))
	for _, callback := range b.stepAnyCallbacks {
		stepRegistrations = append(stepRegistrations, transitionhooks.Registration[State, Input, StateKey]{
			Mode:     transitionhooks.MatchAny,
			Callback: callback,
		})
	}
	for _, callback := range b.stepCallbacks {
		stepCallback := callback.callback
		stepRegistrations = append(stepRegistrations, transitionhooks.Registration[State, Input, StateKey]{
			Mode:    transitionhooks.MatchFromToKey,
			FromKey: callback.from,
			ToKey:   callback.to,
			Callback: func(from State, to State, e Input) {
				stepCallback(e)
			},
		})
	}

	exitRegistrations := make([]transitionhooks.Registration[State, Input, StateKey], 0, len(b.exitAnyCallbacks)+len(b.exitCallbacks))
	for _, callback := range b.exitAnyCallbacks {
		exitAnyCallback := callback
		exitRegistrations = append(exitRegistrations, transitionhooks.Registration[State, Input, StateKey]{
			Mode: transitionhooks.MatchAny,
			Callback: func(from State, to State, e Input) {
				exitAnyCallback(from, e)
			},
		})
	}
	for _, callback := range b.exitCallbacks {
		exitCallback := callback.callback
		exitRegistrations = append(exitRegistrations, transitionhooks.Registration[State, Input, StateKey]{
			Mode:    transitionhooks.MatchFromKey,
			FromKey: callback.target,
			Callback: func(from State, to State, e Input) {
				exitCallback(e)
			},
		})
	}

	enterRegistrations := make([]transitionhooks.Registration[State, Input, StateKey], 0, len(b.enterAnyCallbacks)+len(b.enterCallbacks))
	for _, callback := range b.enterAnyCallbacks {
		enterAnyCallback := callback
		enterRegistrations = append(enterRegistrations, transitionhooks.Registration[State, Input, StateKey]{
			Mode: transitionhooks.MatchAny,
			Callback: func(from State, to State, e Input) {
				enterAnyCallback(to, e)
			},
		})
	}
	for _, callback := range b.enterCallbacks {
		enterCallback := callback.callback
		enterRegistrations = append(enterRegistrations, transitionhooks.Registration[State, Input, StateKey]{
			Mode:  transitionhooks.MatchToKey,
			ToKey: callback.target,
			Callback: func(from State, to State, e Input) {
				enterCallback(e)
			},
		})
	}

	var enterExitGuard transitionhooks.GroupGuard[StateKey]
	if !b.selfTransitionCallbacks {
		enterExitGuard = func(fromKey StateKey, toKey StateKey) bool { return fromKey != toKey }
	}

	return transitionhooks.NewCompiled(
		b.exec,
		func(state State) StateKey { return state.Key() },
		transitionhooks.Group[State, Input, StateKey]{
			Registrations: stepRegistrations,
		},
		transitionhooks.Group[State, Input, StateKey]{
			Guard:         enterExitGuard,
			Registrations: exitRegistrations,
		},
		transitionhooks.Group[State, Input, StateKey]{
			Guard:         enterExitGuard,
			Registrations: enterRegistrations,
		},
	), nil
}

// BuildEngine wires the DFA and transition hooks into an Engine.
func (b *DFABuilder[State, Input, StateKey, InputKey]) BuildEngine() (*engine.Engine[State, Input], error) {
	hooks := b.hooksOverride
	if hooks == nil {
		var err error
		hooks, err = b.buildTransitionHooks()
		if err != nil {
			return nil, err
		}
	}
	d, err := b.BuildDFA()
	if err != nil {
		return nil, err
	}
	return engine.DFA(d).WithTransitionHooks(hooks).Build()
}

// BuildAtomicEngine wires the DFA and transition hooks into a thread-safe Engine.
func (b *DFABuilder[State, Input, StateKey, InputKey]) BuildAtomicEngine() (*engine.AtomicEngine[State, Input], error) {
	hooks := b.hooksOverride
	if hooks == nil {
		var err error
		hooks, err = b.buildTransitionHooks()
		if err != nil {
			return nil, err
		}
	}
	d, err := b.BuildDFA()
	if err != nil {
		return nil, err
	}
	return engine.DFA(d).WithTransitionHooks(hooks).BuildAtomic()
}

// BuildRunner wires the DFA and transition hooks into an Engine-backed Runner.
func (b *DFABuilder[State, Input, StateKey, InputKey]) BuildRunner(buffer int) (*runner.Runner[Input], error) {
	e, err := b.BuildEngine()
	if err != nil {
		return nil, err
	}
	return runner.New(e, buffer), nil
}
