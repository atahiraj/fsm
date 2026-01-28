package atomic

import (
	"sync"
	"testing"

	"github.com/stnhrsprkwns/fsm/pkg/dfa"
)

type state int

const (
	even state = iota
	odd
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

func evenOnesDFA() *dfa.DFA[state, byte] {
	table := deltaTable{
		even: {'0': even, '1': odd},
		odd:  {'0': odd, '1': even},
	}
	return dfa.New(dfa.Config[state, byte]{
		States:    []state{even, odd},
		Alphabet:  []byte{'0', '1'},
		Start:     even,
		Accepting: []state{even},
		Deltaer:   table,
	})
}

func TestAtomicDFAAccepts(t *testing.T) {
	a := New(evenOnesDFA())
	if !a.Accepts([]byte("00")) {
		t.Fatalf("expected acceptance for even ones")
	}
	if a.Accepts([]byte("01")) {
		t.Fatalf("expected rejection for odd ones")
	}
}

func TestAtomicDFAConcurrent(t *testing.T) {
	a := New(evenOnesDFA())
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = a.Accepts([]byte("01"))
				a.AddAccepting(odd)
				a.RemoveAccepting(odd)
			}
		}()
	}
	wg.Wait()
}
