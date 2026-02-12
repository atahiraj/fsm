package engine

import (
	"sync"
	"testing"
)

type testFSM struct{}

func (testFSM) Start() int             { return 0 }
func (testFSM) Step(s int, _ byte) int { return s + 1 }
func (testFSM) IsAccepting(s int) bool { return s%2 == 0 }

type noopObserver struct{}

func (noopObserver) OnStep(int, struct{}, int, byte, struct{}) {}

func TestAtomicEngineConcurrent(t *testing.T) {
	e := New[int, byte, struct{}, struct{}](testFSM{}, noopObserver{})
	a := NewAtomic(e)
	a.Reset(struct{}{})

	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				a.Step('a', struct{}{})
				_ = a.Accepting()
				_ = a.Cur()
			}
		}()
	}
	wg.Wait()
}
