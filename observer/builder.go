package observer

import "errors"

// Builder constructs an Observer by collecting callbacks.
type Builder[S any, E any, SP any, EP any, M any] struct {
	cfg Config[S, E, SP, EP, M]
}

// New creates an empty Builder.
// Preconditions: an executor is provided before Build.
func New[S any, E any, SP any, EP any, M any]() *Builder[S, E, SP, EP, M] {
	return &Builder[S, E, SP, EP, M]{
		cfg: Config[S, E, SP, EP, M]{
			OnStepAny:        []func(from S, sp SP, to S, e E, ep EP){},
			OnStep:           []transitionCallback[S, E, SP, EP]{},
			OnExitAny:        []func(from S, sp SP, e E, ep EP){},
			OnEnterAny:       []func(to S, sp SP, e E, ep EP){},
			OnExit:           []stateCallback[S, E, SP, EP]{},
			OnEnter:          []stateCallback[S, E, SP, EP]{},
			OnExitMemberAny:  []func(member M, sp SP, e E, ep EP){},
			OnEnterMemberAny: []func(member M, sp SP, e E, ep EP){},
			OnExitMember:     []memberCallback[M, E, SP, EP]{},
			OnEnterMember:    []memberCallback[M, E, SP, EP]{},
		},
	}
}

// NewBuilder creates an empty Builder.
func NewBuilder[S any, E any, SP any, EP any, M any]() *Builder[S, E, SP, EP, M] {
	return New[S, E, SP, EP, M]()
}

// WithExecutor replaces the executor.
func (b *Builder[S, E, SP, EP, M]) WithExecutor(executor Executor[S, E, SP, EP]) *Builder[S, E, SP, EP, M] {
	b.cfg.Executor = executor
	return b
}

// WithEqualState sets state/configuration equality.
func (b *Builder[S, E, SP, EP, M]) WithEqualState(equal Equal[S]) *Builder[S, E, SP, EP, M] {
	b.cfg.EqualState = equal
	return b
}

// WithProjection sets member projection and member equality.
func (b *Builder[S, E, SP, EP, M]) WithProjection(p Projection[S, M]) *Builder[S, E, SP, EP, M] {
	b.cfg.Projection = &p
	return b
}

// OnStepAny registers a callback for every step.
func (b *Builder[S, E, SP, EP, M]) OnStepAny(f func(from S, sp SP, to S, e E, ep EP)) *Builder[S, E, SP, EP, M] {
	b.cfg.OnStepAny = append(b.cfg.OnStepAny, f)
	return b
}

// OnStep registers a callback for a specific transition from -> to.
func (b *Builder[S, E, SP, EP, M]) OnStep(from S, to S, f func(from S, sp SP, to S, e E, ep EP)) *Builder[S, E, SP, EP, M] {
	b.cfg.OnStep = append(b.cfg.OnStep, transitionCallback[S, E, SP, EP]{from: from, to: to, f: f})
	return b
}

// OnExit registers a callback when leaving state s.
func (b *Builder[S, E, SP, EP, M]) OnExit(s S, f func(from S, sp SP, e E, ep EP)) *Builder[S, E, SP, EP, M] {
	b.cfg.OnExit = append(b.cfg.OnExit, stateCallback[S, E, SP, EP]{
		state: s,
		f: func(state S, sp SP, e E, ep EP) {
			f(state, sp, e, ep)
		},
	})
	return b
}

// OnEnter registers a callback when entering state s.
func (b *Builder[S, E, SP, EP, M]) OnEnter(s S, f func(to S, sp SP, e E, ep EP)) *Builder[S, E, SP, EP, M] {
	b.cfg.OnEnter = append(b.cfg.OnEnter, stateCallback[S, E, SP, EP]{
		state: s,
		f: func(state S, sp SP, e E, ep EP) {
			f(state, sp, e, ep)
		},
	})
	return b
}

// OnExitAny registers a callback when leaving any state.
func (b *Builder[S, E, SP, EP, M]) OnExitAny(f func(from S, sp SP, e E, ep EP)) *Builder[S, E, SP, EP, M] {
	b.cfg.OnExitAny = append(b.cfg.OnExitAny, f)
	return b
}

// OnEnterAny registers a callback when entering any state.
func (b *Builder[S, E, SP, EP, M]) OnEnterAny(f func(to S, sp SP, e E, ep EP)) *Builder[S, E, SP, EP, M] {
	b.cfg.OnEnterAny = append(b.cfg.OnEnterAny, f)
	return b
}

// OnExitMemberAny registers a callback when any member exits.
func (b *Builder[S, E, SP, EP, M]) OnExitMemberAny(f func(member M, sp SP, e E, ep EP)) *Builder[S, E, SP, EP, M] {
	b.cfg.OnExitMemberAny = append(b.cfg.OnExitMemberAny, f)
	return b
}

// OnEnterMemberAny registers a callback when any member enters.
func (b *Builder[S, E, SP, EP, M]) OnEnterMemberAny(f func(member M, sp SP, e E, ep EP)) *Builder[S, E, SP, EP, M] {
	b.cfg.OnEnterMemberAny = append(b.cfg.OnEnterMemberAny, f)
	return b
}

// OnExitMember registers a callback when a specific member exits.
func (b *Builder[S, E, SP, EP, M]) OnExitMember(member M, f func(sp SP, e E, ep EP)) *Builder[S, E, SP, EP, M] {
	b.cfg.OnExitMember = append(b.cfg.OnExitMember, memberCallback[M, E, SP, EP]{
		member: member,
		f: func(_ M, sp SP, e E, ep EP) {
			f(sp, e, ep)
		},
	})
	return b
}

// OnEnterMember registers a callback when a specific member enters.
func (b *Builder[S, E, SP, EP, M]) OnEnterMember(member M, f func(sp SP, e E, ep EP)) *Builder[S, E, SP, EP, M] {
	b.cfg.OnEnterMember = append(b.cfg.OnEnterMember, memberCallback[M, E, SP, EP]{
		member: member,
		f: func(_ M, sp SP, e E, ep EP) {
			f(sp, e, ep)
		},
	})
	return b
}

// Build constructs the Observer.
// Returns an error if required matching config is missing.
func (b *Builder[S, E, SP, EP, M]) Build() (*Observer[S, E, SP, EP, M], error) {
	if b.cfg.Executor == nil {
		return nil, errors.New("observer builder: executor is nil")
	}
	if b.cfg.EqualState == nil {
		return nil, errors.New("observer builder: state equality is nil")
	}
	needsProjection := len(b.cfg.OnExitMemberAny) != 0 || len(b.cfg.OnEnterMemberAny) != 0 || len(b.cfg.OnExitMember) != 0 || len(b.cfg.OnEnterMember) != 0
	if needsProjection {
		if b.cfg.Projection == nil {
			return nil, errors.New("observer builder: projection is nil")
		}
		if b.cfg.Projection.Members == nil || b.cfg.Projection.Equal == nil {
			return nil, errors.New("observer builder: projection is incomplete")
		}
	}
	return NewObserver(b.cfg), nil
}
