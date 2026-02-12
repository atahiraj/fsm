package graph

import (
	"sync"
)

// AtomicGraph is a thread-safe wrapper around Graph.
// Use NewAtomic to construct a non-nil graph.
type AtomicGraph[T comparable, L comparable] struct {
	mu sync.RWMutex
	g  *Graph[T, L]
}

// NewAtomic constructs an empty labeled digraph (V,Λ,E) with V=∅, E=∅.
func NewAtomic[T comparable, L comparable]() *AtomicGraph[T, L] {
	return &AtomicGraph[T, L]{g: New[T, L]()}
}

// AddVertex inserts v into V if absent.
func (g *AtomicGraph[T, L]) AddVertex(v T) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.g.AddVertex(v)
}

// AddEdge inserts the edge into E and ensures both endpoints are in V.
func (g *AtomicGraph[T, L]) AddEdge(from, to T, label L) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.g.AddEdge(from, to, label)
}

// Clone creates a deep copy of the graph.
func (g *AtomicGraph[T, L]) Clone() *Graph[T, L] {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.g.Clone()
}

// Delta returns the one step adjacent vertices for a given vertex and label.
func (g *AtomicGraph[T, L]) Delta(from T, label L) []T {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.g.Delta(from, label)
}

// Vertices enumerates the vertex set V.
func (g *AtomicGraph[T, L]) Vertices() []T {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.g.Vertices()
}

// Edges enumerates the edge relation E as values.
func (g *AtomicGraph[T, L]) Edges() []Edge[T, L] {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.g.Edges()
}

// Labels returns all labels.
func (g *AtomicGraph[T, L]) Labels() []L {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.g.Labels()
}

// HasEdge checks if the graph contains the edge e.
func (g *AtomicGraph[T, L]) HasEdge(e Edge[T, L]) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.g.HasEdge(e)
}

// HasVertex checks if the graph contains the vertex v.
func (g *AtomicGraph[T, L]) HasVertex(v T) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.g.HasVertex(v)
}

// HasLabel checks if the graph contains the label l.
func (g *AtomicGraph[T, L]) HasLabel(l L) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.g.HasLabel(l)
}
