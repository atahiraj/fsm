package fsm

import (
	"errors"

	"github.com/stnhrsprkwns/fsm/engine"
	"github.com/stnhrsprkwns/fsm/internal/set"
	"github.com/stnhrsprkwns/fsm/key"
	"github.com/stnhrsprkwns/fsm/nfa"
	"github.com/stnhrsprkwns/fsm/observer"
	"github.com/stnhrsprkwns/fsm/runner"
)

// NFAGraph is a minimal graph interface for NFA builders.
type NFAGraph[State any, Symbol any] interface {
	Delta(from State, sym Symbol) []State
	Epsilon(from State) []State
}

type nfaObserverCallback[State any, Symbol any, SP any, IP any] func(from []State, sp SP, to []State, e Symbol, ep IP)

// NFA constructs a top-level NFA builder.
func NFA[
	State key.Keyer[StateKey],
	Symbol key.Keyer[SymbolKey],
	StateKey comparable,
	SymbolKey comparable,
	SP any,
	IP any,
]() *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, IP] {
	return &NFABuilder[State, Symbol, StateKey, SymbolKey, SP, IP]{
		nfa:  nfa.NewBuilder[State, Symbol](),
		exec: DefaultExecutor[[]State, Symbol, SP, IP]{},
	}
}

// NewNFABuilder constructs a low-level NFA builder.
func NewNFABuilder[
	State key.Keyer[StateKey],
	Symbol key.Keyer[SymbolKey],
	StateKey comparable,
	SymbolKey comparable,
	SP any,
	IP any,
]() *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, IP] {
	return &NFABuilder[State, Symbol, StateKey, SymbolKey, SP, IP]{
		nfa: nfa.NewBuilder[State, Symbol](),
	}
}

// NFABuilder wires NFA + Observer + Engine in one fluent flow.
type NFABuilder[
	State key.Keyer[StateKey],
	Symbol key.Keyer[SymbolKey],
	StateKey comparable,
	SymbolKey comparable,
	SP any,
	IP any,
] struct {
	nfa *nfa.Builder[State, Symbol, StateKey, SymbolKey]

	exec Executor[[]State, Symbol, SP, IP]

	// Dispatch order is fixed by phase:
	// exit-state -> exit -> step-any -> step -> enter-state -> enter
	exitStateCallbacks  []nfaObserverCallback[State, Symbol, SP, IP]
	exitCallbacks       []nfaObserverCallback[State, Symbol, SP, IP]
	stepAnyCallbacks    []nfaObserverCallback[State, Symbol, SP, IP]
	stepCallbacks       []nfaObserverCallback[State, Symbol, SP, IP]
	enterStateCallbacks []nfaObserverCallback[State, Symbol, SP, IP]
	enterCallbacks      []nfaObserverCallback[State, Symbol, SP, IP]

	nfaOverride *nfa.NFA[State, Symbol, StateKey, SymbolKey]
	obsOverride engine.Observer[[]State, Symbol, SP, IP]
}

// WithNFA overrides the NFA built by this builder.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, IP]) WithNFA(n *nfa.NFA[State, Symbol, StateKey, SymbolKey]) *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, IP] {
	b.nfaOverride = n
	return b
}

// WithGraph replaces the graph used by the NFA builder.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, IP]) WithGraph(g NFAGraph[State, Symbol]) *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, IP] {
	b.nfa.WithGraph(g)
	return b
}

// WithStart sets q₀.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, IP]) WithStart(state State) *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, IP] {
	b.nfa.SetStart(state)
	return b
}

// WithStates adds states to Q.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, IP]) WithStates(states ...State) *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, IP] {
	b.nfa.AddStates(states...)
	return b
}

// WithAlphabet adds symbols to Σ.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, IP]) WithAlphabet(symbols ...Symbol) *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, IP] {
	b.nfa.AddAlphabet(symbols...)
	return b
}

// WithAccepting adds states to F.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, IP]) WithAccepting(states ...State) *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, IP] {
	b.nfa.AddAccepting(states...)
	return b
}

// WithTransition inserts (from, a, to) into δ.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, IP]) WithTransition(from State, symbol Symbol, to State) *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, IP] {
	b.nfa.Transition(from, symbol, to)
	return b
}

// WithEpsilon inserts (from, ε, to) into δ.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, IP]) WithEpsilon(from State, to State) *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, IP] {
	b.nfa.Epsilon(from, to)
	return b
}

// WithObserver overrides the observer used by the Engine build.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, IP]) WithObserver(obs engine.Observer[[]State, Symbol, SP, IP]) *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, IP] {
	b.obsOverride = obs
	return b
}

// WithExecutor sets the observer executor.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, IP]) WithExecutor(exec Executor[[]State, Symbol, SP, IP]) *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, IP] {
	b.exec = exec
	return b
}

// WithOnStepAny registers a callback for every step.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, IP]) WithOnStepAny(f func(from []State, sp SP, to []State, e Symbol, ep IP)) *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, IP] {
	b.stepAnyCallbacks = append(b.stepAnyCallbacks, f)
	return b
}

