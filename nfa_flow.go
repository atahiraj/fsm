package fsm

import (
	"errors"

	"github.com/stnhrsprkwns/fsm/engine"
	"github.com/stnhrsprkwns/fsm/internal/set"
	"github.com/stnhrsprkwns/fsm/key"
	"github.com/stnhrsprkwns/fsm/nfa"
	"github.com/stnhrsprkwns/fsm/runner"
	"github.com/stnhrsprkwns/fsm/transitionhooks"
)

// NFAGraph is a minimal graph interface for NFA builders.
type NFAGraph[State any, Symbol any] interface {
	Delta(from State, sym Symbol) []State
	Epsilon(from State) []State
}

type nfaTransitionCallback[State any, Symbol any] func(from []State, to []State, e Symbol)

// NFA constructs a top-level NFA builder.
func NFA[
	State key.Keyer[StateKey],
	Symbol key.Keyer[SymbolKey],
	StateKey comparable,
	SymbolKey comparable,
]() *NFABuilder[State, Symbol, StateKey, SymbolKey] {
	return &NFABuilder[State, Symbol, StateKey, SymbolKey]{
		nfa:  nfa.NewBuilder[State, Symbol](),
		exec: DefaultExecutor[[]State, Symbol]{},
	}
}

// NewNFABuilder constructs a low-level NFA builder.
func NewNFABuilder[
	State key.Keyer[StateKey],
	Symbol key.Keyer[SymbolKey],
	StateKey comparable,
	SymbolKey comparable,
]() *NFABuilder[State, Symbol, StateKey, SymbolKey] {
	return &NFABuilder[State, Symbol, StateKey, SymbolKey]{
		nfa: nfa.NewBuilder[State, Symbol](),
	}
}

// NFABuilder wires NFA + TransitionHooks + Engine in one fluent flow.
type NFABuilder[
	State key.Keyer[StateKey],
	Symbol key.Keyer[SymbolKey],
	StateKey comparable,
	SymbolKey comparable,
] struct {
	nfa *nfa.Builder[State, Symbol, StateKey, SymbolKey]

	exec Executor[[]State, Symbol]

	selfTransitionCallbacks bool

	// Dispatch order is fixed by phase:
	// exit-state -> exit -> step-any -> step -> enter-state -> enter
	exitStateCallbacks  []nfaTransitionCallback[State, Symbol]
	exitCallbacks       []nfaTransitionCallback[State, Symbol]
	stepAnyCallbacks    []nfaTransitionCallback[State, Symbol]
	stepCallbacks       []nfaTransitionCallback[State, Symbol]
	enterStateCallbacks []nfaTransitionCallback[State, Symbol]
	enterCallbacks      []nfaTransitionCallback[State, Symbol]

	nfaOverride   *nfa.NFA[State, Symbol, StateKey, SymbolKey]
	hooksOverride engine.TransitionHooks[[]State, Symbol]
}

// WithNFA overrides the NFA built by this builder.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey]) WithNFA(n *nfa.NFA[State, Symbol, StateKey, SymbolKey]) *NFABuilder[State, Symbol, StateKey, SymbolKey] {
	b.nfaOverride = n
	return b
}

// WithGraph replaces the graph used by the NFA builder.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey]) WithGraph(g NFAGraph[State, Symbol]) *NFABuilder[State, Symbol, StateKey, SymbolKey] {
	b.nfa.WithGraph(g)
	return b
}

// WithStart sets q₀.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey]) WithStart(state State) *NFABuilder[State, Symbol, StateKey, SymbolKey] {
	b.nfa.SetStart(state)
	return b
}

// WithStates adds states to Q.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey]) WithStates(states ...State) *NFABuilder[State, Symbol, StateKey, SymbolKey] {
	b.nfa.AddStates(states...)
	return b
}

// WithAlphabet adds symbols to Σ.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey]) WithAlphabet(symbols ...Symbol) *NFABuilder[State, Symbol, StateKey, SymbolKey] {
	b.nfa.AddAlphabet(symbols...)
	return b
}

// WithAccepting adds states to F.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey]) WithAccepting(states ...State) *NFABuilder[State, Symbol, StateKey, SymbolKey] {
	b.nfa.AddAccepting(states...)
	return b
}

// WithTransition inserts (from, a, to) into δ.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey]) WithTransition(from State, symbol Symbol, to State) *NFABuilder[State, Symbol, StateKey, SymbolKey] {
	b.nfa.Transition(from, symbol, to)
	return b
}

// WithEpsilon inserts (from, ε, to) into δ.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey]) WithEpsilon(from State, to State) *NFABuilder[State, Symbol, StateKey, SymbolKey] {
	b.nfa.Epsilon(from, to)
	return b
}

// WithTransitionHooks overrides the transition hooks used by the Engine build.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey]) WithTransitionHooks(hooks engine.TransitionHooks[[]State, Symbol]) *NFABuilder[State, Symbol, StateKey, SymbolKey] {
	b.hooksOverride = hooks
	return b
}

// WithExecutor sets the transition hooks executor.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey]) WithExecutor(exec Executor[[]State, Symbol]) *NFABuilder[State, Symbol, StateKey, SymbolKey] {
	b.exec = exec
	return b
}

// WithSelfTransitionCallbacks enables config-level enter/exit callbacks when the configuration is unchanged.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey]) WithSelfTransitionCallbacks() *NFABuilder[State, Symbol, StateKey, SymbolKey] {
	b.selfTransitionCallbacks = true
	return b
}

// WithOnStepAny registers a callback for every step.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey]) WithOnStepAny(f func(from []State, to []State, e Symbol)) *NFABuilder[State, Symbol, StateKey, SymbolKey] {
	b.stepAnyCallbacks = append(b.stepAnyCallbacks, f)
	return b
}

