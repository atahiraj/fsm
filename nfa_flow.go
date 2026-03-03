package fsm

import (
	"errors"
	"sort"

	"github.com/atahiraj/fsm/engine"
	"github.com/atahiraj/fsm/key"
	"github.com/atahiraj/fsm/nfa"
	"github.com/atahiraj/fsm/runner"
	"github.com/atahiraj/fsm/transitionhooks"
)

// NFAGraph is a minimal graph interface for NFA builders.
type NFAGraph[State any, Input any] interface {
	Delta(from State, sym Input) []State
	Epsilon(from State) []State
}

type nfaStepCallback[State key.Keyer[StateKey], StateKey comparable, Input any] struct {
	fromKeys []StateKey
	toKeys   []StateKey
	callback func(e Input)
}

type nfaConfigCallback[State key.Keyer[StateKey], StateKey comparable, Input any] struct {
	keys     []StateKey
	callback func(e Input)
}

type nfaStateCallback[StateKey comparable, Input any] struct {
	target   StateKey
	callback func(e Input)
}

// NFA constructs a top-level NFA builder.
func NFA[
	State key.Keyer[StateKey],
	Input key.Keyer[InputKey],
	StateKey comparable,
	InputKey comparable,
]() *NFABuilder[State, Input, StateKey, InputKey] {
	return &NFABuilder[State, Input, StateKey, InputKey]{
		nfa:  nfa.NewBuilder[State, Input](),
		exec: DefaultExecutor[[]State, Input]{},
	}
}

// NewNFABuilder constructs a low-level NFA builder.
func NewNFABuilder[
	State key.Keyer[StateKey],
	Input key.Keyer[InputKey],
	StateKey comparable,
	InputKey comparable,
]() *NFABuilder[State, Input, StateKey, InputKey] {
	return &NFABuilder[State, Input, StateKey, InputKey]{
		nfa: nfa.NewBuilder[State, Input](),
	}
}

// NFABuilder wires NFA + TransitionHooks + Engine in one fluent flow.
type NFABuilder[
	State key.Keyer[StateKey],
	Input key.Keyer[InputKey],
	StateKey comparable,
	InputKey comparable,
] struct {
	nfa *nfa.Builder[State, Input, StateKey, InputKey]

	exec Executor[[]State, Input]

	selfTransitionCallbacks bool

	// Dispatch order is fixed by phase:
	// exit-state -> exit -> step-any -> step -> enter-state -> enter
	exitStateCallbacks  []nfaStateCallback[StateKey, Input]
	exitCallbacks       []nfaConfigCallback[State, StateKey, Input]
	stepAnyCallbacks    []func(from []State, to []State, e Input)
	stepCallbacks       []nfaStepCallback[State, StateKey, Input]
	enterStateCallbacks []nfaStateCallback[StateKey, Input]
	enterCallbacks      []nfaConfigCallback[State, StateKey, Input]

	nfaOverride   *nfa.NFA[State, Input, StateKey, InputKey]
	hooksOverride engine.TransitionHooks[[]State, Input]
}

// WithNFA overrides the NFA built by this builder.
func (b *NFABuilder[State, Input, StateKey, InputKey]) WithNFA(n *nfa.NFA[State, Input, StateKey, InputKey]) *NFABuilder[State, Input, StateKey, InputKey] {
	b.nfaOverride = n
	return b
}

// WithGraph replaces the graph used by the NFA builder.
//
// The graph only provides the transition relation (δ, ε). You must still
// provide the universe (Q, Σ), typically with WithStates and WithAlphabet.
func (b *NFABuilder[State, Input, StateKey, InputKey]) WithGraph(g NFAGraph[State, Input]) *NFABuilder[State, Input, StateKey, InputKey] {
	b.nfa.WithGraph(g)
	return b
}

