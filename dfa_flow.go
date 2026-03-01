package fsm

import (
	"errors"

	"github.com/stnhrsprkwns/fsm/dfa"
	"github.com/stnhrsprkwns/fsm/engine"
	"github.com/stnhrsprkwns/fsm/key"
	"github.com/stnhrsprkwns/fsm/runner"
	"github.com/stnhrsprkwns/fsm/transitionhooks"
)

// DFAGraph is a minimal graph interface for DFA builders.
type DFAGraph[State any, Symbol any] interface {
	Delta(from State, label Symbol) []State
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
	Symbol key.Keyer[SymbolKey],
	StateKey comparable,
	SymbolKey comparable,
]() *DFABuilder[State, Symbol, StateKey, SymbolKey] {
	return &DFABuilder[State, Symbol, StateKey, SymbolKey]{
		dfa:  dfa.NewBuilder[State, Symbol](),
		exec: DefaultExecutor[State, Symbol]{},
	}
}

// NewDFABuilder constructs a low-level DFA builder.
func NewDFABuilder[
	State key.Keyer[StateKey],
	Symbol key.Keyer[SymbolKey],
	StateKey comparable,
	SymbolKey comparable,
]() *DFABuilder[State, Symbol, StateKey, SymbolKey] {
	return &DFABuilder[State, Symbol, StateKey, SymbolKey]{
		dfa: dfa.NewBuilder[State, Symbol](),
	}
}

// DFABuilder wires DFA + TransitionHooks + Engine in one fluent flow.
type DFABuilder[
	State key.Keyer[StateKey],
	Symbol key.Keyer[SymbolKey],
	StateKey comparable,
	SymbolKey comparable,
] struct {
	dfa *dfa.Builder[State, Symbol, StateKey, SymbolKey]

	exec Executor[State, Symbol]

	selfTransitionCallbacks bool

	// Dispatch order is fixed by phase:
	// step-any -> step -> exit-any -> exit -> enter-any -> enter
	stepAnyCallbacks  []func(from State, to State, e Symbol)
	stepCallbacks     []dfaStepCallback[StateKey, Symbol]
	exitAnyCallbacks  []func(from State, e Symbol)
	exitCallbacks     []dfaStateCallback[StateKey, Symbol]
	enterAnyCallbacks []func(to State, e Symbol)
	enterCallbacks    []dfaStateCallback[StateKey, Symbol]

	dfaOverride   *dfa.DFA[State, Symbol, StateKey, SymbolKey]
	hooksOverride engine.TransitionHooks[State, Symbol]
	err           error
}

type dfaStepCallback[StateKey comparable, Symbol any] struct {
	from     StateKey
	to       StateKey
	callback func(e Symbol)
}

type dfaStateCallback[StateKey comparable, Symbol any] struct {
	target   StateKey
	callback func(e Symbol)
}

// WithDFA overrides the DFA built by this builder.
func (b *DFABuilder[State, Symbol, StateKey, SymbolKey]) WithDFA(d *dfa.DFA[State, Symbol, StateKey, SymbolKey]) *DFABuilder[State, Symbol, StateKey, SymbolKey] {
	b.dfaOverride = d
	return b
}

// WithGraph replaces the graph used by the DFA builder.
//
// The graph only provides the transition relation (δ). You must still provide
// the universe (Q, Σ), typically with WithStates and WithAlphabet.
func (b *DFABuilder[State, Symbol, StateKey, SymbolKey]) WithGraph(g DFAGraph[State, Symbol]) *DFABuilder[State, Symbol, StateKey, SymbolKey] {
	b.dfa.WithGraph(g)
	return b
}

// WithStart sets q₀.
func (b *DFABuilder[State, Symbol, StateKey, SymbolKey]) WithStart(state State) *DFABuilder[State, Symbol, StateKey, SymbolKey] {
	b.dfa.SetStart(state)
	return b
}

// WithStates adds states to Q.
func (b *DFABuilder[State, Symbol, StateKey, SymbolKey]) WithStates(states ...State) *DFABuilder[State, Symbol, StateKey, SymbolKey] {
	b.dfa.AddStates(states...)
	return b
}

// WithAlphabet adds symbols to Σ.
func (b *DFABuilder[State, Symbol, StateKey, SymbolKey]) WithAlphabet(symbols ...Symbol) *DFABuilder[State, Symbol, StateKey, SymbolKey] {
	b.dfa.AddAlphabet(symbols...)
	return b
}

// WithAccepting adds states to F.
func (b *DFABuilder[State, Symbol, StateKey, SymbolKey]) WithAccepting(states ...State) *DFABuilder[State, Symbol, StateKey, SymbolKey] {
	b.dfa.AddAccepting(states...)
	return b
}

// WithTransition inserts (from, a, to) into δ.
// Preconditions: transition does not introduce nondeterminism.
func (b *DFABuilder[State, Symbol, StateKey, SymbolKey]) WithTransition(from State, symbol Symbol, to State) *DFABuilder[State, Symbol, StateKey, SymbolKey] {
	if b.err != nil {
		return b
	}
	if err := b.dfa.Transition(from, symbol, to); err != nil {
		b.err = err
	}
	return b
}

