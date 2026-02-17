package dfa_test

import (
	"sync"
	"testing"

	"github.com/stnhrsprkwns/fsm/dfa"
)

type state int

func (s state) Key() int { return int(s) }

type symbol byte

func (s symbol) Key() byte { return byte(s) }

const (
	even state = iota
	odd
)

type deltaTable map[state]map[symbol]state

func (t deltaTable) Delta(s int, a byte) (int, bool) {
	row := t[state(s)]
	if row == nil {
		return 0, false
	}
	next, ok := row[symbol(a)]
	return next.Key(), ok
}

func word(xs ...byte) []symbol {
	out := make([]symbol, 0, len(xs))
	for _, x := range xs {
		out = append(out, symbol(x))
	}
	return out
}

func evenOnesDFA() *dfa.DFA[state, symbol, int, byte] {
	table := deltaTable{
		even: {'0': even, '1': odd},
		odd:  {'0': odd, '1': even},
	}
	return dfa.New(dfa.Config[state, symbol, int, byte]{
		States:    []state{even, odd},
		Alphabet:  []symbol{'0', '1'},
		Start:     even,
		Accepting: []state{even},
		Deltaer:   table,
	})
}

func TestAtomicDFAAccepts(t *testing.T) {
	a := dfa.NewAtomic(evenOnesDFA())
	if !a.Accepts(word('0', '0')) {
		t.Fatalf("expected acceptance for even ones")
	}
	if a.Accepts(word('0', '1')) {
		t.Fatalf("expected rejection for odd ones")
	}
}

func TestAtomicDFAConcurrent(t *testing.T) {
	a := dfa.NewAtomic(evenOnesDFA())
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = a.Accepts(word('0', '1'))
				a.AddAccepting(odd)
				a.RemoveAccepting(odd)
			}
		}()
	}
	wg.Wait()
}
