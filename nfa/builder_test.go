package nfa

import (
	"github.com/stnhrsprkwns/fsm/graph"
	"testing"
)

func TestBuildAtomic(t *testing.T) {
	b := NewBuilder[int, byte]()
	b.SetStart(0)
	b.AddAccepting(2)
	b.Epsilon(0, 1)
	b.Transition(1, 'a', 2)

	a, err := b.BuildAtomic()
	if err != nil {
		t.Fatalf("BuildAtomic() error = %v", err)
	}
	if !a.Accepts([]byte{'a'}) {
		t.Fatalf("expected acceptance via epsilon transition")
	}
	if a.Accepts(nil) {
		t.Fatalf("expected empty word to be rejected")
	}
}

func TestWithGraphAtomicGraph(t *testing.T) {
	g := graph.NewAtomic[int, Label[byte]]()
	g.AddEdge(0, 1, Epsilon[byte]())
	g.AddEdge(1, 2, Sym(byte('a')))
	b := NewBuilder[int, byte]()
	b.SetStart(0)
	b.AddAccepting(2)
	b.AddStates(1)
	b.WithGraph(g)
	n, err := b.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if !n.Accepts([]byte{'a'}) {
		t.Fatalf("expected acceptance via epsilon transition")
	}
	if n.Accepts(nil) {
		t.Fatalf("expected empty word to be rejected")
	}
}
