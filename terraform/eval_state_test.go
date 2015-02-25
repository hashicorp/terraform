package terraform

import (
	"sync"
	"testing"
)

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
