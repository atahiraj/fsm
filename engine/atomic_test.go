package engine

import (
	"sync"
	"testing"
)

type testFSM struct{}

func (testFSM) Start() int                     { return 0 }
func (testFSM) Step(s int, _ byte) (int, bool) { return s + 1, true }
func (testFSM) IsAccepting(s int) bool         { return s%2 == 0 }

type noopTransitionHooks struct{}

func (noopTransitionHooks) OnTransition(int, int, byte) {}

func TestAtomicEngineConcurrent(t *testing.T) {
	e := New[int, byte](testFSM{}, noopTransitionHooks{})
	a := NewAtomic(e)
	a.Reset()

	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				a.Step('a')
				_ = a.Accepting()
				_ = a.Cur()
			}
		}()
	}
	wg.Wait()
}
