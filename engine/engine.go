package engine

// FSM is the minimal contract a machine must satisfy.
// S = configuration (DFA: a state; NFA: a set of states)
// I = input symbol.
type FSM[S any, I any] interface {
	Start() S
	// Step returns the next configuration and whether a transition exists.
	Step(s S, a I) (S, bool)
	IsAccepting(s S) bool
}

// Observer receives step notifications.
// All methods are synchronous and called from the Engine's goroutine.
type Observer[S any, I any, SP any, IP any] interface {
	OnStep(from S, sp SP, to S, e I, ep IP)
}

// Engine executes an FSM and notifies an Observer.
// It is synchronous and not safe for concurrent use by design.
//
// SP = state payload (associated with the current state).
// IP = payload associated with the input symbol passed to Step.
type Engine[S any, I any, SP any, IP any] struct {
	fsm     FSM[S, I]
	obs     Observer[S, I, SP, IP]
	cur     S
	payload SP
}

// New constructs a Engine and positions it at the FSM's start configuration.
func New[S any, I any, SP any, IP any](fsm FSM[S, I], obs Observer[S, I, SP, IP]) *Engine[S, I, SP, IP] {
	return &Engine[S, I, SP, IP]{fsm: fsm, obs: obs, cur: fsm.Start()}
}

// Reset returns the Engine to the FSM's start configuration.
func (e *Engine[S, I, SP, IP]) Reset(p SP) {
	e.cur = e.fsm.Start()
	e.payload = p
}

func (e *Engine[S, I, SP, IP]) Cur() S          { return e.cur }
func (e *Engine[S, I, SP, IP]) Accepting() bool { return e.fsm.IsAccepting(e.cur) }

// Step attempts to advance the machine by one symbol.
// It is a no-op when no transition exists.
func (e *Engine[S, I, SP, IP]) Step(symbol I, payload IP) {
	_ = e.TryStep(symbol, payload)
}

// TryStep advances the machine by one symbol and reports whether a transition existed.
// The observer is notified only when a transition exists.
func (e *Engine[S, I, SP, IP]) TryStep(symbol I, payload IP) bool {
	to, ok := e.fsm.Step(e.cur, symbol)
	if !ok {
		return false
	}
	e.obs.OnStep(e.cur, e.payload, to, symbol, payload)
	e.cur = to
	return true
}

// CanStep reports whether a transition exists for the current configuration and symbol.
// It does not update engine state or notify the observer.
func (e *Engine[S, I, SP, IP]) CanStep(symbol I) bool {
	_, ok := e.PeekStep(symbol)
	return ok
}

// PeekStep reports the next configuration for a symbol without mutating engine state.
// It does not update engine state or notify the observer.
func (e *Engine[S, I, SP, IP]) PeekStep(symbol I) (S, bool) {
	next, ok := e.fsm.Step(e.cur, symbol)
	return next, ok
}