// WithStart sets q₀.
func (b *NFABuilder[State, Input, StateKey, InputKey]) WithStart(state State) *NFABuilder[State, Input, StateKey, InputKey] {
	b.nfa.SetStart(state)
	return b
}

// WithStates adds states to Q.
func (b *NFABuilder[State, Input, StateKey, InputKey]) WithStates(states ...State) *NFABuilder[State, Input, StateKey, InputKey] {
	b.nfa.AddStates(states...)
	return b
}

// WithAlphabet adds inputs to Σ.
func (b *NFABuilder[State, Input, StateKey, InputKey]) WithAlphabet(inputs ...Input) *NFABuilder[State, Input, StateKey, InputKey] {
	b.nfa.AddAlphabet(inputs...)
	return b
}

// WithAccepting adds states to F.
func (b *NFABuilder[State, Input, StateKey, InputKey]) WithAccepting(states ...State) *NFABuilder[State, Input, StateKey, InputKey] {
	b.nfa.AddAccepting(states...)
	return b
}

// WithTransition inserts (from, a, to) into δ using an input key.
func (b *NFABuilder[State, Input, StateKey, InputKey]) WithTransition(from State, inputKey InputKey, to State) *NFABuilder[State, Input, StateKey, InputKey] {
	b.nfa.TransitionKey(from, inputKey, to)
	return b
}

// WithEpsilon inserts (from, ε, to) into δ.
func (b *NFABuilder[State, Input, StateKey, InputKey]) WithEpsilon(from State, to State) *NFABuilder[State, Input, StateKey, InputKey] {
	b.nfa.Epsilon(from, to)
	return b
}

// WithTransitionHooks overrides the transition hooks used by the Engine build.
func (b *NFABuilder[State, Input, StateKey, InputKey]) WithTransitionHooks(hooks engine.TransitionHooks[[]State, Input]) *NFABuilder[State, Input, StateKey, InputKey] {
	b.hooksOverride = hooks
	return b
}

// WithExecutor sets the transition hooks executor.
func (b *NFABuilder[State, Input, StateKey, InputKey]) WithExecutor(exec Executor[[]State, Input]) *NFABuilder[State, Input, StateKey, InputKey] {
	b.exec = exec
	return b
}

// WithSelfTransitionCallbacks enables config-level enter/exit callbacks when the configuration is unchanged.
func (b *NFABuilder[State, Input, StateKey, InputKey]) WithSelfTransitionCallbacks() *NFABuilder[State, Input, StateKey, InputKey] {
	b.selfTransitionCallbacks = true
	return b
}

// WithOnTransitionAny registers a callback for every transition.
func (b *NFABuilder[State, Input, StateKey, InputKey]) WithOnTransitionAny(f func(from []State, to []State, e Input)) *NFABuilder[State, Input, StateKey, InputKey] {
	b.stepAnyCallbacks = append(b.stepAnyCallbacks, f)
	return b
}

// WithOnTransition registers a callback for a specific transition from -> to.
func (b *NFABuilder[State, Input, StateKey, InputKey]) WithOnTransition(from []State, to []State, f func(e Input)) *NFABuilder[State, Input, StateKey, InputKey] {
	b.stepCallbacks = append(b.stepCallbacks, nfaStepCallback[State, StateKey, Input]{
		fromKeys: stateSliceToUniqueKeys(from),
		toKeys:   stateSliceToUniqueKeys(to),
		callback: f,
	})
	return b
}

// WithOnStepAny registers a callback for every transition.
// Deprecated: use WithOnTransitionAny.
func (b *NFABuilder[State, Input, StateKey, InputKey]) WithOnStepAny(f func(from []State, to []State, e Input)) *NFABuilder[State, Input, StateKey, InputKey] {
	return b.WithOnTransitionAny(f)
}

// WithOnStep registers a callback for a specific transition from -> to.
// Deprecated: use WithOnTransition.
func (b *NFABuilder[State, Input, StateKey, InputKey]) WithOnStep(from []State, to []State, f func(e Input)) *NFABuilder[State, Input, StateKey, InputKey] {
	return b.WithOnTransition(from, to, f)
}

