// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/lang/ephemeral"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

type graphNodeImportState struct {
	Addr             addrs.AbsResourceInstance // Addr is the resource address to import into
	ID               string                    // ID is the ID to import as
	ProviderAddr     addrs.AbsProviderConfig   // Provider address given by the user, or implied by the resource type
	ResolvedProvider addrs.AbsProviderConfig   // provider node address after resolution

	states []providers.ImportedResource
}

var (
	_ GraphNodeModulePath        = (*graphNodeImportState)(nil)
	_ GraphNodeExecutable        = (*graphNodeImportState)(nil)
	_ GraphNodeProviderConsumer  = (*graphNodeImportState)(nil)
	_ GraphNodeDynamicExpandable = (*graphNodeImportState)(nil)
)

func (n *graphNodeImportState) Name() string {
	return fmt.Sprintf("%s (import id %q)", n.Addr, n.ID)
}

// GraphNodeProviderConsumer
func (n *graphNodeImportState) ProvidedBy() (addrs.ProviderConfig, bool) {
	// We assume that n.ProviderAddr has been properly populated here.
	// It's the responsibility of the code creating a graphNodeImportState
	// to populate this, possibly by calling DefaultProviderConfig() on the
	// resource address to infer an implied provider from the resource type
	// name.
	return n.ProviderAddr, false
}

// GraphNodeProviderConsumer
func (n *graphNodeImportState) Provider() addrs.Provider {
	// We assume that n.ProviderAddr has been properly populated here.
	// It's the responsibility of the code creating a graphNodeImportState
	// to populate this, possibly by calling DefaultProviderConfig() on the
	// resource address to infer an implied provider from the resource type
	// name.
	return n.ProviderAddr.Provider
}

// GraphNodeProviderConsumer
func (n *graphNodeImportState) SetProvider(addr addrs.AbsProviderConfig) {
	n.ResolvedProvider = addr
}

// GraphNodeModuleInstance
func (n *graphNodeImportState) Path() addrs.ModuleInstance {
	return n.Addr.Module
}

// GraphNodeModulePath
func (n *graphNodeImportState) ModulePath() addrs.Module {
	return n.Addr.Module.Module()
}

// GraphNodeExecutable impl.
func (n *graphNodeImportState) Execute(ctx EvalContext, op walkOperation) (diags tfdiags.Diagnostics) {
	// Reset our states
	n.states = nil

	provider, providerSchema, err := getProvider(ctx, n.ResolvedProvider)
	diags = diags.Append(err)
	if diags.HasErrors() {
		return diags
	}

	schema, _ := providerSchema.SchemaForResourceType(n.Addr.Resource.Resource.Mode, n.Addr.Resource.Resource.Type)
	if schema == nil {
		// Should be caught during validation, so we don't bother with a pretty error here
		diags = diags.Append(fmt.Errorf("provider does not support resource type %q", n.Addr.Resource.Resource.Type))
		return diags
	}

	// import state
	absAddr := n.Addr.Resource.Absolute(ctx.Path())
	hookResourceID := HookResourceIdentity{
		Addr:         absAddr,
		ProviderAddr: n.ResolvedProvider.Provider,
	}

	// Call pre-import hook
	diags = diags.Append(ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PreImportState(hookResourceID, n.ID)
	}))
	if diags.HasErrors() {
		return diags
	}

	resp := provider.ImportResourceState(providers.ImportResourceStateRequest{
		TypeName:           n.Addr.Resource.Resource.Type,
		ID:                 n.ID,
		ClientCapabilities: ctx.ClientCapabilities(),
	})
	diags = diags.Append(resp.Diagnostics)
	if diags.HasErrors() {
		return diags
	}

	// Providers are supposed to return null values for all write-only attributes
	var writeOnlyDiags tfdiags.Diagnostics
	for _, imported := range resp.ImportedResources {
		writeOnlyDiags = ephemeral.ValidateWriteOnlyAttributes(
			"Import returned a non-null value for a write-only attribute",
			func(path cty.Path) string {
				return fmt.Sprintf(
					"Provider %q returned a value for the write-only attribute \"%s%s\" during import. Write-only attributes cannot be read back from the provider. This is a bug in the provider, which should be reported in the provider's own issue tracker.",
					n.ResolvedProvider, n.Addr, tfdiags.FormatCtyPath(path),
				)
			},
			imported.State,
			schema,
		)
		diags = diags.Append(writeOnlyDiags)
	}

	if writeOnlyDiags.HasErrors() {
		return diags
	}

	imported := resp.ImportedResources
	for _, obj := range imported {
		log.Printf("[TRACE] graphNodeImportState: import %s %q produced instance object of type %s", absAddr.String(), n.ID, obj.TypeName)
	}
	n.states = imported

	// Call post-import hook
	diags = diags.Append(ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PostImportState(hookResourceID, imported)
	}))
	return diags
}

