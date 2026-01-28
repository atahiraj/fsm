package builder

import (
	"errors"

	"github.com/stnhrsprkwns/fsm/pkg/engine"
	engatomic "github.com/stnhrsprkwns/fsm/pkg/engine/atomic"
	"github.com/stnhrsprkwns/fsm/pkg/set"
)

// Builder wires an FSM and Observer into an Engine.
type Builder[S any, E any, SP any, EP any] struct {
	fsm engine.FSM[S, E]
	obs engine.Observer[S, E, SP, EP]
}

// New constructs a Builder for the given FSM.
func New[S any, E any, SP any, EP any](fsm engine.FSM[S, E]) *Builder[S, E, SP, EP] {
	return &Builder[S, E, SP, EP]{fsm: fsm}
}

// NewBuilder constructs a Builder for the given FSM.
func NewBuilder[S any, E any, SP any, EP any](fsm engine.FSM[S, E]) *Builder[S, E, SP, EP] {
	return New[S, E, SP, EP](fsm)
}

// WithObserver sets the observer.
func (b *Builder[S, E, SP, EP]) WithObserver(obs engine.Observer[S, E, SP, EP]) *Builder[S, E, SP, EP] {
	b.obs = obs
	return b
}

// Build constructs the Engine.
// Returns an error if fsm or observer is missing.
func (b *Builder[S, E, SP, EP]) Build() (*engine.Engine[S, E, SP, EP], error) {
	if b.fsm == nil {
		return nil, errors.New("engine builder: fsm is nil")
	}
	if b.obs == nil {
		return nil, errors.New("engine builder: observer is nil")
	}
	return engine.New(b.fsm, b.obs), nil
}

// BuildAtomic constructs a thread-safe Engine.
func (b *Builder[S, E, SP, EP]) BuildAtomic() (*engatomic.Engine[S, E, SP, EP], error) {
	e, err := b.Build()
	if err != nil {
		return nil, err
	}
	return engatomic.New(e), nil
}

// DFA constructs a Builder backed by a DFA-like machine.
func DFA[State comparable, Symbol comparable, SP any, EP any](d dfaLike[State, Symbol]) *Builder[State, Symbol, SP, EP] {
	return New[State, Symbol, SP, EP](dfaFSM[State, Symbol]{d: d})
}

// NFA constructs a Builder backed by an NFA-like machine.
func NFA[State comparable, Symbol comparable, SP any, EP any](n nfaLike[State, Symbol]) *Builder[*set.Set[State], Symbol, SP, EP] {
	return New[*set.Set[State], Symbol, SP, EP](nfaFSM[State, Symbol]{n: n})
}

type dfaLike[State comparable, Symbol comparable] interface {
	Start() State
	Delta(state State, symbol Symbol) (State, bool)
	IsAccepting(state State) bool
}

type nfaLike[State comparable, Symbol comparable] interface {
	StartSet() *set.Set[State]
	EpsilonClosureSet(Q *set.Set[State]) *set.Set[State]
	DeltaSet(Q *set.Set[State], a Symbol) *set.Set[State]
	IsAccepting(s *set.Set[State]) bool
}

type dfaFSM[State comparable, Symbol comparable] struct {
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

type nfaFSM[State comparable, Symbol comparable] struct {
	n nfaLike[State, Symbol]
}

func (f nfaFSM[State, Symbol]) Start() *set.Set[State] {
	return f.n.EpsilonClosureSet(f.n.StartSet())
}

func (f nfaFSM[State, Symbol]) Step(conf *set.Set[State], a Symbol) *set.Set[State] {
	cur := f.n.EpsilonClosureSet(conf)
	cur = f.n.DeltaSet(cur, a)
	return f.n.EpsilonClosureSet(cur)
}

func (f nfaFSM[State, Symbol]) IsAccepting(conf *set.Set[State]) bool {
	return f.n.IsAccepting(conf)
}
