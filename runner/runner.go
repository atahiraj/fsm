package runner

import "context"

// Stepper is the minimal contract required by Runner.
type Stepper[E any, EP any] interface {
	Step(event E, payload EP)
}

// Event is a single input delivered to Stepper.
type Event[E any, EP any] struct {
	Event   E
	Payload EP
}

// Runner owns an event queue and feeds events to a Stepper in order.
type Runner[E any, EP any] struct {
	stepper Stepper[E, EP]
	events  chan Event[E, EP]
}

// New constructs a Runner.
//
// buffer controls the internal event channel capacity.
func New[E any, EP any](stepper Stepper[E, EP], buffer int) *Runner[E, EP] {
	if buffer < 0 {
		buffer = 0
	}
	return &Runner[E, EP]{
		stepper: stepper,
		events:  make(chan Event[E, EP], buffer),
	}
}

// Events exposes the event channel used by the runner loop.
func (r *Runner[E, EP]) Events() chan<- Event[E, EP] {
	return r.events
}

// Run consumes events until the channel is closed or ctx is canceled.
//
// Returns:
//   - nil when the event channel is closed.
//   - ctx.Err() when the context is canceled.
func (r *Runner[E, EP]) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case evt, ok := <-r.events:
			if !ok {
				return nil
			}
			r.stepper.Step(evt.Event, evt.Payload)
		}
	}
}
