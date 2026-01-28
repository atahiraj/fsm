package atomic

import (
	"sync"

	"github.com/stnhrsprkwns/fsm/pkg/graph"
	"github.com/stnhrsprkwns/fsm/pkg/set"
)

// Graph is a thread-safe wrapper around graph.Graph.
// Use New to construct a non-nil graph.
type Graph[T comparable, L comparable] struct {
	mu sync.RWMutex
	g  *graph.Graph[T, L]
}

// New constructs an empty labeled digraph (V,Λ,E) with V=∅, E=∅.
func New[T comparable, L comparable]() *Graph[T, L] {
	return &Graph[T, L]{g: graph.New[T, L]()}
}

// AddVertex inserts v into V if absent.
func (g *Graph[T, L]) AddVertex(v T) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.g.AddVertex(v)
}

// AddEdge inserts the edge into E and ensures both endpoints are in V.
func (g *Graph[T, L]) AddEdge(from, to T, label L) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.g.AddEdge(from, to, label)
}

// Clone creates a deep copy of the graph.
func (g *Graph[T, L]) Clone() *graph.Graph[T, L] {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.g.Clone()
}

// Delta returns the one step adjacent vertices for a given vertex and label.
func (g *Graph[T, L]) Delta(from T, label L) []T {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.g.Delta(from, label)
}

// Vertices enumerates the vertex set V.
func (g *Graph[T, L]) Vertices() []T {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.g.Vertices()
}

// VerticesSet returns a set of all the vertices.
func (g *Graph[T, L]) VerticesSet() *set.Set[T] {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.g.VerticesSet()
}

// Edges enumerates the edge relation E as values.
func (g *Graph[T, L]) Edges() []graph.Edge[T, L] {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.g.Edges()
}

// EdgesSet returns a set of all the edges.
func (g *Graph[T, L]) EdgesSet() *set.Set[graph.Edge[T, L]] {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.g.EdgesSet()
}

// Labels returns all labels.
func (g *Graph[T, L]) Labels() []L {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.g.Labels()
}

// LabelsSet returns a set of all the labels.
func (g *Graph[T, L]) LabelsSet() *set.Set[L] {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.g.LabelsSet()
}

// HasEdge checks if the graph contains the edge e.
func (g *Graph[T, L]) HasEdge(e graph.Edge[T, L]) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.g.HasEdge(e)
}

// HasVertex checks if the graph contains the vertex v.
func (g *Graph[T, L]) HasVertex(v T) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.g.HasVertex(v)
}

// HasLabel checks if the graph contains the label l.
func (g *Graph[T, L]) HasLabel(l L) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.g.HasLabel(l)
}
