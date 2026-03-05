// Package fsm executes state machines and dispatches transition lifecycle
// notifications.
//
// A Machine supplies transition behavior. Engine owns only the current state;
// definitions, persistence, synchronization, and callback scheduling remain
// injectable concerns.
package fsm