// WithTransitionHook inserts (from, a, to) into δ and registers transition callbacks for from -> to.
// Callbacks are executed in the same order as provided.
// Preconditions: transition does not introduce nondeterminism.
func (b *DFABuilder[State, Symbol, StateKey, SymbolKey]) WithTransitionHook(from State, symbol Symbol, to State, hooks ...func(e Symbol)) *DFABuilder[State, Symbol, StateKey, SymbolKey] {
	if b.err != nil {
		return b
	}
	if err := b.dfa.Transition(from, symbol, to); err != nil {
		b.err = err
		return b
	}
	for _, hook := range hooks {
		if hook == nil {
			continue
		}
		b.stepCallbacks = append(b.stepCallbacks, dfaStepCallback[StateKey, Symbol]{
			from:     from.Key(),
			to:       to.Key(),
			callback: hook,
		})
	}
	return b
}

// WithTransitionHooks overrides the transition hooks used by the Engine build.
func (b *DFABuilder[State, Symbol, StateKey, SymbolKey]) WithTransitionHooks(hooks engine.TransitionHooks[State, Symbol]) *DFABuilder[State, Symbol, StateKey, SymbolKey] {
	b.hooksOverride = hooks
	return b
}

// WithExecutor sets the transition hooks executor.
func (b *DFABuilder[State, Symbol, StateKey, SymbolKey]) WithExecutor(exec Executor[State, Symbol]) *DFABuilder[State, Symbol, StateKey, SymbolKey] {
	b.exec = exec
	return b
}

// WithSelfTransitionCallbacks enables enter/exit callbacks for self-transitions (from == to).
func (b *DFABuilder[State, Symbol, StateKey, SymbolKey]) WithSelfTransitionCallbacks() *DFABuilder[State, Symbol, StateKey, SymbolKey] {
	b.selfTransitionCallbacks = true
	return b
}

// WithOnTransitionAny registers a callback for every transition.
func (b *DFABuilder[State, Symbol, StateKey, SymbolKey]) WithOnTransitionAny(f func(from State, to State, e Symbol)) *DFABuilder[State, Symbol, StateKey, SymbolKey] {
	b.stepAnyCallbacks = append(b.stepAnyCallbacks, f)
	return b
}

// WithOnTransition registers a callback for a specific transition from -> to.
func (b *DFABuilder[State, Symbol, StateKey, SymbolKey]) WithOnTransition(from State, to State, f func(e Symbol)) *DFABuilder[State, Symbol, StateKey, SymbolKey] {
	b.stepCallbacks = append(b.stepCallbacks, dfaStepCallback[StateKey, Symbol]{
		from:     from.Key(),
		to:       to.Key(),
		callback: f,
	})
	return b
}

// WithOnStepAny registers a callback for every transition.
// Deprecated: use WithOnTransitionAny.
func (b *DFABuilder[State, Symbol, StateKey, SymbolKey]) WithOnStepAny(f func(from State, to State, e Symbol)) *DFABuilder[State, Symbol, StateKey, SymbolKey] {
	return b.WithOnTransitionAny(f)
}

// WithOnStep registers a callback for a specific transition from -> to.
// Deprecated: use WithOnTransition.
func (b *DFABuilder[State, Symbol, StateKey, SymbolKey]) WithOnStep(from State, to State, f func(e Symbol)) *DFABuilder[State, Symbol, StateKey, SymbolKey] {
	return b.WithOnTransition(from, to, f)
}

// WithOnExit registers a callback when leaving state s.
func (b *DFABuilder[State, Symbol, StateKey, SymbolKey]) WithOnExit(s State, f func(e Symbol)) *DFABuilder[State, Symbol, StateKey, SymbolKey] {
	b.exitCallbacks = append(b.exitCallbacks, dfaStateCallback[StateKey, Symbol]{
		target:   s.Key(),
		callback: f,
	})
	return b
}

// WithOnEnter registers a callback when entering state s.
func (b *DFABuilder[State, Symbol, StateKey, SymbolKey]) WithOnEnter(s State, f func(e Symbol)) *DFABuilder[State, Symbol, StateKey, SymbolKey] {
	b.enterCallbacks = append(b.enterCallbacks, dfaStateCallback[StateKey, Symbol]{
		target:   s.Key(),
		callback: f,
	})
	return b
}

// WithOnExitAny registers a callback when leaving any state.
func (b *DFABuilder[State, Symbol, StateKey, SymbolKey]) WithOnExitAny(f func(from State, e Symbol)) *DFABuilder[State, Symbol, StateKey, SymbolKey] {
	b.exitAnyCallbacks = append(b.exitAnyCallbacks, f)
	return b
}

