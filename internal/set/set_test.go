package set

import (
	"reflect"
	"testing"
)

func TestSetInsertionOrderAndReinsert(t *testing.T) {
	values := New(2, 1, 2, 3)
	if got, want := values.Slice(), []int{2, 1, 3}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Slice() = %v, want %v", got, want)
	}
	values.Remove(1)
	values.Add(1)
	if got, want := values.Slice(), []int{2, 3, 1}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Slice() after reinsert = %v, want %v", got, want)
	}
}

func TestSetNilReceiverIsSafe(t *testing.T) {
	var values *Set[int]
	if values.Add(1) || values.Remove(1) || values.Has(1) || values.Len() != 0 || !values.IsEmpty() {
		t.Fatal("nil set has unexpected state")
	}
	values.Clear()
	if got := values.Slice(); len(got) != 0 {
		t.Fatalf("nil Slice() = %v", got)
	}
	if !values.Equals(nil) || values.Intersects(New(1)) || values.AddSet(New(1)) || values.RemoveSet(New(1)) {
		t.Fatal("nil set operation returned an unexpected result")
	}
	if got := values.Clone().Slice(); len(got) != 0 {
		t.Fatalf("nil Clone() = %v", got)
	}
	if got := values.Union(New(1)).Slice(); !reflect.DeepEqual(got, []int{1}) {
		t.Fatalf("nil Union() = %v", got)
	}
	if got := values.Intersection(New(1)).Slice(); len(got) != 0 {
		t.Fatalf("nil Intersection() = %v", got)
	}
	if got := values.Difference(New(1)).Slice(); len(got) != 0 {
		t.Fatalf("nil Difference() = %v", got)
	}
	values.Iter()(func(int) bool {
		t.Fatal("nil Iter() yielded a value")
		return false
	})
}
