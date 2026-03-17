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
//
// The SourceExpr and VersionExpr fields store the raw HCL expressions for
// both static and dynamic (variable-interpolated) cases. The resolved output
// fields (Source, Type, Requirement) are populated by calling
// ResolveProviderSources on the parent RequiredProviders.
type RequiredProvider struct {
	Name      string
	DeclRange hcl.Range
	Aliases   []addrs.LocalProviderConfig

	// Input: raw HCL expressions, always stored regardless of whether
	// they are static literals or contain variable references.
	SourceExpr  hcl.Expression // HCL expression for source (may be nil)
	VersionExpr hcl.Expression // HCL expression for version (may be nil)

	// Output: populated after resolution via ResolveProviderSources().
	Source      string            // Resolved source string, e.g. "hashicorp/aws"
	Type        addrs.Provider    // Parsed fully-qualified provider address
	Requirement VersionConstraint // Parsed version constraints
	Resolved    bool              // True once expressions have been evaluated
}

type RequiredProviders struct {
	RequiredProviders map[string]*RequiredProvider
	DeclRange         hcl.Range
}

// ResolveProviderSources evaluates all unresolved provider source and version
// expressions against the given EvalContext. For static (literal) expressions,
// pass nil as ctx. For dynamic expressions containing variable references,
// pass an EvalContext populated with const variable values.
//
// This is the single evaluation path for both static and dynamic cases.
func (rps *RequiredProviders) ResolveProviderSources(ctx *hcl.EvalContext) hcl.Diagnostics {
	var diags hcl.Diagnostics
	for _, rp := range rps.RequiredProviders {
		if rp.Resolved {
			continue
		}
		diags = append(diags, rp.resolve(ctx)...)
	}
	return diags
}

// resolve evaluates the source and version expressions for a single
// RequiredProvider. If no source was given, an implied type is derived
// from the provider's local name.
func (rp *RequiredProvider) resolve(ctx *hcl.EvalContext) hcl.Diagnostics {
	var diags hcl.Diagnostics

	// Resolve source expression
	if rp.SourceExpr != nil {
		val, valDiags := rp.SourceExpr.Value(ctx)
		diags = append(diags, valDiags...)
		if valDiags.HasErrors() {
			return diags
		}
		if val.Type() != cty.String {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid source",
				Detail:   "Source must be specified as a string.",
				Subject:  rp.SourceExpr.Range().Ptr(),
			})
			return diags
		}
		sourceStr := val.AsString()

		fqn, sourceDiags := addrs.ParseProviderSourceString(sourceStr)
		if sourceDiags.HasErrors() {
			hclDiags := sourceDiags.ToHCL()
			for _, diag := range hclDiags {
				if diag.Subject == nil {
					diag.Subject = rp.SourceExpr.Range().Ptr()
				}
			}
			diags = append(diags, hclDiags...)
			return diags
		}

		rp.Source = sourceStr
		rp.Type = fqn
	}

	// Resolve version expression
	if rp.VersionExpr != nil {
		val, valDiags := rp.VersionExpr.Value(ctx)
		diags = append(diags, valDiags...)
		if valDiags.HasErrors() {
			return diags
		}
		if val.Type() != cty.String {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid version constraint",
				Detail:   "Version must be specified as a string.",
				Subject:  rp.VersionExpr.Range().Ptr(),
			})
			return diags
		}

		constraintStr := val.AsString()
		constraints, err := version.NewConstraint(constraintStr)
		if err != nil {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid version constraint",
				Detail:   "This string does not use correct version constraint syntax.",
				Subject:  rp.VersionExpr.Range().Ptr(),
			})
			return diags
		}
		rp.Requirement = VersionConstraint{
			Required:  constraints,
			DeclRange: rp.VersionExpr.Range(),
		}
	}

	// If no source was given, derive an implied type from the local name.
	if rp.Type.IsZero() {
		pType, err := addrs.ParseProviderPart(rp.Name)
		if err != nil {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid provider name",
				Detail:   err.Error(),
				Subject:  rp.DeclRange.Ptr(),
			})
			return diags
		}
		rp.Type = addrs.ImpliedProviderForUnqualifiedType(pType)
	}

	rp.Resolved = true
	return diags
}

func decodeRequiredProvidersBlock(block *hcl.Block) (*RequiredProviders, hcl.Diagnostics) {
	attrs, diags := block.Body.JustAttributes()
	if diags.HasErrors() {
		return nil, diags
	}

	ret := &RequiredProviders{
		RequiredProviders: make(map[string]*RequiredProvider),
		DeclRange:         block.DefRange,
	}

	for name, attr := range attrs {
		rp := &RequiredProvider{
			Name:      name,
			DeclRange: attr.Expr.Range(),
		}

		// Look for a single static string, in case we have the legacy
		// version-only format in the configuration.
		if expr, err := attr.Expr.Value(nil); err == nil && expr.Type().IsPrimitiveType() {
			pType, pErr := addrs.ParseProviderPart(rp.Name)
			if pErr != nil {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid provider name",
					Detail:   pErr.Error(),
					Subject:  attr.Expr.Range().Ptr(),
				})
				continue
			}

			// Store version expression for deferred resolution, but
			// resolve the implied type immediately since there's no
			// source expression in the legacy format.
			rp.VersionExpr = attr.Expr
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
				rp.VersionExpr = kv.Value

			case "source":
				rp.SourceExpr = kv.Value

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

		ret.RequiredProviders[rp.Name] = rp
	}

	// Resolve all static providers immediately (nil context).
	// Dynamic providers with variable references will fail here and
	// remain unresolved until ResolveProviderSources is called with
	// a proper EvalContext containing const variable values.
	resolveDiags := ret.ResolveProviderSources(nil)
	diags = append(diags, resolveDiags...)

	// Remove any providers that failed static resolution.
	// Providers that remain unresolved without errors are dynamic
	// and will be resolved later with a const variable context.
	if resolveDiags.HasErrors() {
		for name, rp := range ret.RequiredProviders {
			if !rp.Resolved {
				delete(ret.RequiredProviders, name)
			}
		}
	}

	return ret, diags
}