// WithOnEnterAny registers a callback when entering any state.
func (b *DFABuilder[State, Symbol, StateKey, SymbolKey]) WithOnEnterAny(f func(to State, e Symbol)) *DFABuilder[State, Symbol, StateKey, SymbolKey] {
	b.enterAnyCallbacks = append(b.enterAnyCallbacks, f)
	return b
}

// BuildDFA returns the DFA built from the configured pieces.
func (b *DFABuilder[State, Symbol, StateKey, SymbolKey]) BuildDFA() (*dfa.DFA[State, Symbol, StateKey, SymbolKey], error) {
	if b.err != nil {
		return nil, b.err
	}
	if b.dfaOverride != nil {
		return b.dfaOverride, nil
	}
	return b.dfa.Build()
}

// BuildAtomicDFA returns a thread-safe DFA.
func (b *DFABuilder[State, Symbol, StateKey, SymbolKey]) BuildAtomicDFA() (*dfa.AtomicDFA[State, Symbol, StateKey, SymbolKey], error) {
	d, err := b.BuildDFA()
	if err != nil {
		return nil, err
	}
	return dfa.NewAtomic(d), nil
}

func (b *DFABuilder[State, Symbol, StateKey, SymbolKey]) buildTransitionHooks() (engine.TransitionHooks[State, Symbol], error) {
	if b.exec == nil {
		return nil, errors.New("transition hooks builder: executor is nil")
	}

	stepRegistrations := make([]transitionhooks.Registration[State, Symbol, StateKey], 0, len(b.stepAnyCallbacks)+len(b.stepCallbacks))
	for _, callback := range b.stepAnyCallbacks {
		stepRegistrations = append(stepRegistrations, transitionhooks.Registration[State, Symbol, StateKey]{
			Mode:     transitionhooks.MatchAny,
			Callback: callback,
		})
	}
	for _, callback := range b.stepCallbacks {
		stepCallback := callback.callback
		stepRegistrations = append(stepRegistrations, transitionhooks.Registration[State, Symbol, StateKey]{
			Mode:    transitionhooks.MatchFromToKey,
			FromKey: callback.from,
			ToKey:   callback.to,
			Callback: func(from State, to State, e Symbol) {
				stepCallback(e)
			},
		})
	}

	exitRegistrations := make([]transitionhooks.Registration[State, Symbol, StateKey], 0, len(b.exitAnyCallbacks)+len(b.exitCallbacks))
	for _, callback := range b.exitAnyCallbacks {
		exitAnyCallback := callback
		exitRegistrations = append(exitRegistrations, transitionhooks.Registration[State, Symbol, StateKey]{
			Mode: transitionhooks.MatchAny,
			Callback: func(from State, to State, e Symbol) {
				exitAnyCallback(from, e)
			},
		})
	}
	for _, callback := range b.exitCallbacks {
		exitCallback := callback.callback
		exitRegistrations = append(exitRegistrations, transitionhooks.Registration[State, Symbol, StateKey]{
			Mode:    transitionhooks.MatchFromKey,
			FromKey: callback.target,
			Callback: func(from State, to State, e Symbol) {
				exitCallback(e)
			},
		})
	}

	enterRegistrations := make([]transitionhooks.Registration[State, Symbol, StateKey], 0, len(b.enterAnyCallbacks)+len(b.enterCallbacks))
	for _, callback := range b.enterAnyCallbacks {
		enterAnyCallback := callback
		enterRegistrations = append(enterRegistrations, transitionhooks.Registration[State, Symbol, StateKey]{
			Mode: transitionhooks.MatchAny,
			Callback: func(from State, to State, e Symbol) {
				enterAnyCallback(to, e)
			},
		})
	}
	for _, callback := range b.enterCallbacks {
		enterCallback := callback.callback
		enterRegistrations = append(enterRegistrations, transitionhooks.Registration[State, Symbol, StateKey]{
			Mode:  transitionhooks.MatchToKey,
			ToKey: callback.target,
			Callback: func(from State, to State, e Symbol) {
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
		transitionhooks.Group[State, Symbol, StateKey]{
			Registrations: stepRegistrations,
		},
		transitionhooks.Group[State, Symbol, StateKey]{
			Guard:         enterExitGuard,
			Registrations: exitRegistrations,
		},
		transitionhooks.Group[State, Symbol, StateKey]{
			Guard:         enterExitGuard,
			Registrations: enterRegistrations,
		},
	), nil
}

// BuildEngine wires the DFA and transition hooks into an Engine.
func (b *DFABuilder[State, Symbol, StateKey, SymbolKey]) BuildEngine() (*engine.Engine[State, Symbol], error) {
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
func (b *DFABuilder[State, Symbol, StateKey, SymbolKey]) BuildAtomicEngine() (*engine.AtomicEngine[State, Symbol], error) {
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
func (b *DFABuilder[State, Symbol, StateKey, SymbolKey]) BuildRunner(buffer int) (*runner.Runner[Symbol], error) {
	e, err := b.BuildEngine()
	if err != nil {
		return nil, err
	}
	return runner.New(e, buffer), nil
}
