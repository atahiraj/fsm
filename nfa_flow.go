package fsm

import (
	"errors"

	"github.com/stnhrsprkwns/fsm/engine"
	"github.com/stnhrsprkwns/fsm/internal/set"
	"github.com/stnhrsprkwns/fsm/nfa"
	"github.com/stnhrsprkwns/fsm/observer"
	"github.com/stnhrsprkwns/fsm/runner"
)

// NFAGraph is a minimal graph interface for NFA builders.
type NFAGraph[State any, Symbol any] interface {
	Delta(from State, sym Symbol) []State
	Epsilon(from State) []State
}

type nfaObserverCallback[State any, Symbol any, SP any, EP any] func(from []State, sp SP, to []State, e Symbol, ep EP)

// NFA constructs a top-level NFA builder.
func NFA[
	State nfa.Keyed[StateKey],
	Symbol nfa.Keyed[SymbolKey],
	StateKey comparable,
	SymbolKey comparable,
	SP any,
	EP any,
]() *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP] {
	return &NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP]{
		nfa:  nfa.NewBuilder[State, Symbol](),
		exec: DefaultExecutor[[]State, Symbol, SP, EP]{},
	}
}

// NewNFABuilder constructs a low-level NFA builder.
func NewNFABuilder[
	State nfa.Keyed[StateKey],
	Symbol nfa.Keyed[SymbolKey],
	StateKey comparable,
	SymbolKey comparable,
	SP any,
	EP any,
]() *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP] {
	return &NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP]{
		nfa: nfa.NewBuilder[State, Symbol](),
	}
}

// NFABuilder wires NFA + Observer + Engine in one fluent flow.
type NFABuilder[
	State nfa.Keyed[StateKey],
	Symbol nfa.Keyed[SymbolKey],
	StateKey comparable,
	SymbolKey comparable,
	SP any,
	EP any,
] struct {
	nfa *nfa.Builder[State, Symbol, StateKey, SymbolKey]

	exec Executor[[]State, Symbol, SP, EP]

	// Dispatch order is fixed by phase:
	// exit-state -> exit -> step-any -> step -> enter-state -> enter
	exitStateCallbacks  []nfaObserverCallback[State, Symbol, SP, EP]
	exitCallbacks       []nfaObserverCallback[State, Symbol, SP, EP]
	stepAnyCallbacks    []nfaObserverCallback[State, Symbol, SP, EP]
	stepCallbacks       []nfaObserverCallback[State, Symbol, SP, EP]
	enterStateCallbacks []nfaObserverCallback[State, Symbol, SP, EP]
	enterCallbacks      []nfaObserverCallback[State, Symbol, SP, EP]

	nfaOverride *nfa.NFA[State, Symbol, StateKey, SymbolKey]
	obsOverride engine.Observer[[]State, Symbol, SP, EP]
}

// WithNFA overrides the NFA built by this builder.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP]) WithNFA(n *nfa.NFA[State, Symbol, StateKey, SymbolKey]) *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP] {
	b.nfaOverride = n
	return b
}

// WithGraph replaces the graph used by the NFA builder.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP]) WithGraph(g NFAGraph[State, Symbol]) *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP] {
	b.nfa.WithGraph(g)
	return b
}

// WithStart sets q₀.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP]) WithStart(state State) *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP] {
	b.nfa.SetStart(state)
	return b
}

// WithStates adds states to Q.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP]) WithStates(states ...State) *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP] {
	b.nfa.AddStates(states...)
	return b
}

// WithAlphabet adds symbols to Σ.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP]) WithAlphabet(symbols ...Symbol) *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP] {
	b.nfa.AddAlphabet(symbols...)
	return b
}

// WithAccepting adds states to F.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP]) WithAccepting(states ...State) *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP] {
	b.nfa.AddAccepting(states...)
	return b
}

// WithTransition inserts (from, a, to) into δ.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP]) WithTransition(from State, symbol Symbol, to State) *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP] {
	b.nfa.Transition(from, symbol, to)
	return b
}

// WithEpsilon inserts (from, ε, to) into δ.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP]) WithEpsilon(from State, to State) *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP] {
	b.nfa.Epsilon(from, to)
	return b
}

// WithObserver overrides the observer used by the Engine build.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP]) WithObserver(obs engine.Observer[[]State, Symbol, SP, EP]) *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP] {
	b.obsOverride = obs
	return b
}

// WithExecutor sets the observer executor.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP]) WithExecutor(exec Executor[[]State, Symbol, SP, EP]) *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP] {
	b.exec = exec
	return b
}

// WithOnStepAny registers a callback for every step.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP]) WithOnStepAny(f func(from []State, sp SP, to []State, e Symbol, ep EP)) *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP] {
	decoratedFunc := func(from []State, sp SP, to []State, e Symbol, ep EP) {
		f(from, sp, to, e, ep)
	}
	b.stepAnyCallbacks = append(b.stepAnyCallbacks, decoratedFunc)
	return b
}

// WithOnStep registers a callback for a specific transition from -> to.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP]) WithOnStep(from []State, to []State, f func(from []State, sp SP, to []State, e Symbol, ep EP)) *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP] {
	fromKeySet := stateSetToKeySet(from)
	toKeySet := stateSetToKeySet(to)
	decoratedFunc := func(from []State, sp SP, to []State, e Symbol, ep EP) {
		if fromKeySet.Equals(stateSetToKeySet(from)) &&
			toKeySet.Equals(stateSetToKeySet(to)) {
			f(from, sp, to, e, ep)
		}
	}
	b.stepCallbacks = append(b.stepCallbacks, decoratedFunc)
	return b
}

