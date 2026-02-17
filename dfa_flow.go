package fsm

import (
	"errors"

	"github.com/stnhrsprkwns/fsm/dfa"
	"github.com/stnhrsprkwns/fsm/engine"
	"github.com/stnhrsprkwns/fsm/key"
	"github.com/stnhrsprkwns/fsm/observer"
	"github.com/stnhrsprkwns/fsm/runner"
)

// DFAGraph is a minimal graph interface for DFA builders.
type DFAGraph[State any, Symbol any] interface {
	Delta(from State, label Symbol) []State
}

// Executor controls how callbacks are executed.
type Executor[S any, I, SP, IP any] interface {
	Execute(f func(from S, sp SP, to S, e I, ep IP), from S, sp SP, to S, e I, ep IP)
}

// DefaultExecutor calls the callback directly.
type DefaultExecutor[S any, I, SP, IP any] struct{}

func (DefaultExecutor[S, I, SP, IP]) Execute(
	f func(from S, sp SP, to S, e I, ep IP),
	from S,
	sp SP,
	to S,
	e I,
	ep IP,
) {
	f(from, sp, to, e, ep)
}

// DFA constructs a top-level DFA builder.
func DFA[
	State key.Keyer[StateKey],
	Symbol key.Keyer[SymbolKey],
	SP any,
	IP any,
	StateKey comparable,
	SymbolKey comparable,
]() *DFABuilder[State, Symbol, SP, IP, StateKey, SymbolKey] {
	return &DFABuilder[State, Symbol, SP, IP, StateKey, SymbolKey]{
		dfa:  dfa.NewBuilder[State, Symbol](),
		exec: DefaultExecutor[State, Symbol, SP, IP]{},
	}
}

// NewDFABuilder constructs a low-level DFA builder.
func NewDFABuilder[
	State key.Keyer[StateKey],
	Symbol key.Keyer[SymbolKey],
	SP any,
	IP any,
	StateKey comparable,
	SymbolKey comparable,
]() *DFABuilder[State, Symbol, SP, IP, StateKey, SymbolKey] {
	return &DFABuilder[State, Symbol, SP, IP, StateKey, SymbolKey]{
		dfa: dfa.NewBuilder[State, Symbol](),
	}
}

// DFABuilder wires DFA + Observer + Engine in one fluent flow.
type DFABuilder[
	State key.Keyer[StateKey],
	Symbol key.Keyer[SymbolKey],
	SP any,
	IP any,
	StateKey comparable,
	SymbolKey comparable,
] struct {
	dfa *dfa.Builder[State, Symbol, StateKey, SymbolKey]

	exec Executor[State, Symbol, SP, IP]

	// Dispatch order is fixed by phase:
	// step-any -> step -> exit-any -> exit -> enter-any -> enter
	stepAnyCallbacks  []func(from State, sp SP, to State, e Symbol, ep IP)
	stepCallbacks     []func(from State, sp SP, to State, e Symbol, ep IP)
	exitAnyCallbacks  []func(from State, sp SP, to State, e Symbol, ep IP)
	exitCallbacks     []func(from State, sp SP, to State, e Symbol, ep IP)
	enterAnyCallbacks []func(from State, sp SP, to State, e Symbol, ep IP)
	enterCallbacks    []func(from State, sp SP, to State, e Symbol, ep IP)

	dfaOverride *dfa.DFA[State, Symbol, StateKey, SymbolKey]
	obsOverride engine.Observer[State, Symbol, SP, IP]
	err         error
}

// WithDFA overrides the DFA built by this builder.
func (b *DFABuilder[State, Symbol, SP, IP, StateKey, SymbolKey]) WithDFA(d *dfa.DFA[State, Symbol, StateKey, SymbolKey]) *DFABuilder[State, Symbol, SP, IP, StateKey, SymbolKey] {
	b.dfaOverride = d
	return b
}

// WithGraph replaces the graph used by the DFA builder.
func (b *DFABuilder[State, Symbol, SP, IP, StateKey, SymbolKey]) WithGraph(g DFAGraph[State, Symbol]) *DFABuilder[State, Symbol, SP, IP, StateKey, SymbolKey] {
	b.dfa.WithGraph(g)
	return b
}

// WithStart sets q₀.
func (b *DFABuilder[State, Symbol, SP, IP, StateKey, SymbolKey]) WithStart(state State) *DFABuilder[State, Symbol, SP, IP, StateKey, SymbolKey] {
	b.dfa.SetStart(state)
	return b
}

