// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"context"
	"fmt"
	"log"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/plans/objchange"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/resources/ephemeral"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type ephemeralResourceInput struct {
	addr           addrs.AbsResourceInstance
	config         *configs.Resource
	providerConfig addrs.AbsProviderConfig
}

// ephemeralResourceOpen implements the "open" step of the ephemeral resource
// instance lifecycle, which behaves the same way in both the plan and apply
// walks.
func ephemeralResourceOpen(ctx EvalContext, inp ephemeralResourceInput) (*providers.Deferred, tfdiags.Diagnostics) {
	log.Printf("[TRACE] ephemeralResourceOpen: opening %s", inp.addr)
	var diags tfdiags.Diagnostics

	provider, providerSchema, err := getProvider(ctx, inp.providerConfig)
	if err != nil {
		diags = diags.Append(err)
		return nil, diags
	}

	config := inp.config
	schema, _ := providerSchema.SchemaForResourceAddr(inp.addr.ContainingResource().Resource)
	if schema == nil {
		// Should be caught during validation, so we don't bother with a pretty error here
		diags = diags.Append(
			fmt.Errorf("provider %q does not support ephemeral resource %q",
				inp.providerConfig, inp.addr.ContainingResource().Resource.Type,
			),
		)
		return nil, diags
	}

	rId := HookResourceIdentity{
		Addr:         inp.addr,
		ProviderAddr: inp.providerConfig.Provider,
	}

	ephemerals := ctx.EphemeralResources()
	allInsts := ctx.InstanceExpander()
	keyData := allInsts.GetResourceInstanceRepetitionData(inp.addr)

	checkDiags := evalCheckRules(
		addrs.ResourcePrecondition,
		config.Preconditions,
		ctx, inp.addr, keyData,
		tfdiags.Error,
	)
	diags = diags.Append(checkDiags)
	if diags.HasErrors() {
		return nil, diags // failed preconditions prevent further evaluation
	}

	configVal, _, configDiags := ctx.EvaluateBlock(config.Config, schema, nil, keyData)
	diags = diags.Append(configDiags)
	if diags.HasErrors() {
		return nil, diags
	}
	unmarkedConfigVal, configMarks := configVal.UnmarkDeepWithPaths()

	if !unmarkedConfigVal.IsWhollyKnown() {
		log.Printf("[DEBUG] ehpemeralResourceOpen: configuration for %s contains unknown values, cannot open resource", inp.addr)

		// We don't know what the result will be, but we need to keep the
		// configured attributes for consistent evaluation. We can use the same
		// technique we used for data sources to create the plan-time value.
		unknownResult := objchange.PlannedDataResourceObject(schema, unmarkedConfigVal)
		// add back any configured marks
		unknownResult = unknownResult.MarkWithPaths(configMarks)
		// and mark the entire value as ephemeral, since it's coming from an ephemeral context.
		unknownResult = unknownResult.Mark(marks.Ephemeral)

		// The state of ephemerals all comes from the registered instances, so
		// we still need to register something so evaluation doesn't fail.
		ephemerals.RegisterInstance(ctx.StopCtx(), inp.addr, ephemeral.ResourceInstanceRegistration{
			Value:      unknownResult,
			ConfigBody: config.Config,
		})

		ctx.Hook(func(h Hook) (HookAction, error) {
			// ephemeral resources aren't stored in the plan, so use a hook to
			// give some feedback to the user that this can't be opened
			return h.PreEphemeralOp(rId, plans.Read)
		})

		return nil, diags
	}

	validateResp := provider.ValidateEphemeralResourceConfig(providers.ValidateEphemeralResourceConfigRequest{
		TypeName: inp.addr.Resource.Resource.Type,
		Config:   unmarkedConfigVal,
	})

	diags = diags.Append(validateResp.Diagnostics)
	if diags.HasErrors() {
		return nil, diags
	}

	ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PreEphemeralOp(rId, plans.Open)
	})
	resp := provider.OpenEphemeralResource(providers.OpenEphemeralResourceRequest{
		TypeName: inp.addr.ContainingResource().Resource.Type,
		Config:   unmarkedConfigVal,
	})
	ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PostEphemeralOp(rId, plans.Open, resp.Diagnostics.Err())
	})
	diags = diags.Append(resp.Diagnostics.InConfigBody(config.Config, inp.addr.String()))
	if diags.HasErrors() {
		return nil, diags
	}
	if resp.Deferred != nil {
		return resp.Deferred, diags
	}
	resultVal := resp.Result.MarkWithPaths(configMarks)

	errs := objchange.AssertPlanValid(schema, cty.NullVal(schema.ImpliedType()), configVal, resultVal)
	for _, err := range errs {
		diags = diags.Append(tfdiags.AttributeValue(
			tfdiags.Error,
			"Provider produced invalid ephemeral resource instance",
			fmt.Sprintf(
				"The provider for %s produced an inconsistent result: %s.",
				inp.addr.Resource.Resource.Type,
				tfdiags.FormatError(err),
			),
			nil,
		)).InConfigBody(config.Config, inp.addr.String())
	}
	if diags.HasErrors() {
		return nil, diags
	}

	// We are going to wholesale mark the entire resource as ephemeral. This
	// simplifies the model as any references to ephemeral resources can be
	// considered as such. Any input values that don't need to be ephemeral can
	// be referenced directly.
	resultVal = resultVal.Mark(marks.Ephemeral)

	impl := &ephemeralResourceInstImpl{
		addr:        inp.addr,
		providerCfg: inp.providerConfig,
		provider:    provider,
		hook:        ctx.Hook,
		internal:    resp.Private,
	}

	ephemerals.RegisterInstance(ctx.StopCtx(), inp.addr, ephemeral.ResourceInstanceRegistration{
		Value:      resultVal,
		ConfigBody: config.Config,
		Impl:       impl,
		RenewAt:    resp.RenewAt,
		Private:    resp.Private,
	})

	// Postconditions for ephemerals validate only what is returned by
	// OpenEphemeralResource. These will block downstream dependency operations
	// if an error is returned, but don't prevent renewal or closing of the
	// resource.
	checkDiags = evalCheckRules(
		addrs.ResourcePostcondition,
		config.Postconditions,
		ctx, inp.addr, keyData,
		tfdiags.Error,
	)
	diags = diags.Append(checkDiags)

	return nil, diags
}

