package dfa

import "testing"

type state int

const (
	even state = iota
	odd
	outside
)

type deltaTable map[state]map[byte]state

func (t deltaTable) Delta(s state, a byte) (state, bool) {
	row := t[s]
	if row == nil {
		return 0, false
	}
	next, ok := row[a]
	return next, ok
}

func evenOnesDFA() *DFA[state, byte] {
	table := deltaTable{
		even: {'0': even, '1': odd},
		odd:  {'0': odd, '1': even},
	}
	return New(Config[state, byte]{
		States:    []state{even, odd},
		Alphabet:  []byte{'0', '1'},
		Start:     even,
		Accepting: []state{even},
		Deltaer:   table,
	})
}

func TestAccepts(t *testing.T) {
	d := evenOnesDFA()
	cases := []struct {
		name string
		word []byte
		want bool
	}{
		{name: "empty", word: nil, want: true},
		{name: "one", word: []byte{'1'}, want: false},
		{name: "two", word: []byte{'1', '1'}, want: true},
		{name: "one-zero-one", word: []byte{'1', '0', '1'}, want: true},
		{name: "three", word: []byte{'1', '1', '1'}, want: false},
	}
	for _, tc := range cases {
		if got := d.Accepts(tc.word); got != tc.want {
			t.Fatalf("Accepts(%s) = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestDeltaRejectsUnknownState(t *testing.T) {
	table := deltaTable{
		even: {'0': outside},
	}
	d := New(Config[state, byte]{
		States:    []state{even, odd},
		Alphabet:  []byte{'0'},
		Start:     even,
		Accepting: []state{even},
		Deltaer:   table,
	})
	if _, ok := d.Delta(even, '0'); ok {
		t.Fatalf("Delta should reject transitions to states outside Q")
	}
}

func TestDeltaStarMissingTransition(t *testing.T) {
	table := deltaTable{
		even: {'0': even},
	}
	d := New(Config[state, byte]{
		States:    []state{even},
		Alphabet:  []byte{'0', '1'},
		Start:     even,
		Accepting: []state{even},
		Deltaer:   table,
	})
	if _, ok := d.DeltaStar(even, []byte{'1'}); ok {
		t.Fatalf("DeltaStar should report missing transitions")
	}
	if d.Accepts([]byte{'1'}) {
		t.Fatalf("Accepts should be false when a transition is missing")
	}
}
