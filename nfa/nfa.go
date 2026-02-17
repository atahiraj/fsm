package nfa

import "github.com/stnhrsprkwns/fsm/internal/set"

// Keyed exposes a stable comparable key for a value.
type Keyed[K comparable] interface {
	Key() K
}

// Deltaer abstracts the NFA transition relation δ.
//
// δ : Q × Σ → 𝒫(Q)
//
// Q and Σ are represented by comparable keys.
type Deltaer[StateKey comparable, SymbolKey comparable] interface {
	Delta(state StateKey, symbol SymbolKey) []StateKey
	// Epsilon yields ε-transitions from state.
	Epsilon(state StateKey) []StateKey
}

// NFA models a nondeterministic finite automaton.
type NFA[S Keyed[StateKey], A Keyed[SymbolKey], StateKey comparable, SymbolKey comparable] struct {
	deltaer    Deltaer[StateKey, SymbolKey] // δ, transition relation
	states     set.Set[StateKey]            // Q, all state keys.
	alphabet   set.Set[SymbolKey]           // Σ, all symbol keys.
	start      S                            // q₀, start state.
	accepting  set.Set[StateKey]            // F, accepting state keys.
	stateByKey map[StateKey]S
	symByKey   map[SymbolKey]A
}

// Config holds the data needed to construct an NFA (Q, Σ, δ, q₀, F).
type Config[S Keyed[StateKey], A Keyed[SymbolKey], StateKey comparable, SymbolKey comparable] struct {
	// States is Q, the set of all states.
	States []S
	// Alphabet is Σ, the input alphabet.
	Alphabet []A
	// Start is q₀, the start state.
	Start S
	// Accepting is F, the accepting states.
	Accepting []S
	// Deltaer provides δ and ε over keys.
	Deltaer Deltaer[StateKey, SymbolKey]
}

// New constructs an NFA from (Q, Σ, δ, q₀, F).
func New[S Keyed[StateKey], A Keyed[SymbolKey], StateKey comparable, SymbolKey comparable](cfg Config[S, A, StateKey, SymbolKey]) *NFA[S, A, StateKey, SymbolKey] {
	n := &NFA[S, A, StateKey, SymbolKey]{
		deltaer:    cfg.Deltaer,
		start:      cfg.Start,
		stateByKey: make(map[StateKey]S, len(cfg.States)),
		symByKey:   make(map[SymbolKey]A, len(cfg.Alphabet)),
	}
	n.AddStates(cfg.States...)
	n.AddAlphabet(cfg.Alphabet...)
	n.AddAccepting(cfg.Accepting...)
	return n
}

// NewNFA constructs an NFA from (Q, Σ, δ, q₀, F).
func NewNFA[S Keyed[StateKey], A Keyed[SymbolKey], StateKey comparable, SymbolKey comparable](states []S, alphabet []A, start S, accepting []S, deltaer Deltaer[StateKey, SymbolKey]) *NFA[S, A, StateKey, SymbolKey] {
	return New(Config[S, A, StateKey, SymbolKey]{
		States:    states,
		Alphabet:  alphabet,
		Start:     start,
		Accepting: accepting,
		Deltaer:   deltaer,
	})
}

func (n *NFA[S, A, StateKey, SymbolKey]) keysToStates(keys []StateKey) []S {
	if len(keys) == 0 {
		return nil
	}
	out := make([]S, 0, len(keys))
	for _, key := range keys {
		if state, ok := n.stateByKey[key]; ok {
			out = append(out, state)
		}
	}
	return out
}

// Delta applies δ : Q × Σ → P(Q) to a state on a single symbol.
func (n *NFA[S, A, StateKey, SymbolKey]) Delta(s S, a A) []S {
	stateKey := s.Key()
	if !n.states.Has(stateKey) {
		return nil
	}
	next := n.deltaer.Delta(stateKey, a.Key())
	if len(next) == 0 {
		return nil
	}
	out := make([]StateKey, 0, len(next))
	for _, ns := range next {
		if n.states.Has(ns) {
			out = append(out, ns)
		}
	}
	return n.keysToStates(out)
}

// DeltaStar applies δ repeatedly over a word. It implements δ*.
func (n *NFA[S, A, StateKey, SymbolKey]) DeltaStar(s S, w []A) []S {
	return n.DeltaStarSet([]S{s}, w)
}

// DeltaSet applies δ : 𝒫(Q) × Σ → 𝒫(Q) to a set of states.
func (n *NFA[S, A, StateKey, SymbolKey]) DeltaSet(states []S, a A) []S {
	out := set.New[StateKey]()
	if len(states) == 0 {
		return nil
	}
	for _, s := range states {
		for _, ns := range n.Delta(s, a) {
			out.Add(ns.Key())
		}
	}
	return n.keysToStates(out.Slice())
}

// DeltaStarSet applies δ* to a set of states over a word.
func (n *NFA[S, A, StateKey, SymbolKey]) DeltaStarSet(states []S, w []A) []S {
	cur := n.EpsilonClosureSet(states)
	for _, a := range w {
		cur = n.DeltaSet(cur, a)
		cur = n.EpsilonClosureSet(cur)
	}
	return cur
}

