package observer

import (
	"errors"
)

// Builder constructs an Observer by collecting callbacks.
type Builder[S comparable, E, SP, EP any] struct {
	cfg Config[S, E, SP, EP]
}

// New creates an empty Builder.
// Preconditions: an executor is provided before Build.
func New[S comparable, E, SP, EP any]() *Builder[S, E, SP, EP] {
	return &Builder[S, E, SP, EP]{
		cfg: Config[S, E, SP, EP]{
			OnStepAny:  []func(from S, sp SP, to S, e E, ep EP){},
			OnStep:     map[transition[S]][]func(from S, sp SP, to S, e E, ep EP){},
			OnExitAny:  []func(from S, sp SP, e E, ep EP){},
			OnEnterAny: []func(to S, sp SP, e E, ep EP){},
			OnExit:     map[S][]func(from S, sp SP, e E, ep EP){},
			OnEnter:    map[S][]func(to S, sp SP, e E, ep EP){},
		},
	}
}

// NewBuilder creates an empty Builder.
func NewBuilder[S comparable, E, SP, EP any]() *Builder[S, E, SP, EP] {
	return New[S, E, SP, EP]()
}

// WithExecutor replaces the executor.
func (b *Builder[S, E, SP, EP]) WithExecutor(executor Executor[S, E, SP, EP]) *Builder[S, E, SP, EP] {
	b.cfg.Executor = executor
	return b
}

// OnStepAny registers a callback for every step.
func (b *Builder[S, E, SP, EP]) OnStepAny(f func(from S, sp SP, to S, e E, ep EP)) *Builder[S, E, SP, EP] {
	b.cfg.OnStepAny = append(b.cfg.OnStepAny, f)
	return b
}

// OnStep registers a callback for a specific transition from -> to.
func (b *Builder[S, E, SP, EP]) OnStep(from S, to S, f func(from S, sp SP, to S, e E, ep EP)) *Builder[S, E, SP, EP] {
	key := transition[S]{from: from, to: to}
	b.cfg.OnStep[key] = append(b.cfg.OnStep[key], f)
	return b
}

// OnExit registers a callback when leaving state s.
func (b *Builder[S, E, SP, EP]) OnExit(s S, f func(from S, sp SP, e E, ep EP)) *Builder[S, E, SP, EP] {
	b.cfg.OnExit[s] = append(b.cfg.OnExit[s], f)
	return b
}

// OnEnter registers a callback when entering state s.
func (b *Builder[S, E, SP, EP]) OnEnter(s S, f func(to S, sp SP, e E, ep EP)) *Builder[S, E, SP, EP] {
	b.cfg.OnEnter[s] = append(b.cfg.OnEnter[s], f)
	return b
}

// OnExitAny registers a callback when leaving any state.
func (b *Builder[S, E, SP, EP]) OnExitAny(f func(from S, sp SP, e E, ep EP)) *Builder[S, E, SP, EP] {
	b.cfg.OnExitAny = append(b.cfg.OnExitAny, f)
	return b
}

// OnEnterAny registers a callback when entering any state.
func (b *Builder[S, E, SP, EP]) OnEnterAny(f func(to S, sp SP, e E, ep EP)) *Builder[S, E, SP, EP] {
	b.cfg.OnEnterAny = append(b.cfg.OnEnterAny, f)
	return b
}

// Build constructs the Observer.
// Returns an error if the executor is missing.
func (b *Builder[S, E, SP, EP]) Build() (*Observer[S, E, SP, EP], error) {
	if b.cfg.Executor == nil {
		return nil, errors.New("observer builder: executor is nil")
	}
	return NewObserver(b.cfg), nil
}
