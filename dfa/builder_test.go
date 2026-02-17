package dfa

import (
	"testing"

	"github.com/stnhrsprkwns/fsm/graph"
)

type bState int

func (s bState) Key() int { return int(s) }

type bSymbol byte

func (s bSymbol) Key() byte { return byte(s) }

func bWord(xs ...byte) []bSymbol {
	out := make([]bSymbol, 0, len(xs))
	for _, x := range xs {
		out = append(out, bSymbol(x))
	}
	return out
}

func TestTransitionDeterminism(t *testing.T) {
	b := NewBuilder[bState, bSymbol, int, byte]()
	if err := b.Transition(0, 'a', 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := b.Transition(0, 'a', 1); err != nil {
		t.Fatalf("expected idempotent transition, got error: %v", err)
	}
	if err := b.Transition(0, 'a', 2); err == nil {
		t.Fatalf("expected error on nondeterministic transition")
	}
}

func TestBuildAccepts(t *testing.T) {
	b := NewBuilder[bState, bSymbol, int, byte]()
	b.SetStart(0)
	b.AddAccepting(0)
	if err := b.Transition(0, '0', 0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := b.Transition(0, '1', 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := b.Transition(1, '0', 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := b.Transition(1, '1', 0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	d, err := b.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if !d.Accepts(bWord('0', '0')) {
		t.Fatalf("expected acceptance for even ones")
	}
	if d.Accepts(bWord('0', '1')) {
		t.Fatalf("expected rejection for odd ones")
	}
}

func TestBuildAtomic(t *testing.T) {
	b := NewBuilder[bState, bSymbol, int, byte]()
	b.SetStart(0)
	b.AddAccepting(0)
	if err := b.Transition(0, '0', 0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := b.Transition(0, '1', 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := b.Transition(1, '0', 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := b.Transition(1, '1', 0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	a, err := b.BuildAtomic()
	if err != nil {
		t.Fatalf("BuildAtomic() error = %v", err)
	}
	if !a.Accepts(bWord('0', '0')) {
		t.Fatalf("expected acceptance for even ones")
	}
	if a.Accepts(bWord('0', '1')) {
		t.Fatalf("expected rejection for odd ones")
	}
}

func TestWithGraph(t *testing.T) {
	g := graph.New[bState, bSymbol]()
	g.AddEdge(0, 0, '0')
	g.AddEdge(0, 1, '1')
	g.AddEdge(1, 1, '0')
	g.AddEdge(1, 0, '1')
	b := NewBuilder[bState, bSymbol, int, byte]()
	b.SetStart(0)
	b.AddAccepting(0)
	b.AddAlphabet('0', '1')
	b.WithGraph(g)
	d, err := b.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if !d.Accepts(bWord('0', '0')) {
		t.Fatalf("expected acceptance for even ones")
	}
	if d.Accepts(bWord('0', '1')) {
		t.Fatalf("expected rejection for odd ones")
	}
}

func TestWithGraphAtomicGraph(t *testing.T) {
	g := graph.NewAtomic[bState, bSymbol]()
	g.AddEdge(0, 0, '0')
	g.AddEdge(0, 1, '1')
	g.AddEdge(1, 1, '0')
	g.AddEdge(1, 0, '1')
	b := NewBuilder[bState, bSymbol, int, byte]()
	b.SetStart(0)
	b.AddAccepting(0)
	b.AddAlphabet('0', '1')
	b.WithGraph(g)
	d, err := b.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if !d.Accepts(bWord('0', '0')) {
		t.Fatalf("expected acceptance for even ones")
	}
	if d.Accepts(bWord('0', '1')) {
		t.Fatalf("expected rejection for odd ones")
	}
}
