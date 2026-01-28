package atomic

import (
	"sync"
	"testing"

	"github.com/stnhrsprkwns/fsm/pkg/graph"
)

func TestAtomicGraphBasic(t *testing.T) {
	g := New[int, string]()
	g.AddEdge(1, 2, "a")
	g.AddEdge(2, 3, "b")

	if got := g.Delta(1, "a"); len(got) != 1 || got[0] != 2 {
		t.Fatalf("Delta(1,'a') = %v, want [2]", got)
	}
	if !g.HasVertex(1) || !g.HasVertex(2) {
		t.Fatalf("expected vertices to be present")
	}
	if !g.HasLabel("a") || !g.HasLabel("b") {
		t.Fatalf("expected labels to be present")
	}
	if !g.HasEdge(graph.Edge[int, string]{From: 1, To: 2, Label: "a"}) {
		t.Fatalf("expected edge to be present")
	}
}

func TestAtomicGraphConcurrent(t *testing.T) {
	g := New[int, int]()
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				g.AddEdge(id, j, j)
				_ = g.Delta(id, j)
			}
		}(i)
	}
	wg.Wait()
}
