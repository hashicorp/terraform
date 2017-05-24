package local

import (
	"sync"

	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/terraform"
)

// StateHook is a hook that continuously updates the state by calling
// WriteState on a state.State.
type StateHook struct {
	terraform.NilHook
	sync.Mutex

	State state.State
}

func (h *StateHook) PostStateUpdate(
	s *terraform.State) (terraform.HookAction, error) {
	h.Lock()
	defer h.Unlock()

	if h.State != nil {
		// Write the new state
		if err := h.State.WriteState(s); err != nil {
			return terraform.HookActionHalt, err
		}
	}

	// Continue forth
	return terraform.HookActionContinue, nil
}
