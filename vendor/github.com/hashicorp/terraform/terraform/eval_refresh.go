package terraform

import (
	"fmt"
	"log"
)

// EvalRefresh is an EvalNode implementation that does a refresh for
// a resource.
type EvalRefresh struct {
	Provider *ResourceProvider
	State    **InstanceState
	Info     *InstanceInfo
	Output   **InstanceState
}

// TODO: test
func (n *EvalRefresh) Eval(ctx EvalContext) (interface{}, error) {
	provider := *n.Provider
	state := *n.State

	// If we have no state, we don't do any refreshing
	if state == nil {
		log.Printf("[DEBUG] refresh: %s: no state, not refreshing", n.Info.Id)
		return nil, nil
	}

	// Call pre-refresh hook
	err := ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PreRefresh(n.Info, state)
	})
	if err != nil {
		return nil, err
	}

	// Refresh!
	state, err = provider.Refresh(n.Info, state)
	if err != nil {
		return nil, fmt.Errorf("%s: %s", n.Info.Id, err.Error())
	}

	// Call post-refresh hook
	err = ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PostRefresh(n.Info, state)
	})
	if err != nil {
		return nil, err
	}

	if n.Output != nil {
		*n.Output = state
	}

	return nil, nil
}