// WithOnStep registers a callback for a specific transition from -> to.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, IP]) WithOnStep(from []State, to []State, f func(sp SP, e Symbol, ep IP)) *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, IP] {
	fromKeySet := stateSetToKeySet(from)
	toKeySet := stateSetToKeySet(to)
	decoratedFunc := func(from []State, sp SP, to []State, e Symbol, ep IP) {
		if fromKeySet.Equals(stateSetToKeySet(from)) &&
			toKeySet.Equals(stateSetToKeySet(to)) {
			f(sp, e, ep)
		}
	}
	b.stepCallbacks = append(b.stepCallbacks, decoratedFunc)
	return b
}

// WithOnExit registers a callback when leaving configuration s.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, IP]) WithOnExit(s []State, f func(sp SP, e Symbol, ep IP)) *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, IP] {
	keySet := stateSetToKeySet(s)
	decoratedFunc := func(from []State, sp SP, to []State, e Symbol, ep IP) {
		fromKeySet := stateSetToKeySet(from)
		toKeySet := stateSetToKeySet(to)
		if keySet.Equals(fromKeySet) && !fromKeySet.Equals(toKeySet) {
			f(sp, e, ep)
		}
	}
	b.exitCallbacks = append(b.exitCallbacks, decoratedFunc)
	return b
}

// WithOnEnter registers a callback when entering configuration s.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, IP]) WithOnEnter(s []State, f func(sp SP, e Symbol, ep IP)) *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, IP] {
	keySet := stateSetToKeySet(s)
	decoratedFunc := func(from []State, sp SP, to []State, e Symbol, ep IP) {
		fromKeySet := stateSetToKeySet(from)
		toKeySet := stateSetToKeySet(to)
		if keySet.Equals(toKeySet) && !fromKeySet.Equals(toKeySet) {
			f(sp, e, ep)
		}
	}
	b.enterCallbacks = append(b.enterCallbacks, decoratedFunc)
	return b
}

// WithOnExitState registers a callback when a specific state exits the active set.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, IP]) WithOnExitState(state State, f func(sp SP, e Symbol, ep IP)) *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, IP] {
	target := state.Key()
	decoratedFunc := func(from []State, sp SP, to []State, e Symbol, ep IP) {
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
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, IP]) WithOnEnterState(state State, f func(sp SP, e Symbol, ep IP)) *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, IP] {
	target := state.Key()
	decoratedFunc := func(from []State, sp SP, to []State, e Symbol, ep IP) {
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
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, IP]) BuildNFA() (*nfa.NFA[State, Symbol, StateKey, SymbolKey], error) {
	if b.nfaOverride != nil {
		return b.nfaOverride, nil
	}
	return b.nfa.Build()
}

// BuildAtomicNFA returns a thread-safe NFA.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, IP]) BuildAtomicNFA() (*nfa.AtomicNFA[State, Symbol, StateKey, SymbolKey], error) {
	n, err := b.BuildNFA()
	if err != nil {
		return nil, err
	}
	return nfa.NewAtomic(n), nil
}

func (b *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, IP]) buildObserver() (engine.Observer[[]State, Symbol, SP, IP], error) {
	if b.exec == nil {
		return nil, errors.New("observer builder: executor is nil")
	}

	total := len(b.exitStateCallbacks) + len(b.exitCallbacks) + len(b.stepAnyCallbacks) + len(b.stepCallbacks) + len(b.enterStateCallbacks) + len(b.enterCallbacks)
	callbacks := make([]func(from []State, sp SP, to []State, e Symbol, ep IP), 0, total)
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

	return observer.NewObserver[[]State, Symbol, SP, IP, State](b.exec, callbacks...), nil
}

// BuildEngine wires the NFA and observer into an Engine.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, IP]) BuildEngine() (*engine.Engine[[]State, Symbol, SP, IP], error) {
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
	return engine.NFA[State, Symbol, SP, IP](n).WithObserver(obs).Build()
}

// BuildAtomicEngine wires the NFA and observer into a thread-safe Engine.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, IP]) BuildAtomicEngine() (*engine.AtomicEngine[[]State, Symbol, SP, IP], error) {
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
	return engine.NFA[State, Symbol, SP, IP](n).WithObserver(obs).BuildAtomic()
}

// BuildRunner wires the NFA and observer into an Engine-backed Runner.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, IP]) BuildRunner(buffer int) (*runner.Runner[Symbol, IP], error) {
	e, err := b.BuildEngine()
	if err != nil {
		return nil, err
	}
	return runner.New(e, buffer), nil
}

func stateSetToKeySet[State key.Keyer[StateKey], StateKey comparable](states []State) *set.Set[StateKey] {
	keys := make([]StateKey, 0, len(states))
	for _, state := range states {
		keys = append(keys, state.Key())
	}
	out := set.New(keys...)
	return out
}
