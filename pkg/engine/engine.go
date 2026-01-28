package engine

// FSM is the minimal contract a machine must satisfy.
// S = configuration (DFA: a state; NFA: a set of states)
// E = input symbol/event.
type FSM[S any, E any] interface {
	Start() S
	Step(s S, a E) S
	IsAccepting(s S) bool
}

// Observer receives step events.
// All methods are synchronous and called from the Engine's goroutine.
type Observer[S any, E any, SP any, EP any] interface {
	OnStep(from S, sp SP, to S, e E, ep EP)
}

// Engine executes an FSM and notifies an Observer.
// It is synchronous and not safe for concurrent use by design.
//
// SP = state payload (associated with the current state).
// EP = event payload (associated with the input/event passed to Step).
type Engine[S any, E any, SP any, EP any] struct {
	fsm     FSM[S, E]
	obs     Observer[S, E, SP, EP]
	cur     S
	payload SP
}

// New constructs a Engine and positions it at the FSM's start configuration.
func New[S any, E any, SP any, EP any](fsm FSM[S, E], obs Observer[S, E, SP, EP]) *Engine[S, E, SP, EP] {
	return &Engine[S, E, SP, EP]{fsm: fsm, obs: obs, cur: fsm.Start()}
}

// Reset returns the Engine to the FSM's start configuration.
func (e *Engine[S, E, SP, EP]) Reset(p SP) {
	e.cur = e.fsm.Start()
	e.payload = p
}

func (e *Engine[S, E, SP, EP]) Cur() S          { return e.cur }
func (e *Engine[S, E, SP, EP]) Accepting() bool { return e.fsm.IsAccepting(e.cur) }

// Step advances the machine by one symbol and notifies the observer.
func (e *Engine[S, E, SP, EP]) Step(event E, payload EP) {
	to := e.fsm.Step(e.cur, event)
	e.obs.OnStep(e.cur, e.payload, to, event, payload)
	e.cur = to
}
