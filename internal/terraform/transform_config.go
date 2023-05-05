// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package terraform

import (
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
)

// ConfigTransformer is a GraphTransformer that adds all the resources
// from the configuration to the graph.
//
// The module used to configure this transformer must be the root module.
//
// Only resources are added to the graph. Variables, outputs, and
// providers must be added via other transforms.
//
// Unlike ConfigTransformerOld, this transformer creates a graph with
// all resources including module resources, rather than creating module
// nodes that are then "flattened".
type ConfigTransformer struct {
	Concrete ConcreteResourceNodeFunc

	// Module is the module to add resources from.
	Config *configs.Config

	// Mode will only add resources that match the given mode
	ModeFilter bool
	Mode       addrs.ResourceMode

	// Do not apply this transformer.
	skip bool

	// importTargets specifies a slice of addresses that will have state
	// imported for them.
	importTargets []*ImportTarget

	// generateConfigForImportTargets tells the graph to generate config for any
	// import targets that are not contained within config.
	//
	// If this is false and an import target has no config, the graph will
	// simply import the state for the target and any follow-up operations will
	// try to delete the imported resource unless the config is updated
	// manually.
	generateConfigForImportTargets bool
}

func (t *ConfigTransformer) Transform(g *Graph) error {
	if t.skip {
		return nil
	}

	// If no configuration is available, we don't do anything
	if t.Config == nil {
		return nil
	}

	// Start the transformation process
	return t.transform(g, t.Config, t.generateConfigForImportTargets)
}

func (t *ConfigTransformer) transform(g *Graph, config *configs.Config, generateConfig bool) error {
	// If no config, do nothing
	if config == nil {
		return nil
	}

	// Add our resources
	if err := t.transformSingle(g, config, generateConfig); err != nil {
		return err
	}

	// Transform all the children without generating config.
	for _, c := range config.Children {
		if err := t.transform(g, c, false); err != nil {
			return err
		}
	}

	return nil
}

func (t *ConfigTransformer) transformSingle(g *Graph, config *configs.Config, generateConfig bool) error {
	path := config.Path
	module := config.Module
	log.Printf("[TRACE] ConfigTransformer: Starting for path: %v", path)

	allResources := make([]*configs.Resource, 0, len(module.ManagedResources)+len(module.DataResources))
	for _, r := range module.ManagedResources {
		allResources = append(allResources, r)
	}
	for _, r := range module.DataResources {
		allResources = append(allResources, r)
	}

	// Take a copy of the import targets, so we can edit them as we go.
	var importTargets []*ImportTarget
	importTargets = append(importTargets, t.importTargets...)

	for _, r := range allResources {
		relAddr := r.Addr()

		if t.ModeFilter && relAddr.Mode != t.Mode {
			// Skip non-matching modes
			continue
		}

		// If any of the import targets can apply to this node's instances,
		// filter them down to the applicable addresses.
		var imports []*ImportTarget
		configAddr := relAddr.InModule(path)

		var matchedIndices []int
		for ix, i := range importTargets {
			if target := i.Addr.ContainingResource().Config(); target.Equal(configAddr) {
				// This import target has been claimed by an actual resource,
				// let's make a note of this to remove it from the targets.
				matchedIndices = append(matchedIndices, ix)
				imports = append(imports, i)
			}
		}

		for ix := len(matchedIndices) - 1; ix >= 0; ix-- {
			tIx := matchedIndices[ix]

			// We do this backwards, since it means we don't have to adjust the
			// later indices as we change the length of import targets.
			//
			// We need to do this separately, as a single resource could match
			// multiple import targets.
			importTargets = append(importTargets[:tIx], importTargets[tIx+1:]...)
		}

		abstract := &NodeAbstractResource{
			Addr: addrs.ConfigResource{
				Resource: relAddr,
				Module:   path,
			},
			importTargets: imports,
		}

		var node dag.Vertex = abstract
		if f := t.Concrete; f != nil {
			node = f(abstract)
		}

		g.Add(node)
	}

	if generateConfig {
		// If any import targets were not claimed by resources, then we will
		// generate config for them.
		for _, i := range importTargets {
			if !i.Addr.Module.IsRoot() {
				// We only generate config for resources imported into the root
				// module.
				continue
			}

			abstract := &NodeAbstractResource{
				Addr:          i.Addr.ConfigResource(),
				importTargets: []*ImportTarget{i},
			}

			var node dag.Vertex = abstract
			if f := t.Concrete; f != nil {
				node = f(abstract)
			}

			g.Add(node)
		}
	}

	return nil
}
