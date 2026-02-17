package nfa

import "testing"

type bState int

func (s bState) Key() int { return int(s) }

type bSymbol byte

func (s bSymbol) Key() byte { return byte(s) }

type bGraph struct {
	bySym map[bState]map[bSymbol][]bState
	eps   map[bState][]bState
}

func (g bGraph) Delta(from bState, sym bSymbol) []bState {
	if row, ok := g.bySym[from]; ok {
		return row[sym]
	}
	return nil
}

func (g bGraph) Epsilon(from bState) []bState {
	return g.eps[from]
}

func bword(xs ...byte) []bSymbol {
	out := make([]bSymbol, 0, len(xs))
	for _, x := range xs {
		out = append(out, bSymbol(x))
	}
	return out
}

func TestBuildAtomic(t *testing.T) {
	b := NewBuilder[bState, bSymbol, int, byte]()
	b.SetStart(0)
	b.AddAccepting(2)
	b.Epsilon(0, 1)
	b.Transition(1, 'a', 2)

	a, err := b.BuildAtomic()
	if err != nil {
		t.Fatalf("BuildAtomic() error = %v", err)
	}
	if !a.Accepts(bword('a')) {
		t.Fatalf("expected acceptance via epsilon transition")
	}
	if a.Accepts(nil) {
		t.Fatalf("expected empty word to be rejected")
	}
}

func TestWithGraphAtomicGraph(t *testing.T) {
	g := bGraph{
		bySym: map[bState]map[bSymbol][]bState{
			1: {'a': {2}},
		},
		eps: map[bState][]bState{
			0: {1},
		},
	}
	b := NewBuilder[bState, bSymbol, int, byte]()
	b.SetStart(0)
	b.AddAccepting(2)
	b.AddStates(1)
	b.AddAlphabet('a')
	b.WithGraph(g)
	n, err := b.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if !n.Accepts(bword('a')) {
		t.Fatalf("expected acceptance via epsilon transition")
	}
	if n.Accepts(nil) {
		t.Fatalf("expected empty word to be rejected")
	}
}
