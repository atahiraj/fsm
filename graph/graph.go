package graph

// Edge is an element of the edge relation E ∈ V×Λ×V.
// It is returned by enumeration APIs as a value.
type Edge[T comparable, L comparable] struct {
	From, To T
	Label    L
}

// Graph is a finite directed labeled graph.
// Mathematically a triple (V, Λ, E) where
//   - V is the set of vertices,
//   - Λ is the set of edge labels (alphabet of relation names),
//   - E ∈ V × Λ × V is the edge relation.
//
// Use New to create a new graph.
type Graph[T comparable, L comparable] struct {
	adj map[T]map[L]map[T]struct{}
}

// New constructs an empty labeled digraph (V,Λ,E) with V=∅, E=∅.
// Λ is implicit in the type parameter L; it grows as labels are inserted.
func New[T comparable, L comparable]() *Graph[T, L] {
	return &Graph[T, L]{
		adj: make(map[T]map[L]map[T]struct{}),
	}
}

// AddVertex inserts v into V if absent.
// Returns true iff v was not already present.
func (g *Graph[T, L]) AddVertex(v T) bool {
	if _, ok := g.adj[v]; ok {
		return false
	}
	g.adj[v] = make(map[L]map[T]struct{})
	return true
}

// AddEdge inserts the edge into E and ensures both endpoints are in V.
// Returns true if either vertex or the edge were not added.
func (g *Graph[T, L]) AddEdge(from, to T, label L) bool {
	addedFrom := g.AddVertex(from)
	addedTo := g.AddVertex(to)
	byLabel := g.adj[from]
	if byLabel == nil {
		byLabel = make(map[L]map[T]struct{})
		g.adj[from] = byLabel
	}
	tos := byLabel[label]
	if tos == nil {
		tos = make(map[T]struct{})
		byLabel[label] = tos
	}
	_, labelExists := tos[to]
	tos[to] = struct{}{}
	return addedFrom || addedTo || !labelExists
}

// Clone creates a deep copy of the graph.
func (g *Graph[T, L]) Clone() *Graph[T, L] {
	clone := New[T, L]()
	for from, byLabel := range g.adj {
		for label, tos := range byLabel {
			for to := range tos {
				clone.AddEdge(from, to, label)
			}
		}
	}
	return clone
}

// Delta returns the one step adjacent vertices for a given vertex and label.
func (g *Graph[T, L]) Delta(from T, label L) []T {
	byLabel, ok := g.adj[from]
	if !ok || byLabel == nil {
		return nil
	}
	tos, ok := byLabel[label]
	if !ok || tos == nil {
		return nil
	}
	out := make([]T, 0, len(tos))
	for to := range tos {
		out = append(out, to)
	}
	return out
}

// Vertices enumerates the vertex set V.
func (g *Graph[T, L]) Vertices() []T {
	vs := make([]T, 0, len(g.adj))
	for v := range g.adj {
		vs = append(vs, v)
	}
	return vs
}

// Edges enumerates the edge relation E as values.
func (g *Graph[T, L]) Edges() []Edge[T, L] {
	edges := make([]Edge[T, L], 0)
	for from, byLabel := range g.adj {
		for label, tos := range byLabel {
			for to := range tos {
				edges = append(edges, Edge[T, L]{From: from, To: to, Label: label})
			}
		}
	}
	return edges
}

// Labels returns all labels.
func (g *Graph[T, L]) Labels() []L {
	labels := make(map[L]struct{})
	for _, byLabel := range g.adj {
		for label := range byLabel {
			labels[label] = struct{}{}
		}
	}
	out := make([]L, 0, len(labels))
	for label := range labels {
		out = append(out, label)
	}
	return out
}

// HasEdge checks if the graph contains the edge e.
func (g *Graph[T, L]) HasEdge(e Edge[T, L]) bool {
	byLabel := g.adj[e.From]
	if byLabel == nil {
		return false
	}
	tos := byLabel[e.Label]
	if tos == nil {
		return false
	}
	_, ok := tos[e.To]
	return ok
}

// HasVertex checks if the graph contains the vertex v.
func (g *Graph[T, L]) HasVertex(v T) bool {
	_, ok := g.adj[v]
	return ok
}

// HasLabel checks if the graph contains the label l.
func (g *Graph[T, L]) HasLabel(l L) bool {
	for _, byLabel := range g.adj {
		if _, ok := byLabel[l]; ok {
			return true
		}
	}
	return false
}