// Epsilon applies δ to a state on ε.
func (n *NFA[S, A, StateKey, SymbolKey]) Epsilon(s S) []S {
	stateKey := s.Key()
	if !n.states.Has(stateKey) {
		return nil
	}
	next := n.deltaer.Epsilon(stateKey)
	if len(next) == 0 {
		return nil
	}
	out := make([]StateKey, 0, len(next))
	for _, ns := range next {
		if n.states.Has(ns) {
			out = append(out, ns)
		}
	}
	return n.keysToStates(out)
}

// EpsilonClosure returns ε-closure(s).
func (n *NFA[S, A, StateKey, SymbolKey]) EpsilonClosure(s S) []S {
	return n.EpsilonClosureSet([]S{s})
}

// EpsilonSet applies δ to a set of states on ε.
func (n *NFA[S, A, StateKey, SymbolKey]) EpsilonSet(states []S) []S {
	out := set.New[StateKey]()
	if len(states) == 0 {
		return nil
	}
	for _, s := range states {
		for _, ns := range n.Epsilon(s) {
			out.Add(ns.Key())
		}
	}
	return n.keysToStates(out.Slice())
}

// EpsilonClosureSet returns ε-closure(Q).
func (n *NFA[S, A, StateKey, SymbolKey]) EpsilonClosureSet(states []S) []S {
	if len(states) == 0 {
		return nil
	}
	out := set.New[StateKey]()
	stack := make([]StateKey, 0, len(states))
	for _, s := range states {
		k := s.Key()
		if !out.Has(k) {
			out.Add(k)
			stack = append(stack, k)
		}
	}
	for len(stack) > 0 {
		sk := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		state, ok := n.stateByKey[sk]
		if !ok {
			continue
		}
		for _, ns := range n.Epsilon(state) {
			nk := ns.Key()
			if !out.Has(nk) {
				out.Add(nk)
				stack = append(stack, nk)
			}
		}
	}
	return n.keysToStates(out.Slice())
}

// Start returns q₀, the start state.
func (n *NFA[S, A, StateKey, SymbolKey]) Start() S {
	return n.start
}

// StartSet returns {q₀}, the singleton set containing the start state.
func (n *NFA[S, A, StateKey, SymbolKey]) StartSet() []S {
	return []S{n.start}
}

// States returns Q, the set of all states.
func (n *NFA[S, A, StateKey, SymbolKey]) States() []S {
	return n.keysToStates(n.states.Clone().Slice())
}

// Alphabet returns Σ, the input alphabet.
func (n *NFA[S, A, StateKey, SymbolKey]) Alphabet() []A {
	keys := n.alphabet.Clone().Slice()
	out := make([]A, 0, len(keys))
	for _, key := range keys {
		if sym, ok := n.symByKey[key]; ok {
			out = append(out, sym)
		}
	}
	return out
}

// Accepting returns F, the set of accepting states.
func (n *NFA[S, A, StateKey, SymbolKey]) Accepting() []S {
	return n.keysToStates(n.accepting.Clone().Slice())
}

// Accepts reports whether the NFA accepts the given word.
func (n *NFA[S, A, StateKey, SymbolKey]) Accepts(w []A) bool {
	return n.IsAccepting(n.DeltaStarSet(n.StartSet(), w))
}

// HasState reports whether state ∈ Q.
func (n *NFA[S, A, StateKey, SymbolKey]) HasState(s S) bool {
	return n.states.Has(s.Key())
}

// HasSymbol reports whether symbol ∈ Σ.
func (n *NFA[S, A, StateKey, SymbolKey]) HasSymbol(a A) bool {
	return n.alphabet.Has(a.Key())
}

// IsAccepting reports whether any state in s is an accepting state.
func (n *NFA[S, A, StateKey, SymbolKey]) IsAccepting(states []S) bool {
	for _, s := range states {
		if n.accepting.Has(s.Key()) {
			return true
		}
	}
	return false
}

// SetStart sets q₀, the start state.
func (n *NFA[S, A, StateKey, SymbolKey]) SetStart(s S) {
	n.start = s
}

// AddStates inserts states into Q.
func (n *NFA[S, A, StateKey, SymbolKey]) AddStates(states ...S) {
	for _, state := range states {
		key := state.Key()
		n.states.Add(key)
		n.stateByKey[key] = state
	}
}

// AddAlphabet inserts symbols into Σ.
func (n *NFA[S, A, StateKey, SymbolKey]) AddAlphabet(symbols ...A) {
	for _, symbol := range symbols {
		key := symbol.Key()
		n.alphabet.Add(key)
		n.symByKey[key] = symbol
	}
}

// AddAccepting inserts states into F.
func (n *NFA[S, A, StateKey, SymbolKey]) AddAccepting(states ...S) {
	for _, state := range states {
		key := state.Key()
		n.accepting.Add(key)
		n.stateByKey[key] = state
	}
}

// RemoveAccepting removes states from F.
func (n *NFA[S, A, StateKey, SymbolKey]) RemoveAccepting(states ...S) {
	keys := make([]StateKey, 0, len(states))
	for _, state := range states {
		keys = append(keys, state.Key())
	}
	n.accepting.Remove(keys...)
}

// SetDeltaer sets δ, the transition relation.
func (n *NFA[S, A, StateKey, SymbolKey]) SetDeltaer(deltaer Deltaer[StateKey, SymbolKey]) {
	n.deltaer = deltaer
}
