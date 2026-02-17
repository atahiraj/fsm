package fsm

import (
	"errors"

	"github.com/stnhrsprkwns/fsm/engine"
	"github.com/stnhrsprkwns/fsm/internal/set"
	"github.com/stnhrsprkwns/fsm/nfa"
	"github.com/stnhrsprkwns/fsm/runner"
)

// NFAGraph is a minimal graph interface for NFA builders.
type NFAGraph[State comparable, Symbol comparable] interface {
	Delta(from State, sym Symbol) []State
	Epsilon(from State) []State
}

// NFA constructs a top-level NFA builder.
func NFA[State comparable, Symbol comparable, SP any, EP any]() *NFABuilder[State, Symbol, SP, EP] {
	b := &NFABuilder[State, Symbol, SP, EP]{
		nfa: nfa.NewBuilder[State, Symbol](),
	}
	b.obs = newNFAObserverBuilder[State, Symbol, SP, EP]().WithExecutor(DefaultExecutor[[]State, Symbol, SP, EP]{})
	return b
}

// NewNFABuilder constructs a low-level NFA builder.
func NewNFABuilder[State comparable, Symbol comparable, SP any, EP any]() *NFABuilder[State, Symbol, SP, EP] {
	return &NFABuilder[State, Symbol, SP, EP]{
		nfa: nfa.NewBuilder[State, Symbol](),
		obs: newNFAObserverBuilder[State, Symbol, SP, EP](),
	}
}

// NFABuilder wires NFA + Observer + Engine in one fluent flow.
type NFABuilder[State comparable, Symbol comparable, SP any, EP any] struct {
	nfa         *nfa.Builder[State, Symbol]
	obs         *nfaObserverBuilder[State, Symbol, SP, EP]
	nfaOverride *nfa.NFA[State, Symbol]
	obsOverride engine.Observer[[]State, Symbol, SP, EP]
}

type nfaGraphAdapter[State comparable, Symbol comparable] struct {
	g NFAGraph[State, Symbol]
}

func (a nfaGraphAdapter[State, Symbol]) Delta(from State, label nfa.Label[Symbol]) []State {
	if label.IsEpsilon {
		return a.g.Epsilon(from)
	}
	return a.g.Delta(from, label.Symbol)
}

// WithNFA overrides the NFA built by this builder.
func (b *NFABuilder[State, Symbol, SP, EP]) WithNFA(n *nfa.NFA[State, Symbol]) *NFABuilder[State, Symbol, SP, EP] {
	b.nfaOverride = n
	return b
}

// WithGraph replaces the graph used by the NFA builder.
func (b *NFABuilder[State, Symbol, SP, EP]) WithGraph(g NFAGraph[State, Symbol]) *NFABuilder[State, Symbol, SP, EP] {
	b.nfa.WithGraph(nfaGraphAdapter[State, Symbol]{g: g})
	return b
}

// WithStart sets q₀.
func (b *NFABuilder[State, Symbol, SP, EP]) WithStart(state State) *NFABuilder[State, Symbol, SP, EP] {
	b.nfa.SetStart(state)
	return b
}

// WithStates adds states to Q.
func (b *NFABuilder[State, Symbol, SP, EP]) WithStates(states ...State) *NFABuilder[State, Symbol, SP, EP] {
	b.nfa.AddStates(states...)
	return b
}

// WithAlphabet adds symbols to Σ.
func (b *NFABuilder[State, Symbol, SP, EP]) WithAlphabet(symbols ...Symbol) *NFABuilder[State, Symbol, SP, EP] {
	b.nfa.AddAlphabet(symbols...)
	return b
}

// WithAccepting adds states to F.
func (b *NFABuilder[State, Symbol, SP, EP]) WithAccepting(states ...State) *NFABuilder[State, Symbol, SP, EP] {
	b.nfa.AddAccepting(states...)
	return b
}

// WithTransition inserts (from, a, to) into δ.
func (b *NFABuilder[State, Symbol, SP, EP]) WithTransition(from State, symbol Symbol, to State) *NFABuilder[State, Symbol, SP, EP] {
	b.nfa.Transition(from, symbol, to)
	return b
}

// WithEpsilon inserts (from, ε, to) into δ.
func (b *NFABuilder[State, Symbol, SP, EP]) WithEpsilon(from State, to State) *NFABuilder[State, Symbol, SP, EP] {
	b.nfa.Epsilon(from, to)
	return b
}

// WithObserver overrides the observer used by the Engine build.
func (b *NFABuilder[State, Symbol, SP, EP]) WithObserver(obs engine.Observer[[]State, Symbol, SP, EP]) *NFABuilder[State, Symbol, SP, EP] {
	b.obsOverride = obs
	return b
}

// WithExecutor sets the observer executor.
func (b *NFABuilder[State, Symbol, SP, EP]) WithExecutor(exec Executor[[]State, Symbol, SP, EP]) *NFABuilder[State, Symbol, SP, EP] {
	b.obs.WithExecutor(exec)
	return b
}

