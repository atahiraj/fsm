package transitionhooks

// MatchMode describes how a registration is matched.
type MatchMode uint8

const (
	MatchAny MatchMode = iota
	MatchFromKey
	MatchToKey
	MatchFromToKey
	MatchPredicate
)

// GroupGuard controls whether a group should dispatch for a transition.
type GroupGuard[K comparable] func(fromKey K, toKey K) bool

// Registration describes one callback registration for compiled dispatch.
type Registration[S any, I any, K comparable] struct {
	Mode      MatchMode
	FromKey   K
	ToKey     K
	Predicate func(from S, to S, e I) bool
	Callback  func(from S, to S, e I)
}

// Group is a phase in the compiled dispatcher.
type Group[S any, I any, K comparable] struct {
	Guard         GroupGuard[K]
	Registrations []Registration[S, I, K]
}

type fromToKey[K comparable] struct {
	from K
	to   K
}

type predicateCallback[S any, I any] struct {
	predicate func(from S, to S, e I) bool
	callback  func(from S, to S, e I)
}

type compiledGroup[S any, I any, K comparable] struct {
	guard      GroupGuard[K]
	any        []func(from S, to S, e I)
	from       map[K][]func(from S, to S, e I)
	to         map[K][]func(from S, to S, e I)
	fromTo     map[fromToKey[K]][]func(from S, to S, e I)
	predicates []predicateCallback[S, I]
}

// CompiledTransitionHooks dispatches callbacks using compiled buckets.
type CompiledTransitionHooks[S any, I any, K comparable] struct {
	executor    Executor[S, I]
	keySelector func(S) K
	groups      []compiledGroup[S, I, K]
}

// NewCompiled constructs a compiled transition hook dispatcher.
func NewCompiled[S any, I any, K comparable](
	executor Executor[S, I],
	keySelector func(S) K,
	groups ...Group[S, I, K],
) *CompiledTransitionHooks[S, I, K] {
	compiledGroups := make([]compiledGroup[S, I, K], len(groups))
	for i, group := range groups {
		cg := compiledGroup[S, I, K]{
			guard:      group.Guard,
			any:        make([]func(from S, to S, e I), 0),
			from:       make(map[K][]func(from S, to S, e I)),
			to:         make(map[K][]func(from S, to S, e I)),
			fromTo:     make(map[fromToKey[K]][]func(from S, to S, e I)),
			predicates: make([]predicateCallback[S, I], 0),
		}

		for _, registration := range group.Registrations {
			switch registration.Mode {
			case MatchAny:
				cg.any = append(cg.any, registration.Callback)
			case MatchFromKey:
				cg.from[registration.FromKey] = append(cg.from[registration.FromKey], registration.Callback)
			case MatchToKey:
				cg.to[registration.ToKey] = append(cg.to[registration.ToKey], registration.Callback)
			case MatchFromToKey:
				key := fromToKey[K]{from: registration.FromKey, to: registration.ToKey}
				cg.fromTo[key] = append(cg.fromTo[key], registration.Callback)
			case MatchPredicate:
				cg.predicates = append(cg.predicates, predicateCallback[S, I]{
					predicate: registration.Predicate,
					callback:  registration.Callback,
				})
			default:
				cg.any = append(cg.any, registration.Callback)
			}
		}
		compiledGroups[i] = cg
	}

	return &CompiledTransitionHooks[S, I, K]{
		executor:    executor,
		keySelector: keySelector,
		groups:      compiledGroups,
	}
}

// OnTransition dispatches callbacks from compiled groups in order.
func (o *CompiledTransitionHooks[S, I, K]) OnTransition(from S, to S, e I) {
	fromKey := o.keySelector(from)
	toKey := o.keySelector(to)
	fromTo := fromToKey[K]{from: fromKey, to: toKey}

	for _, group := range o.groups {
		if group.guard != nil && !group.guard(fromKey, toKey) {
			continue
		}

		for _, callback := range group.any {
			o.executor.Execute(callback, from, to, e)
		}
		if callbacks, ok := group.from[fromKey]; ok {
			for _, callback := range callbacks {
				o.executor.Execute(callback, from, to, e)
			}
		}
		if callbacks, ok := group.to[toKey]; ok {
			for _, callback := range callbacks {
				o.executor.Execute(callback, from, to, e)
			}
		}
		if callbacks, ok := group.fromTo[fromTo]; ok {
			for _, callback := range callbacks {
				o.executor.Execute(callback, from, to, e)
			}
		}
		for _, callback := range group.predicates {
			if callback.predicate != nil && !callback.predicate(from, to, e) {
				continue
			}
			o.executor.Execute(callback.callback, from, to, e)
		}
	}
}
