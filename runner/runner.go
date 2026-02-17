package runner

import "context"

// Stepper is the minimal contract required by Runner.
type Stepper[I any, IP any] interface {
	Step(symbol I, payload IP)
}

// Event is a transport envelope delivered to Stepper.
// Event.Event carries the FSM input symbol.
type Event[I any, IP any] struct {
	Event   I
	Payload IP
}

// Runner owns an event queue and feeds events to a Stepper in order.
type Runner[I any, IP any] struct {
	stepper Stepper[I, IP]
	events  chan Event[I, IP]
}

// New constructs a Runner.
//
// buffer controls the internal event channel capacity.
func New[I any, IP any](stepper Stepper[I, IP], buffer int) *Runner[I, IP] {
	if buffer < 0 {
		buffer = 0
	}
	return &Runner[I, IP]{
		stepper: stepper,
		events:  make(chan Event[I, IP], buffer),
	}
}

// Events exposes the event channel used by the runner loop.
func (r *Runner[I, IP]) Events() chan<- Event[I, IP] {
	return r.events
}

// Run consumes events until the channel is closed or ctx is canceled.
//
// Returns:
//   - nil when the event channel is closed.
//   - ctx.Err() when the context is canceled.
func (r *Runner[I, IP]) Run(ctx context.Context) error {
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