// WithOnStepAny registers a callback for every step.
func (b *NFABuilder[State, Symbol, SP, EP]) WithOnStepAny(f func(from []State, sp SP, to []State, e Symbol, ep EP)) *NFABuilder[State, Symbol, SP, EP] {
	b.obs.OnStepAny(f)
	return b
}

// WithOnStep registers a callback for a specific transition from -> to.
func (b *NFABuilder[State, Symbol, SP, EP]) WithOnStep(from []State, to []State, f func(from []State, sp SP, to []State, e Symbol, ep EP)) *NFABuilder[State, Symbol, SP, EP] {
	b.obs.OnStep(from, to, f)
	return b
}

// WithOnExit registers a callback when leaving state s.
func (b *NFABuilder[State, Symbol, SP, EP]) WithOnExit(s []State, f func(from []State, sp SP, e Symbol, ep EP)) *NFABuilder[State, Symbol, SP, EP] {
	b.obs.OnExit(s, f)
	return b
}

// WithOnEnter registers a callback when entering state s.
func (b *NFABuilder[State, Symbol, SP, EP]) WithOnEnter(s []State, f func(to []State, sp SP, e Symbol, ep EP)) *NFABuilder[State, Symbol, SP, EP] {
	b.obs.OnEnter(s, f)
	return b
}

// WithOnExitAny registers a callback when leaving any state.
func (b *NFABuilder[State, Symbol, SP, EP]) WithOnExitAny(f func(from []State, sp SP, e Symbol, ep EP)) *NFABuilder[State, Symbol, SP, EP] {
	b.obs.OnExitAny(f)
	return b
}

// WithOnEnterAny registers a callback when entering any state.
func (b *NFABuilder[State, Symbol, SP, EP]) WithOnEnterAny(f func(to []State, sp SP, e Symbol, ep EP)) *NFABuilder[State, Symbol, SP, EP] {
	b.obs.OnEnterAny(f)
	return b
}

// BuildNFA returns the NFA built from the configured pieces.
func (b *NFABuilder[State, Symbol, SP, EP]) BuildNFA() (*nfa.NFA[State, Symbol], error) {
	if b.nfaOverride != nil {
		return b.nfaOverride, nil
	}
	return b.nfa.Build()
}

// BuildAtomicNFA returns a thread-safe NFA.
func (b *NFABuilder[State, Symbol, SP, EP]) BuildAtomicNFA() (*nfa.AtomicNFA[State, Symbol], error) {
	n, err := b.BuildNFA()
	if err != nil {
		return nil, err
	}
	return nfa.NewAtomic(n), nil
}

