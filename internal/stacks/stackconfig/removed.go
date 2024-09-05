// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackconfig

import (
	"github.com/apparentlymart/go-versions/versions/constraints"
	"github.com/hashicorp/go-slug/sourceaddrs"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Removed represents a component that was removed from the configuration.
//
// Removed blocks don't have labels associated with them, instead they have
// a "from" attribute that points directly to the old component that was
// removed. Removed blocks can also point to component instances specifically,
// using an index expression. The "for_each" attribute also means that the
// "from" attribute can't always be evaluated statically.
//
// Removed blocks are, therefore, represented by the FromComponent and FromIndex
// fields, which together represent the address of the removed component. The
// FromComponent field is the address of the component itself, and the FromIndex
// field is the index expression that will be evaluated to determine the
// specific instance of the component that was removed.
//
// FromIndex can be null if either the removed block is pointing to a component
// that was not instanced, or is pointing to all the instances of a removed
// component.
//
// For this reason, multiple Removed blocks can be associated with the same
// FromComponent, but with different FromIndex values. When the FromIndex values
// are evaluated, during the planning stage, we will validate that the FromIndex
// values are unique.
type Removed struct {
	FromComponent stackaddrs.Component
	FromIndex     hcl.Expression

	SourceAddr                               sourceaddrs.Source
	VersionConstraints                       constraints.IntersectionSpec
	SourceAddrRange, VersionConstraintsRange tfdiags.SourceRange

	// FinalSourceAddr is populated only when a configuration is loaded
	// through [LoadConfigDir], and in that case contains the finalized
	// address produced by resolving the SourceAddr field relative to
	// the address of the file where the component was declared. This
	// is the address to use if you intend to load the component's
	// root module from a source bundle.
	FinalSourceAddr sourceaddrs.FinalSource

	ForEach hcl.Expression

	// ProviderConfigs describes the mapping between the static provider
	// configuration slots declared in the component's root module and the
	// dynamic provider configuration objects in scope in the calling
	// stack configuration.
	//
	// This map deals with the slight schism between the stacks language's
	// treatment of provider configurations as regular values of a special
	// data type vs. the main Terraform language's treatment of provider
	// configurations as something special passed out of band from the
	// input variables. The overall structure and the map keys are fixed
	// statically during decoding, but the final provider configuration objects
	// are determined only at runtime by normal expression evaluation.
	//
	// The keys of this map refer to provider configuration slots inside
	// the module being called, but use the local names defined in the
	// calling stack configuration. The stacks language runtime will
	// translate the caller's local names into the callee's declared provider
	// configurations by using the stack configuration's table of local
	// provider names.
	ProviderConfigs map[addrs.LocalProviderConfig]hcl.Expression

	// Destroy controls whether this removed block will actually destroy all
	// instances of resources within this component, or just removed them from
	// the state. Defaults to true.
	Destroy bool

	DeclRange tfdiags.SourceRange
}

func decodeRemovedBlock(block *hcl.Block) (*Removed, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	ret := &Removed{
		DeclRange: tfdiags.SourceRangeFromHCL(block.DefRange),
	}

	content, hclDiags := block.Body.Content(removedBlockSchema)
	diags = diags.Append(hclDiags)
	if hclDiags.HasErrors() {
		return nil, diags
	}

	// We're splitting out the component and the index now, as we can decode and
	// analyse the component now. The index might be referencing the for_each
	// variable, which we can't decode yet.
	component, index, moreDiags := stackaddrs.ParseRemovedFrom(content.Attributes["from"].Expr)
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		return nil, diags
	}
	ret.FromComponent = component
	ret.FromIndex = index

	sourceAddr, versionConstraints, moreDiags := decodeSourceAddrArguments(
		content.Attributes["source"],
		content.Attributes["version"],
	)
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		return nil, diags
	}

	ret.SourceAddr = sourceAddr
	ret.VersionConstraints = versionConstraints
	ret.SourceAddrRange = tfdiags.SourceRangeFromHCL(content.Attributes["source"].Range)
	if content.Attributes["version"] != nil {
		ret.VersionConstraintsRange = tfdiags.SourceRangeFromHCL(content.Attributes["version"].Range)
	}
	// Now that we've populated the mandatory source location fields we can
	// safely return a partial ret if we encounter any further errors, as
	// long as we leave the other fields either unset or in some other
	// reasonable state for careful partial analysis.

	if attr, ok := content.Attributes["for_each"]; ok {
		ret.ForEach = attr.Expr
	}
	if attr, ok := content.Attributes["providers"]; ok {
		var providerDiags tfdiags.Diagnostics
		ret.ProviderConfigs, providerDiags = decodeProvidersAttribute(attr)
		diags = diags.Append(providerDiags)
	}

	ret.Destroy = true // default to true
	for _, block := range content.Blocks {
		switch block.Type {
		case "lifecycle":
			lcContent, lcDiags := block.Body.Content(removedLifecycleBlockSchema)
			diags = diags.Append(lcDiags)

			if attr, ok := lcContent.Attributes["destroy"]; ok {
				valDiags := gohcl.DecodeExpression(attr.Expr, nil, &ret.Destroy)
				diags = diags.Append(valDiags)
			}
		}
	}

	return ret, diags
}

var removedBlockSchema = &hcl.BodySchema{
	Blocks: []hcl.BlockHeaderSchema{
		{Type: "lifecycle"},
	},
	Attributes: []hcl.AttributeSchema{
		{Name: "from", Required: true},
		{Name: "source", Required: true},
		{Name: "version", Required: false},
		{Name: "for_each", Required: false},
		{Name: "providers", Required: false},
	},
}

var removedLifecycleBlockSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{Name: "destroy"},
	},
}
