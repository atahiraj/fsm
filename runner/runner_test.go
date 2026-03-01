package runner

import (
	"context"
	"sync"
	"testing"
	"time"
)

type recordedEvent struct {
	event string
}

type recordingStepper struct {
	mu     sync.Mutex
	events []recordedEvent
}

func (s *recordingStepper) Step(event string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, recordedEvent{event: event})
}

func (s *recordingStepper) snapshot() []recordedEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]recordedEvent, len(s.events))
	copy(out, s.events)
	return out
}

func TestRunnerRunConsumesEventsInOrder(t *testing.T) {
	stepper := &recordingStepper{}
	r := New[string](stepper, 2)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- r.Run(ctx)
	}()

	events := r.Events()
	events <- Event[string]{Event: "a"}
	events <- Event[string]{Event: "b"}
	close(events)

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run() error = %v, want nil", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run() did not exit")
	}

	got := stepper.snapshot()
	want := []recordedEvent{{event: "a"}, {event: "b"}}
	if len(got) != len(want) {
		t.Fatalf("events len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("event[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestRunnerRunReturnsContextErrorOnCancel(t *testing.T) {
	stepper := &recordingStepper{}
	r := New[string](stepper, 0)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- r.Run(ctx)
	}()

	cancel()

	select {
	case err := <-done:
		if err != context.Canceled {
			t.Fatalf("Run() error = %v, want %v", err, context.Canceled)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run() did not exit after cancellation")
	}
}

func TestRunnerNegativeBufferFallsBackToUnbuffered(t *testing.T) {
	stepper := &recordingStepper{}
	r := New[string](stepper, -1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- r.Run(ctx)
	}()

	events := r.Events()
	go func() {
		events <- Event[string]{Event: "x"}
		close(events)
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run() error = %v, want nil", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run() did not exit")
	}

	got := stepper.snapshot()
	if len(got) != 1 || got[0] != (recordedEvent{event: "x"}) {
		t.Fatalf("events = %+v, want [{event:x}]", got)
	}
}
