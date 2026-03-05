package graph

import (
	"reflect"
	"testing"
)

func TestDeterministicCloneIncludesIsolatedVertices(t *testing.T) {
	graph := NewDeterministic(
		func(state string) string { return state },
		func(input byte) byte { return input },
	)
	graph.AddVertex("isolated")
	if err := graph.AddTransition("a", 'x', "b"); err != nil {
		t.Fatal(err)
	}
	clone := graph.Clone()
	if got, want := clone.Vertices(), []string{"isolated", "a", "b"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Vertices() = %v, want %v", got, want)
	}
	if next, ok := clone.Delta("a", 'x'); !ok || next != "b" {
		t.Fatalf("Delta() = (%q, %v)", next, ok)
	}
}

func TestNondeterministicCloneIsIndependentAndOrdered(t *testing.T) {
	graph := NewNondeterministic(
		func(state string) string { return state },
		func(input byte) byte { return input },
	)
	graph.AddVertex("isolated")
	graph.AddTransition("a", 'x', "c")
	graph.AddTransition("a", 'x', "b")
	graph.AddEpsilon("a", "d")
	clone := graph.Clone()
	graph.RemoveTransition("a", 'x', "c")
	graph.RemoveEpsilon("a", "d")
	if got, want := clone.Vertices(), []string{"isolated", "a", "c", "b", "d"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Vertices() = %v, want %v", got, want)
	}
	if got, want := clone.Delta("a", 'x'), []string{"c", "b"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Delta() = %v, want %v", got, want)
	}
	if got, want := clone.Epsilon("a"), []string{"d"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Epsilon() = %v, want %v", got, want)
	}
}
