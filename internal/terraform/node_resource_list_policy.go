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

// listResourcePolicyUnknownReason classifies why a discovered resource cannot
// be evaluated for policy. The zero value unknownReasonNone indicates the
// resource is a valid policy input with no skip.
type listResourcePolicyUnknownReason uint8

const (
	unknownReasonNone            listResourcePolicyUnknownReason = iota
	unknownReasonNoState                                         // include_resource = false or state absent from list response
	unknownReasonConfigGenFailed                                 // provider RPC or legacy fallback could not produce config
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

	// ListBlockAddr is the AbsResourceInstance address of the originating list block.
	ListBlockAddr addrs.AbsResourceInstance

	// Unknown is true when the resource had no "state" attribute in the list
	// response (include_resource = false), preventing config generation.
	Unknown bool

	// UnknownReason classifies why Unknown is true. unknownReasonNone when Unknown is false.
	UnknownReason listResourcePolicyUnknownReason
}

// generateListResourcePolicyData iterates over the discovered resources in a
// list block response and generates per-resource config data required for
// policy evaluation.
func (n *NodePlannableResourceInstance) generateListResourcePolicyData(
	ctx EvalContext,
	listBlockAddr addrs.AbsResourceInstance,
	data cty.Value,
) ([]listResourcePolicy, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// Check null or unknwon data
	if !data.CanIterateElements() {
		// no resources to process.
		return nil, diags
	}

	// Expansion enum required for the keyed synthetic address formula.
	expansionEnum := ctx.InstanceExpander().ResourceExpansionEnum(listBlockAddr)

	var results []listResourcePolicy
	// Counters to aggregate possible diagnostics from enumerated list.
	var unknownCount, configErrCount int

	iter := data.ElementIterator()
	for idx := 0; iter.Next(); idx++ {
		_, val := iter.Element()

		// Build the synthetic address using the same formula as genconfig.GenerateListResourceContents.
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

		// Absent "state" means include_resource is false so we'll skip with Unknown outcome.
		hasState := val.Type().HasAttribute("state") && !val.GetAttr("state").IsNull()
		if !hasState {
			unknownCount++
			results = append(results, listResourcePolicy{
				SyntheticAddr:  syntheticAddr,
				Identity:       identity,
				ResourceConfig: n.Config,
				ListBlockAddr:  listBlockAddr,
				Unknown:        true,
				UnknownReason:  unknownReasonNoState,
			})
			continue
		}

		stateVal := val.GetAttr("state")

		// Call Provider RPC to generate configuration with fallback to legacy extraction config from state
		generatedConfig, configDiags := n.generateResourceConfig(ctx, stateVal)
		// Handle Provider GenerateResourceConfig RPC failure.
		if configDiags.HasErrors() {
			configErrCount++
			results = append(results, listResourcePolicy{
				SyntheticAddr:  syntheticAddr,
				Identity:       identity,
				ResourceConfig: n.Config,
				ListBlockAddr:  listBlockAddr,
				Unknown:        true,
				UnknownReason:  unknownReasonConfigGenFailed,
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

	// Emit one consolidated warning per skip category rather than one per resource.
	if unknownCount > 0 {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Warning,
			"Policy evaluation skipped",
			fmt.Sprintf(
				"%d resource(s) in list block %s have no state (include_resource = false). "+
					"Policy evaluation cannot be performed without resource state.",
				unknownCount, listBlockAddr.String(),
			),
		))
	}
	if configErrCount > 0 {
		// Config generation errors are unexpected but possible.
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Warning,
			"Policy evaluation skipped",
			fmt.Sprintf(
				"%d resource(s) in list block %s could not generate config for policy evaluation.",
				configErrCount, listBlockAddr.String(),
			),
		))
	}

	return results, diags
}
