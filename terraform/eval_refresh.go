package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/states"
)

// EvalRefresh is an EvalNode implementation that does a refresh for
// a resource.
type EvalRefresh struct {
	Addr     addrs.ResourceInstance
	Provider *ResourceProvider
	State    **states.ResourceInstanceObject
	Output   **states.ResourceInstanceObject
}

// TODO: test
func (n *EvalRefresh) Eval(ctx EvalContext) (interface{}, error) {
	state := *n.State
	absAddr := n.Addr.Absolute(ctx.Path())

	// If we have no state, we don't do any refreshing
	if state == nil {
		log.Printf("[DEBUG] refresh: %s: no state, so not refreshing", n.Addr.Absolute(ctx.Path()))
		return nil, nil
	}

	// Call pre-refresh hook
	err := ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PreRefresh(absAddr, states.CurrentGen, state.Value)
	})
	if err != nil {
		return nil, err
	}

	// Refresh!
	priorVal := state.Value
	// TODO: Shim our new state type into the old one
	//provider := *n.Provider
	//state, err = provider.Refresh(legacyInfo, state)
	return nil, fmt.Errorf("EvalRefresh is not yet updated for new state type")
	if err != nil {
		return nil, fmt.Errorf("%s: %s", n.Addr.Absolute(ctx.Path()), err.Error())
	}
	if state == nil {
		log.Printf("[TRACE] EvalRefresh: after refresh, %s has nil state", n.Addr)
	}

	// Call post-refresh hook
	err = ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PostRefresh(absAddr, states.CurrentGen, priorVal, state.Value)
	})
	if err != nil {
		return nil, err
	}

	if n.Output != nil {
		*n.Output = state
	}

	return nil, nil
}
