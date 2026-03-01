package dfa

import "testing"

type state int

func (s state) Key() int { return int(s) }

type input byte

func (s input) Key() byte { return byte(s) }

const (
	even state = iota
	odd
	outside
)

type deltaTable map[state]map[input]state

func (t deltaTable) Delta(s int, a byte) (int, bool) {
	row := t[state(s)]
	if row == nil {
		return 0, false
	}
	next, ok := row[input(a)]
	return next.Key(), ok
}

func word(xs ...byte) []input {
	out := make([]input, 0, len(xs))
	for _, x := range xs {
		out = append(out, input(x))
	}
	return out
}

func evenOnesDFA() *DFA[state, input, int, byte] {
	table := deltaTable{
		even: {'0': even, '1': odd},
		odd:  {'0': odd, '1': even},
	}
	return New(Config[state, input, int, byte]{
		States:    []state{even, odd},
		Alphabet:  []input{'0', '1'},
		Start:     even,
		Accepting: []state{even},
		Deltaer:   table,
	})
}

func TestAccepts(t *testing.T) {
	d := evenOnesDFA()
	cases := []struct {
		name string
		word []input
		want bool
	}{
		{name: "empty", word: nil, want: true},
		{name: "one", word: word('1'), want: false},
		{name: "two", word: word('1', '1'), want: true},
		{name: "one-zero-one", word: word('1', '0', '1'), want: true},
		{name: "three", word: word('1', '1', '1'), want: false},
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
	d := New(Config[state, input, int, byte]{
		States:    []state{even, odd},
		Alphabet:  []input{'0'},
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
	d := New(Config[state, input, int, byte]{
		States:    []state{even},
		Alphabet:  []input{'0', '1'},
		Start:     even,
		Accepting: []state{even},
		Deltaer:   table,
	})
	if _, ok := d.DeltaStar(even, word('1')); ok {
		t.Fatalf("DeltaStar should report missing transitions")
	}
	if d.Accepts(word('1')) {
		t.Fatalf("Accepts should be false when a transition is missing")
	}
}
