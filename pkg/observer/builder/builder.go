package builder

import (
	"errors"

	"github.com/stnhrsprkwns/fsm/pkg/observer"
)

// Builder constructs an Observer by collecting callbacks.
type Builder[S comparable, E, SP, EP any] struct {
	cfg observer.Config[S, E, SP, EP]
}

// New creates an empty Builder.
// Preconditions: an executor is provided before Build.
func New[S comparable, E, SP, EP any]() *Builder[S, E, SP, EP] {
	return &Builder[S, E, SP, EP]{
		cfg: observer.Config[S, E, SP, EP]{
			OnExitAny:  []func(from S, sp SP, to S, e E, ep EP){},
			OnEnterAny: []func(from S, sp SP, to S, e E, ep EP){},
			OnExit:     map[S][]func(from S, sp SP, to S, e E, ep EP){},
			OnEnter:    map[S][]func(from S, sp SP, to S, e E, ep EP){},
		},
	}
}

// WithExecutor replaces the executor.
func (b *Builder[S, E, SP, EP]) WithExecutor(executor observer.Executor[S, E, SP, EP]) *Builder[S, E, SP, EP] {
	b.cfg.Executor = executor
	return b
}

// OnExit registers a callback when leaving state s.
func (b *Builder[S, E, SP, EP]) OnExit(s S, f func(from S, sp SP, to S, e E, ep EP)) *Builder[S, E, SP, EP] {
	b.cfg.OnExit[s] = append(b.cfg.OnExit[s], f)
	return b
}

// OnEnter registers a callback when entering state s.
func (b *Builder[S, E, SP, EP]) OnEnter(s S, f func(from S, sp SP, to S, e E, ep EP)) *Builder[S, E, SP, EP] {
	b.cfg.OnEnter[s] = append(b.cfg.OnEnter[s], f)
	return b
}

// OnExitAny registers a callback when leaving any state.
func (b *Builder[S, E, SP, EP]) OnExitAny(f func(from S, sp SP, to S, e E, ep EP)) *Builder[S, E, SP, EP] {
	b.cfg.OnExitAny = append(b.cfg.OnExitAny, f)
	return b
}

// OnEnterAny registers a callback when entering any state.
func (b *Builder[S, E, SP, EP]) OnEnterAny(f func(from S, sp SP, to S, e E, ep EP)) *Builder[S, E, SP, EP] {
	b.cfg.OnEnterAny = append(b.cfg.OnEnterAny, f)
	return b
}

// Build constructs the Observer.
// Returns an error if the executor is missing.
func (b *Builder[S, E, SP, EP]) Build() (*observer.Observer[S, E, SP, EP], error) {
	if b.cfg.Executor == nil {
		return nil, errors.New("observer builder: executor is nil")
	}
	return observer.NewObserver(b.cfg), nil
}
