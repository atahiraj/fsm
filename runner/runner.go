package runner

import "context"

// Stepper is the minimal contract required by Runner.
type Stepper[I any] interface {
	Step(symbol I)
}

// Event is a transport envelope delivered to Stepper.
// Event.Event carries the FSM input symbol.
type Event[I any] struct {
	Event I
}

// Runner owns an event queue and feeds events to a Stepper in order.
type Runner[I any] struct {
	stepper Stepper[I]
	events  chan Event[I]
}

// New constructs a Runner.
//
// buffer controls the internal event channel capacity.
func New[I any](stepper Stepper[I], buffer int) *Runner[I] {
	if buffer < 0 {
		buffer = 0
	}
	return &Runner[I]{
		stepper: stepper,
		events:  make(chan Event[I], buffer),
	}
}

// Events exposes the event channel used by the runner loop.
func (r *Runner[I]) Events() chan<- Event[I] {
	return r.events
}

// Run consumes events until the channel is closed or ctx is canceled.
//
// Returns:
//   - nil when the event channel is closed.
//   - ctx.Err() when the context is canceled.
func (r *Runner[I]) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case evt, ok := <-r.events:
			if !ok {
				return nil
			}
			r.stepper.Step(evt.Event)
		}
	}
}
