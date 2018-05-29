package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/addrs"
)

// EvalRefresh is an EvalNode implementation that does a refresh for
// a resource.
type EvalRefresh struct {
	Addr     addrs.ResourceInstance
	Provider *ResourceProvider
	State    **InstanceState
	Output   **InstanceState
}

// TODO: test
func (n *EvalRefresh) Eval(ctx EvalContext) (interface{}, error) {
	provider := *n.Provider
	state := *n.State

	// The provider and hook APIs still expect our legacy InstanceInfo type.
	legacyInfo := NewInstanceInfo(n.Addr.Absolute(ctx.Path()))

	// If we have no state, we don't do any refreshing
	if state == nil {
		log.Printf("[DEBUG] refresh: %s: no state, so not refreshing", n.Addr.Absolute(ctx.Path()))
		return nil, nil
	}

	// Call pre-refresh hook
	err := ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PreRefresh(legacyInfo, state)
	})
	if err != nil {
		return nil, err
	}

	// Refresh!
	state, err = provider.Refresh(legacyInfo, state)
	if err != nil {
		return nil, fmt.Errorf("%s: %s", n.Addr.Absolute(ctx.Path()), err.Error())
	}
	if state == nil {
		log.Printf("[TRACE] EvalRefresh: after refresh, %s has nil state", n.Addr)
	}

	// Call post-refresh hook
	err = ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PostRefresh(legacyInfo, state)
	})
	if err != nil {
		return nil, err
	}

	if n.Output != nil {
		*n.Output = state
	}

	return nil, nil
}
