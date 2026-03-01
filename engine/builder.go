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
func DFA[State any, Input any](d dfaLike[State, Input]) *Builder[State, Input] {
	return NewBuilder[State, Input](dfaFSM[State, Input]{d: d})
}

// NFA constructs a Builder backed by an NFA-like machine.
func NFA[State any, Input any](n nfaLike[State, Input]) *Builder[[]State, Input] {
	return NewBuilder[[]State, Input](nfaFSM[State, Input]{n: n})
}

type dfaLike[State any, Input any] interface {
	Start() State
	Delta(state State, input Input) (State, bool)
	IsAccepting(state State) bool
}

type nfaLike[State any, Input any] interface {
	StartSet() []State
	EpsilonClosureSet(states []State) []State
	DeltaSet(states []State, a Input) []State
	IsAccepting(states []State) bool
}

type dfaFSM[State any, Input any] struct {
	d dfaLike[State, Input]
}

func (f dfaFSM[State, Input]) Start() State {
	return f.d.Start()
}

func (f dfaFSM[State, Input]) Step(s State, a Input) (State, bool) {
	next, ok := f.d.Delta(s, a)
	if !ok {
		return s, false
	}
	return next, true
}

func (f dfaFSM[State, Input]) IsAccepting(s State) bool {
	return f.d.IsAccepting(s)
}

type nfaFSM[State any, Input any] struct {
	n nfaLike[State, Input]
}

func (f nfaFSM[State, Input]) Start() []State {
	return f.n.EpsilonClosureSet(f.n.StartSet())
}

func (f nfaFSM[State, Input]) Step(conf []State, a Input) ([]State, bool) {
	cur := f.n.EpsilonClosureSet(conf)
	next := f.n.DeltaSet(cur, a)
	return f.n.EpsilonClosureSet(next), len(next) > 0
}

func (f nfaFSM[State, Input]) IsAccepting(conf []State) bool {
	return f.n.IsAccepting(conf)
}
