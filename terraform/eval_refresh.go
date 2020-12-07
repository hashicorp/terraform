package terraform

import (
	"fmt"
	"log"
	"strings"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/plans/objchange"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/tfdiags"
)

// EvalRefresh is an EvalNode implementation that does a refresh for
// a resource.
type EvalRefreshRequest struct {
	Addr           addrs.ResourceInstance
	ProviderAddr   addrs.AbsProviderConfig
	Provider       *providers.Interface
	ProviderMetas  map[addrs.Provider]*configs.ProviderMeta
	ProviderSchema *ProviderSchema
	State          *states.ResourceInstanceObject
}

// TODO: test
func Refresh(req *EvalRefreshRequest, ctx EvalContext) (*states.ResourceInstanceObject, tfdiags.Diagnostics) {
	state := req.State
	absAddr := req.Addr.Absolute(ctx.Path())
	var diags tfdiags.Diagnostics

	// If we have no state, we don't do any refreshing
	if state == nil {
		log.Printf("[DEBUG] refresh: %s: no state, so not refreshing", absAddr)
		return state, diags
	}

	schema, _ := req.ProviderSchema.SchemaForResourceAddr(req.Addr.ContainingResource())
	if schema == nil {
		// Should be caught during validation, so we don't bother with a pretty error here
		diags = diags.Append(fmt.Errorf("provider does not support resource type %q", req.Addr.Resource.Type))
		return state, diags
	}

	metaConfigVal := cty.NullVal(cty.DynamicPseudoType)
	if req.ProviderMetas != nil {
		if m, ok := req.ProviderMetas[req.ProviderAddr.Provider]; ok && m != nil {
			log.Printf("[DEBUG] EvalRefresh: ProviderMeta config value set")
			// if the provider doesn't support this feature, throw an error
			if req.ProviderSchema.ProviderMeta == nil {
				log.Printf("[DEBUG] EvalRefresh: no ProviderMeta schema")
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  fmt.Sprintf("Provider %s doesn't support provider_meta", req.ProviderAddr.Provider.String()),
					Detail:   fmt.Sprintf("The resource %s belongs to a provider that doesn't support provider_meta blocks", req.Addr),
					Subject:  &m.ProviderRange,
				})
			} else {
				log.Printf("[DEBUG] EvalRefresh: ProviderMeta schema found: %+v", (*req.ProviderSchema).ProviderMeta)
				var configDiags tfdiags.Diagnostics
				metaConfigVal, _, configDiags = ctx.EvaluateBlock(m.Config, (*req.ProviderSchema).ProviderMeta, nil, EvalDataForNoInstanceKey)
				diags = diags.Append(configDiags)
				if configDiags.HasErrors() {
					return state, diags
				}
			}
		}
	}

	// Call pre-refresh hook
	diags = diags.Append(ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PreRefresh(absAddr, states.CurrentGen, state.Value)
	}))
	if diags.HasErrors() {
		return state, diags
	}

	// Refresh!
	priorVal := state.Value

	// Unmarked before sending to provider
	var priorPaths []cty.PathValueMarks
	if priorVal.ContainsMarked() {
		priorVal, priorPaths = priorVal.UnmarkDeepWithPaths()
	}

	providerReq := providers.ReadResourceRequest{
		TypeName:     req.Addr.Resource.Type,
		PriorState:   priorVal,
		Private:      state.Private,
		ProviderMeta: metaConfigVal,
	}

	provider := *req.Provider
	resp := provider.ReadResource(providerReq)
	diags = diags.Append(resp.Diagnostics)
	if diags.HasErrors() {
		return state, diags
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
				req.ProviderAddr.Provider.String(), absAddr, tfdiags.FormatError(err),
			),
		))
	}
	if diags.HasErrors() {
		return state, diags
	}

	// We have no way to exempt provider using the legacy SDK from this check,
	// so we can only log inconsistencies with the updated state values.
	// In most cases these are not errors anyway, and represent "drift" from
	// external changes which will be handled by the subsequent plan.
	if errs := objchange.AssertObjectCompatible(schema, priorVal, resp.NewState); len(errs) > 0 {
		var buf strings.Builder
		fmt.Fprintf(&buf, "[WARN] Provider %q produced an unexpected new value for %s during refresh.", req.ProviderAddr.Provider.String(), absAddr)
		for _, err := range errs {
			fmt.Fprintf(&buf, "\n      - %s", tfdiags.FormatError(err))
		}
		log.Print(buf.String())
	}

	ret := state.DeepCopy()
	ret.Value = resp.NewState
	ret.Private = resp.Private
	ret.Dependencies = state.Dependencies
	ret.CreateBeforeDestroy = state.CreateBeforeDestroy

	// Call post-refresh hook
	diags = diags.Append(ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PostRefresh(absAddr, states.CurrentGen, priorVal, ret.Value)
	}))
	if diags.HasErrors() {
		return ret, diags
	}

	// Mark the value if necessary
	if len(priorPaths) > 0 {
		ret.Value = ret.Value.MarkWithPaths(priorPaths)
	}

	return ret, diags
}
