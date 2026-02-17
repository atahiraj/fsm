package nfa_test

import (
	"sync"
	"testing"

	"github.com/stnhrsprkwns/fsm/nfa"
)

type nstate int

func (s nstate) Key() int { return int(s) }

type nsymbol byte

func (s nsymbol) Key() byte { return byte(s) }

const (
	s0 nstate = iota
	s1
	s2
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

func simpleNFA() *nfa.NFA[nstate, nsymbol, int, byte] {
	table := deltaTable{
		bySym: map[nstate]map[nsymbol][]nstate{
			s1: {
				'a': {s1, s2},
			},
		},
		eps: map[nstate][]nstate{
			s0: {s1},
		},
	}
	return nfa.New(nfa.Config[nstate, nsymbol, int, byte]{
		States:    []nstate{s0, s1, s2},
		Alphabet:  []nsymbol{'a'},
		Start:     s0,
		Accepting: []nstate{s2},
		Deltaer:   table,
	})
}

func TestAtomicNFAAccepts(t *testing.T) {
	a := nfa.NewAtomic(simpleNFA())
	if !a.Accepts(nword('a')) {
		t.Fatalf("expected acceptance via epsilon transition")
	}
	if a.Accepts(nil) {
		t.Fatalf("expected empty word to be rejected")
	}
}

func TestAtomicNFAConcurrent(t *testing.T) {
	a := nfa.NewAtomic(simpleNFA())
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = a.Accepts(nword('a'))
				a.AddAccepting(s1)
				a.RemoveAccepting(s1)
			}
		}()
	}
	wg.Wait()
}
