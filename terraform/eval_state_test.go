package terraform

import (
	"sync"
	"testing"
)

func TestEvalRequireState(t *testing.T) {
	ctx := new(MockEvalContext)

	cases := []struct {
		State *InstanceState
		Exit  bool
	}{
		{
			nil,
			true,
		},
		{
			&InstanceState{},
			true,
		},
		{
			&InstanceState{ID: "foo"},
			false,
		},
	}

	var exitVal EvalEarlyExitError
	for _, tc := range cases {
		node := &EvalRequireState{State: &tc.State}
		_, err := node.Eval(ctx)
		if tc.Exit {
			if err != exitVal {
				t.Fatalf("should've exited: %#v", tc.State)
			}

			continue
		}
		if !tc.Exit && err != nil {
			t.Fatalf("shouldn't exit: %#v", tc.State)
		}
		if err != nil {
			t.Fatalf("err: %s", err)
		}
	}
}

func TestEvalUpdateStateHook(t *testing.T) {
	mockHook := new(MockHook)

	ctx := new(MockEvalContext)
	ctx.HookHook = mockHook
	ctx.StateState = &State{Serial: 42}
	ctx.StateLock = new(sync.RWMutex)

	node := &EvalUpdateStateHook{}
	if _, err := node.Eval(ctx); err != nil {
		t.Fatalf("err: %s", err)
	}

	if !mockHook.PostStateUpdateCalled {
		t.Fatal("should call PostStateUpdate")
	}
	if mockHook.PostStateUpdateState.Serial != 42 {
		t.Fatalf("bad: %#v", mockHook.PostStateUpdateState)
	}
}
