package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/states"
)

// EvalRefresh is an EvalNode implementation that does a refresh for
// a resource.
type EvalRefresh struct {
	Addr     addrs.ResourceInstance
	Provider *providers.Interface
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
	req := providers.ReadResourceRequest{
		TypeName:   n.Addr.Resource.Type,
		PriorState: priorVal,
	}

	provider := *n.Provider
	resp := provider.ReadResource(req)
	if resp.Diagnostics.HasErrors() {
		return nil, fmt.Errorf("%s: %s", n.Addr.Absolute(ctx.Path()), resp.Diagnostics.Err())
	}

	state.Value = resp.NewState

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
