// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/lang/langrefs"
	"github.com/hashicorp/terraform/internal/providers"
)

// GraphNodeConfigAction is implemented by any nodes that represent an action.
// The type of operation cannot be assumed, only that this node represents
// the given resource.
type GraphNodeConfigAction interface {
	ActionAddr() addrs.ConfigAction
}

// nodeAbstractActionDeclaration represents an action config block in a configuration module.
type nodeAbstractActionDeclaration struct {
	Addr   addrs.ConfigAction
	Config configs.Action

	Schema           *providers.ActionSchema
	ResolvedProvider addrs.AbsProviderConfig
}

var (
	_ GraphNodeConfigAction       = (*nodeAbstractActionDeclaration)(nil)
	_ GraphNodeReferenceable      = (*nodeAbstractActionDeclaration)(nil)
	_ GraphNodeReferencer         = (*nodeAbstractActionDeclaration)(nil)
	_ GraphNodeProviderConsumer   = (*nodeAbstractActionDeclaration)(nil)
	_ GraphNodeAttachActionSchema = (*nodeAbstractActionDeclaration)(nil)
)

func (n *nodeAbstractActionDeclaration) Name() string {
	return n.Addr.String()
}

func (n *nodeAbstractActionDeclaration) ActionAddr() addrs.ConfigAction {
	return n.Addr
}

func (n *nodeAbstractActionDeclaration) ReferenceableAddrs() []addrs.Referenceable {
	return []addrs.Referenceable{n.Addr.Action}
}

// GraphNodeModulePath
func (n *nodeAbstractActionDeclaration) ModulePath() addrs.Module {
	return n.Addr.Module
}

// GraphNodeAttachActionSchema
func (n *nodeAbstractActionDeclaration) AttachActionSchema(schema *providers.ActionSchema) {
	n.Schema = schema
}

// GraphNodeReferencer
func (n *nodeAbstractActionDeclaration) References() []*addrs.Reference {
	var result []*addrs.Reference
	c := n.Config

	refs, _ := langrefs.ReferencesInExpr(addrs.ParseRef, c.Count)
	result = append(result, refs...)
	refs, _ = langrefs.ReferencesInExpr(addrs.ParseRef, c.ForEach)
	result = append(result, refs...)

	if n.Schema != nil {
		refs, _ = langrefs.ReferencesInBlock(addrs.ParseRef, c.Config, n.Schema.ConfigSchema)
		result = append(result, refs...)
	}

	return result
}

// GraphNodeProviderConsumer
func (n *nodeAbstractActionDeclaration) ProvidedBy() (addrs.ProviderConfig, bool) {
	// Once the provider is fully resolved, we can return the known value.
	if n.ResolvedProvider.Provider.Type != "" {
		return n.ResolvedProvider, true
	}

	// Since we always have a config, we can use it
	relAddr := n.Config.ProviderConfigAddr()
	return addrs.LocalProviderConfig{
		LocalName: relAddr.LocalName,
		Alias:     relAddr.Alias,
	}, false
}

// GraphNodeProviderConsumer
func (n *nodeAbstractActionDeclaration) Provider() addrs.Provider {
	return n.Config.Provider
}

// GraphNodeProviderConsumer
func (n *nodeAbstractActionDeclaration) SetProvider(p addrs.AbsProviderConfig) {
	n.ResolvedProvider = p
}
