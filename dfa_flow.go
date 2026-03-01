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
	stepCallbacks     []func(from State, to State, e Symbol)
	exitAnyCallbacks  []func(from State, to State, e Symbol)
	exitCallbacks     []func(from State, to State, e Symbol)
	enterAnyCallbacks []func(from State, to State, e Symbol)
	enterCallbacks    []func(from State, to State, e Symbol)

	dfaOverride   *dfa.DFA[State, Symbol, StateKey, SymbolKey]
	hooksOverride engine.TransitionHooks[State, Symbol]
	err           error
}

// WithDFA overrides the DFA built by this builder.
func (b *DFABuilder[State, Symbol, StateKey, SymbolKey]) WithDFA(d *dfa.DFA[State, Symbol, StateKey, SymbolKey]) *DFABuilder[State, Symbol, StateKey, SymbolKey] {
	b.dfaOverride = d
	return b
}

// WithGraph replaces the graph used by the DFA builder.
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

// WithOnStepAny registers a callback for every step.
func (b *DFABuilder[State, Symbol, StateKey, SymbolKey]) WithOnStepAny(f func(from State, to State, e Symbol)) *DFABuilder[State, Symbol, StateKey, SymbolKey] {
	b.stepAnyCallbacks = append(b.stepAnyCallbacks, f)
	return b
}

// WithOnStep registers a callback for a specific transition from -> to.
func (b *DFABuilder[State, Symbol, StateKey, SymbolKey]) WithOnStep(from State, to State, f func(e Symbol)) *DFABuilder[State, Symbol, StateKey, SymbolKey] {
	fromKey := from.Key()
	toKey := to.Key()
	decoratedFunc := func(from State, to State, e Symbol) {
		if from.Key() == fromKey && to.Key() == toKey {
			f(e)
		}
	}
	b.stepCallbacks = append(b.stepCallbacks, decoratedFunc)
	return b
}

// WithOnExit registers a callback when leaving state s.
func (b *DFABuilder[State, Symbol, StateKey, SymbolKey]) WithOnExit(s State, f func(e Symbol)) *DFABuilder[State, Symbol, StateKey, SymbolKey] {
	target := s.Key()
	decoratedFunc := func(from State, to State, e Symbol) {
		fromKey := from.Key()
		toKey := to.Key()
		if fromKey == target && (b.selfTransitionCallbacks || fromKey != toKey) {
			f(e)
		}
	}
	b.exitCallbacks = append(b.exitCallbacks, decoratedFunc)
	return b
}

// WithOnEnter registers a callback when entering state s.
func (b *DFABuilder[State, Symbol, StateKey, SymbolKey]) WithOnEnter(s State, f func(e Symbol)) *DFABuilder[State, Symbol, StateKey, SymbolKey] {
	target := s.Key()
	decoratedFunc := func(from State, to State, e Symbol) {
		fromKey := from.Key()
		toKey := to.Key()
		if toKey == target && (b.selfTransitionCallbacks || fromKey != toKey) {
			f(e)
		}
	}
	b.enterCallbacks = append(b.enterCallbacks, decoratedFunc)
	return b
}

// WithOnExitAny registers a callback when leaving any state.
func (b *DFABuilder[State, Symbol, StateKey, SymbolKey]) WithOnExitAny(f func(from State, e Symbol)) *DFABuilder[State, Symbol, StateKey, SymbolKey] {
	decoratedFunc := func(from State, to State, e Symbol) {
		if b.selfTransitionCallbacks || from.Key() != to.Key() {
			f(from, e)
		}
	}
	b.exitAnyCallbacks = append(b.exitAnyCallbacks, decoratedFunc)
	return b
}

// WithOnEnterAny registers a callback when entering any state.
func (b *DFABuilder[State, Symbol, StateKey, SymbolKey]) WithOnEnterAny(f func(to State, e Symbol)) *DFABuilder[State, Symbol, StateKey, SymbolKey] {
	decoratedFunc := func(from State, to State, e Symbol) {
		if b.selfTransitionCallbacks || from.Key() != to.Key() {
			f(to, e)
		}
	}
	b.enterAnyCallbacks = append(b.enterAnyCallbacks, decoratedFunc)
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

	total := len(b.stepAnyCallbacks) + len(b.stepCallbacks) + len(b.exitAnyCallbacks) + len(b.exitCallbacks) + len(b.enterAnyCallbacks) + len(b.enterCallbacks)
	callbacks := make([]func(from State, to State, e Symbol), 0, total)
	for _, callback := range b.stepAnyCallbacks {
		callbacks = append(callbacks, callback)
	}
	for _, callback := range b.stepCallbacks {
		callbacks = append(callbacks, callback)
	}
	for _, callback := range b.exitAnyCallbacks {
		callbacks = append(callbacks, callback)
	}
	for _, callback := range b.exitCallbacks {
		callbacks = append(callbacks, callback)
	}
	for _, callback := range b.enterAnyCallbacks {
		callbacks = append(callbacks, callback)
	}
	for _, callback := range b.enterCallbacks {
		callbacks = append(callbacks, callback)
	}

	return transitionhooks.New[State, Symbol](b.exec, callbacks...), nil
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
	return engine.DFA[State, Symbol](d).WithTransitionHooks(hooks).Build()
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
	return engine.DFA[State, Symbol](d).WithTransitionHooks(hooks).BuildAtomic()
}

// BuildRunner wires the DFA and transition hooks into an Engine-backed Runner.
func (b *DFABuilder[State, Symbol, StateKey, SymbolKey]) BuildRunner(buffer int) (*runner.Runner[Symbol], error) {
	e, err := b.BuildEngine()
	if err != nil {
		return nil, err
	}
	return runner.New(e, buffer), nil
}
