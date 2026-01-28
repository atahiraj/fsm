package builder

import (
	"github.com/stnhrsprkwns/fsm/pkg/engine"
	engatomic "github.com/stnhrsprkwns/fsm/pkg/engine/atomic"
	engbuilder "github.com/stnhrsprkwns/fsm/pkg/engine/builder"

	"github.com/stnhrsprkwns/fsm/pkg/dfa"
	dfaatomic "github.com/stnhrsprkwns/fsm/pkg/dfa/atomic"
	dfabuilder "github.com/stnhrsprkwns/fsm/pkg/dfa/builder"

	"github.com/stnhrsprkwns/fsm/pkg/nfa"
	nfaatomic "github.com/stnhrsprkwns/fsm/pkg/nfa/atomic"
	nfabuilder "github.com/stnhrsprkwns/fsm/pkg/nfa/builder"

	"github.com/stnhrsprkwns/fsm/pkg/observer"
	obsbuilder "github.com/stnhrsprkwns/fsm/pkg/observer/builder"

	"github.com/stnhrsprkwns/fsm/pkg/set"
)

// DFAGraph is a minimal graph interface for DFA builders.
type DFAGraph[State comparable, Symbol comparable] interface {
	Delta(from State, label Symbol) []State
}

// NFAGraph is a minimal graph interface for NFA builders.
type NFAGraph[State comparable, Symbol comparable] interface {
	Delta(from State, sym Symbol) []State
	Epsilon(from State) []State
}

// DefaultExecutor calls the callback directly.
type DefaultExecutor[S comparable, E, SP, EP any] = observer.DefaultExecutor[S, E, SP, EP]

// DFA constructs a top-level DFA builder.
func DFA[State comparable, Symbol comparable, SP any, EP any]() *DFABuilder[State, Symbol, SP, EP] {
	b := &DFABuilder[State, Symbol, SP, EP]{
		dfa: dfabuilder.New[State, Symbol](),
	}
	b.obs = obsbuilder.New[State, Symbol, SP, EP]().WithExecutor(DefaultExecutor[State, Symbol, SP, EP]{})
	return b
}

// NFA constructs a top-level NFA builder.
func NFA[State comparable, Symbol comparable, SP any, EP any]() *NFABuilder[State, Symbol, SP, EP] {
	b := &NFABuilder[State, Symbol, SP, EP]{
		nfa: nfabuilder.New[State, Symbol](),
	}
	b.obs = obsbuilder.New[*set.Set[State], Symbol, SP, EP]().WithExecutor(DefaultExecutor[*set.Set[State], Symbol, SP, EP]{})
	return b
}

// DFABuilder wires DFA + Observer + Engine in one fluent flow.
type DFABuilder[State comparable, Symbol comparable, SP any, EP any] struct {
	dfa         *dfabuilder.Builder[State, Symbol]
	obs         *obsbuilder.Builder[State, Symbol, SP, EP]
	dfaOverride *dfa.DFA[State, Symbol]
	obsOverride engine.Observer[State, Symbol, SP, EP]
	err         error
}

// WithDFA overrides the DFA built by this builder.
func (b *DFABuilder[State, Symbol, SP, EP]) WithDFA(d *dfa.DFA[State, Symbol]) *DFABuilder[State, Symbol, SP, EP] {
	b.dfaOverride = d
	return b
}

// WithGraph replaces the graph used by the DFA builder.
func (b *DFABuilder[State, Symbol, SP, EP]) WithGraph(g DFAGraph[State, Symbol]) *DFABuilder[State, Symbol, SP, EP] {
	b.dfa.WithGraph(g)
	return b
}

// WithStart sets q₀.
func (b *DFABuilder[State, Symbol, SP, EP]) WithStart(state State) *DFABuilder[State, Symbol, SP, EP] {
	b.dfa.SetStart(state)
	return b
}

// WithStates adds states to Q.
func (b *DFABuilder[State, Symbol, SP, EP]) WithStates(states ...State) *DFABuilder[State, Symbol, SP, EP] {
	b.dfa.AddStates(states...)
	return b
}