// WithOnExit registers a callback when leaving configuration s.
func (b *NFABuilder[State, Input, StateKey, InputKey]) WithOnExit(s []State, f func(e Input)) *NFABuilder[State, Input, StateKey, InputKey] {
	b.exitCallbacks = append(b.exitCallbacks, nfaConfigCallback[State, StateKey, Input]{
		keys:     stateSliceToUniqueKeys(s),
		callback: f,
	})
	return b
}

// WithOnEnter registers a callback when entering configuration s.
func (b *NFABuilder[State, Input, StateKey, InputKey]) WithOnEnter(s []State, f func(e Input)) *NFABuilder[State, Input, StateKey, InputKey] {
	b.enterCallbacks = append(b.enterCallbacks, nfaConfigCallback[State, StateKey, Input]{
		keys:     stateSliceToUniqueKeys(s),
		callback: f,
	})
	return b
}

// WithOnExitState registers a callback when a specific state exits the active set.
func (b *NFABuilder[State, Input, StateKey, InputKey]) WithOnExitState(state State, f func(e Input)) *NFABuilder[State, Input, StateKey, InputKey] {
	b.exitStateCallbacks = append(b.exitStateCallbacks, nfaStateCallback[StateKey, Input]{
		target:   state.Key(),
		callback: f,
	})
	return b
}

// WithOnEnterState registers a callback when a specific state enters the active set.
func (b *NFABuilder[State, Input, StateKey, InputKey]) WithOnEnterState(state State, f func(e Input)) *NFABuilder[State, Input, StateKey, InputKey] {
	b.enterStateCallbacks = append(b.enterStateCallbacks, nfaStateCallback[StateKey, Input]{
		target:   state.Key(),
		callback: f,
	})
	return b
}

// BuildNFA returns the NFA built from the configured pieces.
func (b *NFABuilder[State, Input, StateKey, InputKey]) BuildNFA() (*nfa.NFA[State, Input, StateKey, InputKey], error) {
	if b.nfaOverride != nil {
		return b.nfaOverride, nil
	}
	return b.nfa.Build()
}

// BuildAtomicNFA returns a thread-safe NFA.
func (b *NFABuilder[State, Input, StateKey, InputKey]) BuildAtomicNFA() (*nfa.AtomicNFA[State, Input, StateKey, InputKey], error) {
	n, err := b.BuildNFA()
	if err != nil {
		return nil, err
	}
	return nfa.NewAtomic(n), nil
}

