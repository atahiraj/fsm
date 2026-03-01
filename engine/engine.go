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

// TransitionHooks receives transition notifications.
// All methods are synchronous and called from the Engine's goroutine.
type TransitionHooks[S any, I any] interface {
	OnTransition(from S, to S, e I)
}

// Engine executes an FSM and notifies TransitionHooks.
// It is synchronous and not safe for concurrent use by design.
type Engine[S any, I any] struct {
	fsm   FSM[S, I]
	hooks TransitionHooks[S, I]
	cur   S
}

// New constructs a Engine and positions it at the FSM's start configuration.
func New[S any, I any](fsm FSM[S, I], hooks TransitionHooks[S, I]) *Engine[S, I] {
	return &Engine[S, I]{fsm: fsm, hooks: hooks, cur: fsm.Start()}
}

// Reset returns the Engine to the FSM's start configuration.
func (e *Engine[S, I]) Reset() {
	e.cur = e.fsm.Start()
}

func (e *Engine[S, I]) Cur() S          { return e.cur }
func (e *Engine[S, I]) Accepting() bool { return e.fsm.IsAccepting(e.cur) }

// Step attempts to advance the machine by one symbol.
// It is a no-op when no transition exists.
func (e *Engine[S, I]) Step(symbol I) {
	_ = e.TryStep(symbol)
}

// TryStep advances the machine by one symbol and reports whether a transition existed.
// The hooks are notified only when a transition exists.
func (e *Engine[S, I]) TryStep(symbol I) bool {
	to, ok := e.fsm.Step(e.cur, symbol)
	if !ok {
		return false
	}
	e.hooks.OnTransition(e.cur, to, symbol)
	e.cur = to
	return true
}

// CanStep reports whether a transition exists for the current configuration and symbol.
// It does not update engine state or notify the hooks.
func (e *Engine[S, I]) CanStep(symbol I) bool {
	_, ok := e.PeekStep(symbol)
	return ok
}

// PeekStep reports the next configuration for a symbol without mutating engine state.
// It does not update engine state or notify the hooks.
func (e *Engine[S, I]) PeekStep(symbol I) (S, bool) {
	next, ok := e.fsm.Step(e.cur, symbol)
	return next, ok
}
