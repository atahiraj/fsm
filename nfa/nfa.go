package nfa

import "github.com/stnhrsprkwns/fsm/internal/set"

// Deltaer abstracts the NFA transition relation δ.
//
// δ : Q × Σ → 𝒫(Q)
type Deltaer[State comparable, Symbol comparable] interface {
	Delta(state State, symbol Symbol) []State
	// Epsilon yields ε-transitions from state.
	Epsilon(state State) []State
}

// NFA models a nondeterministic finite automaton.
type NFA[S comparable, A comparable] struct {
	deltaer   Deltaer[S, A] // δ, transition relation
	states    set.Set[S]    // Q, all states.
	alphabet  set.Set[A]    // Σ, all As.
	start     S             // q₀, start state.
	accepting set.Set[S]    // F, accepting states.
}

// Config holds the data needed to construct an NFA (Q, Σ, δ, q₀, F).
type Config[S comparable, A comparable] struct {
	// States is Q, the set of all states.
	States []S
	// Alphabet is Σ, the input alphabet.
	Alphabet []A
	// Start is q₀, the start state.
	Start S
	// Accepting is F, the accepting states.
	Accepting []S
	// Deltaer provides δ and ε.
	Deltaer Deltaer[S, A]
}

// New constructs an NFA from (Q, Σ, δ, q₀, F).
func New[S comparable, A comparable](cfg Config[S, A]) *NFA[S, A] {
	nfa := &NFA[S, A]{
		states:    *set.New(cfg.States...),
		alphabet:  *set.New(cfg.Alphabet...),
		start:     cfg.Start,
		accepting: *set.New(cfg.Accepting...),
		deltaer:   cfg.Deltaer,
	}
	return nfa
}

// NewNFA constructs an NFA from (Q, Σ, δ, q₀, F).
func NewNFA[S comparable, A comparable](states []S, alphabet []A, start S, accepting []S, deltaer Deltaer[S, A]) *NFA[S, A] {
	return New(Config[S, A]{
		States:    states,
		Alphabet:  alphabet,
		Start:     start,
		Accepting: accepting,
		Deltaer:   deltaer,
	})
}

// Delta applies δ : Q × Σ → P(Q) to a state on a single symbol.
func (n *NFA[S, A]) Delta(s S, a A) []S {
	if !n.states.Has(s) {
		return nil
	}
	next := n.deltaer.Delta(s, a)
	if len(next) == 0 {
		return nil
	}
	out := make([]S, 0, len(next))
	for _, ns := range next {
		if n.states.Has(ns) {
			out = append(out, ns)
		}
	}
	return out
}

// DeltaStar applies δ repeatedly over a word. It implements δ*.
func (n *NFA[S, A]) DeltaStar(s S, w []A) []S {
	return n.DeltaStarSet([]S{s}, w)
}

// DeltaSet applies δ : 𝒫(Q) × Σ → 𝒫(Q) to a set of states.
func (n *NFA[S, A]) DeltaSet(states []S, a A) []S {
	out := set.New[S]()
	if len(states) == 0 {
		return nil
	}
	for _, s := range states {
		out.Add(n.Delta(s, a)...)
	}
	return out.Slice()
}

// DeltaStarSet applies δ* to a set of states over a word.
func (n *NFA[S, A]) DeltaStarSet(states []S, w []A) []S {
	cur := n.EpsilonClosureSet(states)
	for _, a := range w {
		cur = n.DeltaSet(cur, a)
		cur = n.EpsilonClosureSet(cur)
	}
	return cur
}

// Epsilon applies δ to a state on ε.
func (n *NFA[S, A]) Epsilon(s S) []S {
	if !n.states.Has(s) {
		return nil
	}
	next := n.deltaer.Epsilon(s)
	if len(next) == 0 {
		return nil
	}
	out := make([]S, 0, len(next))
	for _, ns := range next {
		if n.states.Has(ns) {
			out = append(out, ns)
		}
	}
	return out
}

// EpsilonClosure returns ε-closure(s).
func (n *NFA[S, A]) EpsilonClosure(s S) []S {
	return n.EpsilonClosureSet([]S{s})
}

// EpsilonSet applies δ to a set of states on ε.
func (n *NFA[S, A]) EpsilonSet(states []S) []S {
	out := set.New[S]()
	if len(states) == 0 {
		return nil
	}
	for _, s := range states {
		out.Add(n.Epsilon(s)...)
	}
	return out.Slice()
}

// EpsilonClosureSet returns ε-closure(Q).
func (n *NFA[S, A]) EpsilonClosureSet(states []S) []S {
	if len(states) == 0 {
		return nil
	}
	out := set.New[S]()
	stack := make([]S, 0, len(states))
	for _, s := range states {
		if !out.Has(s) {
			out.Add(s)
			stack = append(stack, s)
		}
	}
	for len(stack) > 0 {
		s := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for _, ns := range n.Epsilon(s) {
			if !out.Has(ns) {
				out.Add(ns)
				stack = append(stack, ns)
			}
		}
	}
	return out.Slice()
}

// Start returns q₀, the start state.
func (n *NFA[S, A]) Start() S {
	return n.start
}

// StartSet returns {q₀}, the singleton set containing the start state.
func (n *NFA[S, A]) StartSet() []S {
	return []S{n.start}
}

// States returns Q, the set of all states.
func (n *NFA[S, A]) States() []S {
	return n.states.Clone().Slice()
}

// Alphabet returns Σ, the input alphabet.
func (n *NFA[S, A]) Alphabet() []A {
	return n.alphabet.Clone().Slice()
}

// Accepting returns F, the set of accepting states.
func (n *NFA[S, A]) Accepting() []S {
	return n.accepting.Clone().Slice()
}

// Accepts reports whether the NFA accepts the given word.
func (n *NFA[S, A]) Accepts(w []A) bool {
	return n.IsAccepting(n.DeltaStarSet(n.StartSet(), w))
}

// HasState reports whether state ∈ Q.
func (n *NFA[S, A]) HasState(s S) bool {
	return n.states.Has(s)
}

// HasSymbol reports whether symbol ∈ Σ.
func (n *NFA[S, A]) HasSymbol(a A) bool {
	return n.alphabet.Has(a)
}

// IsAccepting reports whether any state in s is an accepting state.
func (n *NFA[S, A]) IsAccepting(states []S) bool {
	for _, s := range states {
		if n.accepting.Has(s) {
			return true
		}
	}
	return false
}

// SetStart sets q₀, the start state.
func (n *NFA[S, A]) SetStart(s S) {
	n.start = s
}

// AddStates inserts states into Q.
func (n *NFA[S, A]) AddStates(states ...S) {
	n.states.Add(states...)
}

// AddAlphabet inserts symbols into Σ.
func (n *NFA[S, A]) AddAlphabet(symbols ...A) {
	n.alphabet.Add(symbols...)
}

// AddAccepting inserts states into F.
func (n *NFA[S, A]) AddAccepting(states ...S) {
	n.accepting.Add(states...)
}

// RemoveAccepting removes states from F.
func (n *NFA[S, A]) RemoveAccepting(states ...S) {
	n.accepting.Remove(states...)
}

// SetDeltaer sets δ, the transition relation.
func (n *NFA[S, A]) SetDeltaer(deltaer Deltaer[S, A]) {
	n.deltaer = deltaer
}