// nodeEphemeralResourceClose is the node type for closing the previously-opened
// instances of a particular ephemeral resource.
//
// Although ephemeral resource instances will always all get closed once a
// graph walk has completed anyway, the inclusion of explicit nodes for this
// allows closing ephemeral resource instances more promptly after all work
// that uses them has been completed, rather than always just waiting until
// the end of the graph walk.
//
// This is scoped to config-level resources rather than dynamic resource
// instances as a concession to allow using the same node type in both the plan
// and apply graphs, where the former only deals in whole resources while the
// latter contains individual instances.
type nodeEphemeralResourceClose struct {
	// The provider must remain active for the lifetime of the value. Proxy the
	// provider methods from the original resource to ensure the references are
	// create correctly.
	resourceNode GraphNodeProviderConsumer
	addr         addrs.ConfigResource
}

var _ GraphNodeExecutable = (*nodeEphemeralResourceClose)(nil)
var _ GraphNodeModulePath = (*nodeEphemeralResourceClose)(nil)
var _ GraphNodeProviderConsumer = (*nodeEphemeralResourceClose)(nil)

func (n *nodeEphemeralResourceClose) Name() string {
	return n.addr.String() + " (close)"
}

// ModulePath implements GraphNodeModulePath.
func (n *nodeEphemeralResourceClose) ModulePath() addrs.Module {
	return n.addr.Module
}

// Execute implements GraphNodeExecutable.
func (n *nodeEphemeralResourceClose) Execute(ctx EvalContext, op walkOperation) tfdiags.Diagnostics {
	log.Printf("[TRACE] nodeEphemeralResourceClose: closing all instances of %s", n.addr)
	resources := ctx.EphemeralResources()
	return resources.CloseInstances(ctx.StopCtx(), n.addr)
}

func (n *nodeEphemeralResourceClose) ProvidedBy() (addrs.ProviderConfig, bool) {
	return n.resourceNode.ProvidedBy()
}

func (n *nodeEphemeralResourceClose) Provider() addrs.Provider {
	return n.resourceNode.Provider()
}

func (n *nodeEphemeralResourceClose) SetProvider(provider addrs.AbsProviderConfig) {
	// the provider should not be set through this proxy node
}

// ephemeralResourceInstImpl implements ephemeral.ResourceInstance as an
// adapter to the relevant provider API calls.
type ephemeralResourceInstImpl struct {
	addr        addrs.AbsResourceInstance
	providerCfg addrs.AbsProviderConfig
	provider    providers.Interface
	hook        hookFunc
	internal    []byte
}

var _ ephemeral.ResourceInstance = (*ephemeralResourceInstImpl)(nil)

// Close implements ephemeral.ResourceInstance.
func (impl *ephemeralResourceInstImpl) Close(ctx context.Context) tfdiags.Diagnostics {
	log.Printf("[TRACE] ephemeralResourceInstImpl: closing %s", impl.addr)
	rId := HookResourceIdentity{
		Addr:         impl.addr,
		ProviderAddr: impl.providerCfg.Provider,
	}
	impl.hook(func(h Hook) (HookAction, error) {
		return h.PreEphemeralOp(rId, plans.Close)
	})
	resp := impl.provider.CloseEphemeralResource(providers.CloseEphemeralResourceRequest{
		TypeName: impl.addr.Resource.Resource.Type,
		Private:  impl.internal,
	})
	impl.hook(func(h Hook) (HookAction, error) {
		return h.PostEphemeralOp(rId, plans.Close, resp.Diagnostics.Err())
	})
	return resp.Diagnostics
}

// Renew implements ephemeral.ResourceInstance.
func (impl *ephemeralResourceInstImpl) Renew(ctx context.Context, req providers.EphemeralRenew) (nextRenew *providers.EphemeralRenew, diags tfdiags.Diagnostics) {
	log.Printf("[TRACE] ephemeralResourceInstImpl: renewing %s", impl.addr)

	rId := HookResourceIdentity{
		Addr:         impl.addr,
		ProviderAddr: impl.providerCfg.Provider,
	}
	impl.hook(func(h Hook) (HookAction, error) {
		return h.PreEphemeralOp(rId, plans.Renew)
	})
	resp := impl.provider.RenewEphemeralResource(providers.RenewEphemeralResourceRequest{
		TypeName: impl.addr.Resource.Resource.Type,
		Private:  req.Private,
	})
	impl.hook(func(h Hook) (HookAction, error) {
		return h.PostEphemeralOp(rId, plans.Renew, resp.Diagnostics.Err())
	})
	if !resp.RenewAt.IsZero() {
		nextRenew = &providers.EphemeralRenew{
			RenewAt: resp.RenewAt,
			Private: resp.Private,
		}
	}

	return nextRenew, resp.Diagnostics
}
