package fsm

import "github.com/stnhrsprkwns/fsm/observer"

// NewObserverBuilder constructs an observer callback builder.
func NewObserverBuilder[S any, E any, SP any, EP any, M any](equal observer.Equal[S]) *observer.Builder[S, E, SP, EP, M] {
	return observer.NewBuilder[S, E, SP, EP, M]().WithEqualState(equal)
}