// BuildEngine wires the NFA and observer into an Engine.
func (b *NFABuilder[State, Symbol, SP, EP]) BuildEngine() (*engine.Engine[[]State, Symbol, SP, EP], error) {
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
func (b *NFABuilder[State, Symbol, SP, EP]) BuildAtomicEngine() (*engine.AtomicEngine[[]State, Symbol, SP, EP], error) {
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
func (b *NFABuilder[State, Symbol, SP, EP]) BuildRunner(buffer int) (*runner.Runner[Symbol, EP], error) {
	e, err := b.BuildEngine()
	if err != nil {
		return nil, err
	}
	return runner.New(e, buffer), nil
}

type nfaExitStateCallback[State comparable, Symbol comparable, SP any, EP any] struct {
	state []State
	f     func(from []State, sp SP, e Symbol, ep EP)
}

type nfaEnterStateCallback[State comparable, Symbol comparable, SP any, EP any] struct {
	state []State
	f     func(to []State, sp SP, e Symbol, ep EP)
}

type nfaTransitionCallback[State comparable, Symbol comparable, SP any, EP any] struct {
	from []State
	to   []State
	f    func(from []State, sp SP, to []State, e Symbol, ep EP)
}

type nfaObserverBuilder[State comparable, Symbol comparable, SP any, EP any] struct {
	exec       Executor[[]State, Symbol, SP, EP]
	onStepAny  []func(from []State, sp SP, to []State, e Symbol, ep EP)
	onStep     []nfaTransitionCallback[State, Symbol, SP, EP]
	onExitAny  []func(from []State, sp SP, e Symbol, ep EP)
	onEnterAny []func(to []State, sp SP, e Symbol, ep EP)
	onExit     []nfaExitStateCallback[State, Symbol, SP, EP]
	onEnter    []nfaEnterStateCallback[State, Symbol, SP, EP]
}

func newNFAObserverBuilder[State comparable, Symbol comparable, SP any, EP any]() *nfaObserverBuilder[State, Symbol, SP, EP] {
	return &nfaObserverBuilder[State, Symbol, SP, EP]{
		onStepAny:  []func(from []State, sp SP, to []State, e Symbol, ep EP){},
		onExitAny:  []func(from []State, sp SP, e Symbol, ep EP){},
		onEnterAny: []func(to []State, sp SP, e Symbol, ep EP){},
	}
}

func (b *nfaObserverBuilder[State, Symbol, SP, EP]) WithExecutor(exec Executor[[]State, Symbol, SP, EP]) *nfaObserverBuilder[State, Symbol, SP, EP] {
	b.exec = exec
	return b
}

func (b *nfaObserverBuilder[State, Symbol, SP, EP]) OnStepAny(f func(from []State, sp SP, to []State, e Symbol, ep EP)) *nfaObserverBuilder[State, Symbol, SP, EP] {
	b.onStepAny = append(b.onStepAny, f)
	return b
}

func (b *nfaObserverBuilder[State, Symbol, SP, EP]) OnStep(from []State, to []State, f func(from []State, sp SP, to []State, e Symbol, ep EP)) *nfaObserverBuilder[State, Symbol, SP, EP] {
	b.onStep = append(b.onStep, nfaTransitionCallback[State, Symbol, SP, EP]{
		from: cloneStates(from),
		to:   cloneStates(to),
		f:    f,
	})
	return b
}

func (b *nfaObserverBuilder[State, Symbol, SP, EP]) OnExit(s []State, f func(from []State, sp SP, e Symbol, ep EP)) *nfaObserverBuilder[State, Symbol, SP, EP] {
	b.onExit = append(b.onExit, nfaExitStateCallback[State, Symbol, SP, EP]{state: cloneStates(s), f: f})
	return b
}

func (b *nfaObserverBuilder[State, Symbol, SP, EP]) OnEnter(s []State, f func(to []State, sp SP, e Symbol, ep EP)) *nfaObserverBuilder[State, Symbol, SP, EP] {
	b.onEnter = append(b.onEnter, nfaEnterStateCallback[State, Symbol, SP, EP]{state: cloneStates(s), f: f})
	return b
}

func (b *nfaObserverBuilder[State, Symbol, SP, EP]) OnExitAny(f func(from []State, sp SP, e Symbol, ep EP)) *nfaObserverBuilder[State, Symbol, SP, EP] {
	b.onExitAny = append(b.onExitAny, f)
	return b
}

func (b *nfaObserverBuilder[State, Symbol, SP, EP]) OnEnterAny(f func(to []State, sp SP, e Symbol, ep EP)) *nfaObserverBuilder[State, Symbol, SP, EP] {
	b.onEnterAny = append(b.onEnterAny, f)
	return b
}

func (b *nfaObserverBuilder[State, Symbol, SP, EP]) Build() (engine.Observer[[]State, Symbol, SP, EP], error) {
	if b.exec == nil {
		return nil, errors.New("observer builder: executor is nil")
	}
	return &nfaObserver[State, Symbol, SP, EP]{
		exec:       b.exec,
		onStepAny:  b.onStepAny,
		onStep:     b.onStep,
		onExitAny:  b.onExitAny,
		onEnterAny: b.onEnterAny,
		onExit:     b.onExit,
		onEnter:    b.onEnter,
	}, nil
}

type nfaObserver[State comparable, Symbol comparable, SP any, EP any] struct {
	exec       Executor[[]State, Symbol, SP, EP]
	onStepAny  []func(from []State, sp SP, to []State, e Symbol, ep EP)
	onStep     []nfaTransitionCallback[State, Symbol, SP, EP]
	onExitAny  []func(from []State, sp SP, e Symbol, ep EP)
	onEnterAny []func(to []State, sp SP, e Symbol, ep EP)
	onExit     []nfaExitStateCallback[State, Symbol, SP, EP]
	onEnter    []nfaEnterStateCallback[State, Symbol, SP, EP]
}

func (o *nfaObserver[State, Symbol, SP, EP]) OnStep(from []State, sp SP, to []State, e Symbol, ep EP) {
	for _, f := range o.onStepAny {
		o.exec.Execute(f, from, sp, to, e, ep)
	}
	for _, item := range o.onStep {
		if stateSetEqual(item.from, from) && stateSetEqual(item.to, to) {
			o.exec.Execute(item.f, from, sp, to, e, ep)
		}
	}
	for _, f := range o.onExitAny {
		fn := f
		o.exec.Execute(
			func(from []State, sp SP, _ []State, e Symbol, ep EP) { fn(from, sp, e, ep) },
			from, sp, to, e, ep,
		)
	}
	for _, item := range o.onExit {
		if stateSetEqual(item.state, from) {
			fn := item.f
			o.exec.Execute(
				func(from []State, sp SP, _ []State, e Symbol, ep EP) { fn(from, sp, e, ep) },
				from, sp, to, e, ep,
			)
		}
	}
	for _, f := range o.onEnterAny {
		fn := f
		o.exec.Execute(
			func(_ []State, sp SP, to []State, e Symbol, ep EP) { fn(to, sp, e, ep) },
			from, sp, to, e, ep,
		)
	}
	for _, item := range o.onEnter {
		if stateSetEqual(item.state, to) {
			fn := item.f
			o.exec.Execute(
				func(_ []State, sp SP, to []State, e Symbol, ep EP) { fn(to, sp, e, ep) },
				from, sp, to, e, ep,
			)
		}
	}
}

func cloneStates[State comparable](states []State) []State {
	if len(states) == 0 {
		return nil
	}
	out := make([]State, len(states))
	copy(out, states)
	return out
}

func stateSetEqual[State comparable](a, b []State) bool {
	return set.New(a...).Equals(set.New(b...))
}
