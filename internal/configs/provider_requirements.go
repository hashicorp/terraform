// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package configs

import (
	"fmt"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/zclconf/go-cty/cty"
)

// RequiredProvider represents a declaration of a dependency on a particular
// provider version or source without actually configuring that provider. This
// is used in child modules that expect a provider to be passed in from their
// parent.
type RequiredProvider struct {
	Name        string
	Source      string
	Type        addrs.Provider
	Requirement VersionConstraint
	DeclRange   hcl.Range
	Aliases     []addrs.LocalProviderConfig
}

type RequiredProviders struct {
	RequiredProviders map[string]*RequiredProvider
	DeclRange         hcl.Range
}

func decodeRequiredProvidersBlock(block *hcl.Block) (
	*RequiredProviders,
	map[string]*ProviderRequirementExpr,
	hcl.Diagnostics,
) {
	attrs, diags := block.Body.JustAttributes()
	if diags.HasErrors() {
		return nil, nil, diags
	}

	ret := &RequiredProviders{
		RequiredProviders: make(map[string]*RequiredProvider),
		DeclRange:         block.DefRange,
	}
	var deferredExprs map[string]*ProviderRequirementExpr

	for name, attr := range attrs {
		rp := &RequiredProvider{
			Name:      name,
			DeclRange: attr.Expr.Range(),
		}

		// Look for a single static string, in case we have the legacy version-only
		// format in the configuration.
		if expr, err := attr.Expr.Value(nil); err == nil && expr.Type().IsPrimitiveType() {
			vc, reqDiags := decodeVersionConstraint(attr)
			diags = append(diags, reqDiags...)

			pType, err := addrs.ParseProviderPart(rp.Name)
			if err != nil {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid provider name",
					Detail:   err.Error(),
					Subject:  attr.Expr.Range().Ptr(),
				})
				continue
			}

			rp.Requirement = vc
			rp.Type = addrs.ImpliedProviderForUnqualifiedType(pType)
			ret.RequiredProviders[name] = rp

			continue
		}

		// verify that the local name is already localized or produce an error.
		nameDiags := checkProviderNameNormalized(name, attr.Expr.Range())
		if nameDiags.HasErrors() {
			diags = append(diags, nameDiags...)
			continue
		}

		kvs, mapDiags := hcl.ExprMap(attr.Expr)
		if mapDiags.HasErrors() {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid required_providers object",
				Detail:   "required_providers entries must be strings or objects.",
				Subject:  attr.Expr.Range().Ptr(),
			})
			continue
		}

		providerExpr := &ProviderRequirementExpr{
			Name:        name,
			SourceExpr:  nil,
			VersionExpr: nil,
			DeclRange:   attr.Expr.Range(),
		}
		var sourceExpr, versionExpr hcl.Expression

	LOOP:
		for _, kv := range kvs {
			key, keyDiags := kv.Key.Value(nil)
			if keyDiags.HasErrors() {
				diags = append(diags, keyDiags...)
				continue
			}

			if key.Type() != cty.String {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid Attribute",
					Detail:   fmt.Sprintf("Invalid attribute value for provider requirement: %#v", key),
					Subject:  kv.Key.Range().Ptr(),
				})
				continue
			}

			switch key.AsString() {
			case "version":
				versionExpr = kv.Value

				// Store the version expression if it contains variable that
				// needs to be evaluated.
				// Skip the "legacy" pure string resolution of the version
				// attribute.
				if vars := kv.Value.Variables(); len(vars) > 0 {
					providerExpr.VersionExpr = kv.Value
					continue
				}

				vc := VersionConstraint{
					DeclRange: attr.Range,
				}

				constraint, valDiags := kv.Value.Value(nil)
				if valDiags.HasErrors() || !constraint.Type().Equals(cty.String) {
					diags = append(diags, &hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Invalid version constraint",
						Detail:   "Version must be specified as a string.",
						Subject:  kv.Value.Range().Ptr(),
					})
					continue
				}

				constraintStr := constraint.AsString()
				constraints, err := version.NewConstraint(constraintStr)
				if err != nil {
					// NewConstraint doesn't return user-friendly errors, so we'll just
					// ignore the provided error and produce our own generic one.
					diags = append(diags, &hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Invalid version constraint",
						Detail:   "This string does not use correct version constraint syntax.",
						Subject:  kv.Value.Range().Ptr(),
					})
					continue
				}

				vc.Required = constraints
				rp.Requirement = vc

			case "source":
				sourceExpr = kv.Value

				// Store the source expression if it contains variable that
				// needs to be evaluated.
				//
				// Skip the "legacy" pure string resolution of the source
				// attribute.
				if vars := kv.Value.Variables(); len(vars) > 0 {
					providerExpr.SourceExpr = kv.Value
					continue
				}

				source, err := kv.Value.Value(nil)
				if err != nil || !source.Type().Equals(cty.String) {
					diags = append(diags, &hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Invalid source",
						Detail:   "Source must be specified as a string.",
						Subject:  kv.Value.Range().Ptr(),
					})
					continue
				}

				fqn, sourceDiags := addrs.ParseProviderSourceString(source.AsString())
				if sourceDiags.HasErrors() {
					hclDiags := sourceDiags.ToHCL()
					// The diagnostics from ParseProviderSourceString don't contain
					// source location information because it has no context to compute
					// them from, and so we'll add those in quickly here before we
					// return.
					for _, diag := range hclDiags {
						if diag.Subject == nil {
							diag.Subject = kv.Value.Range().Ptr()
						}
					}
					diags = append(diags, hclDiags...)
					continue
				}

				rp.Source = source.AsString()
				rp.Type = fqn

			case "configuration_aliases":
				exprs, listDiags := hcl.ExprList(kv.Value)
				if listDiags.HasErrors() {
					diags = append(diags, listDiags...)
					continue
				}

				for _, expr := range exprs {
					traversal, travDiags := hcl.AbsTraversalForExpr(expr)
					if travDiags.HasErrors() {
						diags = append(diags, travDiags...)
						continue
					}

					addr, cfgDiags := ParseProviderConfigCompact(traversal)
					if cfgDiags.HasErrors() {
						diags = append(diags, &hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  "Invalid configuration_aliases value",
							Detail:   `Configuration aliases can only contain references to local provider configuration names in the format of provider.alias`,
							Subject:  kv.Value.Range().Ptr(),
						})
						continue
					}

					if addr.LocalName != name {
						diags = append(diags, &hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  "Invalid configuration_aliases value",
							Detail:   fmt.Sprintf(`Configuration aliases must be prefixed with the provider name. Expected %q, but found %q.`, name, addr.LocalName),
							Subject:  kv.Value.Range().Ptr(),
						})
						continue
					}

					rp.Aliases = append(rp.Aliases, addr)
				}

			default:
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid required_providers object",
					Detail:   `required_providers objects can only contain "version", "source" and "configuration_aliases" attributes. To configure a provider, use a "provider" block.`,
					Subject:  kv.Key.Range().Ptr(),
				})
				break LOOP
			}

		}

		if diags.HasErrors() {
			continue
		}

		// Provider Expression contains either source or version expression.
		// Hydrate the rest, store it into the result map and skip adding it to
		// required providers.
		if !providerExpr.IsEmpty() {
			providerExpr.ConfigAliases = rp.Aliases

			if providerExpr.SourceExpr == nil {
				providerExpr.SourceExpr = sourceExpr
			}

			if providerExpr.VersionExpr == nil {
				providerExpr.VersionExpr = versionExpr
			}

			if deferredExprs == nil {
				deferredExprs = map[string]*ProviderRequirementExpr{}
			}
			deferredExprs[name] = providerExpr

			// Skip adding it to required providers.
			continue
		}

		// We can add the required provider when there are no errors.
		// If a source was not given, create an implied type.
		if rp.Type.IsZero() {
			pType, err := addrs.ParseProviderPart(rp.Name)
			if err != nil {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid provider name",
					Detail:   err.Error(),
					Subject:  attr.Expr.Range().Ptr(),
				})
			} else {
				rp.Type = addrs.ImpliedProviderForUnqualifiedType(pType)
			}
		}

		ret.RequiredProviders[rp.Name] = rp
	}

	return ret, deferredExprs, diags
}
