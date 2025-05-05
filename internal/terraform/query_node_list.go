// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/lang/langrefs"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type NodeQueryList struct {
	// The config for this list
	Config *configs.List

	// Schema is the schema for the list block itself
	Schema *configschema.Block

	// The schema for the resource type
	ResourceSchema *configschema.Block

	// The address of the provider this resource will use
	ResolvedProvider addrs.AbsProviderConfig
}

type AttachListSchema interface {
	Provider() addrs.Provider
	AttachSchema(schema *providers.Schema)
	AttachResourceSchema(schema *providers.Schema)
}

var (
	_ AttachListSchema          = (*NodeQueryList)(nil)
	_ GraphNodeProviderConsumer = (*NodeQueryList)(nil)
)

func (n *NodeQueryList) Addr() addrs.List {
	return n.Config.Addr()
}

func (n *NodeQueryList) Name() string {
	return n.Addr().String()
}

func (n *NodeQueryList) Path() addrs.ModuleInstance {
	// Lists cannot be contained inside an expanded module, so we
	// just return the root module path.
	return addrs.RootModuleInstance
}

func (n *NodeQueryList) ModulePath() addrs.Module {
	// Lists cannot be contained inside an expanded module, so we
	// just return the root module path.
	return addrs.RootModule
}

// GraphNodeProviderConsumer
func (n *NodeQueryList) ProvidedBy() (addrs.ProviderConfig, bool) {
	// If we have a config, then we can build the exact provider config address
	if n.Config != nil && n.Config.ProviderConfigRef != nil {
		return addrs.AbsProviderConfig{
			Module:   addrs.RootModule,
			Provider: n.Config.Provider,
			Alias:    n.Config.ProviderConfigRef.Alias,
		}, true
	}
	panic("NodeQueryList: no provider config")
}

// GraphNodeProviderConsumer
func (n *NodeQueryList) SetProvider(p addrs.AbsProviderConfig) {
	n.ResolvedProvider = p
}

// AttachListSchema
func (n *NodeQueryList) AttachSchema(schema *providers.Schema) {
	n.Schema = schema.Body
}

// AttachListSchema
func (n *NodeQueryList) AttachResourceSchema(schema *providers.Schema) {
	n.ResourceSchema = schema.Body
}

// AttachListSchema
func (n *NodeQueryList) Provider() addrs.Provider {
	return n.Config.Provider
}

func (n *NodeQueryList) References() []*addrs.Reference {
	if n.Config == nil {
		return nil
	}
	refs := ReferencesFromConfig(n.Config.Config, n.Schema)
	if n.Config.Count != nil {
		countRefs, _ := langrefs.ReferencesInExpr(addrs.ParseRef, n.Config.Count)
		refs = append(refs, countRefs...)
	}
	if n.Config.ForEach != nil {
		forEachRefs, _ := langrefs.ReferencesInExpr(addrs.ParseRef, n.Config.ForEach)
		refs = append(refs, forEachRefs...)
	}
	return refs
}

func (n *NodeQueryList) String() string {
	return n.Addr().String()
}

