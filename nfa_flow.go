package fsm

import (
	"errors"
	"sort"

	"github.com/stnhrsprkwns/fsm/engine"
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

type nfaStepCallback[State key.Keyer[StateKey], StateKey comparable, Symbol any] struct {
	fromKeys []StateKey
	toKeys   []StateKey
	callback func(e Symbol)
}

type nfaConfigCallback[State key.Keyer[StateKey], StateKey comparable, Symbol any] struct {
	keys     []StateKey
	callback func(e Symbol)
}

type nfaStateCallback[StateKey comparable, Symbol any] struct {
	target   StateKey
	callback func(e Symbol)
}

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
	exitStateCallbacks  []nfaStateCallback[StateKey, Symbol]
	exitCallbacks       []nfaConfigCallback[State, StateKey, Symbol]
	stepAnyCallbacks    []func(from []State, to []State, e Symbol)
	stepCallbacks       []nfaStepCallback[State, StateKey, Symbol]
	enterStateCallbacks []nfaStateCallback[StateKey, Symbol]
	enterCallbacks      []nfaConfigCallback[State, StateKey, Symbol]

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
	b.stepCallbacks = append(b.stepCallbacks, nfaStepCallback[State, StateKey, Symbol]{
		fromKeys: stateSliceToUniqueKeys(from),
		toKeys:   stateSliceToUniqueKeys(to),
		callback: f,
	})
	return b
}

// WithOnExit registers a callback when leaving configuration s.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey]) WithOnExit(s []State, f func(e Symbol)) *NFABuilder[State, Symbol, StateKey, SymbolKey] {
	b.exitCallbacks = append(b.exitCallbacks, nfaConfigCallback[State, StateKey, Symbol]{
		keys:     stateSliceToUniqueKeys(s),
		callback: f,
	})
	return b
}

// WithOnEnter registers a callback when entering configuration s.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey]) WithOnEnter(s []State, f func(e Symbol)) *NFABuilder[State, Symbol, StateKey, SymbolKey] {
	b.enterCallbacks = append(b.enterCallbacks, nfaConfigCallback[State, StateKey, Symbol]{
		keys:     stateSliceToUniqueKeys(s),
		callback: f,
	})
	return b
}

// WithOnExitState registers a callback when a specific state exits the active set.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey]) WithOnExitState(state State, f func(e Symbol)) *NFABuilder[State, Symbol, StateKey, SymbolKey] {
	b.exitStateCallbacks = append(b.exitStateCallbacks, nfaStateCallback[StateKey, Symbol]{
		target:   state.Key(),
		callback: f,
	})
	return b
}

