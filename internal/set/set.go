package set

// Set is a generic hash set backed by map[T]struct{}.
// Use New() to obtain a mutable set.
type Set[T comparable] struct{ m map[T]struct{} }

// New constructs a Set containing elems.
func New[T comparable](elems ...T) *Set[T] {
	s := newWithHint[T](len(elems))
	for _, e := range elems {
		s.m[e] = struct{}{}
	}
	return s
}

func newWithHint[T comparable](hint int) *Set[T] {
	return &Set[T]{m: make(map[T]struct{}, hint)}
}

// Add inserts each x into the set. Duplicates are ignored.
// Returns true if any new elements were added.
// On a nil receiver, it is a no-op and returns false.
func (s *Set[T]) Add(xs ...T) bool {
	if s.m == nil {
		s.m = make(map[T]struct{}, len(xs))
	}
	before := len(s.m)
	for _, x := range xs {
		s.m[x] = struct{}{}
	}
	return before != len(s.m)
}

// Remove deletes each x from the set. Missing values are ignored.
// Returns true if any elements were removed.
// On a nil receiver, it is a no-op and returns false.
func (s *Set[T]) Remove(xs ...T) bool {
	if len(s.m) == 0 {
		return false
	}
	before := len(s.m)
	for _, x := range xs {
		delete(s.m, x)
	}
	return before != len(s.m)
}

// AddSet inserts into s every element from o.
// Returns true if any new elements were added.
// On a nil receiver, it is a no-op and returns false.
// A nil o is treated as empty.
func (s *Set[T]) AddSet(o *Set[T]) bool {
	if o == nil || len(o.m) == 0 {
		return false
	}
	if s.m == nil {
		s.m = make(map[T]struct{}, len(o.m))
	}
	before := len(s.m)
	for k := range o.m {
		s.m[k] = struct{}{}
	}
	return before != len(s.m)
}

// RemoveSet deletes from s every element that appears in o.
// Returns true if any elements were removed.
// On a nil receiver, it is a no-op and returns false.
// A nil o is treated as empty.
func (s *Set[T]) RemoveSet(o *Set[T]) bool {
	if len(s.m) == 0 || o == nil || len(o.m) == 0 {
		return false
	}
	before := len(s.m)
	for k := range o.m {
		delete(s.m, k)
	}
	return before != len(s.m)
}

// Has reports whether x is in the set.
// Safe on a nil receiver (returns false).
func (s *Set[T]) Has(x T) bool {
	_, ok := s.m[x]
	return ok
}

// Len returns the number of elements in the set.
// Safe on a nil receiver (returns 0).
func (s *Set[T]) Len() int {
	return len(s.m)
}

// Cardinality of Len.
func (s *Set[T]) Cardinality() int { return s.Len() }

// IsEmpty reports whether the set is empty.
// Safe on a nil receiver (returns true).
func (s *Set[T]) IsEmpty() bool {
	return len(s.m) == 0
}

// Clear removes all elements from the set.
// On a nil receiver, it is a no-op.
func (s *Set[T]) Clear() {
	clear(s.m)
}

// Equals reports whether the set is equal to another set.
// Nil receivers/args are treated as empty.
func (s *Set[T]) Equals(o *Set[T]) bool {
	lns := 0
	if s != nil {
		lns = len(s.m)
	}
	lno := 0
	if o != nil {
		lno = len(o.m)
	}
	if lns != lno {
		return false
	}
	if lns == 0 { // both empty
		return true
	}
	// s and o non-nil with same length
	for x := range s.m {
		if _, ok := o.m[x]; !ok {
			return false
		}
	}
	return true
}

// Intersects reports whether two sets share at least one element.
// Nil receivers/args are treated as empty.
func (s *Set[T]) Intersects(o *Set[T]) bool {
	if o == nil || len(s.m) == 0 || len(o.m) == 0 {
		return false
	}
	a, b := s, o
	if len(b.m) < len(a.m) {
		a, b = b, a
	}
	for x := range a.m {
		if _, ok := b.m[x]; ok {
			return true
		}
	}
	return false
}

// Slice returns the elements in unspecified order.
// The returned slice is independent from the set.
// Safe on a nil receiver (returns an empty slice).
func (s *Set[T]) Slice() []T {
	n := 0
	if s != nil {
		n = len(s.m)
	}
	out := make([]T, 0, n)
	if n == 0 {
		return out
	}
	for x := range s.m {
		out = append(out, x)
	}
	return out
}

// Clone returns a shallow copy of the set.
// Nil receiver returns an empty set.
func (s *Set[T]) Clone() *Set[T] {
	if len(s.m) == 0 {
		return newWithHint[T](0)
	}
	out := newWithHint[T](len(s.m))
	for x := range s.m {
		out.m[x] = struct{}{}
	}
	return out
}

// Union returns the union of two sets.
// Nil receivers/args are treated as empty.
func (s *Set[T]) Union(o *Set[T]) *Set[T] {
	na, nb := 0, 0
	if s != nil {
		na = len(s.m)
	}
	if o != nil {
		nb = len(o.m)
	}
	out := newWithHint[T](na + nb)
	if na != 0 {
		for x := range s.m {
			out.m[x] = struct{}{}
		}
	}
	if nb != 0 {
		for x := range o.m {
			out.m[x] = struct{}{}
		}
	}
	return out
}

// Intersection returns the intersection of two sets.
// Nil receivers/args or empty inputs yield an empty set.
func (s *Set[T]) Intersection(o *Set[T]) *Set[T] {
	if o == nil || len(s.m) == 0 || len(o.m) == 0 {
		return newWithHint[T](0)
	}
	a, b := s, o
	if len(b.m) < len(a.m) {
		a, b = b, a
	}
	out := newWithHint[T](len(a.m)) // cap to smaller
	for x := range a.m {
		if _, ok := b.m[x]; ok {
			out.m[x] = struct{}{}
		}
	}
	return out
}

// Difference returns the elements in s that are not in o.
// Nil receiver is empty; nil o is treated as empty.
func (s *Set[T]) Difference(o *Set[T]) *Set[T] {
	if len(s.m) == 0 {
		return newWithHint[T](0)
	}
	if len(o.m) == 0 {
		return s.Clone()
	}
	out := newWithHint[T](len(s.m))
	for x := range s.m {
		if _, ok := o.m[x]; !ok {
			out.m[x] = struct{}{}
		}
	}
	return out
}

// Iterate over the elements in the set, calling the yield function for each element.
func (s *Set[T]) Iter() func(func(T) bool) {
	return func(yield func(T) bool) {
		for x := range s.m {
			if !yield(x) {
				return
			}
		}
	}
}