// WithStates adds states to Q.
func (b *DFABuilder[State, Symbol, SP, IP, StateKey, SymbolKey]) WithStates(states ...State) *DFABuilder[State, Symbol, SP, IP, StateKey, SymbolKey] {
	b.dfa.AddStates(states...)
	return b
}

// WithAlphabet adds symbols to Σ.
func (b *DFABuilder[State, Symbol, SP, IP, StateKey, SymbolKey]) WithAlphabet(symbols ...Symbol) *DFABuilder[State, Symbol, SP, IP, StateKey, SymbolKey] {
	b.dfa.AddAlphabet(symbols...)
	return b
}

// WithAccepting adds states to F.
func (b *DFABuilder[State, Symbol, SP, IP, StateKey, SymbolKey]) WithAccepting(states ...State) *DFABuilder[State, Symbol, SP, IP, StateKey, SymbolKey] {
	b.dfa.AddAccepting(states...)
	return b
}

// WithTransition inserts (from, a, to) into δ.
// Preconditions: transition does not introduce nondeterminism.
func (b *DFABuilder[State, Symbol, SP, IP, StateKey, SymbolKey]) WithTransition(from State, symbol Symbol, to State) *DFABuilder[State, Symbol, SP, IP, StateKey, SymbolKey] {
	if b.err != nil {
		return b
	}
	if err := b.dfa.Transition(from, symbol, to); err != nil {
		b.err = err
	}
	return b
}

// WithObserver overrides the observer used by the Engine build.
func (b *DFABuilder[State, Symbol, SP, IP, StateKey, SymbolKey]) WithObserver(obs engine.Observer[State, Symbol, SP, IP]) *DFABuilder[State, Symbol, SP, IP, StateKey, SymbolKey] {
	b.obsOverride = obs
	return b
}

// WithExecutor sets the observer executor.
func (b *DFABuilder[State, Symbol, SP, IP, StateKey, SymbolKey]) WithExecutor(exec Executor[State, Symbol, SP, IP]) *DFABuilder[State, Symbol, SP, IP, StateKey, SymbolKey] {
	b.exec = exec
	return b
}

// WithOnStepAny registers a callback for every step.
func (b *DFABuilder[State, Symbol, SP, IP, StateKey, SymbolKey]) WithOnStepAny(f func(from State, sp SP, to State, e Symbol, ep IP)) *DFABuilder[State, Symbol, SP, IP, StateKey, SymbolKey] {
	decoratedFunc := func(from State, sp SP, to State, e Symbol, ep IP) {
		f(from, sp, to, e, ep)
	}
	b.stepAnyCallbacks = append(b.stepAnyCallbacks, decoratedFunc)
	return b
}

// WithOnStep registers a callback for a specific transition from -> to.
func (b *DFABuilder[State, Symbol, SP, IP, StateKey, SymbolKey]) WithOnStep(from State, to State, f func(from State, sp SP, to State, e Symbol, ep IP)) *DFABuilder[State, Symbol, SP, IP, StateKey, SymbolKey] {
	fromKey := from.Key()
	toKey := to.Key()
	decoratedFunc := func(from State, sp SP, to State, e Symbol, ep IP) {
		if from.Key() == fromKey && to.Key() == toKey {
			f(from, sp, to, e, ep)
		}
	}
	b.stepCallbacks = append(b.stepCallbacks, decoratedFunc)
	return b
}

// WithOnExit registers a callback when leaving state s.
func (b *DFABuilder[State, Symbol, SP, IP, StateKey, SymbolKey]) WithOnExit(s State, f func(from State, sp SP, e Symbol, ep IP)) *DFABuilder[State, Symbol, SP, IP, StateKey, SymbolKey] {
	target := s.Key()
	decoratedFunc := func(from State, sp SP, to State, e Symbol, ep IP) {
		fromKey := from.Key()
		toKey := to.Key()
		if fromKey == target && fromKey != toKey {
			f(from, sp, e, ep)
		}
	}
	b.exitCallbacks = append(b.exitCallbacks, decoratedFunc)
	return b
}

