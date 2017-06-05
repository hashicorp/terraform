package local

import (
	"sync"
	"time"

	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/terraform"
)

// interval between forced PersistState calls by StateHook
const persistStateHookInterval = 10 * time.Second

// StateHook is a hook that continuously updates the state by calling
// WriteState on a state.State.
type StateHook struct {
	terraform.NilHook
	sync.Mutex

	// lastPersist is the time of the last call to PersistState, for periodic
	// updates to remote state. PostStateUpdate will force a call PersistState
	// if it has been more that persistStateHookInterval since the last call to
	// PersistState.
	lastPersist time.Time

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

		// periodically persist the state
		if time.Since(h.lastPersist) > persistStateHookInterval {
			if err := h.persistState(); err != nil {
				return terraform.HookActionHalt, err
			}
		}
	}

	// Continue forth
	return terraform.HookActionContinue, nil
}

func (h *StateHook) persistState() error {
	if h.State != nil {
		err := h.State.PersistState()
		h.lastPersist = time.Now()
		return err
	}
	return nil
}
