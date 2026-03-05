// Package set provides a small insertion-ordered set for the public automata.
package set

// Set is an insertion-ordered set. Its zero value is ready for use.
type Set[T comparable] struct {
	values map[T]struct{}
	order  []T
}

// New constructs a set containing values in first-insertion order.
func New[T comparable](values ...T) *Set[T] {
	s := &Set[T]{}
	s.Add(values...)
	return s
}

// Add inserts values and reports whether the set changed.
func (s *Set[T]) Add(values ...T) bool {
	if s == nil {
		return false
	}
	if s.values == nil {
		s.values = make(map[T]struct{}, len(values))
	}
	changed := false
	for _, value := range values {
		if _, exists := s.values[value]; exists {
			continue
		}
		s.values[value] = struct{}{}
		s.order = append(s.order, value)
		changed = true
	}
	return changed
}

// Remove removes values and reports whether the set changed.
func (s *Set[T]) Remove(values ...T) bool {
	if s == nil || len(s.values) == 0 {
		return false
	}
	changed := false
	for _, value := range values {
		if _, exists := s.values[value]; !exists {
			continue
		}
		delete(s.values, value)
		changed = true
	}
	if changed {
		order := s.order[:0]
		for _, value := range s.order {
			if s.Has(value) {
				order = append(order, value)
			}
		}
		s.order = order
	}
	return changed
}

// Has reports whether value belongs to the set.
func (s *Set[T]) Has(value T) bool {
	if s == nil {
		return false
	}
	_, exists := s.values[value]
	return exists
}

// Len returns the number of values in the set.
func (s *Set[T]) Len() int {
	if s == nil {
		return 0
	}
	return len(s.values)
}

// IsEmpty reports whether the set is empty.
func (s *Set[T]) IsEmpty() bool { return s.Len() == 0 }

// Clear removes every value.
func (s *Set[T]) Clear() {
	if s == nil {
		return
	}
	clear(s.values)
	s.order = nil
}

// Slice returns an independent slice in insertion order.
func (s *Set[T]) Slice() []T {
	if s == nil || len(s.values) == 0 {
		return []T{}
	}
	out := make([]T, 0, len(s.values))
	for _, value := range s.order {
		if s.Has(value) {
			out = append(out, value)
		}
	}
	return out
}

// Clone returns an independent copy preserving insertion order.
func (s *Set[T]) Clone() *Set[T] {
	if s == nil {
		return New[T]()
	}
	return New(s.Slice()...)
}

// Equals reports whether two sets contain the same values.
func (s *Set[T]) Equals(other *Set[T]) bool {
	if s.Len() != other.Len() {
		return false
	}
	for _, value := range s.Slice() {
		if !other.Has(value) {
			return false
		}
	}
	return true
}

// Intersects reports whether two sets share a value.
func (s *Set[T]) Intersects(other *Set[T]) bool {
	if s == nil || other == nil {
		return false
	}
	for _, value := range s.Slice() {
		if other.Has(value) {
			return true
		}
	}
	return false
}

// AddSet inserts all values from other in its insertion order.
func (s *Set[T]) AddSet(other *Set[T]) bool {
	if s == nil || other == nil {
		return false
	}
	return s.Add(other.Slice()...)
}

// RemoveSet removes all values found in other.
func (s *Set[T]) RemoveSet(other *Set[T]) bool {
	if s == nil || other == nil {
		return false
	}
	return s.Remove(other.Slice()...)
}

// Union returns a new set containing values from both sets.
func (s *Set[T]) Union(other *Set[T]) *Set[T] {
	out := s.Clone()
	out.AddSet(other)
	return out
}

// Intersection returns a new set containing shared values.
func (s *Set[T]) Intersection(other *Set[T]) *Set[T] {
	out := New[T]()
	if s == nil || other == nil {
		return out
	}
	for _, value := range s.Slice() {
		if other.Has(value) {
			out.Add(value)
		}
	}
	return out
}

// Difference returns values from s that are not in other.
func (s *Set[T]) Difference(other *Set[T]) *Set[T] {
	out := New[T]()
	if s == nil {
		return out
	}
	for _, value := range s.Slice() {
		if !other.Has(value) {
			out.Add(value)
		}
	}
	return out
}

// Iter returns an insertion-ordered iterator.
func (s *Set[T]) Iter() func(func(T) bool) {
	return func(yield func(T) bool) {
		for _, value := range s.Slice() {
			if !yield(value) {
				return
			}
		}
	}
}