// WithOnStep registers a callback for a specific transition from -> to.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey]) WithOnStep(from []State, to []State, f func(e Symbol)) *NFABuilder[State, Symbol, StateKey, SymbolKey] {
	fromKeySet := stateSetToKeySet(from)
	toKeySet := stateSetToKeySet(to)
	decoratedFunc := func(from []State, to []State, e Symbol) {
		if fromKeySet.Equals(stateSetToKeySet(from)) &&
			toKeySet.Equals(stateSetToKeySet(to)) {
			f(e)
		}
	}
	b.stepCallbacks = append(b.stepCallbacks, decoratedFunc)
	return b
}

// WithOnExit registers a callback when leaving configuration s.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey]) WithOnExit(s []State, f func(e Symbol)) *NFABuilder[State, Symbol, StateKey, SymbolKey] {
	keySet := stateSetToKeySet(s)
	decoratedFunc := func(from []State, to []State, e Symbol) {
		fromKeySet := stateSetToKeySet(from)
		toKeySet := stateSetToKeySet(to)
		if keySet.Equals(fromKeySet) && (b.selfTransitionCallbacks || !fromKeySet.Equals(toKeySet)) {
			f(e)
		}
	}
	b.exitCallbacks = append(b.exitCallbacks, decoratedFunc)
	return b
}

// WithOnEnter registers a callback when entering configuration s.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey]) WithOnEnter(s []State, f func(e Symbol)) *NFABuilder[State, Symbol, StateKey, SymbolKey] {
	keySet := stateSetToKeySet(s)
	decoratedFunc := func(from []State, to []State, e Symbol) {
		fromKeySet := stateSetToKeySet(from)
		toKeySet := stateSetToKeySet(to)
		if keySet.Equals(toKeySet) && (b.selfTransitionCallbacks || !fromKeySet.Equals(toKeySet)) {
			f(e)
		}
	}
	b.enterCallbacks = append(b.enterCallbacks, decoratedFunc)
	return b
}

// WithOnExitState registers a callback when a specific state exits the active set.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey]) WithOnExitState(state State, f func(e Symbol)) *NFABuilder[State, Symbol, StateKey, SymbolKey] {
	target := state.Key()
	decoratedFunc := func(from []State, to []State, e Symbol) {
		fromKeySet := stateSetToKeySet(from)
		toKeySet := stateSetToKeySet(to)
		if fromKeySet.Has(target) && !toKeySet.Has(target) {
			f(e)
		}
	}
	b.exitStateCallbacks = append(b.exitStateCallbacks, decoratedFunc)
	return b
}

// WithOnEnterState registers a callback when a specific state enters the active set.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey]) WithOnEnterState(state State, f func(e Symbol)) *NFABuilder[State, Symbol, StateKey, SymbolKey] {
	target := state.Key()
	decoratedFunc := func(from []State, to []State, e Symbol) {
		fromKeySet := stateSetToKeySet(from)
		toKeySet := stateSetToKeySet(to)
		if !fromKeySet.Has(target) && toKeySet.Has(target) {
			f(e)
		}
	}
	b.enterStateCallbacks = append(b.enterStateCallbacks, decoratedFunc)
	return b
}

// BuildNFA returns the NFA built from the configured pieces.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey]) BuildNFA() (*nfa.NFA[State, Symbol, StateKey, SymbolKey], error) {
	if b.nfaOverride != nil {
		return b.nfaOverride, nil
	}
	return b.nfa.Build()
}

// BuildAtomicNFA returns a thread-safe NFA.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey]) BuildAtomicNFA() (*nfa.AtomicNFA[State, Symbol, StateKey, SymbolKey], error) {
	n, err := b.BuildNFA()
	if err != nil {
		return nil, err
	}
	return nfa.NewAtomic(n), nil
}

func (b *NFABuilder[State, Symbol, StateKey, SymbolKey]) buildTransitionHooks() (engine.TransitionHooks[[]State, Symbol], error) {
	if b.exec == nil {
		return nil, errors.New("transition hooks builder: executor is nil")
	}

	total := len(b.exitStateCallbacks) + len(b.exitCallbacks) + len(b.stepAnyCallbacks) + len(b.stepCallbacks) + len(b.enterStateCallbacks) + len(b.enterCallbacks)
	callbacks := make([]func(from []State, to []State, e Symbol), 0, total)
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

	return transitionhooks.New[[]State, Symbol](b.exec, callbacks...), nil
}

// BuildEngine wires the NFA and transition hooks into an Engine.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey]) BuildEngine() (*engine.Engine[[]State, Symbol], error) {
	hooks := b.hooksOverride
	if hooks == nil {
		var err error
		hooks, err = b.buildTransitionHooks()
		if err != nil {
			return nil, err
		}
	}
	n, err := b.BuildNFA()
	if err != nil {
		return nil, err
	}
	return engine.NFA[State, Symbol](n).WithTransitionHooks(hooks).Build()
}

// BuildAtomicEngine wires the NFA and transition hooks into a thread-safe Engine.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey]) BuildAtomicEngine() (*engine.AtomicEngine[[]State, Symbol], error) {
	hooks := b.hooksOverride
	if hooks == nil {
		var err error
		hooks, err = b.buildTransitionHooks()
		if err != nil {
			return nil, err
		}
	}
	n, err := b.BuildNFA()
	if err != nil {
		return nil, err
	}
	return engine.NFA[State, Symbol](n).WithTransitionHooks(hooks).BuildAtomic()
}

// BuildRunner wires the NFA and transition hooks into an Engine-backed Runner.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey]) BuildRunner(buffer int) (*runner.Runner[Symbol], error) {
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
