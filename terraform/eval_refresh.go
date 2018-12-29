package terraform

import (
	"fmt"
	"log"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/tfdiags"
)

// EvalRefresh is an EvalNode implementation that does a refresh for
// a resource.
type EvalRefresh struct {
	Addr           addrs.ResourceInstance
	ProviderAddr   addrs.AbsProviderConfig
	Provider       *providers.Interface
	ProviderSchema **ProviderSchema
	State          **states.ResourceInstanceObject
	Output         **states.ResourceInstanceObject
}

// TODO: test
func (n *EvalRefresh) Eval(ctx EvalContext) (interface{}, error) {
	state := *n.State
	absAddr := n.Addr.Absolute(ctx.Path())

	var diags tfdiags.Diagnostics

	// If we have no state, we don't do any refreshing
	if state == nil {
		log.Printf("[DEBUG] refresh: %s: no state, so not refreshing", n.Addr.Absolute(ctx.Path()))
		return nil, diags.ErrWithWarnings()
	}

	schema := (*n.ProviderSchema).ResourceTypes[n.Addr.Resource.Type]
	if schema == nil {
		// Should be caught during validation, so we don't bother with a pretty error here
		return nil, fmt.Errorf("provider does not support resource type %q", n.Addr.Resource.Type)
	}

	// Call pre-refresh hook
	err := ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PreRefresh(absAddr, states.CurrentGen, state.Value)
	})
	if err != nil {
		return nil, diags.ErrWithWarnings()
	}

	// Refresh!
	priorVal := state.Value
	req := providers.ReadResourceRequest{
		TypeName:   n.Addr.Resource.Type,
		PriorState: priorVal,
	}

	provider := *n.Provider
	resp := provider.ReadResource(req)
	diags = diags.Append(resp.Diagnostics)
	if diags.HasErrors() {
		return nil, diags.Err()
	}

	if resp.NewState == cty.NilVal {
		// This ought not to happen in real cases since it's not possible to
		// send NilVal over the plugin RPC channel, but it can come up in
		// tests due to sloppy mocking.
		panic("new state is cty.NilVal")
	}

	for _, err := range schema.ImpliedType().TestConformance(resp.NewState.Type()) {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Provider produced invalid object",
			fmt.Sprintf(
				"Provider %q planned an invalid value for %s: %s during refresh.\n\nThis is a bug in the provider, which should be reported in the provider's own issue tracker.",
				n.ProviderAddr.ProviderConfig.Type, absAddr, tfdiags.FormatError(err),
			),
		))
	}
	if diags.HasErrors() {
		return nil, diags.Err()
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

	return nil, diags.ErrWithWarnings()
}
