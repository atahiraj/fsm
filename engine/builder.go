package engine

import "errors"

// Builder wires an FSM and Observer into an Engine.
type Builder[S any, I any, SP any, IP any] struct {
	fsm FSM[S, I]
	obs Observer[S, I, SP, IP]
}

// NewBuilder constructs a Builder for the given FSM.
func NewBuilder[S any, I any, SP any, IP any](fsm FSM[S, I]) *Builder[S, I, SP, IP] {
	return &Builder[S, I, SP, IP]{fsm: fsm}
}

// WithObserver sets the observer.
func (b *Builder[S, I, SP, IP]) WithObserver(obs Observer[S, I, SP, IP]) *Builder[S, I, SP, IP] {
	b.obs = obs
	return b
}

// Build constructs the Engine.
// Returns an error if fsm or observer is missing.
func (b *Builder[S, I, SP, IP]) Build() (*Engine[S, I, SP, IP], error) {
	if b.fsm == nil {
		return nil, errors.New("engine builder: fsm is nil")
	}
	if b.obs == nil {
		return nil, errors.New("engine builder: observer is nil")
	}
	return New(b.fsm, b.obs), nil
}

// BuildAtomic constructs a thread-safe Engine.
func (b *Builder[S, I, SP, IP]) BuildAtomic() (*AtomicEngine[S, I, SP, IP], error) {
	e, err := b.Build()
	if err != nil {
		return nil, err
	}
	return NewAtomic(e), nil
}

// DFA constructs a Builder backed by a DFA-like machine.
func DFA[State any, Symbol any, SP any, IP any](d dfaLike[State, Symbol]) *Builder[State, Symbol, SP, IP] {
	return NewBuilder[State, Symbol, SP, IP](dfaFSM[State, Symbol]{d: d})
}

// NFA constructs a Builder backed by an NFA-like machine.
func NFA[State any, Symbol any, Config any, SP any, IP any](n nfaLike[State, Symbol, Config]) *Builder[Config, Symbol, SP, IP] {
	return NewBuilder[Config, Symbol, SP, IP](nfaFSM[State, Symbol, Config]{n: n})
}

type dfaLike[State any, Symbol any] interface {
	Start() State
	Delta(state State, symbol Symbol) (State, bool)
	IsAccepting(state State) bool
}

type nfaLike[State any, Symbol any, Config any] interface {
	StartSet() Config
	EpsilonClosureSet(states Config) Config
	DeltaSet(states Config, a Symbol) Config
	IsAccepting(states Config) bool
}

type dfaFSM[State any, Symbol any] struct {
	d dfaLike[State, Symbol]
}

func (f dfaFSM[State, Symbol]) Start() State {
	return f.d.Start()
}

func (f dfaFSM[State, Symbol]) Step(s State, a Symbol) State {
	next, ok := f.d.Delta(s, a)
	if !ok {
		return s
	}
	return next
}

func (f dfaFSM[State, Symbol]) IsAccepting(s State) bool {
	return f.d.IsAccepting(s)
}

type nfaFSM[State any, Symbol any, Config any] struct {
	n nfaLike[State, Symbol, Config]
}

func (f nfaFSM[State, Symbol, Config]) Start() Config {
	return f.n.EpsilonClosureSet(f.n.StartSet())
}

func (f nfaFSM[State, Symbol, Config]) Step(conf Config, a Symbol) Config {
	cur := f.n.EpsilonClosureSet(conf)
	cur = f.n.DeltaSet(cur, a)
	return f.n.EpsilonClosureSet(cur)
}

func (f nfaFSM[State, Symbol, Config]) IsAccepting(conf Config) bool {
	return f.n.IsAccepting(conf)
}
