// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/tfdiags"
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

	// some actions are skipped during the destroy process
	destroy bool

	// importTargets specifies a slice of addresses that will have state
	// imported for them.
	importTargets []*ImportTarget

	// generateConfigPathForImportTargets tells the graph where to write any
	// generated config for import targets that are not contained within config.
	//
	// If this is empty and an import target has no config, the graph will
	// simply import the state for the target and any follow-up operations will
	// try to delete the imported resource unless the config is updated
	// manually.
	generateConfigPathForImportTargets string
}

func (t *ConfigTransformer) Transform(g *Graph) error {
	// If no configuration is available, we don't do anything
	if t.Config == nil {
		return nil
	}

	if err := t.validateImportTargets(); err != nil {
		return err
	}

	// Start the transformation process
	return t.transform(g, t.Config)
}

func (t *ConfigTransformer) transform(g *Graph, config *configs.Config) error {
	// If no config, do nothing
	if config == nil {
		return nil
	}

	// Add our resources
	if err := t.transformSingle(g, config); err != nil {
		return err
	}

	// Transform all the children without generating config.
	for _, c := range config.Children {
		if err := t.transform(g, c); err != nil {
			return err
		}
	}

	return nil
}

func (t *ConfigTransformer) transformSingle(g *Graph, config *configs.Config) error {
	path := config.Path
	module := config.Module
	log.Printf("[TRACE] ConfigTransformer: Starting for path: %v", path)

	var allResources []*configs.Resource
	if !t.destroy {
		for _, r := range module.ManagedResources {
			allResources = append(allResources, r)
		}
		for _, r := range module.DataResources {
			allResources = append(allResources, r)
		}
	}

	// ephemeral resources act like temporary values and must be added to the
	// graph even during destroy operations.
	for _, r := range module.EphemeralResources {
		allResources = append(allResources, r)
	}

	// Take a copy of the import targets, so we can edit them as we go.
	// Only include import targets that are targeting the current module.
	var importTargets []*ImportTarget
	for _, target := range t.importTargets {
		switch {
		case target.Config == nil:
			if target.LegacyAddr.Module.Module().Equal(config.Path) {
				importTargets = append(importTargets, target)
			}
		default:
			if target.Config.ToResource.Module.Equal(config.Path) {
				importTargets = append(importTargets, target)
			}
		}
	}

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
			if i.LegacyAddr.ConfigResource().Equal(configAddr) {
				matchedIndices = append(matchedIndices, ix)
				imports = append(imports, i)

			}
			if i.Config != nil && i.Config.ToResource.Equal(configAddr) {
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

	// If any import targets were not claimed by resources we may be
	// generating configuration. Add them to the graph for validation.
	for _, i := range importTargets {
		log.Printf("[DEBUG] ConfigTransformer: adding config generation node for %s", i.Config.ToResource)

		// TODO: if config generation is ever supported for for_each
		// resources, this will add multiple nodes for the same
		// resource
		abstract := &NodeAbstractResource{
			Addr:               i.Config.ToResource,
			importTargets:      []*ImportTarget{i},
			generateConfigPath: t.generateConfigPathForImportTargets,
		}
		var node dag.Vertex = abstract
		if f := t.Concrete; f != nil {
			node = f(abstract)
		}

		g.Add(node)
	}
	return nil
}

// validateImportTargets ensures that the import target module exists in the
// configuration. Individual resources will be check by the validation node.
func (t *ConfigTransformer) validateImportTargets() error {
	if t.destroy {
		return nil
	}
	var diags tfdiags.Diagnostics

	for _, i := range t.importTargets {
		var toResource addrs.ConfigResource
		switch {
		case i.Config != nil:
			toResource = i.Config.ToResource
		default:
			toResource = i.LegacyAddr.ConfigResource()
		}

		moduleCfg := t.Config.Root.Descendant(toResource.Module)
		if moduleCfg == nil {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Configuration for import target does not exist",
				Detail:   fmt.Sprintf("The configuration for the given import target %s does not exist. All target instances must have an associated configuration to be imported.", i.Config.ToResource),
				Subject:  i.Config.To.Range().Ptr(),
			})
		}
	}

	return diags.Err()
}
