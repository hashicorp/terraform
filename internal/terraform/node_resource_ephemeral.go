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
func ephemeralResourceOpen(ctx EvalContext, inp ephemeralResourceInput) tfdiags.Diagnostics {
	log.Printf("[TRACE] ephemeralResourceOpen: opening %s", inp.addr)
	var diags tfdiags.Diagnostics

	provider, providerSchema, err := getProvider(ctx, inp.providerConfig)
	if err != nil {
		diags = diags.Append(err)
		return diags
	}

	config := inp.config
	schema, _ := providerSchema.SchemaForResourceAddr(inp.addr.ContainingResource().Resource)
	if schema == nil {
		// Should be caught during validation, so we don't bother with a pretty error here
		diags = diags.Append(
			fmt.Errorf("provider %q does not support data source %q",
				inp.providerConfig, inp.addr.ContainingResource().Resource.Type,
			),
		)
		return diags
	}

	resources := ctx.EphemeralResources()
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
		return diags // failed preconditions prevent further evaluation
	}

	configVal, _, configDiags := ctx.EvaluateBlock(config.Config, schema, nil, keyData)
	diags = diags.Append(configDiags)
	if diags.HasErrors() {
		return diags
	}
	unmarkedConfigVal, configMarks := configVal.UnmarkDeepWithPaths()

	// TODO: Call provider.ValidateEphemeralConfig, once such a thing exists.
	// For prototype we'll just let OpenEphemeral be the first line of validation.

	resp := provider.OpenEphemeral(providers.OpenEphemeralRequest{
		TypeName: inp.addr.ContainingResource().Resource.Type,
		Config:   unmarkedConfigVal,
	})
	if resp.Deferred != nil {
		// FIXME: Actually implement this. (Skipped for prototype only because
		// deferred changes is under development concurrently with this
		// prototype, and so don't want to conflict.)
		diags = diags.Append(fmt.Errorf("don't support deferral of ephemeral resource instances yet"))
	}
	diags = diags.Append(resp.Diagnostics.InConfigBody(config.Config, inp.addr.String()))
	if diags.HasErrors() {
		return diags
	}
	resultVal := resp.Result.MarkWithPaths(configMarks)

	errs := objchange.AssertPlanValid(schema, cty.NullVal(schema.ImpliedType()), configVal, resultVal)
	for _, err := range errs {
		// FIXME: Should turn these errors into suitable diagnostics.
		diags = diags.Append(err)
	}
	if diags.HasErrors() {
		return diags
	}

	// FIXME: Ideally anything that was set in the configuration to a
	// non-ephemeral value should remain non-ephemeral in the result. For
	// the sake of initial prototyping though, we'll just make the entire
	// object ephemeral.
	resultVal = resultVal.Mark(marks.Ephemeral)

	impl := &ephemeralResourceInstImpl{
		addr:      inp.addr,
		provider:  provider,
		closeData: resp.InternalContext,
	}
	// TODO: What can we use as a signal to cancel the context we're passing
	// in here, so that the object will stop renewing things when we start
	// shutting down?
	resources.RegisterInstance(context.TODO(), inp.addr, ephemeral.ResourceInstanceRegistration{
		Value:        resultVal,
		ConfigBody:   config.Config,
		Impl:         impl,
		FirstRenewal: resp.Renew,
	})

	return diags
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
	addr addrs.ConfigResource
}

var _ GraphNodeExecutable = (*nodeEphemeralResourceClose)(nil)
var _ GraphNodeModulePath = (*nodeEphemeralResourceClose)(nil)

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
	return resources.CloseInstances(context.TODO(), n.addr)
}

// ephemeralResourceInstImpl implements ephemeral.ResourceInstance as an
// adapter to the relevant provider API calls.
type ephemeralResourceInstImpl struct {
	addr      addrs.AbsResourceInstance
	provider  providers.Interface
	closeData []byte
}

var _ ephemeral.ResourceInstance = (*ephemeralResourceInstImpl)(nil)

// Close implements ephemeral.ResourceInstance.
func (impl *ephemeralResourceInstImpl) Close(ctx context.Context) tfdiags.Diagnostics {
	log.Printf("[TRACE] ephemeralResourceInstImpl: closing %s", impl.addr)
	resp := impl.provider.CloseEphemeral(providers.CloseEphemeralRequest{
		TypeName:        impl.addr.Resource.Resource.Type,
		InternalContext: impl.closeData,
	})
	return resp.Diagnostics
}

// Renew implements ephemeral.ResourceInstance.
func (impl *ephemeralResourceInstImpl) Renew(ctx context.Context, req providers.EphemeralRenew) (nextRenew *providers.EphemeralRenew, diags tfdiags.Diagnostics) {
	log.Printf("[TRACE] ephemeralResourceInstImpl: renewing %s", impl.addr)
	resp := impl.provider.RenewEphemeral(providers.RenewEphemeralRequest{
		TypeName:        impl.addr.Resource.Resource.Type,
		InternalContext: req.InternalContext,
	})
	return resp.RenewAgain, resp.Diagnostics
}
