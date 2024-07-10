// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackconfig

import (
	"fmt"

	"github.com/apparentlymart/go-versions/versions/constraints"
	"github.com/hashicorp/go-slug/sourceaddrs"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type Removed struct {
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

	From       stackaddrs.AbsComponentInstance
	SourceAddr sourceaddrs.Source

	VersionConstraints                       constraints.IntersectionSpec
	SourceAddrRange, VersionConstraintsRange tfdiags.SourceRange

	// FinalSourceAddr is populated only when a configuration is loaded
	// through [LoadConfigDir], and in that case contains the finalized
	// address produced by resolving the SourceAddr field relative to
	// the address of the file where the component was declared. This
	// is the address to use if you intend to load the component's
	// root module from a source bundle.
	//
	// If this Component was created through one of the narrower configuration
	// loading functions, such as [LoadSingleStackConfig] or [ParseFileSource],
	// then this field will be nil and it won't be possible to determine the
	// finalized source location for the root module.
	FinalSourceAddr sourceaddrs.FinalSource

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

	// TODO: Do we want to early-return here?
	if content.Attributes["from"] == nil {
		diags = diags.Append(&hcl.Diagnostic{Severity: hcl.DiagError,
			Summary: "missing required argument from",
			Detail:  "To remove a component, you must specify the source component to remove it from.",
			Subject: block.DefRange.Ptr(),
		})
		return nil, diags
	}
	traversal, traversalDiags := hcl.AbsTraversalForExpr(content.Attributes["from"].Expr)
	if traversalDiags.HasErrors() {
		diags = diags.Append(traversalDiags)
		return nil, diags
	}
	// TODO: What about each key in for_each
	from, parseComponentInstanceDiags := stackaddrs.ParseAbsComponentInstance(traversal)
	if parseComponentInstanceDiags.HasErrors() {
		diags = diags.Append(traversalDiags)
		return nil, diags
	}
	ret.From = from

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
		// This particular argument has some enforced static structure because
		// it's populating an inflexible part of Terraform Core's input.
		// This argument, if present, must always be an object constructor
		// whose attributes are Terraform Core-style provider configuration
		// addresses, but whose values are just arbitrary expressions for now
		// and will be resolved into specific provider configuration addresses
		// dynamically at runtime.
		pairs, hclDiags := hcl.ExprMap(attr.Expr)
		diags = diags.Append(hclDiags)
		if !hclDiags.HasErrors() {
			ret.ProviderConfigs = make(map[addrs.LocalProviderConfig]hcl.Expression, len(pairs))
			for _, pair := range pairs {
				insideAddrExpr := pair.Key
				outsideAddrExpr := pair.Value

				traversal, hclDiags := hcl.AbsTraversalForExpr(insideAddrExpr)
				diags = diags.Append(hclDiags)
				if hclDiags.HasErrors() {
					continue
				}

				if len(traversal) < 1 || len(traversal) > 2 {
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Invalid provider configuration reference",
						Detail:   "Each item in the providers argument requires a provider local name, optionally followed by a period and then a configuration alias, matching one of the provider configuration import slots declared by the component's root module.",
						Subject:  insideAddrExpr.Range().Ptr(),
					})
					continue
				}

				localName := traversal.RootName()
				if !hclsyntax.ValidIdentifier(localName) {
					diags = diags.Append(invalidNameDiagnostic(
						"Invalid provider local name",
						traversal[0].SourceRange(),
					))
					continue
				}

				var alias string
				if len(traversal) > 1 {
					aliasStep, ok := traversal[1].(hcl.TraverseAttr)
					if !ok {
						diags = diags.Append(&hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  "Invalid provider configuration reference",
							Detail:   "Provider local name must either stand alone or be followed by a period and then a configuration alias.",
							Subject:  traversal[1].SourceRange().Ptr(),
						})
						continue
					}
					alias = aliasStep.Name
				}

				addr := addrs.LocalProviderConfig{
					LocalName: localName,
					Alias:     alias,
				}
				if existing, exists := ret.ProviderConfigs[addr]; exists {
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Duplicate provider configuration assignment",
						Detail: fmt.Sprintf(
							"A provider configuration for %s was already assigned at %s.",
							addr.StringCompact(), existing.Range().Ptr(),
						),
						Subject: outsideAddrExpr.Range().Ptr(),
					})
					continue
				} else {
					ret.ProviderConfigs[addr] = outsideAddrExpr
				}
			}
		}
	}

	return ret, diags
}

var removedBlockSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{Name: "from", Required: true},
		{Name: "source", Required: true},
		{Name: "version", Required: false},
		{Name: "for_each", Required: false},
		{Name: "providers", Required: false},
	},
}
