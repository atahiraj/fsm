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

// Observer dispatches step, enter, and exit hooks.
type Observer[S any, E any, SP any, EP any, M any] struct {
	executor  Executor[S, E, SP, EP]
	callbacks []func(from S, sp SP, to S, e E, ep EP)
}

// NewObserver constructs an Observer.
func NewObserver[S any, E any, SP any, EP any, M any](e Executor[S, E, SP, EP], callbacks ...func(from S, sp SP, to S, e E, ep EP)) *Observer[S, E, SP, EP, M] {
	o := &Observer[S, E, SP, EP, M]{
		executor:  e,
		callbacks: callbacks,
	}
	if o.callbacks == nil {
		o.callbacks = make([]func(from S, sp SP, to S, e E, ep EP), 0)
	}
	return o
}

// OnStep dispatches step, exit, and enter hooks.
func (o *Observer[S, E, SP, EP, M]) OnStep(from S, sp SP, to S, e E, ep EP) {
	for _, f := range o.callbacks {
		o.executor.Execute(f, from, sp, to, e, ep)
	}
}