func (n *NodeQueryList) Execute(ctx EvalContext) (diags tfdiags.Diagnostics) {
	expander := ctx.InstanceExpander()
	module := n.Path()
	switch {
	case n.Config.Count != nil:
		count, ctDiags := evaluateCountExpression(n.Config.Count, ctx, false)
		diags = diags.Append(ctDiags)
		if diags.HasErrors() {
			return diags
		}
		if count >= 0 {
			expander.SetQueryListCount(module, n.Addr(), count)
		} else {
			// -1 represents "unknown"
			panic("NodeQueryList: count is unknown")
		}

	case n.Config.ForEach != nil:
		forEach, known, feDiags := evaluateForEachExpression(n.Config.ForEach, ctx, false)
		diags = diags.Append(feDiags)
		if diags.HasErrors() {
			return diags
		}
		if known {
			expander.SetQueryListForEach(module, n.Addr(), forEach)
		} else {
			panic("NodeQueryList: for_each is unknown")
		}

	default:
		expander.SetQueryListSingle(module, n.Addr())
	}

	_, knownKeys, hasUnknown := expander.ListInstanceKeys(n.Addr())
	if hasUnknown {
		panic("NodeQueryList: list instance keys are unknown")
	}
	for _, key := range knownKeys {
		diags = diags.Append(n.execute(ctx, n.Addr().Instance(key)))
		if diags.HasErrors() {
			return diags
		}
	}

	return diags
}
func (n *NodeQueryList) execute(ctx EvalContext, inst addrs.ListInstance) (diags tfdiags.Diagnostics) {
	config := n.Config
	provider, providerSchema, err := getProvider(ctx, n.ResolvedProvider)
	diags = diags.Append(err)
	if diags.HasErrors() {
		return diags
	}

	// validate self ref

	// retrieve list schema? (already done in transformer)

	instKeyData := ctx.InstanceExpander().GetListInstanceRepetitionData(inst)

	// evaluate the config block
	var configDiags tfdiags.Diagnostics
	configVal, _, configDiags := ctx.EvaluateBlock(config.Config, n.Schema, nil, instKeyData)
	diags = diags.Append(configDiags)
	if diags.HasErrors() {
		return diags
	}

	// Unmark before sending to provider, will re-mark before returning
	unmarkedConfigVal, _ := configVal.UnmarkDeepWithPaths()
	configKnown := configVal.IsWhollyKnown()
	if !configKnown {
		diags = diags.Append(fmt.Errorf("config is not known"))
		return diags
	}

	log.Printf("[TRACE] NodeQueryList: Re-validating config for %s", n.Addr())
	validateResp := provider.ValidateListResourceConfig(
		providers.ValidateListResourceConfigRequest{
			TypeName: n.Addr().Type,
			Config:   unmarkedConfigVal,
		},
	)
	diags = diags.Append(validateResp.Diagnostics.InConfigBody(config.Config, n.Addr().String()))
	if diags.HasErrors() {
		return diags
	}

	doneCh := make(chan struct{}, 1)
	// retrieve resource schema
	resourceSchema := providerSchema.SchemaForResourceType(addrs.ManagedResourceMode, n.Addr().Type)
	if resourceSchema.Body == nil {
		// Should be caught during validation, so we don't bother with a pretty error here
		diags = diags.Append(fmt.Errorf("provider %q does not support managed source %q", n.ResolvedProvider, n.Addr().Type))
		return diags
	}

	// If we get down here then our configuration is complete and we're ready
	// to actually call the provider to list the data.
	err = provider.ListResource(providers.ListResourceRequest{
		TypeName:        n.Addr().Type,
		Config:          unmarkedConfigVal,
		DiagEmitter:     n.emitDiags,
		ResourceEmitter: n.emitResource(ctx, resourceSchema, diags),
		DoneCh:          doneCh,
	})
	if err != nil {
		return diags.Append(fmt.Errorf("failed to list %s: %s", n.Addr(), err))
	}

	for {
		select {
		case <-doneCh:
			return diags
		default:
			// Maybe we want to set some limit on how long we wait or how much data can be sent?
			// do nothing
		}
	}
}

func (n *NodeQueryList) Validate(ctx EvalContext) (diags tfdiags.Diagnostics) {
	return nil
}

func (n *NodeQueryList) emitDiags(diags tfdiags.Diagnostics) {
	if diags.HasErrors() {
		diags = diags.Append(diags.InConfigBody(n.Config.Config, n.Addr().String()))
		return
	}
}

func (n *NodeQueryList) emitResource(ctx EvalContext, schema providers.Schema, diags tfdiags.Diagnostics) func(resource providers.ListResult) {
	return func(resource providers.ListResult) {
		obj := &states.ResourceInstanceObject{
			Value:    resource.ResourceObject,
			Identity: resource.Identity,
			Status:   states.ObjectPlanned,
		}
		src, err := obj.Encode(schema)
		if err != nil {
			diags = diags.Append(fmt.Errorf("failed to encode %s in state: %s", n.Addr(), err))
		}
		// store the resources in some transient state or send directly to the consumer
		// Check if there's already an entry for this address and initialize if not
		qState := ctx.Querier().State
		if _, exists := qState.GetOk(n.Addr()); !exists {
			qState.Put(n.Addr(), []*states.ResourceInstanceObjectSrc{})
		}
		qState.Put(n.Addr(), append(qState.Get(n.Addr()), src))
		ctx.Querier().View.Resource(n.Addr(), src)
	}
}
