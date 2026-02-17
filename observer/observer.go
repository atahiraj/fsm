package observer

// Executor controls how observer callbacks are executed.
type Executor[S any, I, SP, IP any] interface {
	Execute(f func(from S, sp SP, to S, e I, ep IP), from S, sp SP, to S, e I, ep IP)
}

// DefaultExecutor calls the callback directly.
type DefaultExecutor[S any, I, SP, IP any] struct{}

func (DefaultExecutor[S, I, SP, IP]) Execute(
	f func(from S, sp SP, to S, e I, ep IP),
	from S,
	sp SP,
	to S,
	e I,
	ep IP,
) {
	f(from, sp, to, e, ep)
}

// Observer dispatches step, enter, and exit hooks.
type Observer[S any, I any, SP any, IP any, M any] struct {
	executor  Executor[S, I, SP, IP]
	callbacks []func(from S, sp SP, to S, e I, ep IP)
}

// NewObserver constructs an Observer.
func NewObserver[S any, I any, SP any, IP any, M any](e Executor[S, I, SP, IP], callbacks ...func(from S, sp SP, to S, e I, ep IP)) *Observer[S, I, SP, IP, M] {
	o := &Observer[S, I, SP, IP, M]{
		executor:  e,
		callbacks: callbacks,
	}
	if o.callbacks == nil {
		o.callbacks = make([]func(from S, sp SP, to S, e I, ep IP), 0)
	}
	return o
}

// OnStep dispatches step, exit, and enter hooks.
func (o *Observer[S, I, SP, IP, M]) OnStep(from S, sp SP, to S, e I, ep IP) {
	for _, f := range o.callbacks {
		o.executor.Execute(f, from, sp, to, e, ep)
	}
}
