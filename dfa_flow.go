package fsm

import (
	"github.com/stnhrsprkwns/fsm/dfa"
	"github.com/stnhrsprkwns/fsm/engine"
	"github.com/stnhrsprkwns/fsm/observer"
)

// DFAGraph is a minimal graph interface for DFA builders.
type DFAGraph[State comparable, Symbol comparable] interface {
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
func DFA[State comparable, Symbol comparable, SP any, EP any]() *DFABuilder[State, Symbol, SP, EP] {
	b := &DFABuilder[State, Symbol, SP, EP]{
		dfa: dfa.NewBuilder[State, Symbol](),
	}
	b.obs = observer.NewBuilder[State, Symbol, SP, EP]().WithExecutor(DefaultExecutor[State, Symbol, SP, EP]{})
	return b
}

// NewDFABuilder constructs a low-level DFA builder.
func NewDFABuilder[State comparable, Symbol comparable, SP, EP any]() *DFABuilder[State, Symbol, SP, EP] {
	return &DFABuilder[State, Symbol, SP, EP]{
		dfa: dfa.NewBuilder[State, Symbol](),
		obs: observer.NewBuilder[State, Symbol, SP, EP](),
	}
}

// DFABuilder wires DFA + Observer + Engine in one fluent flow.
type DFABuilder[State comparable, Symbol comparable, SP any, EP any] struct {
	dfa         *dfa.Builder[State, Symbol]
	obs         *observer.Builder[State, Symbol, SP, EP]
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
func (b *DFABuilder[State, Symbol, SP, EP]) WithExecutor(exec Executor[State, Symbol, SP, EP]) *DFABuilder[State, Symbol, SP, EP] {
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
func (b *DFABuilder[State, Symbol, SP, EP]) BuildAtomicDFA() (*dfa.AtomicDFA[State, Symbol], error) {
	d, err := b.BuildDFA()
	if err != nil {
		return nil, err
	}
	return dfa.NewAtomic(d), nil
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
	return engine.DFA[State, Symbol, SP, EP](d).WithObserver(obs).Build()
}

// BuildAtomicEngine wires the DFA and observer into a thread-safe Engine.
func (b *DFABuilder[State, Symbol, SP, EP]) BuildAtomicEngine() (*engine.AtomicEngine[State, Symbol, SP, EP], error) {
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
	return engine.DFA[State, Symbol, SP, EP](d).WithObserver(obs).BuildAtomic()
}
