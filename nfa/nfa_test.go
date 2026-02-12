package nfa

import "testing"

type nstate int

const (
	s0 nstate = iota
	s1
	s2
	s3
)

type deltaTable struct {
	bySym map[nstate]map[byte][]nstate
	eps   map[nstate][]nstate
}

func (t deltaTable) Delta(state nstate, symbol byte) []nstate {
	row := t.bySym[state]
	if row == nil {
		return nil
	}
	return row[symbol]
}

func (t deltaTable) Epsilon(state nstate) []nstate {
	return t.eps[state]
}

func TestAcceptsWithEpsilon(t *testing.T) {
	table := deltaTable{
		bySym: map[nstate]map[byte][]nstate{
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
	n := New(Config[nstate, byte]{
		States:    []nstate{s0, s1, s2, s3},
		Alphabet:  []byte{'a', 'b'},
		Start:     s0,
		Accepting: []nstate{s2},
		Deltaer:   table,
	})
	if !n.Accepts([]byte{'b'}) {
		t.Fatalf("expected acceptance via epsilon transition")
	}
	if !n.Accepts([]byte{'a', 'b'}) {
		t.Fatalf("expected acceptance for ab")
	}
	if n.Accepts(nil) {
		t.Fatalf("expected empty word to be rejected")
	}
}

func TestDeltaStarSet(t *testing.T) {
	table := deltaTable{
		bySym: map[nstate]map[byte][]nstate{
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
	n := New(Config[nstate, byte]{
		States:    []nstate{s0, s1, s2},
		Alphabet:  []byte{'a'},
		Start:     s0,
		Accepting: []nstate{s2},
		Deltaer:   table,
	})
	end := n.DeltaStarSet(n.StartSet(), []byte{'a', 'a'})
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