func (b *NFABuilder[State, Input, StateKey, InputKey]) buildTransitionHooks() (engine.TransitionHooks[[]State, Input], error) {
	if b.exec == nil {
		return nil, errors.New("transition hooks builder: executor is nil")
	}

	configKeyer := newNFAConfigKeyer[State]()

	var configGuard transitionhooks.GroupGuard[string]
	if !b.selfTransitionCallbacks {
		configGuard = func(fromKey string, toKey string) bool { return fromKey != toKey }
	}

	exitStateRegistrations := make([]transitionhooks.Registration[[]State, Input, string], 0, len(b.exitStateCallbacks))
	for _, callback := range b.exitStateCallbacks {
		target := callback.target
		exitStateCallback := callback.callback
		exitStateRegistrations = append(exitStateRegistrations, transitionhooks.Registration[[]State, Input, string]{
			Mode: transitionhooks.MatchPredicate,
			Predicate: func(from []State, to []State, e Input) bool {
				return stateSliceHasKey(from, target) && !stateSliceHasKey(to, target)
			},
			Callback: func(from []State, to []State, e Input) {
				exitStateCallback(e)
			},
		})
	}

	exitConfigRegistrations := make([]transitionhooks.Registration[[]State, Input, string], 0, len(b.exitCallbacks))
	for _, callback := range b.exitCallbacks {
		exitConfigCallback := callback.callback
		exitConfigRegistrations = append(exitConfigRegistrations, transitionhooks.Registration[[]State, Input, string]{
			Mode:    transitionhooks.MatchFromKey,
			FromKey: configKeyer.keyForStateKeys(callback.keys),
			Callback: func(from []State, to []State, e Input) {
				exitConfigCallback(e)
			},
		})
	}

	stepRegistrations := make([]transitionhooks.Registration[[]State, Input, string], 0, len(b.stepAnyCallbacks)+len(b.stepCallbacks))
	for _, callback := range b.stepAnyCallbacks {
		stepRegistrations = append(stepRegistrations, transitionhooks.Registration[[]State, Input, string]{
			Mode:     transitionhooks.MatchAny,
			Callback: callback,
		})
	}
	for _, callback := range b.stepCallbacks {
		stepCallback := callback.callback
		stepRegistrations = append(stepRegistrations, transitionhooks.Registration[[]State, Input, string]{
			Mode:    transitionhooks.MatchFromToKey,
			FromKey: configKeyer.keyForStateKeys(callback.fromKeys),
			ToKey:   configKeyer.keyForStateKeys(callback.toKeys),
			Callback: func(from []State, to []State, e Input) {
				stepCallback(e)
			},
		})
	}

	enterStateRegistrations := make([]transitionhooks.Registration[[]State, Input, string], 0, len(b.enterStateCallbacks))
	for _, callback := range b.enterStateCallbacks {
		target := callback.target
		enterStateCallback := callback.callback
		enterStateRegistrations = append(enterStateRegistrations, transitionhooks.Registration[[]State, Input, string]{
			Mode: transitionhooks.MatchPredicate,
			Predicate: func(from []State, to []State, e Input) bool {
				return !stateSliceHasKey(from, target) && stateSliceHasKey(to, target)
			},
			Callback: func(from []State, to []State, e Input) {
				enterStateCallback(e)
			},
		})
	}

	enterConfigRegistrations := make([]transitionhooks.Registration[[]State, Input, string], 0, len(b.enterCallbacks))
	for _, callback := range b.enterCallbacks {
		enterConfigCallback := callback.callback
		enterConfigRegistrations = append(enterConfigRegistrations, transitionhooks.Registration[[]State, Input, string]{
			Mode:  transitionhooks.MatchToKey,
			ToKey: configKeyer.keyForStateKeys(callback.keys),
			Callback: func(from []State, to []State, e Input) {
				enterConfigCallback(e)
			},
		})
	}

	return transitionhooks.NewCompiled(
		b.exec,
		func(states []State) string { return configKeyer.keyForStates(states) },
		transitionhooks.Group[[]State, Input, string]{
			Registrations: exitStateRegistrations,
		},
		transitionhooks.Group[[]State, Input, string]{
			Guard:         configGuard,
			Registrations: exitConfigRegistrations,
		},
		transitionhooks.Group[[]State, Input, string]{
			Registrations: stepRegistrations,
		},
		transitionhooks.Group[[]State, Input, string]{
			Registrations: enterStateRegistrations,
		},
		transitionhooks.Group[[]State, Input, string]{
			Guard:         configGuard,
			Registrations: enterConfigRegistrations,
		},
	), nil
}

// BuildEngine wires the NFA and transition hooks into an Engine.
func (b *NFABuilder[State, Input, StateKey, InputKey]) BuildEngine() (*engine.Engine[[]State, Input], error) {
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
func (b *NFABuilder[State, Input, StateKey, InputKey]) BuildAtomicEngine() (*engine.AtomicEngine[[]State, Input], error) {
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
func (b *NFABuilder[State, Input, StateKey, InputKey]) BuildRunner(buffer int) (*runner.Runner[Input], error) {
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
