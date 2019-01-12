package local

import (
	"testing"

	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/states/statemgr"
	"github.com/hashicorp/terraform/terraform"
)

func TestStateHook_impl(t *testing.T) {
	var _ terraform.Hook = new(StateHook)
}

func TestStateHook(t *testing.T) {
	is := statemgr.NewTransientInMemory(nil)
	var hook terraform.Hook = &StateHook{StateMgr: is}

	s := state.TestStateInitial()
	action, err := hook.PostStateUpdate(s)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if action != terraform.HookActionContinue {
		t.Fatalf("bad: %v", action)
	}
	if !is.State().Equal(s) {
		t.Fatalf("bad state: %#v", is.State())
	}
}
