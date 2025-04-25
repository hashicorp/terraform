// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type NodeQueryList struct {
	// The config for this list
	Config *configs.List

	Schema *configschema.Block

	// The address of the provider this resource will use
	ResolvedProvider addrs.AbsProviderConfig
}

type AttachResourceSchema2 interface {
	Provider() addrs.Provider
	AttachResourceSchema(schema *providers.Schema)
}

var (
	_ AttachResourceSchema2     = (*NodeQueryList)(nil)
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

// AttachResourceSchema2
func (n *NodeQueryList) AttachResourceSchema(schema *providers.Schema) {
	n.Schema = schema.Body
}

// AttachResourceSchema2
func (n *NodeQueryList) Provider() addrs.Provider {
	return n.Config.Provider
}

func (n *NodeQueryList) References() []*addrs.Reference {
	if n.Config == nil {
		return nil
	}

	return ReferencesFromConfig(n.Config.Config, n.Schema)
}

func (n *NodeQueryList) String() string {
	return n.Addr().String()
}

func (n *NodeQueryList) Execute(ctx EvalContext) (diags tfdiags.Diagnostics) {
	config := n.Config
	provider, providerSchema, err := getProvider(ctx, n.ResolvedProvider)
	diags = diags.Append(err)
	if diags.HasErrors() {
		return diags
	}

	// validate self ref

	// retrieve list schema
	schema := providerSchema.SchemaForResourceType(addrs.ListResourceMode, n.Addr().Type)
	if schema.Body == nil {
		// Should be caught during validation, so we don't bother with a pretty error here
		diags = diags.Append(fmt.Errorf("provider %q does not support data source %q", n.ResolvedProvider, n.Addr().Type))
		return diags
	}

	// evaluate the config block
	var configDiags tfdiags.Diagnostics
	configVal, _, configDiags := ctx.EvaluateBlock(config.Config, schema.Body, nil, EvalDataForNoInstanceKey)
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

	// If we get down here then our configuration is complete and we're ready
	// to actually call the provider to list the data.
	err = provider.ListResource(providers.ListResourceRequest{
		TypeName:        n.Addr().Type,
		Config:          unmarkedConfigVal,
		DiagEmitter:     n.emitDiags,
		ResourceEmitter: n.emitResource(ctx, schema, diags),
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
