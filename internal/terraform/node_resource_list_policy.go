// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

// listResourcePolicy holds the policy evaluation inputs for a single resource
// discovered during list block execution.
type listResourcePolicy struct {
	// SyntheticAddr is the unique managed-mode address assigned to this
	// discovered resource. The naming formula matches genconfig.GenerateListResourceContents:
	//
	//   no key:  <type>.<list-resource-name>_<idx>
	//   keyed:   <type>.<list-resource-name>_<expansionEnum>_<idx>
	SyntheticAddr addrs.AbsResourceInstance

	// GeneratedConfig is the cty config value from the provider's
	// GenerateResourceConfig RPC or genconfig.ExtractLegacyConfigFromState (fallback).
	// Zero value when Unknown is true.
	GeneratedConfig cty.Value

	// Identity is the identity cty object from the list response element.
	Identity cty.Value

	// ResourceConfig is the list block's *configs.Resource. Downstream policy
	// nodes use this for source location in diagnostics (DeclRange).
	ResourceConfig *configs.Resource

	// ListBlockAddr is the AbsResourceInstance address of the originating list
	// block.
	ListBlockAddr addrs.AbsResourceInstance

	// Unknown is true when the resource had no "state" attribute in the list
	// response (include_resource = false), preventing config generation.
	Unknown bool

	// Diags carries an explanatory Warning when Unknown is true, or error
	// diagnostics when config generation failed.
	Diags tfdiags.Diagnostics
}

// generateListResourcePolicyData iterates over the discovered resources in a
// list block response and generates per-resource config data required for
// policy evaluation.
//
// data must be the "data" attribute of the provider's ListResource response
// (resp.Result.GetAttr("data")). listBlockAddr is the AbsResourceInstance of
// the list block walk node (n.ResourceInstanceAddr()).
func (n *NodePlannableResourceInstance) generateListResourcePolicyData(
	ctx EvalContext,
	listBlockAddr addrs.AbsResourceInstance,
	data cty.Value,
) ([]listResourcePolicy, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	if !data.CanIterateElements() {
		// Null or unknown data — no resources to process.
		return nil, diags
	}

	// Expansion enum required for the keyed synthetic address formula.
	expansionEnum := ctx.InstanceExpander().ResourceExpansionEnum(listBlockAddr)

	var results []listResourcePolicy

	iter := data.ElementIterator()
	for idx := 0; iter.Next(); idx++ {
		_, val := iter.Element()

		// Build the synthetic address using the same formula as
		// genconfig.GenerateListResourceContents.
		var syntheticName string
		if listBlockAddr.Resource.Key == addrs.NoKey {
			syntheticName = fmt.Sprintf("%s_%d", listBlockAddr.Resource.Resource.Name, idx)
		} else {
			syntheticName = fmt.Sprintf("%s_%d_%d", listBlockAddr.Resource.Resource.Name, expansionEnum, idx)
		}
		syntheticAddr := addrs.AbsResourceInstance{
			Module: listBlockAddr.Module,
			Resource: addrs.ResourceInstance{
				Resource: addrs.Resource{
					Mode: addrs.ManagedResourceMode,
					Type: listBlockAddr.Resource.Resource.Type,
					Name: syntheticName,
				},
				Key: addrs.NoKey,
			},
		}

		// Extract the identity object from the element.
		var identity cty.Value
		if val.Type().HasAttribute("identity") {
			identity = val.GetAttr("identity")
		}

		// Absent "state" means include_resource = false.
		// skip with Unknown outcome.
		hasState := val.Type().HasAttribute("state") && !val.GetAttr("state").IsNull()
		if !hasState {
			results = append(results, listResourcePolicy{
				SyntheticAddr:  syntheticAddr,
				Identity:       identity,
				ResourceConfig: n.Config,
				ListBlockAddr:  listBlockAddr,
				Unknown:        true,
				Diags: tfdiags.Diagnostics{tfdiags.Sourceless(
					tfdiags.Warning,
					"Policy evaluation skipped",
					fmt.Sprintf(
						"Resource at index %d in list block %s has no state "+
							"(include_resource = false). Policy evaluation "+
							"cannot be performed without resource state.",
						idx, listBlockAddr.String(),
					),
				)},
			})
			continue
		}

		stateVal := val.GetAttr("state")

		// Provider RPC first, legacy fallback second; on failure record Unknown and continue.
		// Provider GenerateResourceConfig RPC failure is the most likely error source.
		generatedConfig, configDiags := n.generateResourceConfig(ctx, stateVal)
		if configDiags.HasErrors() {
			// FIXME: if configDiags contains errors, consider whether to propagate them
			// in the returned diags (surfacing at the list block execution level) or
			// keep them contained in the listResourcePolicy Diags field only. Currently
			// they are appended to both — see the matching FIXME in listResourceExecute.
			diags = diags.Append(configDiags)
			results = append(results, listResourcePolicy{
				SyntheticAddr:  syntheticAddr,
				Identity:       identity,
				ResourceConfig: n.Config,
				ListBlockAddr:  listBlockAddr,
				Unknown:        true,
				Diags: tfdiags.Diagnostics{tfdiags.Sourceless(
					tfdiags.Warning,
					"Policy evaluation skipped",
					fmt.Sprintf(
						"Resource at index %d in list block %s could not generate config. "+
							"Policy evaluation cannot be performed.",
						idx, listBlockAddr.String(),
					),
				)}.Append(configDiags.InConfigBody(n.Config.Config, listBlockAddr.String())),
			})
			continue
		}

		// Successfully captured policy evaluation inputs for this discovered resource.
		results = append(results, listResourcePolicy{
			SyntheticAddr:   syntheticAddr,
			GeneratedConfig: generatedConfig,
			Identity:        identity,
			ResourceConfig:  n.Config,
			ListBlockAddr:   listBlockAddr,
		})
	}

	return results, diags
}
