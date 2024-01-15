// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackconfig

import (
	"fmt"

	"github.com/apparentlymart/go-versions/versions/constraints"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type ProviderRequirements struct {
	Requirements map[string]ProviderRequirement

	DeclRange tfdiags.SourceRange
}

type ProviderRequirement struct {
	LocalName string

	Provider           addrs.Provider
	VersionConstraints constraints.IntersectionSpec

	DeclRange tfdiags.SourceRange
}

func decodeProviderRequirementsBlock(block *hcl.Block) (*ProviderRequirements, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	attrs, hclDiags := block.Body.JustAttributes()
	diags = diags.Append(hclDiags)
	if len(attrs) == 0 {
		return nil, diags
	}

	reverseMap := make(map[addrs.Provider]string)

	ret := &ProviderRequirements{
		Requirements: make(map[string]ProviderRequirement, len(attrs)),
		DeclRange:    tfdiags.SourceRangeFromHCL(block.DefRange),
	}
	for name, attr := range attrs {
		if !hclsyntax.ValidIdentifier(name) {
			diags = diags.Append(invalidNameDiagnostic(
				"Invalid local name for provider",
				attr.NameRange,
			))
			continue
		}
		if existing, exists := ret.Requirements[name]; exists {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Duplicate provider local name",
				Detail:   fmt.Sprintf("A provider requirement with local name %q was already declared at %s.", name, existing.DeclRange.StartString()),
				Subject:  attr.NameRange.Ptr(),
			})
			continue
		}
		declPairs, hclDiags := hcl.ExprMap(attr.Expr)
		diags = diags.Append(hclDiags)
		if hclDiags.HasErrors() {
			continue
		}
		declAttrs := make(map[string]*hcl.KeyValuePair, len(declPairs))
		for i := range declPairs {
			pair := &declPairs[i]
			name := hcl.ExprAsKeyword(pair.Key)
			if name == "" {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid provider requirement attribute",
					Detail:   "All of the attributes of a required_providers entry must be simple keywords.",
					Subject:  pair.Key.Range().Ptr(),
				})
				continue
			}
			if existing, exists := declAttrs[name]; exists {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Duplicate attribute",
					Detail:   fmt.Sprintf("The attribute %q was already defined at %s.", name, existing.Key.Range()),
					Subject:  pair.Key.Range().Ptr(),
				})
				continue
			}
			declAttrs[name] = pair
		}

		var sourceAddrStr, versionConstraintsStr string
		sourceAddrPair := declAttrs["source"]
		versionConstraintsPair := declAttrs["version"]
		delete(declAttrs, "source")
		delete(declAttrs, "version")

		if sourceAddrPair == nil {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Missing required attribute",
				Detail:   "All required_providers entries must include the attribute \"source\", giving the qualified provider source address to use.",
				Subject:  attr.Expr.StartRange().Ptr(),
			})
			continue
		}
		hclDiags = gohcl.DecodeExpression(sourceAddrPair.Value, nil, &sourceAddrStr)
		diags = diags.Append(hclDiags)
		if diags.HasErrors() {
			continue
		}
		providerAddr, moreDiags := addrs.ParseProviderSourceString(sourceAddrStr)
		// Ugh: ParseProviderSourceString returns sourceless diagnostics,
		// so we need to postprocess the diagnostics to add source locations
		// to them.
		for _, diag := range moreDiags {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: diag.Severity().ToHCL(),
				Summary:  diag.Description().Summary,
				Detail:   diag.Description().Detail,
				Subject:  sourceAddrPair.Value.Range().Ptr(),
			})
		}
		if moreDiags.HasErrors() {
			continue
		}

		var versionConstraints constraints.IntersectionSpec
		if !providerAddr.IsBuiltIn() {
			if versionConstraintsPair == nil {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Missing required attribute",
					Detail:   "Each required_providers entry for an installable provider must include the attribute \"version\", specifying the provider versions that this stack is compatible with.",
					Subject:  attr.Expr.StartRange().Ptr(),
				})
				continue
			}
			for name, pair := range declAttrs {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid provider requirement attribute",
					Detail:   fmt.Sprintf("An attribute named %q is not expected here.", name),
					Subject:  pair.Key.Range().Ptr(),
				})
				continue
			}
			hclDiags = gohcl.DecodeExpression(versionConstraintsPair.Value, nil, &versionConstraintsStr)
			diags = diags.Append(hclDiags)
			if diags.HasErrors() {
				continue
			}
			var err error
			versionConstraints, err = constraints.ParseRubyStyleMulti(versionConstraintsStr)
			if err != nil {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid version constraint",
					Detail:   fmt.Sprintf("Cannot use %q as a version constraint: %s.", versionConstraintsStr, err),
					Subject:  sourceAddrPair.Value.Range().Ptr(),
				})
				continue
			}
		} else {
			if versionConstraintsPair != nil {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Unsupported attribute",
					Detail:   fmt.Sprintf("The provider %q is built in to Terraform, so does not support version constraints.", providerAddr.ForDisplay()),
					Subject:  attr.Expr.StartRange().Ptr(),
				})
				continue
			}
		}

		if existingName, exists := reverseMap[providerAddr]; exists {
			existing := ret.Requirements[existingName]
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Duplicate provider local name",
				Detail: fmt.Sprintf(
					"A requirement for provider %s was already declared with local name %q at %s.",
					providerAddr, existingName, existing.DeclRange.StartString(),
				),
				Subject: attr.NameRange.Ptr(),
			})
			continue
		}

		ret.Requirements[name] = ProviderRequirement{
			LocalName:          name,
			Provider:           providerAddr,
			VersionConstraints: versionConstraints,
			DeclRange:          tfdiags.SourceRangeFromHCL(attr.NameRange),
		}
		reverseMap[providerAddr] = name
	}
	return ret, diags
}

func (pr *ProviderRequirements) ProviderForLocalName(localName string) (addrs.Provider, bool) {
	if pr == nil {
		return addrs.Provider{}, false
	}
	obj, ok := pr.Requirements[localName]
	if !ok {
		return addrs.Provider{}, false
	}
	return obj.Provider, true
}

func (pr *ProviderRequirements) LocalNameForProvider(providerAddr addrs.Provider) (string, bool) {
	if pr == nil {
		return "", false
	}
	for localName, obj := range pr.Requirements {
		if obj.Provider == providerAddr {
			return localName, true
		}
	}
	return "", false
}
