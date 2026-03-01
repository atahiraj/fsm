package transitionhooks

import "testing"

type compiledState struct {
	key string
}

type compiledSymbol string

type recordingCompiledExecutor struct {
	calls int
}

func (e *recordingCompiledExecutor) Execute(
	f func(from compiledState, to compiledState, sym compiledSymbol),
	from compiledState,
	to compiledState,
	sym compiledSymbol,
) {
	e.calls++
	f(from, to, sym)
}

func TestNewCompiledDispatchesMatchingBucketsInOrder(t *testing.T) {
	var got []string
	hooks := NewCompiled[compiledState, compiledSymbol, string](
		DefaultExecutor[compiledState, compiledSymbol]{},
		func(s compiledState) string { return s.key },
		Group[compiledState, compiledSymbol, string]{
			Registrations: []Registration[compiledState, compiledSymbol, string]{
				{
					Mode: MatchAny,
					Callback: func(from compiledState, to compiledState, sym compiledSymbol) {
						got = append(got, "any")
					},
				},
				{
					Mode:    MatchFromKey,
					FromKey: "A",
					Callback: func(from compiledState, to compiledState, sym compiledSymbol) {
						got = append(got, "from:A")
					},
				},
				{
					Mode:  MatchToKey,
					ToKey: "B",
					Callback: func(from compiledState, to compiledState, sym compiledSymbol) {
						got = append(got, "to:B")
					},
				},
				{
					Mode:    MatchFromToKey,
					FromKey: "A",
					ToKey:   "B",
					Callback: func(from compiledState, to compiledState, sym compiledSymbol) {
						got = append(got, "A->B")
					},
				},
				{
					Mode: MatchPredicate,
					Predicate: func(from compiledState, to compiledState, sym compiledSymbol) bool {
						return sym == "go"
					},
					Callback: func(from compiledState, to compiledState, sym compiledSymbol) {
						got = append(got, "predicate:go")
					},
				},
			},
		},
	)

	hooks.OnTransition(compiledState{key: "A"}, compiledState{key: "B"}, "go")

	want := []string{"any", "from:A", "to:B", "A->B", "predicate:go"}
	if len(got) != len(want) {
		t.Fatalf("got %d callbacks, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestNewCompiledPreservesRegistrationOrderWithinBucket(t *testing.T) {
	var got []string
	hooks := NewCompiled[compiledState, compiledSymbol, string](
		DefaultExecutor[compiledState, compiledSymbol]{},
		func(s compiledState) string { return s.key },
		Group[compiledState, compiledSymbol, string]{
			Registrations: []Registration[compiledState, compiledSymbol, string]{
				{
					Mode:    MatchFromToKey,
					FromKey: "A",
					ToKey:   "B",
					Callback: func(from compiledState, to compiledState, sym compiledSymbol) {
						got = append(got, "first")
					},
				},
				{
					Mode:    MatchFromToKey,
					FromKey: "A",
					ToKey:   "B",
					Callback: func(from compiledState, to compiledState, sym compiledSymbol) {
						got = append(got, "second")
					},
				},
			},
		},
	)

	hooks.OnTransition(compiledState{key: "A"}, compiledState{key: "B"}, "x")

	want := []string{"first", "second"}
	if len(got) != len(want) {
		t.Fatalf("got %d callbacks, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestNewCompiledGroupGuard(t *testing.T) {
	var got []string
	hooks := NewCompiled[compiledState, compiledSymbol, string](
		DefaultExecutor[compiledState, compiledSymbol]{},
		func(s compiledState) string { return s.key },
		Group[compiledState, compiledSymbol, string]{
			Guard: func(fromKey string, toKey string) bool {
				return fromKey != toKey
			},
			Registrations: []Registration[compiledState, compiledSymbol, string]{
				{
					Mode: MatchAny,
					Callback: func(from compiledState, to compiledState, sym compiledSymbol) {
						got = append(got, "guarded")
					},
				},
			},
		},
	)

	hooks.OnTransition(compiledState{key: "A"}, compiledState{key: "A"}, "x")
	hooks.OnTransition(compiledState{key: "A"}, compiledState{key: "B"}, "x")

	want := []string{"guarded"}
	if len(got) != len(want) {
		t.Fatalf("got %d callbacks, want %d (%v)", len(got), len(want), got)
	}
	if got[0] != want[0] {
		t.Fatalf("got[0] = %q, want %q", got[0], want[0])
	}
}

func TestNewCompiledExecutorCalledOnlyForMatchedCallbacks(t *testing.T) {
	exec := &recordingCompiledExecutor{}
	hooks := NewCompiled[compiledState, compiledSymbol, string](
		exec,
		func(s compiledState) string { return s.key },
		Group[compiledState, compiledSymbol, string]{
			Registrations: []Registration[compiledState, compiledSymbol, string]{
				{
					Mode: MatchAny,
					Callback: func(from compiledState, to compiledState, sym compiledSymbol) {
					},
				},
				{
					Mode:    MatchFromToKey,
					FromKey: "A",
					ToKey:   "B",
					Callback: func(from compiledState, to compiledState, sym compiledSymbol) {
					},
				},
				{
					Mode: MatchPredicate,
					Predicate: func(from compiledState, to compiledState, sym compiledSymbol) bool {
						return false
					},
					Callback: func(from compiledState, to compiledState, sym compiledSymbol) {
					},
				},
			},
		},
	)

	hooks.OnTransition(compiledState{key: "A"}, compiledState{key: "B"}, "x")
	if exec.calls != 2 {
		t.Fatalf("executor calls = %d, want 2", exec.calls)
	}
}
