package nfa_test

import (
	"sync"
	"testing"

	"github.com/stnhrsprkwns/fsm/nfa"
)

type nstate int

const (
	s0 nstate = iota
	s1
	s2
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

func simpleNFA() *nfa.NFA[nstate, byte] {
	table := deltaTable{
		bySym: map[nstate]map[byte][]nstate{
			s1: {
				'a': {s1, s2},
			},
		},
		eps: map[nstate][]nstate{
			s0: {s1},
		},
	}
	return nfa.New(nfa.Config[nstate, byte]{
		States:    []nstate{s0, s1, s2},
		Alphabet:  []byte{'a'},
		Start:     s0,
		Accepting: []nstate{s2},
		Deltaer:   table,
	})
}

func TestAtomicNFAAccepts(t *testing.T) {
	a := nfa.NewAtomic(simpleNFA())
	if !a.Accepts([]byte{'a'}) {
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
				_ = a.Accepts([]byte{'a'})
				a.AddAccepting(s1)
				a.RemoveAccepting(s1)
			}
		}()
	}
	wg.Wait()
}
