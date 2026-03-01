package transitionhooks

// Executor controls how transition hook callbacks are executed.
type Executor[S any, I any] interface {
	Execute(f func(from S, to S, e I), from S, to S, e I)
}

// DefaultExecutor calls the callback directly.
type DefaultExecutor[S any, I any] struct{}

func (DefaultExecutor[S, I]) Execute(
	f func(from S, to S, e I),
	from S,
	to S,
	e I,
) {
	f(from, to, e)
}

// TransitionHooks dispatches step, enter, and exit hooks.
type TransitionHooks[S any, I any] struct {
	executor  Executor[S, I]
	callbacks []func(from S, to S, e I)
}

// New constructs TransitionHooks.
func New[S any, I any](e Executor[S, I], callbacks ...func(from S, to S, e I)) *TransitionHooks[S, I] {
	o := &TransitionHooks[S, I]{
		executor:  e,
		callbacks: callbacks,
	}
	if o.callbacks == nil {
		o.callbacks = make([]func(from S, to S, e I), 0)
	}
	return o
}

// OnTransition dispatches step, exit, and enter hooks.
func (o *TransitionHooks[S, I]) OnTransition(from S, to S, e I) {
	for _, f := range o.callbacks {
		o.executor.Execute(f, from, to, e)
	}
}
