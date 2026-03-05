package fsm

import "errors"

// Hook receives one transition lifecycle notification.
type Hook[State, Input any] func(Transition[State, Input])

// Predicate selects transition lifecycle notifications.
type Predicate[State, Input any] func(Transition[State, Input]) bool

type hookRegistration[State, Input any] struct {
	predicate Predicate[State, Input]
	callback  Hook[State, Input]
}

// Hooks is a synchronous lifecycle router.
//
// Registrations run in insertion order within each phase. Hooks is mutable and
// is not safe for concurrent use.
type Hooks[State, Input any] struct {
	equal         func(State, State) bool
	registrations map[Phase][]hookRegistration[State, Input]
}

// NewHooks constructs an empty hook router.
func NewHooks[State, Input any](equal func(State, State) bool) (*Hooks[State, Input], error) {
	if equal == nil {
		return nil, errors.New("fsm: hook state equality function is nil")
	}
	return &Hooks[State, Input]{
		equal:         equal,
		registrations: make(map[Phase][]hookRegistration[State, Input]),
	}, nil
}

func validPhase(phase Phase) bool {
	return phase >= PhaseBefore && phase <= PhaseAfter
}

func (h *Hooks[State, Input]) add(phase Phase, predicate Predicate[State, Input], callback Hook[State, Input]) error {
	if h == nil {
		return errors.New("fsm: hooks is nil")
	}
	if !validPhase(phase) {
		return errors.New("fsm: invalid lifecycle phase")
	}
	if callback == nil {
		return errors.New("fsm: hook callback is nil")
	}
	h.registrations[phase] = append(h.registrations[phase], hookRegistration[State, Input]{
		predicate: predicate,
		callback:  callback,
	})
	return nil
}

// OnIf registers callback for phase when predicate matches.
func (h *Hooks[State, Input]) OnIf(phase Phase, predicate Predicate[State, Input], callback Hook[State, Input]) error {
	if predicate == nil {
		return errors.New("fsm: hook predicate is nil")
	}
	return h.add(phase, predicate, callback)
}

// OnBefore registers a callback before every successful transition.
func (h *Hooks[State, Input]) OnBefore(callback Hook[State, Input]) error {
	return h.add(PhaseBefore, nil, callback)
}

// OnExitAny registers a callback whenever a state is exited.
func (h *Hooks[State, Input]) OnExitAny(callback Hook[State, Input]) error {
	return h.add(PhaseExit, nil, callback)
}

// OnExit registers a callback when state is exited.
func (h *Hooks[State, Input]) OnExit(state State, callback Hook[State, Input]) error {
	return h.add(PhaseExit, func(transition Transition[State, Input]) bool {
		return h.equal(transition.From, state)
	}, callback)
}

// OnTransitionAny registers a callback for every successful transition.
func (h *Hooks[State, Input]) OnTransitionAny(callback Hook[State, Input]) error {
	return h.add(PhaseTransition, nil, callback)
}

// OnTransition registers a callback for a specific from-to transition.
func (h *Hooks[State, Input]) OnTransition(from, to State, callback Hook[State, Input]) error {
	return h.add(PhaseTransition, func(transition Transition[State, Input]) bool {
		return h.equal(transition.From, from) && h.equal(transition.To, to)
	}, callback)
}

// OnEnterAny registers a callback whenever a state is entered.
func (h *Hooks[State, Input]) OnEnterAny(callback Hook[State, Input]) error {
	return h.add(PhaseEnter, nil, callback)
}

// OnEnter registers a callback when state is entered.
func (h *Hooks[State, Input]) OnEnter(state State, callback Hook[State, Input]) error {
	return h.add(PhaseEnter, func(transition Transition[State, Input]) bool {
		return h.equal(transition.To, state)
	}, callback)
}

// OnAfter registers a callback after every successful transition.
func (h *Hooks[State, Input]) OnAfter(callback Hook[State, Input]) error {
	return h.add(PhaseAfter, nil, callback)
}

// Handle dispatches matching callbacks for phase.
func (h *Hooks[State, Input]) Handle(phase Phase, transition Transition[State, Input]) {
	if h == nil {
		return
	}
	for _, registration := range h.registrations[phase] {
		if registration.predicate != nil && !registration.predicate(transition) {
			continue
		}
		registration.callback(transition)
	}
}