// GraphNodeDynamicExpandable impl.
//
// We use DynamicExpand as a way to generate the subgraph of refreshes
// and state inserts we need to do for our import state. Since they're new
// resources they don't depend on anything else and refreshes are isolated
// so this is nearly a perfect use case for dynamic expand.
func (n *graphNodeImportState) DynamicExpand(ctx EvalContext) (*Graph, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	g := &Graph{Path: ctx.Path()}

	// nameCounter is used to de-dup names in the state.
	nameCounter := make(map[string]int)

	// Compile the list of addresses that we'll be inserting into the state.
	// We do this ahead of time so we can verify that we aren't importing
	// something that already exists.
	addrs := make([]addrs.AbsResourceInstance, len(n.states))
	for i, state := range n.states {
		addr := n.Addr
		if t := state.TypeName; t != "" {
			addr.Resource.Resource.Type = t
		}

		// Determine if we need to suffix the name to de-dup
		key := addr.String()
		count, ok := nameCounter[key]
		if ok {
			count++
			addr.Resource.Resource.Name += fmt.Sprintf("-%d", count)
		}
		nameCounter[key] = count

		// Add it to our list
		addrs[i] = addr
	}

	// Verify that all the addresses are clear
	state := ctx.State()
	for _, addr := range addrs {
		existing := state.ResourceInstance(addr)
		if existing != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Resource already managed by Terraform",
				fmt.Sprintf("Terraform is already managing a remote object for %s. To import to this address you must first remove the existing object from the state.", addr),
			))
			continue
		}
	}
	if diags.HasErrors() {
		// Bail out early, then.
		return nil, diags
	}

	// For each of the states, we add a node to handle the refresh/add to state.
	// "n.states" is populated by our own Execute with the result of
	// ImportState. Since DynamicExpand is always called after Execute, this is
	// safe.
	for i, state := range n.states {
		g.Add(&graphNodeImportStateSub{
			TargetAddr:       addrs[i],
			State:            state,
			ResolvedProvider: n.ResolvedProvider,
		})
	}

	addRootNodeToGraph(g)

	// Done!
	return g, diags
}

// graphNodeImportStateSub is the sub-node of graphNodeImportState
// and is part of the subgraph. This node is responsible for refreshing
// and adding a resource to the state once it is imported.
type graphNodeImportStateSub struct {
	TargetAddr       addrs.AbsResourceInstance
	State            providers.ImportedResource
	ResolvedProvider addrs.AbsProviderConfig
}

var (
	_ GraphNodeModuleInstance = (*graphNodeImportStateSub)(nil)
	_ GraphNodeExecutable     = (*graphNodeImportStateSub)(nil)
)

func (n *graphNodeImportStateSub) Name() string {
	return fmt.Sprintf("import %s result", n.TargetAddr)
}

func (n *graphNodeImportStateSub) Path() addrs.ModuleInstance {
	return n.TargetAddr.Module
}

// GraphNodeExecutable impl.
func (n *graphNodeImportStateSub) Execute(ctx EvalContext, op walkOperation) (diags tfdiags.Diagnostics) {
	// If the Ephemeral type isn't set, then it is an error
	if n.State.TypeName == "" {
		diags = diags.Append(fmt.Errorf("import of %s didn't set type", n.TargetAddr.String()))
		return diags
	}

	state := n.State.AsInstanceObject()

	// Refresh
	riNode := &NodeAbstractResourceInstance{
		Addr: n.TargetAddr,
		NodeAbstractResource: NodeAbstractResource{
			ResolvedProvider: n.ResolvedProvider,
		},
	}
	state, deferred, refreshDiags := riNode.refresh(ctx, states.NotDeposed, state, false)
	diags = diags.Append(refreshDiags)
	if diags.HasErrors() {
		return diags
	}

	// If the refresh is deferred we will need to do another cycle to import the resource
	if deferred != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Cannot import deferred remote object",
			fmt.Sprintf(
				"While attempting to import an existing object to %q, "+
					"the provider deferred reading the resource. "+
					"This is a bug in the provider since deferrals are not supported when importing through the CLI, please file an issue."+
					"Please either use an import block for importing this resource "+
					"or remove the to be imported resource from your configuration, "+
					"apply the configuration using \"terraform apply\", "+
					"add the to be imported resource again, and retry the import operation.",
				n.TargetAddr,
			),
		))
	} else {
		// Verify the existance of the imported resource
		if state.Value.IsNull() {
			var diags tfdiags.Diagnostics
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Cannot import non-existent remote object",
				fmt.Sprintf(
					"While attempting to import an existing object to %q, "+
						"the provider detected that no object exists with the given id. "+
						"Only pre-existing objects can be imported; check that the id "+
						"is correct and that it is associated with the provider's "+
						"configured region or endpoint, or use \"terraform apply\" to "+
						"create a new remote object for this resource.",
					n.TargetAddr,
				),
			))
			return diags
		}
	}

	diags = diags.Append(riNode.writeResourceInstanceState(ctx, state, workingState))
	return diags
}