// WithAlphabet adds symbols to Σ.
func (b *DFABuilder[State, Symbol, SP, EP]) WithAlphabet(symbols ...Symbol) *DFABuilder[State, Symbol, SP, EP] {
	b.dfa.AddAlphabet(symbols...)
	return b
}

// WithAccepting adds states to F.
func (b *DFABuilder[State, Symbol, SP, EP]) WithAccepting(states ...State) *DFABuilder[State, Symbol, SP, EP] {
	b.dfa.AddAccepting(states...)
	return b
}

// WithTransition inserts (from, a, to) into δ.
// Preconditions: transition does not introduce nondeterminism.
func (b *DFABuilder[State, Symbol, SP, EP]) WithTransition(from State, symbol Symbol, to State) *DFABuilder[State, Symbol, SP, EP] {
	if b.err != nil {
		return b
	}
	if err := b.dfa.Transition(from, symbol, to); err != nil {
		b.err = err
	}
	return b
}

// WithObserver overrides the observer used by the Engine build.
func (b *DFABuilder[State, Symbol, SP, EP]) WithObserver(obs engine.Observer[State, Symbol, SP, EP]) *DFABuilder[State, Symbol, SP, EP] {
	b.obsOverride = obs
	return b
}

// WithExecutor sets the observer executor.
func (b *DFABuilder[State, Symbol, SP, EP]) WithExecutor(exec observer.Executor[State, Symbol, SP, EP]) *DFABuilder[State, Symbol, SP, EP] {
	b.obs.WithExecutor(exec)
	return b
}

// WithOnExit registers a callback when leaving state s.
func (b *DFABuilder[State, Symbol, SP, EP]) WithOnExit(s State, f func(from State, sp SP, to State, e Symbol, ep EP)) *DFABuilder[State, Symbol, SP, EP] {
	b.obs.OnExit(s, f)
	return b
}

// WithOnEnter registers a callback when entering state s.
func (b *DFABuilder[State, Symbol, SP, EP]) WithOnEnter(s State, f func(from State, sp SP, to State, e Symbol, ep EP)) *DFABuilder[State, Symbol, SP, EP] {
	b.obs.OnEnter(s, f)
	return b
}

// WithOnExitAny registers a callback when leaving any state.
func (b *DFABuilder[State, Symbol, SP, EP]) WithOnExitAny(f func(from State, sp SP, to State, e Symbol, ep EP)) *DFABuilder[State, Symbol, SP, EP] {
	b.obs.OnExitAny(f)
	return b
}

// WithOnEnterAny registers a callback when entering any state.
func (b *DFABuilder[State, Symbol, SP, EP]) WithOnEnterAny(f func(from State, sp SP, to State, e Symbol, ep EP)) *DFABuilder[State, Symbol, SP, EP] {
	b.obs.OnEnterAny(f)
	return b
}

// BuildDFA returns the DFA built from the configured pieces.
func (b *DFABuilder[State, Symbol, SP, EP]) BuildDFA() (*dfa.DFA[State, Symbol], error) {
	if b.err != nil {
		return nil, b.err
	}
	if b.dfaOverride != nil {
		return b.dfaOverride, nil
	}
	return b.dfa.Build()
}

// BuildAtomicDFA returns a thread-safe DFA.
func (b *DFABuilder[State, Symbol, SP, EP]) BuildAtomicDFA() (*dfaatomic.DFA[State, Symbol], error) {
	d, err := b.BuildDFA()
	if err != nil {
		return nil, err
	}
	return dfaatomic.New(d), nil
}

// BuildEngine wires the DFA and observer into an Engine.
func (b *DFABuilder[State, Symbol, SP, EP]) BuildEngine() (*engine.Engine[State, Symbol, SP, EP], error) {
	obs := b.obsOverride
	if obs == nil {
		var err error
		obs, err = b.obs.Build()
		if err != nil {
			return nil, err
		}
	}
	d, err := b.BuildDFA()
	if err != nil {
		return nil, err
	}
	return engbuilder.DFA[State, Symbol, SP, EP](d).WithObserver(obs).Build()
}

