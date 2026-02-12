package fsm

import "github.com/stnhrsprkwns/fsm/observer"

// NewObserverBuilder constructs an observer callback builder.
func NewObserverBuilder[S comparable, E, SP, EP any]() *observer.Builder[S, E, SP, EP] {
	return observer.NewBuilder[S, E, SP, EP]()
}