// WithOnExit registers a callback when leaving configuration s.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP]) WithOnExit(s []State, f func(from []State, sp SP, e Symbol, ep EP)) *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP] {
	keySet := stateSetToKeySet(s)
	decoratedFunc := func(from []State, sp SP, _ []State, e Symbol, ep EP) {
		if keySet.Equals(stateSetToKeySet(from)) {
			f(from, sp, e, ep)
		}
	}
	b.exitCallbacks = append(b.exitCallbacks, decoratedFunc)
	return b
}

// WithOnEnter registers a callback when entering configuration s.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP]) WithOnEnter(s []State, f func(to []State, sp SP, e Symbol, ep EP)) *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP] {
	keySet := stateSetToKeySet(s)
	decoratedFunc := func(_ []State, sp SP, to []State, e Symbol, ep EP) {
		if keySet.Equals(stateSetToKeySet(to)) {
			f(to, sp, e, ep)
		}
	}
	b.enterCallbacks = append(b.enterCallbacks, decoratedFunc)
	return b
}

// WithOnExitState registers a callback when a specific state exits the active set.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP]) WithOnExitState(state State, f func(sp SP, e Symbol, ep EP)) *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP] {
	target := state.Key()
	decoratedFunc := func(from []State, sp SP, to []State, e Symbol, ep EP) {
		fromKeySet := stateSetToKeySet(from)
		toKeySet := stateSetToKeySet(to)
		if fromKeySet.Has(target) && !toKeySet.Has(target) {
			f(sp, e, ep)
		}
	}
	b.exitStateCallbacks = append(b.exitStateCallbacks, decoratedFunc)
	return b
}

// WithOnEnterState registers a callback when a specific state enters the active set.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP]) WithOnEnterState(state State, f func(sp SP, e Symbol, ep EP)) *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP] {
	target := state.Key()
	decoratedFunc := func(from []State, sp SP, to []State, e Symbol, ep EP) {
		fromKeySet := stateSetToKeySet(from)
		toKeySet := stateSetToKeySet(to)
		if !fromKeySet.Has(target) && toKeySet.Has(target) {
			f(sp, e, ep)
		}
	}
	b.enterStateCallbacks = append(b.enterStateCallbacks, decoratedFunc)
	return b
}

// BuildNFA returns the NFA built from the configured pieces.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP]) BuildNFA() (*nfa.NFA[State, Symbol, StateKey, SymbolKey], error) {
	if b.nfaOverride != nil {
		return b.nfaOverride, nil
	}
	return b.nfa.Build()
}

// BuildAtomicNFA returns a thread-safe NFA.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP]) BuildAtomicNFA() (*nfa.AtomicNFA[State, Symbol, StateKey, SymbolKey], error) {
	n, err := b.BuildNFA()
	if err != nil {
		return nil, err
	}
	return nfa.NewAtomic(n), nil
}

func (b *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP]) buildObserver() (engine.Observer[[]State, Symbol, SP, EP], error) {
	if b.exec == nil {
		return nil, errors.New("observer builder: executor is nil")
	}

	total := len(b.exitStateCallbacks) + len(b.exitCallbacks) + len(b.stepAnyCallbacks) + len(b.stepCallbacks) + len(b.enterStateCallbacks) + len(b.enterCallbacks)
	callbacks := make([]func(from []State, sp SP, to []State, e Symbol, ep EP), 0, total)
	for _, callback := range b.exitStateCallbacks {
		callbacks = append(callbacks, callback)
	}
	for _, callback := range b.exitCallbacks {
		callbacks = append(callbacks, callback)
	}
	for _, callback := range b.stepAnyCallbacks {
		callbacks = append(callbacks, callback)
	}
	for _, callback := range b.stepCallbacks {
		callbacks = append(callbacks, callback)
	}
	for _, callback := range b.enterStateCallbacks {
		callbacks = append(callbacks, callback)
	}
	for _, callback := range b.enterCallbacks {
		callbacks = append(callbacks, callback)
	}

	return observer.NewObserver[[]State, Symbol, SP, EP, State](b.exec, callbacks...), nil
}

// BuildEngine wires the NFA and observer into an Engine.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP]) BuildEngine() (*engine.Engine[[]State, Symbol, SP, EP], error) {
	obs := b.obsOverride
	if obs == nil {
		var err error
		obs, err = b.buildObserver()
		if err != nil {
			return nil, err
		}
	}
	n, err := b.BuildNFA()
	if err != nil {
		return nil, err
	}
	return engine.NFA[State, Symbol, []State, SP, EP](n).WithObserver(obs).Build()
}

// BuildAtomicEngine wires the NFA and observer into a thread-safe Engine.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP]) BuildAtomicEngine() (*engine.AtomicEngine[[]State, Symbol, SP, EP], error) {
	obs := b.obsOverride
	if obs == nil {
		var err error
		obs, err = b.buildObserver()
		if err != nil {
			return nil, err
		}
	}
	n, err := b.BuildNFA()
	if err != nil {
		return nil, err
	}
	return engine.NFA[State, Symbol, []State, SP, EP](n).WithObserver(obs).BuildAtomic()
}

// BuildRunner wires the NFA and observer into an Engine-backed Runner.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP]) BuildRunner(buffer int) (*runner.Runner[Symbol, EP], error) {
	e, err := b.BuildEngine()
	if err != nil {
		return nil, err
	}
	return runner.New(e, buffer), nil
}

func stateSetToKeySet[State nfa.Keyed[StateKey], StateKey comparable](states []State) *set.Set[StateKey] {
	keys := make([]StateKey, 0, len(states))
	for _, state := range states {
		keys = append(keys, state.Key())
	}
	out := set.New(keys...)
	return out
}
