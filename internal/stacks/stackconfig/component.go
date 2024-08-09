// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackconfig

import (
	"fmt"

	"github.com/apparentlymart/go-versions/versions/constraints"
	"github.com/hashicorp/go-slug/sourceaddrs"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Component represents the declaration of a single component within a
// particular [Stack].
//
// Components are the most important object in a stack configuration, just as
// resources are the most important object in a Terraform module: each one
// refers to a Terraform module that describes the infrastructure that the
// component is "made of".
type Component struct {
	Name string

	SourceAddr                               sourceaddrs.Source
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

	ForEach hcl.Expression

	// Inputs is an expression that should produce a value that can convert
	// to an object type derived from the component's input variable
	// declarations, and whose attribute values will then be used to populate
	// those input variables.
	Inputs hcl.Expression

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

	// DependsOn forces a dependency between this resource and the list
	// resources, allowing users to specify ordering of components without
	// direct references.
	DependsOn []hcl.Traversal

	DeclRange tfdiags.SourceRange
}

func decodeComponentBlock(block *hcl.Block) (*Component, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	ret := &Component{
		Name:      block.Labels[0],
		DeclRange: tfdiags.SourceRangeFromHCL(block.DefRange),
	}
	if !hclsyntax.ValidIdentifier(ret.Name) {
		diags = diags.Append(invalidNameDiagnostic(
			"Invalid component name",
			block.LabelRanges[0],
		))
		return nil, diags
	}

	content, hclDiags := block.Body.Content(componentBlockSchema)
	diags = diags.Append(hclDiags)
	if hclDiags.HasErrors() {
		return nil, diags
	}

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
	if attr, ok := content.Attributes["inputs"]; ok {
		ret.Inputs = attr.Expr
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
	if attr, exists := content.Attributes["depends_on"]; exists {
		ret.DependsOn, hclDiags = configs.DecodeDependsOn(attr)
		diags = diags.Append(hclDiags)
	}

	return ret, diags
}

func decodeSourceAddrArguments(sourceAttr, versionAttr *hcl.Attribute) (sourceaddrs.Source, constraints.IntersectionSpec, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	var sourceStr string
	hclDiags := gohcl.DecodeExpression(sourceAttr.Expr, nil, &sourceStr)
	diags = diags.Append(hclDiags)
	if hclDiags.HasErrors() {
		return nil, nil, diags
	}

	sourceAddr, err := sourceaddrs.ParseSource(sourceStr)
	if err != nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid source address",
			Detail: fmt.Sprintf(
				"Cannot parse %q as a source address: %s.",
				sourceStr, err,
			),
			Subject: sourceAttr.Expr.Range().Ptr(),
		})
		return nil, nil, diags
	}

	var versionConstraints constraints.IntersectionSpec
	if sourceAddr.SupportsVersionConstraints() {
		if versionAttr == nil {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Missing required version constraints",
				Detail:   "The specified source address requires version constraints specified in a separate \"version\" argument.",
				Subject:  sourceAttr.Expr.Range().Ptr(),
			})
			return nil, nil, diags
		}
		var versionStr string
		hclDiags := gohcl.DecodeExpression(versionAttr.Expr, nil, &versionStr)
		diags = diags.Append(hclDiags)
		if hclDiags.HasErrors() {
			return nil, nil, diags
		}
		versionConstraints, err = constraints.ParseRubyStyleMulti(versionStr)
		if err != nil {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid version constraints",
				Detail: fmt.Sprintf(
					"Cannot parse %q as source package version constraints: %s.",
					versionStr, err,
				),
				Subject: versionAttr.Expr.Range().Ptr(),
			})
			return nil, nil, diags
		}
	} else {
		if versionAttr != nil {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Unsupported version constraints",
				Detail:   "The specified source address does not support version constraints.",
				Subject:  versionAttr.Range.Ptr(),
			})
			return nil, nil, diags
		}
	}

	return sourceAddr, versionConstraints, diags
}

var componentBlockSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{Name: "source", Required: true},
		{Name: "version", Required: false},
		{Name: "for_each", Required: false},
		{Name: "inputs", Required: false},
		{Name: "providers", Required: false},
		{Name: "depends_on", Required: false},
	},
}
