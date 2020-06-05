package terraform

import (
	"fmt"
	"log"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
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
	ProviderMetas  map[addrs.Provider]*configs.ProviderMeta
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

	schema, _ := (*n.ProviderSchema).SchemaForResourceAddr(n.Addr.ContainingResource())
	if schema == nil {
		// Should be caught during validation, so we don't bother with a pretty error here
		return nil, fmt.Errorf("provider does not support resource type %q", n.Addr.Resource.Type)
	}

	metaConfigVal := cty.NullVal(cty.DynamicPseudoType)
	if n.ProviderMetas != nil {
		if m, ok := n.ProviderMetas[n.ProviderAddr.Provider]; ok && m != nil {
			log.Printf("[DEBUG] EvalRefresh: ProviderMeta config value set")
			// if the provider doesn't support this feature, throw an error
			if (*n.ProviderSchema).ProviderMeta == nil {
				log.Printf("[DEBUG] EvalRefresh: no ProviderMeta schema")
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  fmt.Sprintf("Provider %s doesn't support provider_meta", n.ProviderAddr.Provider.String()),
					Detail:   fmt.Sprintf("The resource %s belongs to a provider that doesn't support provider_meta blocks", n.Addr),
					Subject:  &m.ProviderRange,
				})
			} else {
				log.Printf("[DEBUG] EvalRefresh: ProviderMeta schema found: %+v", (*n.ProviderSchema).ProviderMeta)
				var configDiags tfdiags.Diagnostics
				metaConfigVal, _, configDiags = ctx.EvaluateBlock(m.Config, (*n.ProviderSchema).ProviderMeta, nil, EvalDataForNoInstanceKey)
				diags = diags.Append(configDiags)
				if configDiags.HasErrors() {
					return nil, diags.Err()
				}
			}
		}
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
		TypeName:     n.Addr.Resource.Type,
		PriorState:   priorVal,
		Private:      state.Private,
		ProviderMeta: metaConfigVal,
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

	for _, err := range resp.NewState.Type().TestConformance(schema.ImpliedType()) {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Provider produced invalid object",
			fmt.Sprintf(
				"Provider %q planned an invalid value for %s during refresh: %s.\n\nThis is a bug in the provider, which should be reported in the provider's own issue tracker.",
				n.ProviderAddr.Provider.String(), absAddr, tfdiags.FormatError(err),
			),
		))
	}
	if diags.HasErrors() {
		return nil, diags.Err()
	}

	newState := state.DeepCopy()
	newState.Value = resp.NewState
	newState.Private = resp.Private
	newState.Dependencies = state.Dependencies
	newState.CreateBeforeDestroy = state.CreateBeforeDestroy

	// Call post-refresh hook
	err = ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PostRefresh(absAddr, states.CurrentGen, priorVal, newState.Value)
	})
	if err != nil {
		return nil, err
	}

	if n.Output != nil {
		*n.Output = newState
	}

	return nil, diags.ErrWithWarnings()
}