// BuildAtomicEngine wires the DFA and observer into a thread-safe Engine.
func (b *DFABuilder[State, Symbol, SP, EP]) BuildAtomicEngine() (*engatomic.Engine[State, Symbol, SP, EP], error) {
	obs := b.obsOverride
	if obs == nil {
		var err error
		obs, err = b.obs.Build()
		if err != nil {
			return nil, err
		}
	}
	d, err := b.BuildDFA()
	if err != nil {
		return nil, err
	}
	return engbuilder.DFA[State, Symbol, SP, EP](d).WithObserver(obs).BuildAtomic()
}

// NFABuilder wires NFA + Observer + Engine in one fluent flow.
type NFABuilder[State comparable, Symbol comparable, SP any, EP any] struct {
	nfa         *nfabuilder.Builder[State, Symbol]
	obs         *obsbuilder.Builder[*set.Set[State], Symbol, SP, EP]
	nfaOverride *nfa.NFA[State, Symbol]
	obsOverride engine.Observer[*set.Set[State], Symbol, SP, EP]
}

type nfaGraphAdapter[State comparable, Symbol comparable] struct {
	g NFAGraph[State, Symbol]
}

func (a nfaGraphAdapter[State, Symbol]) Delta(from State, label nfabuilder.Label[Symbol]) []State {
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
func (b *NFABuilder[State, Symbol, SP, EP]) WithObserver(obs engine.Observer[*set.Set[State], Symbol, SP, EP]) *NFABuilder[State, Symbol, SP, EP] {
	b.obsOverride = obs
	return b
}

// WithExecutor sets the observer executor.
func (b *NFABuilder[State, Symbol, SP, EP]) WithExecutor(exec observer.Executor[*set.Set[State], Symbol, SP, EP]) *NFABuilder[State, Symbol, SP, EP] {
	b.obs.WithExecutor(exec)
	return b
}

// WithOnExit registers a callback when leaving state s.
func (b *NFABuilder[State, Symbol, SP, EP]) WithOnExit(s *set.Set[State], f func(from *set.Set[State], sp SP, to *set.Set[State], e Symbol, ep EP)) *NFABuilder[State, Symbol, SP, EP] {
	b.obs.OnExit(s, f)
	return b
}

// WithOnEnter registers a callback when entering state s.
func (b *NFABuilder[State, Symbol, SP, EP]) WithOnEnter(s *set.Set[State], f func(from *set.Set[State], sp SP, to *set.Set[State], e Symbol, ep EP)) *NFABuilder[State, Symbol, SP, EP] {
	b.obs.OnEnter(s, f)
	return b
}

// WithOnExitAny registers a callback when leaving any state.
func (b *NFABuilder[State, Symbol, SP, EP]) WithOnExitAny(f func(from *set.Set[State], sp SP, to *set.Set[State], e Symbol, ep EP)) *NFABuilder[State, Symbol, SP, EP] {
	b.obs.OnExitAny(f)
	return b
}

// WithOnEnterAny registers a callback when entering any state.
func (b *NFABuilder[State, Symbol, SP, EP]) WithOnEnterAny(f func(from *set.Set[State], sp SP, to *set.Set[State], e Symbol, ep EP)) *NFABuilder[State, Symbol, SP, EP] {
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
func (b *NFABuilder[State, Symbol, SP, EP]) BuildAtomicNFA() (*nfaatomic.NFA[State, Symbol], error) {
	n, err := b.BuildNFA()
	if err != nil {
		return nil, err
	}
	return nfaatomic.New(n), nil
}

// BuildEngine wires the NFA and observer into an Engine.
func (b *NFABuilder[State, Symbol, SP, EP]) BuildEngine() (*engine.Engine[*set.Set[State], Symbol, SP, EP], error) {
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
	return engbuilder.NFA[State, Symbol, SP, EP](n).WithObserver(obs).Build()
}

// BuildAtomicEngine wires the NFA and observer into a thread-safe Engine.
func (b *NFABuilder[State, Symbol, SP, EP]) BuildAtomicEngine() (*engatomic.Engine[*set.Set[State], Symbol, SP, EP], error) {
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
	return engbuilder.NFA[State, Symbol, SP, EP](n).WithObserver(obs).BuildAtomic()
}
