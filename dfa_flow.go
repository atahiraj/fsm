package fsm

import (
	"errors"

	"github.com/stnhrsprkwns/fsm/dfa"
	"github.com/stnhrsprkwns/fsm/engine"
	"github.com/stnhrsprkwns/fsm/observer"
	"github.com/stnhrsprkwns/fsm/runner"
)

// DFAGraph is a minimal graph interface for DFA builders.
type DFAGraph[State any, Symbol any] interface {
	Delta(from State, label Symbol) []State
}

// Executor controls how callbacks are executed.
type Executor[S any, E, SP, EP any] interface {
	Execute(f func(from S, sp SP, to S, e E, ep EP), from S, sp SP, to S, e E, ep EP)
}

// DefaultExecutor calls the callback directly.
type DefaultExecutor[S any, E, SP, EP any] struct{}

func (DefaultExecutor[S, E, SP, EP]) Execute(
	f func(from S, sp SP, to S, e E, ep EP),
	from S,
	sp SP,
	to S,
	e E,
	ep EP,
) {
	f(from, sp, to, e, ep)
}

// DFA constructs a top-level DFA builder.
func DFA[
	State dfa.Keyed[StateKey],
	Symbol dfa.Keyed[SymbolKey],
	StateKey comparable,
	SymbolKey comparable,
	SP any,
	EP any,
]() *DFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP] {
	return &DFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP]{
		dfa:  dfa.NewBuilder[State, Symbol](),
		exec: DefaultExecutor[State, Symbol, SP, EP]{},
	}
}

// NewDFABuilder constructs a low-level DFA builder.
func NewDFABuilder[
	State dfa.Keyed[StateKey],
	Symbol dfa.Keyed[SymbolKey],
	StateKey comparable,
	SymbolKey comparable,
	SP any,
	EP any,
]() *DFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP] {
	return &DFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP]{
		dfa: dfa.NewBuilder[State, Symbol](),
	}
}

// DFABuilder wires DFA + Observer + Engine in one fluent flow.
type DFABuilder[
	State dfa.Keyed[StateKey],
	Symbol dfa.Keyed[SymbolKey],
	StateKey comparable,
	SymbolKey comparable,
	SP any,
	EP any,
] struct {
	dfa *dfa.Builder[State, Symbol, StateKey, SymbolKey]

	exec Executor[State, Symbol, SP, EP]

	// Dispatch order is fixed by phase:
	// step-any -> step -> exit-any -> exit -> enter-any -> enter
	stepAnyCallbacks  []func(from State, sp SP, to State, e Symbol, ep EP)
	stepCallbacks     []func(from State, sp SP, to State, e Symbol, ep EP)
	exitAnyCallbacks  []func(from State, sp SP, to State, e Symbol, ep EP)
	exitCallbacks     []func(from State, sp SP, to State, e Symbol, ep EP)
	enterAnyCallbacks []func(from State, sp SP, to State, e Symbol, ep EP)
	enterCallbacks    []func(from State, sp SP, to State, e Symbol, ep EP)

	dfaOverride *dfa.DFA[State, Symbol, StateKey, SymbolKey]
	obsOverride engine.Observer[State, Symbol, SP, EP]
	err         error
}

// WithDFA overrides the DFA built by this builder.
func (b *DFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP]) WithDFA(d *dfa.DFA[State, Symbol, StateKey, SymbolKey]) *DFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP] {
	b.dfaOverride = d
	return b
}

// WithGraph replaces the graph used by the DFA builder.
func (b *DFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP]) WithGraph(g DFAGraph[State, Symbol]) *DFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP] {
	b.dfa.WithGraph(g)
	return b
}

// WithStart sets q₀.
func (b *DFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP]) WithStart(state State) *DFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP] {
	b.dfa.SetStart(state)
	return b
}

// WithStates adds states to Q.
func (b *DFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP]) WithStates(states ...State) *DFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP] {
	b.dfa.AddStates(states...)
	return b
}

// WithAlphabet adds symbols to Σ.
func (b *DFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP]) WithAlphabet(symbols ...Symbol) *DFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP] {
	b.dfa.AddAlphabet(symbols...)
	return b
}

// WithAccepting adds states to F.
func (b *DFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP]) WithAccepting(states ...State) *DFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP] {
	b.dfa.AddAccepting(states...)
	return b
}

// WithTransition inserts (from, a, to) into δ.
// Preconditions: transition does not introduce nondeterminism.
func (b *DFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP]) WithTransition(from State, symbol Symbol, to State) *DFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP] {
	if b.err != nil {
		return b
	}
	if err := b.dfa.Transition(from, symbol, to); err != nil {
		b.err = err
	}
	return b
}

// WithObserver overrides the observer used by the Engine build.
func (b *DFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP]) WithObserver(obs engine.Observer[State, Symbol, SP, EP]) *DFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP] {
	b.obsOverride = obs
	return b
}

// WithExecutor sets the observer executor.
func (b *DFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP]) WithExecutor(exec Executor[State, Symbol, SP, EP]) *DFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP] {
	b.exec = exec
	return b
}