// WithOnEnter registers a callback when entering state s.
func (b *DFABuilder[State, Symbol, SP, IP, StateKey, SymbolKey]) WithOnEnter(s State, f func(to State, sp SP, e Symbol, ep IP)) *DFABuilder[State, Symbol, SP, IP, StateKey, SymbolKey] {
	target := s.Key()
	decoratedFunc := func(from State, sp SP, to State, e Symbol, ep IP) {
		fromKey := from.Key()
		toKey := to.Key()
		if toKey == target && fromKey != toKey {
			f(to, sp, e, ep)
		}
	}
	b.enterCallbacks = append(b.enterCallbacks, decoratedFunc)
	return b
}

// WithOnExitAny registers a callback when leaving any state.
func (b *DFABuilder[State, Symbol, SP, IP, StateKey, SymbolKey]) WithOnExitAny(f func(from State, sp SP, e Symbol, ep IP)) *DFABuilder[State, Symbol, SP, IP, StateKey, SymbolKey] {
	decoratedFunc := func(from State, sp SP, to State, e Symbol, ep IP) {
		if from.Key() != to.Key() {
			f(from, sp, e, ep)
		}
	}
	b.exitAnyCallbacks = append(b.exitAnyCallbacks, decoratedFunc)
	return b
}

// WithOnEnterAny registers a callback when entering any state.
func (b *DFABuilder[State, Symbol, SP, IP, StateKey, SymbolKey]) WithOnEnterAny(f func(to State, sp SP, e Symbol, ep IP)) *DFABuilder[State, Symbol, SP, IP, StateKey, SymbolKey] {
	decoratedFunc := func(from State, sp SP, to State, e Symbol, ep IP) {
		if from.Key() != to.Key() {
			f(to, sp, e, ep)
		}
	}
	b.enterAnyCallbacks = append(b.enterAnyCallbacks, decoratedFunc)
	return b
}

// BuildDFA returns the DFA built from the configured pieces.
func (b *DFABuilder[State, Symbol, SP, IP, StateKey, SymbolKey]) BuildDFA() (*dfa.DFA[State, Symbol, StateKey, SymbolKey], error) {
	if b.err != nil {
		return nil, b.err
	}
	if b.dfaOverride != nil {
		return b.dfaOverride, nil
	}
	return b.dfa.Build()
}

// BuildAtomicDFA returns a thread-safe DFA.
func (b *DFABuilder[State, Symbol, SP, IP, StateKey, SymbolKey]) BuildAtomicDFA() (*dfa.AtomicDFA[State, Symbol, StateKey, SymbolKey], error) {
	d, err := b.BuildDFA()
	if err != nil {
		return nil, err
	}
	return dfa.NewAtomic(d), nil
}

func (b *DFABuilder[State, Symbol, SP, IP, StateKey, SymbolKey]) buildObserver() (engine.Observer[State, Symbol, SP, IP], error) {
	if b.exec == nil {
		return nil, errors.New("observer builder: executor is nil")
	}

	total := len(b.stepAnyCallbacks) + len(b.stepCallbacks) + len(b.exitAnyCallbacks) + len(b.exitCallbacks) + len(b.enterAnyCallbacks) + len(b.enterCallbacks)
	callbacks := make([]func(from State, sp SP, to State, e Symbol, ep IP), 0, total)
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

	return observer.NewObserver[State, Symbol, SP, IP, State](b.exec, callbacks...), nil
}

// BuildEngine wires the DFA and observer into an Engine.
func (b *DFABuilder[State, Symbol, SP, IP, StateKey, SymbolKey]) BuildEngine() (*engine.Engine[State, Symbol, SP, IP], error) {
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
	return engine.DFA[State, Symbol, SP, IP](d).WithObserver(obs).Build()
}

// BuildAtomicEngine wires the DFA and observer into a thread-safe Engine.
func (b *DFABuilder[State, Symbol, SP, IP, StateKey, SymbolKey]) BuildAtomicEngine() (*engine.AtomicEngine[State, Symbol, SP, IP], error) {
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
	return engine.DFA[State, Symbol, SP, IP](d).WithObserver(obs).BuildAtomic()
}

// BuildRunner wires the DFA and observer into an Engine-backed Runner.
func (b *DFABuilder[State, Symbol, SP, IP, StateKey, SymbolKey]) BuildRunner(buffer int) (*runner.Runner[Symbol, IP], error) {
	e, err := b.BuildEngine()
	if err != nil {
		return nil, err
	}
	return runner.New(e, buffer), nil
}
