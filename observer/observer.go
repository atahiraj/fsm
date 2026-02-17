package observer

// Executor controls how observer callbacks are executed.
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

// Equal reports whether two values are equivalent.
type Equal[T any] func(a, b T) bool

// Projection extracts member values from a state/configuration.
// It enables state-delta callbacks such as "state entered"/"state exited"
// for compound configurations (for example, NFA active-state sets).
type Projection[S any, M any] struct {
	Members func(state S) []M
	Equal   Equal[M]
}

type transitionCallback[S any, E any, SP any, EP any] struct {
	from S
	to   S
	f    func(from S, sp SP, to S, e E, ep EP)
}

type stateCallback[S any, E any, SP any, EP any] struct {
	state S
	f     func(state S, sp SP, e E, ep EP)
}

type memberCallback[M any, E any, SP any, EP any] struct {
	member M
	f      func(member M, sp SP, e E, ep EP)
}

// Config defines observer callbacks and matching behavior.
type Config[S any, E any, SP any, EP any, M any] struct {
	Executor   Executor[S, E, SP, EP]
	EqualState Equal[S]
	Projection *Projection[S, M]

	OnStepAny  []func(from S, sp SP, to S, e E, ep EP)
	OnStep     []transitionCallback[S, E, SP, EP]
	OnExitAny  []func(from S, sp SP, e E, ep EP)
	OnEnterAny []func(to S, sp SP, e E, ep EP)
	OnExit     []stateCallback[S, E, SP, EP]
	OnEnter    []stateCallback[S, E, SP, EP]

	OnExitMemberAny  []func(member M, sp SP, e E, ep EP)
	OnEnterMemberAny []func(member M, sp SP, e E, ep EP)
	OnExitMember     []memberCallback[M, E, SP, EP]
	OnEnterMember    []memberCallback[M, E, SP, EP]
}

// Observer dispatches step, enter, and exit hooks.
type Observer[S any, E any, SP any, EP any, M any] struct {
	executor   Executor[S, E, SP, EP]
	equalState Equal[S]
	projection *Projection[S, M]

	onStepAny  []func(from S, sp SP, to S, e E, ep EP)
	onStep     []transitionCallback[S, E, SP, EP]
	onExitAny  []func(from S, sp SP, e E, ep EP)
	onEnterAny []func(to S, sp SP, e E, ep EP)
	onExit     []stateCallback[S, E, SP, EP]
	onEnter    []stateCallback[S, E, SP, EP]

	onExitMemberAny  []func(member M, sp SP, e E, ep EP)
	onEnterMemberAny []func(member M, sp SP, e E, ep EP)
	onExitMember     []memberCallback[M, E, SP, EP]
	onEnterMember    []memberCallback[M, E, SP, EP]
}

// NewObserver constructs an Observer.
func NewObserver[S any, E any, SP any, EP any, M any](cfg Config[S, E, SP, EP, M]) *Observer[S, E, SP, EP, M] {
	return &Observer[S, E, SP, EP, M]{
		executor:         cfg.Executor,
		equalState:       cfg.EqualState,
		projection:       cfg.Projection,
		onStepAny:        cfg.OnStepAny,
		onStep:           cfg.OnStep,
		onExitAny:        cfg.OnExitAny,
		onEnterAny:       cfg.OnEnterAny,
		onExit:           cfg.OnExit,
		onEnter:          cfg.OnEnter,
		onExitMemberAny:  cfg.OnExitMemberAny,
		onEnterMemberAny: cfg.OnEnterMemberAny,
		onExitMember:     cfg.OnExitMember,
		onEnterMember:    cfg.OnEnterMember,
	}
}

// OnExit registers a callback when leaving state s.
func (o *Observer[S, E, SP, EP, M]) OnExit(s S, f func(state S, sp SP, e E, ep EP)) {
	o.onExit = append(o.onExit, stateCallback[S, E, SP, EP]{state: s, f: f})
}

