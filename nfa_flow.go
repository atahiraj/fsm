package fsm

import (
	"github.com/stnhrsprkwns/fsm/engine"
	"github.com/stnhrsprkwns/fsm/nfa"
	"github.com/stnhrsprkwns/fsm/observer"
	"github.com/stnhrsprkwns/fsm/runner"
)

// NFAGraph is a minimal graph interface for NFA builders.
type NFAGraph[State any, Symbol any] interface {
	Delta(from State, sym Symbol) []State
	Epsilon(from State) []State
}

func cloneStates[State any](states []State) []State {
	if len(states) == 0 {
		return nil
	}
	out := make([]State, len(states))
	copy(out, states)
	return out
}

func stateSetEqualByKey[State nfa.Keyed[StateKey], StateKey comparable](a, b []State) bool {
	left := make(map[StateKey]struct{}, len(a))
	right := make(map[StateKey]struct{}, len(b))
	for _, state := range a {
		left[state.Key()] = struct{}{}
	}
	for _, state := range b {
		right[state.Key()] = struct{}{}
	}
	if len(left) != len(right) {
		return false
	}
	for key := range left {
		if _, ok := right[key]; !ok {
			return false
		}
	}
	return true
}

func newNFAObserverBuilder[
	State nfa.Keyed[StateKey],
	Symbol nfa.Keyed[SymbolKey],
	StateKey comparable,
	SymbolKey comparable,
	SP any,
	EP any,
]() *observer.Builder[[]State, Symbol, SP, EP, State] {
	stateEqual := func(a, b State) bool { return a.Key() == b.Key() }
	configEqual := func(a, b []State) bool { return stateSetEqualByKey[State, StateKey](a, b) }

	return observer.NewBuilder[[]State, Symbol, SP, EP, State]().
		WithEqualState(configEqual).
		WithProjection(observer.Projection[[]State, State]{
			Members: func(state []State) []State { return cloneStates(state) },
			Equal:   stateEqual,
		})
}

// NFA constructs a top-level NFA builder.
func NFA[
	State nfa.Keyed[StateKey],
	Symbol nfa.Keyed[SymbolKey],
	StateKey comparable,
	SymbolKey comparable,
	SP any,
	EP any,
]() *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP] {
	b := &NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP]{
		nfa: nfa.NewBuilder[State, Symbol, StateKey, SymbolKey](),
	}
	b.obs = newNFAObserverBuilder[State, Symbol, StateKey, SymbolKey, SP, EP]().WithExecutor(DefaultExecutor[[]State, Symbol, SP, EP]{})
	return b
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
		nfa: nfa.NewBuilder[State, Symbol, StateKey, SymbolKey](),
		obs: newNFAObserverBuilder[State, Symbol, StateKey, SymbolKey, SP, EP](),
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
	nfa         *nfa.Builder[State, Symbol, StateKey, SymbolKey]
	obs         *observer.Builder[[]State, Symbol, SP, EP, State]
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
	b.obs.WithExecutor(exec)
	return b
}

// WithOnStepAny registers a callback for every step.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP]) WithOnStepAny(f func(from []State, sp SP, to []State, e Symbol, ep EP)) *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP] {
	b.obs.OnStepAny(f)
	return b
}

// WithOnStep registers a callback for a specific transition from -> to.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP]) WithOnStep(from []State, to []State, f func(from []State, sp SP, to []State, e Symbol, ep EP)) *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP] {
	b.obs.OnStep(from, to, f)
	return b
}

// WithOnExit registers a callback when leaving configuration s.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP]) WithOnExit(s []State, f func(from []State, sp SP, e Symbol, ep EP)) *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP] {
	b.obs.OnExit(s, f)
	return b
}

// WithOnEnter registers a callback when entering configuration s.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP]) WithOnEnter(s []State, f func(to []State, sp SP, e Symbol, ep EP)) *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP] {
	b.obs.OnEnter(s, f)
	return b
}

// WithOnExitState registers a callback when a specific state exits the active set.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP]) WithOnExitState(state State, f func(sp SP, e Symbol, ep EP)) *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP] {
	b.obs.OnExitMember(state, f)
	return b
}

// WithOnEnterState registers a callback when a specific state enters the active set.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP]) WithOnEnterState(state State, f func(sp SP, e Symbol, ep EP)) *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP] {
	b.obs.OnEnterMember(state, f)
	return b
}

// WithOnExitAny registers a callback when leaving any state.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP]) WithOnExitAny(f func(from []State, sp SP, e Symbol, ep EP)) *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP] {
	b.obs.OnExitAny(f)
	return b
}

// WithOnEnterAny registers a callback when entering any state.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP]) WithOnEnterAny(f func(to []State, sp SP, e Symbol, ep EP)) *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP] {
	b.obs.OnEnterAny(f)
	return b
}

// WithOnExitStateAny registers a callback when any state exits the active set.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP]) WithOnExitStateAny(f func(state State, sp SP, e Symbol, ep EP)) *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP] {
	b.obs.OnExitMemberAny(f)
	return b
}

// WithOnEnterStateAny registers a callback when any state enters the active set.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP]) WithOnEnterStateAny(f func(state State, sp SP, e Symbol, ep EP)) *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP] {
	b.obs.OnEnterMemberAny(f)
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

// BuildEngine wires the NFA and observer into an Engine.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey, SP, EP]) BuildEngine() (*engine.Engine[[]State, Symbol, SP, EP], error) {
	obs := b.obsOverride
	if obs == nil {
		var err error
		obs, err = b.obs.Build()
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
		obs, err = b.obs.Build()
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