// WithOnStepAny registers a callback for every step.
func (b *DFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP]) WithOnStepAny(f func(from State, sp SP, to State, e Symbol, ep EP)) *DFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP] {
	decoratedFunc := func(from State, sp SP, to State, e Symbol, ep EP) {
		f(from, sp, to, e, ep)
	}
	b.stepAnyCallbacks = append(b.stepAnyCallbacks, decoratedFunc)
	return b
}

// WithOnStep registers a callback for a specific transition from -> to.
func (b *DFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP]) WithOnStep(from State, to State, f func(from State, sp SP, to State, e Symbol, ep EP)) *DFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP] {
	fromKey := from.Key()
	toKey := to.Key()
	decoratedFunc := func(from State, sp SP, to State, e Symbol, ep EP) {
		if from.Key() == fromKey && to.Key() == toKey {
			f(from, sp, to, e, ep)
		}
	}
	b.stepCallbacks = append(b.stepCallbacks, decoratedFunc)
	return b
}

// WithOnExit registers a callback when leaving state s.
func (b *DFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP]) WithOnExit(s State, f func(from State, sp SP, e Symbol, ep EP)) *DFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP] {
	target := s.Key()
	decoratedFunc := func(from State, sp SP, _ State, e Symbol, ep EP) {
		if from.Key() == target {
			f(from, sp, e, ep)
		}
	}
	b.exitCallbacks = append(b.exitCallbacks, decoratedFunc)
	return b
}

// WithOnEnter registers a callback when entering state s.
func (b *DFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP]) WithOnEnter(s State, f func(to State, sp SP, e Symbol, ep EP)) *DFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP] {
	target := s.Key()
	decoratedFunc := func(_ State, sp SP, to State, e Symbol, ep EP) {
		if to.Key() == target {
			f(to, sp, e, ep)
		}
	}
	b.enterCallbacks = append(b.enterCallbacks, decoratedFunc)
	return b
}

// WithOnExitAny registers a callback when leaving any state.
func (b *DFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP]) WithOnExitAny(f func(from State, sp SP, e Symbol, ep EP)) *DFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP] {
	decoratedFunc := func(from State, sp SP, _ State, e Symbol, ep EP) {
		f(from, sp, e, ep)
	}
	b.exitAnyCallbacks = append(b.exitAnyCallbacks, decoratedFunc)
	return b
}

// WithOnEnterAny registers a callback when entering any state.
func (b *DFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP]) WithOnEnterAny(f func(to State, sp SP, e Symbol, ep EP)) *DFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP] {
	decoratedFunc := func(_ State, sp SP, to State, e Symbol, ep EP) {
		f(to, sp, e, ep)
	}
	b.enterAnyCallbacks = append(b.enterAnyCallbacks, decoratedFunc)
	return b
}

// BuildDFA returns the DFA built from the configured pieces.
func (b *DFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP]) BuildDFA() (*dfa.DFA[State, Symbol, StateKey, SymbolKey], error) {
	if b.err != nil {
		return nil, b.err
	}
	if b.dfaOverride != nil {
		return b.dfaOverride, nil
	}
	return b.dfa.Build()
}

// BuildAtomicDFA returns a thread-safe DFA.
func (b *DFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP]) BuildAtomicDFA() (*dfa.AtomicDFA[State, Symbol, StateKey, SymbolKey], error) {
	d, err := b.BuildDFA()
	if err != nil {
		return nil, err
	}
	return dfa.NewAtomic(d), nil
}

func (b *DFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP]) buildObserver() (engine.Observer[State, Symbol, SP, EP], error) {
	if b.exec == nil {
		return nil, errors.New("observer builder: executor is nil")
	}

	total := len(b.stepAnyCallbacks) + len(b.stepCallbacks) + len(b.exitAnyCallbacks) + len(b.exitCallbacks) + len(b.enterAnyCallbacks) + len(b.enterCallbacks)
	callbacks := make([]func(from State, sp SP, to State, e Symbol, ep EP), 0, total)
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

	return observer.NewObserver[State, Symbol, SP, EP, State](b.exec, callbacks...), nil
}

// BuildEngine wires the DFA and observer into an Engine.
func (b *DFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP]) BuildEngine() (*engine.Engine[State, Symbol, SP, EP], error) {
	obs := b.obsOverride
	if obs == nil {
		var err error
		obs, err = b.buildObserver()
		if err != nil {
			return nil, err
		}
	}
	d, err := b.BuildDFA()
	if err != nil {
		return nil, err
	}
	return engine.DFA[State, Symbol, SP, EP](d).WithObserver(obs).Build()
}

// BuildAtomicEngine wires the DFA and observer into a thread-safe Engine.
func (b *DFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP]) BuildAtomicEngine() (*engine.AtomicEngine[State, Symbol, SP, EP], error) {
	obs := b.obsOverride
	if obs == nil {
		var err error
		obs, err = b.buildObserver()
		if err != nil {
			return nil, err
		}
	}
	d, err := b.BuildDFA()
	if err != nil {
		return nil, err
	}
	return engine.DFA[State, Symbol, SP, EP](d).WithObserver(obs).BuildAtomic()
}

// BuildRunner wires the DFA and observer into an Engine-backed Runner.
func (b *DFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP]) BuildRunner(buffer int) (*runner.Runner[Symbol, EP], error) {
	e, err := b.BuildEngine()
	if err != nil {
		return nil, err
	}
	return runner.New(e, buffer), nil
}