// OnEnter registers a callback when entering state s.
func (o *Observer[S, E, SP, EP, M]) OnEnter(s S, f func(state S, sp SP, e E, ep EP)) {
	o.onEnter = append(o.onEnter, stateCallback[S, E, SP, EP]{state: s, f: f})
}

func (o *Observer[S, E, SP, EP, M]) runExit(f func(from S, sp SP, e E, ep EP), from S, sp SP, to S, e E, ep EP) {
	o.executor.Execute(
		func(from S, sp SP, _ S, e E, ep EP) { f(from, sp, e, ep) },
		from, sp, to, e, ep,
	)
}

func (o *Observer[S, E, SP, EP, M]) runEnter(f func(to S, sp SP, e E, ep EP), from S, sp SP, to S, e E, ep EP) {
	o.executor.Execute(
		func(_ S, sp SP, to S, e E, ep EP) { f(to, sp, e, ep) },
		from, sp, to, e, ep,
	)
}

func containsByEqual[T any](xs []T, x T, equal Equal[T]) bool {
	for _, item := range xs {
		if equal(item, x) {
			return true
		}
	}
	return false
}

func differenceOrdered[T any](left []T, right []T, equal Equal[T]) []T {
	if len(left) == 0 {
		return nil
	}
	out := make([]T, 0, len(left))
	for _, item := range left {
		if containsByEqual(right, item, equal) || containsByEqual(out, item, equal) {
			continue
		}
		out = append(out, item)
	}
	return out
}

// OnStep dispatches step, exit, and enter hooks.
func (o *Observer[S, E, SP, EP, M]) OnStep(from S, sp SP, to S, e E, ep EP) {
	for _, f := range o.onStepAny {
		o.executor.Execute(f, from, sp, to, e, ep)
	}
	for _, item := range o.onStep {
		if o.equalState(item.from, from) && o.equalState(item.to, to) {
			o.executor.Execute(item.f, from, sp, to, e, ep)
		}
	}
	for _, f := range o.onExitAny {
		o.runExit(f, from, sp, to, e, ep)
	}
	for _, item := range o.onExit {
		if o.equalState(item.state, from) {
			fn := item.f
			o.executor.Execute(
				func(from S, sp SP, _ S, e E, ep EP) { fn(from, sp, e, ep) },
				from, sp, to, e, ep,
			)
		}
	}
	for _, f := range o.onEnterAny {
		o.runEnter(f, from, sp, to, e, ep)
	}
	for _, item := range o.onEnter {
		if o.equalState(item.state, to) {
			fn := item.f
			o.executor.Execute(
				func(_ S, sp SP, to S, e E, ep EP) { fn(to, sp, e, ep) },
				from, sp, to, e, ep,
			)
		}
	}

	if o.projection == nil {
		return
	}

	exited := differenceOrdered(o.projection.Members(from), o.projection.Members(to), o.projection.Equal)
	entered := differenceOrdered(o.projection.Members(to), o.projection.Members(from), o.projection.Equal)

	for _, member := range exited {
		for _, f := range o.onExitMemberAny {
			m := member
			fn := f
			o.executor.Execute(
				func(_ S, sp SP, _ S, e E, ep EP) { fn(m, sp, e, ep) },
				from, sp, to, e, ep,
			)
		}
		for _, item := range o.onExitMember {
			if o.projection.Equal(item.member, member) {
				m := member
				fn := item.f
				o.executor.Execute(
					func(_ S, sp SP, _ S, e E, ep EP) { fn(m, sp, e, ep) },
					from, sp, to, e, ep,
				)
			}
		}
	}

	for _, member := range entered {
		for _, f := range o.onEnterMemberAny {
			m := member
			fn := f
			o.executor.Execute(
				func(_ S, sp SP, _ S, e E, ep EP) { fn(m, sp, e, ep) },
				from, sp, to, e, ep,
			)
		}
		for _, item := range o.onEnterMember {
			if o.projection.Equal(item.member, member) {
				m := member
				fn := item.f
				o.executor.Execute(
					func(_ S, sp SP, _ S, e E, ep EP) { fn(m, sp, e, ep) },
					from, sp, to, e, ep,
				)
			}
		}
	}
}
