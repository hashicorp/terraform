// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configschema"

	"github.com/hashicorp/terraform/internal/dag"
)

// ConcreteProviderNodeFunc is a callback type used to convert an
// abstract provider to a concrete one of some type.
type ConcreteProviderNodeFunc func(*NodeAbstractProvider) dag.Vertex

// NodeAbstractProvider represents a provider that has no associated operations.
// It registers all the common interfaces across operations for providers.
type NodeAbstractProvider struct {
	Addr addrs.AbsProviderConfig

	// The fields below will be automatically set using the Attach
	// interfaces if you're running those transforms, but also be explicitly
	// set if you already have that information.

	Config *configs.Provider
	Schema *configschema.Block
}

var (
	_ GraphNodeModulePath                 = (*NodeAbstractProvider)(nil)
	_ GraphNodeReferencer                 = (*NodeAbstractProvider)(nil)
	_ GraphNodeProvider                   = (*NodeAbstractProvider)(nil)
	_ GraphNodeAttachProvider             = (*NodeAbstractProvider)(nil)
	_ GraphNodeAttachProviderConfigSchema = (*NodeAbstractProvider)(nil)
	_ dag.GraphNodeDotter                 = (*NodeAbstractProvider)(nil)
)

func (n *NodeAbstractProvider) Name() string {
	return n.Addr.String()
}

// GraphNodeModuleInstance
func (n *NodeAbstractProvider) Path() addrs.ModuleInstance {
	// Providers cannot be contained inside an expanded module, so this shim
	// converts our module path to the correct ModuleInstance.
	return n.Addr.Module.UnkeyedInstanceShim()
}

// GraphNodeModulePath
func (n *NodeAbstractProvider) ModulePath() addrs.Module {
	return n.Addr.Module
}

// GraphNodeReferencer
func (n *NodeAbstractProvider) References() []*addrs.Reference {
	if n.Config == nil || n.Schema == nil {
		return nil
	}

	return ReferencesFromConfig(n.Config.Config, n.Schema)
}

// GraphNodeProvider
func (n *NodeAbstractProvider) ProviderAddr() addrs.AbsProviderConfig {
	return n.Addr
}

// GraphNodeProvider
func (n *NodeAbstractProvider) ProviderConfig() *configs.Provider {
	if n.Config == nil {
		return nil
	}

	return n.Config
}

// GraphNodeAttachProvider
func (n *NodeAbstractProvider) AttachProvider(c *configs.Provider) {
	n.Config = c
}

// GraphNodeAttachProviderConfigSchema impl.
func (n *NodeAbstractProvider) AttachProviderConfigSchema(schema *configschema.Block) {
	n.Schema = schema
}

// GraphNodeDotter impl.
func (n *NodeAbstractProvider) DotNode(name string, opts *dag.DotOpts) *dag.DotNode {
	return &dag.DotNode{
		Name: name,
		Attrs: map[string]string{
			"label": n.Name(),
			"shape": "diamond",
		},
	}
}
