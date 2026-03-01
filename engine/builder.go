package engine

import "errors"

// Builder wires an FSM and TransitionHooks into an Engine.
type Builder[S any, I any] struct {
	fsm   FSM[S, I]
	hooks TransitionHooks[S, I]
}

// NewBuilder constructs a Builder for the given FSM.
func NewBuilder[S any, I any](fsm FSM[S, I]) *Builder[S, I] {
	return &Builder[S, I]{fsm: fsm}
}

// WithTransitionHooks sets transition hooks.
func (b *Builder[S, I]) WithTransitionHooks(hooks TransitionHooks[S, I]) *Builder[S, I] {
	b.hooks = hooks
	return b
}

// Build constructs the Engine.
// Returns an error if fsm or transition hooks is missing.
func (b *Builder[S, I]) Build() (*Engine[S, I], error) {
	if b.fsm == nil {
		return nil, errors.New("engine builder: fsm is nil")
	}
	if b.hooks == nil {
		return nil, errors.New("engine builder: transition hooks are nil")
	}
	return New(b.fsm, b.hooks), nil
}

// BuildAtomic constructs a thread-safe Engine.
func (b *Builder[S, I]) BuildAtomic() (*AtomicEngine[S, I], error) {
	e, err := b.Build()
	if err != nil {
		return nil, err
	}
	return NewAtomic(e), nil
}

// DFA constructs a Builder backed by a DFA-like machine.
func DFA[State any, Symbol any](d dfaLike[State, Symbol]) *Builder[State, Symbol] {
	return NewBuilder[State, Symbol](dfaFSM[State, Symbol]{d: d})
}

// NFA constructs a Builder backed by an NFA-like machine.
func NFA[State any, Symbol any](n nfaLike[State, Symbol]) *Builder[[]State, Symbol] {
	return NewBuilder[[]State, Symbol](nfaFSM[State, Symbol]{n: n})
}

type dfaLike[State any, Symbol any] interface {
	Start() State
	Delta(state State, symbol Symbol) (State, bool)
	IsAccepting(state State) bool
}

type nfaLike[State any, Symbol any] interface {
	StartSet() []State
	EpsilonClosureSet(states []State) []State
	DeltaSet(states []State, a Symbol) []State
	IsAccepting(states []State) bool
}

type dfaFSM[State any, Symbol any] struct {
	d dfaLike[State, Symbol]
}

func (f dfaFSM[State, Symbol]) Start() State {
	return f.d.Start()
}

func (f dfaFSM[State, Symbol]) Step(s State, a Symbol) (State, bool) {
	next, ok := f.d.Delta(s, a)
	if !ok {
		return s, false
	}
	return next, true
}

func (f dfaFSM[State, Symbol]) IsAccepting(s State) bool {
	return f.d.IsAccepting(s)
}

type nfaFSM[State any, Symbol any] struct {
	n nfaLike[State, Symbol]
}

func (f nfaFSM[State, Symbol]) Start() []State {
	return f.n.EpsilonClosureSet(f.n.StartSet())
}

func (f nfaFSM[State, Symbol]) Step(conf []State, a Symbol) ([]State, bool) {
	cur := f.n.EpsilonClosureSet(conf)
	next := f.n.DeltaSet(cur, a)
	return f.n.EpsilonClosureSet(next), len(next) > 0
}

func (f nfaFSM[State, Symbol]) IsAccepting(conf []State) bool {
	return f.n.IsAccepting(conf)
}