// WithOnEnterState registers a callback when a specific state enters the active set.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey]) WithOnEnterState(state State, f func(e Symbol)) *NFABuilder[State, Symbol, StateKey, SymbolKey] {
	b.enterStateCallbacks = append(b.enterStateCallbacks, nfaStateCallback[StateKey, Symbol]{
		target:   state.Key(),
		callback: f,
	})
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

	configKeyer := newNFAConfigKeyer[State]()

	var configGuard transitionhooks.GroupGuard[string]
	if !b.selfTransitionCallbacks {
		configGuard = func(fromKey string, toKey string) bool { return fromKey != toKey }
	}

	exitStateRegistrations := make([]transitionhooks.Registration[[]State, Symbol, string], 0, len(b.exitStateCallbacks))
	for _, callback := range b.exitStateCallbacks {
		target := callback.target
		exitStateCallback := callback.callback
		exitStateRegistrations = append(exitStateRegistrations, transitionhooks.Registration[[]State, Symbol, string]{
			Mode: transitionhooks.MatchPredicate,
			Predicate: func(from []State, to []State, e Symbol) bool {
				return stateSliceHasKey(from, target) && !stateSliceHasKey(to, target)
			},
			Callback: func(from []State, to []State, e Symbol) {
				exitStateCallback(e)
			},
		})
	}

	exitConfigRegistrations := make([]transitionhooks.Registration[[]State, Symbol, string], 0, len(b.exitCallbacks))
	for _, callback := range b.exitCallbacks {
		exitConfigCallback := callback.callback
		exitConfigRegistrations = append(exitConfigRegistrations, transitionhooks.Registration[[]State, Symbol, string]{
			Mode:    transitionhooks.MatchFromKey,
			FromKey: configKeyer.keyForStateKeys(callback.keys),
			Callback: func(from []State, to []State, e Symbol) {
				exitConfigCallback(e)
			},
		})
	}

	stepRegistrations := make([]transitionhooks.Registration[[]State, Symbol, string], 0, len(b.stepAnyCallbacks)+len(b.stepCallbacks))
	for _, callback := range b.stepAnyCallbacks {
		stepRegistrations = append(stepRegistrations, transitionhooks.Registration[[]State, Symbol, string]{
			Mode:     transitionhooks.MatchAny,
			Callback: callback,
		})
	}
	for _, callback := range b.stepCallbacks {
		stepCallback := callback.callback
		stepRegistrations = append(stepRegistrations, transitionhooks.Registration[[]State, Symbol, string]{
			Mode:    transitionhooks.MatchFromToKey,
			FromKey: configKeyer.keyForStateKeys(callback.fromKeys),
			ToKey:   configKeyer.keyForStateKeys(callback.toKeys),
			Callback: func(from []State, to []State, e Symbol) {
				stepCallback(e)
			},
		})
	}

	enterStateRegistrations := make([]transitionhooks.Registration[[]State, Symbol, string], 0, len(b.enterStateCallbacks))
	for _, callback := range b.enterStateCallbacks {
		target := callback.target
		enterStateCallback := callback.callback
		enterStateRegistrations = append(enterStateRegistrations, transitionhooks.Registration[[]State, Symbol, string]{
			Mode: transitionhooks.MatchPredicate,
			Predicate: func(from []State, to []State, e Symbol) bool {
				return !stateSliceHasKey(from, target) && stateSliceHasKey(to, target)
			},
			Callback: func(from []State, to []State, e Symbol) {
				enterStateCallback(e)
			},
		})
	}

	enterConfigRegistrations := make([]transitionhooks.Registration[[]State, Symbol, string], 0, len(b.enterCallbacks))
	for _, callback := range b.enterCallbacks {
		enterConfigCallback := callback.callback
		enterConfigRegistrations = append(enterConfigRegistrations, transitionhooks.Registration[[]State, Symbol, string]{
			Mode:  transitionhooks.MatchToKey,
			ToKey: configKeyer.keyForStateKeys(callback.keys),
			Callback: func(from []State, to []State, e Symbol) {
				enterConfigCallback(e)
			},
		})
	}

	return transitionhooks.NewCompiled(
		b.exec,
		func(states []State) string { return configKeyer.keyForStates(states) },
		transitionhooks.Group[[]State, Symbol, string]{
			Registrations: exitStateRegistrations,
		},
		transitionhooks.Group[[]State, Symbol, string]{
			Guard:         configGuard,
			Registrations: exitConfigRegistrations,
		},
		transitionhooks.Group[[]State, Symbol, string]{
			Registrations: stepRegistrations,
		},
		transitionhooks.Group[[]State, Symbol, string]{
			Registrations: enterStateRegistrations,
		},
		transitionhooks.Group[[]State, Symbol, string]{
			Guard:         configGuard,
			Registrations: enterConfigRegistrations,
		},
	), nil
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
	return engine.NFA(n).WithTransitionHooks(hooks).Build()
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
	return engine.NFA(n).WithTransitionHooks(hooks).BuildAtomic()
}

// BuildRunner wires the NFA and transition hooks into an Engine-backed Runner.
func (b *NFABuilder[State, Symbol, StateKey, SymbolKey]) BuildRunner(buffer int) (*runner.Runner[Symbol], error) {
	e, err := b.BuildEngine()
	if err != nil {
		return nil, err
	}
	return runner.New(e, buffer), nil
}

type nfaConfigKeyer[State key.Keyer[StateKey], StateKey comparable] struct {
	ids  map[StateKey]uint32
	next uint32
}

func newNFAConfigKeyer[State key.Keyer[StateKey], StateKey comparable]() *nfaConfigKeyer[State, StateKey] {
	return &nfaConfigKeyer[State, StateKey]{
		ids: make(map[StateKey]uint32),
	}
}

func (k *nfaConfigKeyer[State, StateKey]) keyForStates(states []State) string {
	keys := make([]StateKey, 0, len(states))
	for _, state := range states {
		keys = append(keys, state.Key())
	}
	return k.keyForStateKeys(keys)
}

func (k *nfaConfigKeyer[State, StateKey]) keyForStateKeys(keys []StateKey) string {
	if len(keys) == 0 {
		return ""
	}

	ids := make([]uint32, 0, len(keys))
	seen := make(map[uint32]struct{}, len(keys))
	for _, stateKey := range keys {
		id, ok := k.ids[stateKey]
		if !ok {
			id = k.next
			k.ids[stateKey] = id
			k.next++
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}

	sort.Slice(ids, func(i int, j int) bool { return ids[i] < ids[j] })
	encoded := make([]byte, 0, len(ids)*4)
	for _, id := range ids {
		encoded = append(encoded,
			byte(id>>24),
			byte(id>>16),
			byte(id>>8),
			byte(id),
		)
	}
	return string(encoded)
}

func stateSliceToUniqueKeys[State key.Keyer[StateKey], StateKey comparable](states []State) []StateKey {
	keys := make([]StateKey, 0, len(states))
	seen := make(map[StateKey]struct{}, len(states))
	for _, state := range states {
		stateKey := state.Key()
		if _, ok := seen[stateKey]; ok {
			continue
		}
		seen[stateKey] = struct{}{}
		keys = append(keys, stateKey)
	}
	return keys
}

func stateSliceHasKey[State key.Keyer[StateKey], StateKey comparable](states []State, target StateKey) bool {
	for _, state := range states {
		if state.Key() == target {
			return true
		}
	}
	return false
}
