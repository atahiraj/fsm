package observer

// Executor controls how observer callbacks are executed.
type Executor[S comparable, E, SP, EP any] interface {
	Execute(f func(from S, sp SP, to S, e E, ep EP), from S, sp SP, to S, e E, ep EP)
}

// DefaultExecutor calls the callback directly.
type DefaultExecutor[S comparable, E, SP, EP any] struct{}

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

type transition[S comparable] struct {
	from S
	to   S
}

// Config defines the Observer callbacks and executor.
// Preconditions: Executor, OnStepAny, OnStep, OnExitAny, OnEnterAny, OnExit, and OnEnter are non-nil.
type Config[S comparable, E, SP, EP any] struct {
	Executor   Executor[S, E, SP, EP]
	OnStepAny  []func(from S, sp SP, to S, e E, ep EP)
	OnStep     map[transition[S]][]func(from S, sp SP, to S, e E, ep EP)
	OnExitAny  []func(from S, sp SP, e E, ep EP)
	OnEnterAny []func(to S, sp SP, e E, ep EP)
	OnExit     map[S][]func(from S, sp SP, e E, ep EP)
	OnEnter    map[S][]func(to S, sp SP, e E, ep EP)
}

// Observer dispatches per-state enter/exit hooks.
type Observer[S comparable, E, SP, EP any] struct {
	onStepAny  []func(from S, sp SP, to S, e E, ep EP)
	onStep     map[transition[S]][]func(from S, sp SP, to S, e E, ep EP)
	onExitAny  []func(from S, sp SP, e E, ep EP)
	onEnterAny []func(to S, sp SP, e E, ep EP)
	onExit     map[S][]func(from S, sp SP, e E, ep EP)
	onEnter    map[S][]func(to S, sp SP, e E, ep EP)
	executor   Executor[S, E, SP, EP]
}

// NewObserver constructs an Observer.
// Preconditions: cfg.Executor, cfg.OnStepAny, cfg.OnStep, cfg.OnExitAny, cfg.OnEnterAny, cfg.OnExit, and cfg.OnEnter are non-nil.
func NewObserver[S comparable, E, SP, EP any](cfg Config[S, E, SP, EP]) *Observer[S, E, SP, EP] {
	return &Observer[S, E, SP, EP]{
		onStepAny:  cfg.OnStepAny,
		onStep:     cfg.OnStep,
		onExitAny:  cfg.OnExitAny,
		onEnterAny: cfg.OnEnterAny,
		executor:   cfg.Executor,
		onExit:     cfg.OnExit,
		onEnter:    cfg.OnEnter,
	}
}

// OnExit registers a callback when leaving state s.
func (o *Observer[S, E, SP, EP]) OnExit(s S, f func(from S, sp SP, e E, ep EP)) {
	o.onExit[s] = append(o.onExit[s], f)
}

// OnEnter registers a callback when entering state s.
func (o *Observer[S, E, SP, EP]) OnEnter(s S, f func(to S, sp SP, e E, ep EP)) {
	o.onEnter[s] = append(o.onEnter[s], f)
}

func (o *Observer[S, E, SP, EP]) runExit(f func(from S, sp SP, e E, ep EP), from S, sp SP, to S, e E, ep EP) {
	o.executor.Execute(
		func(from S, sp SP, _ S, e E, ep EP) { f(from, sp, e, ep) },
		from, sp, to, e, ep,
	)
}

func (o *Observer[S, E, SP, EP]) runEnter(f func(to S, sp SP, e E, ep EP), from S, sp SP, to S, e E, ep EP) {
	o.executor.Execute(
		func(_ S, sp SP, to S, e E, ep EP) { f(to, sp, e, ep) },
		from, sp, to, e, ep,
	)
}

// OnStep dispatches step, exit, and enter hooks.
func (o *Observer[S, E, SP, EP]) OnStep(from S, sp SP, to S, e E, ep EP) {
	for _, f := range o.onStepAny {
		o.executor.Execute(f, from, sp, to, e, ep)
	}
	if fs := o.onStep[transition[S]{from: from, to: to}]; len(fs) != 0 {
		for _, f := range fs {
			o.executor.Execute(f, from, sp, to, e, ep)
		}
	}
	for _, f := range o.onExitAny {
		o.runExit(f, from, sp, to, e, ep)
	}
	if fs := o.onExit[from]; len(fs) != 0 {
		for _, f := range fs {
			o.runExit(f, from, sp, to, e, ep)
		}
	}
	for _, f := range o.onEnterAny {
		o.runEnter(f, from, sp, to, e, ep)
	}
	if fs := o.onEnter[to]; len(fs) != 0 {
		for _, f := range fs {
			o.runEnter(f, from, sp, to, e, ep)
		}
	}
}
