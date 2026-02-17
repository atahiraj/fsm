package nfa

import "testing"

type nstate int

func (s nstate) Key() int { return int(s) }

type nsymbol byte

func (s nsymbol) Key() byte { return byte(s) }

const (
	s0 nstate = iota
	s1
	s2
	s3
)

type deltaTable struct {
	bySym map[nstate]map[nsymbol][]nstate
	eps   map[nstate][]nstate
}

func (t deltaTable) Delta(state int, symbol byte) []int {
	row := t.bySym[nstate(state)]
	if row == nil {
		return nil
	}
	next := row[nsymbol(symbol)]
	out := make([]int, 0, len(next))
	for _, s := range next {
		out = append(out, s.Key())
	}
	return out
}

func (t deltaTable) Epsilon(state int) []int {
	next := t.eps[nstate(state)]
	out := make([]int, 0, len(next))
	for _, s := range next {
		out = append(out, s.Key())
	}
	return out
}

func nword(xs ...byte) []nsymbol {
	out := make([]nsymbol, 0, len(xs))
	for _, x := range xs {
		out = append(out, nsymbol(x))
	}
	return out
}

func TestAcceptsWithEpsilon(t *testing.T) {
	table := deltaTable{
		bySym: map[nstate]map[nsymbol][]nstate{
			s1: {
				'a': {s1, s2},
				'b': {s2},
			},
			s2: {
				'b': {s2},
				'a': {s3},
			},
			s3: {
				'b': {s3},
			},
		},
		eps: map[nstate][]nstate{
			s0: {s1},
		},
	}
	n := New(Config[nstate, nsymbol, int, byte]{
		States:    []nstate{s0, s1, s2, s3},
		Alphabet:  []nsymbol{'a', 'b'},
		Start:     s0,
		Accepting: []nstate{s2},
		Deltaer:   table,
	})
	if !n.Accepts(nword('b')) {
		t.Fatalf("expected acceptance via epsilon transition")
	}
	if !n.Accepts(nword('a', 'b')) {
		t.Fatalf("expected acceptance for ab")
	}
	if n.Accepts(nil) {
		t.Fatalf("expected empty word to be rejected")
	}
}

func TestDeltaStarSet(t *testing.T) {
	table := deltaTable{
		bySym: map[nstate]map[nsymbol][]nstate{
			s0: {
				'a': {s0, s1},
			},
			s1: {
				'a': {s2},
			},
		},
		eps: map[nstate][]nstate{
			s0: {s1},
		},
	}
	n := New(Config[nstate, nsymbol, int, byte]{
		States:    []nstate{s0, s1, s2},
		Alphabet:  []nsymbol{'a'},
		Start:     s0,
		Accepting: []nstate{s2},
		Deltaer:   table,
	})
	end := n.DeltaStarSet(n.StartSet(), nword('a', 'a'))
	found := false
	for _, s := range end {
		if s == s2 {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected s2 to be reachable after aa")
	}
}
