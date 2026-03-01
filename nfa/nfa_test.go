package nfa

import "testing"

type nstate int

func (s nstate) Key() int { return int(s) }

type ninput byte

func (s ninput) Key() byte { return byte(s) }

const (
	s0 nstate = iota
	s1
	s2
	s3
)

type deltaTable struct {
	bySym map[nstate]map[ninput][]nstate
	eps   map[nstate][]nstate
}

func (t deltaTable) Delta(state int, input byte) []int {
	row := t.bySym[nstate(state)]
	if row == nil {
		return nil
	}
	next := row[ninput(input)]
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

func nword(xs ...byte) []ninput {
	out := make([]ninput, 0, len(xs))
	for _, x := range xs {
		out = append(out, ninput(x))
	}
	return out
}

func TestAcceptsWithEpsilon(t *testing.T) {
	table := deltaTable{
		bySym: map[nstate]map[ninput][]nstate{
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
	n := New(Config[nstate, ninput, int, byte]{
		States:    []nstate{s0, s1, s2, s3},
		Alphabet:  []ninput{'a', 'b'},
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
		bySym: map[nstate]map[ninput][]nstate{
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
	n := New(Config[nstate, ninput, int, byte]{
		States:    []nstate{s0, s1, s2},
		Alphabet:  []ninput{'a'},
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
