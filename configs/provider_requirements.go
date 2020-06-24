package configs

import (
	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/addrs"
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
}

type RequiredProviders struct {
	RequiredProviders map[string]*RequiredProvider
	DeclRange         hcl.Range
}

func decodeRequiredProvidersBlock(block *hcl.Block) (*RequiredProviders, hcl.Diagnostics) {
	attrs, diags := block.Body.JustAttributes()
	ret := &RequiredProviders{
		RequiredProviders: make(map[string]*RequiredProvider),
		DeclRange:         block.DefRange,
	}
	for name, attr := range attrs {
		expr, err := attr.Expr.Value(nil)
		if err != nil {
			diags = append(diags, err...)
		}

		// verify that the local name is already localized or produce an error.
		nameDiags := checkProviderNameNormalized(name, attr.Expr.Range())
		diags = append(diags, nameDiags...)

		rp := &RequiredProvider{
			Name:      name,
			DeclRange: attr.Expr.Range(),
		}

		switch {
		case expr.Type().IsPrimitiveType():
			vc, reqDiags := decodeVersionConstraint(attr)
			diags = append(diags, reqDiags...)
			rp.Requirement = vc

		case expr.Type().IsObjectType():
			if expr.Type().HasAttribute("version") {
				vc := VersionConstraint{
					DeclRange: attr.Range,
				}
				constraint := expr.GetAttr("version")
				if !constraint.Type().Equals(cty.String) || constraint.IsNull() {
					diags = append(diags, &hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Invalid version constraint",
						Detail:   "Version must be specified as a string.",
						Subject:  attr.Expr.Range().Ptr(),
					})
				} else {
					constraintStr := constraint.AsString()
					constraints, err := version.NewConstraint(constraintStr)
					if err != nil {
						// NewConstraint doesn't return user-friendly errors, so we'll just
						// ignore the provided error and produce our own generic one.
						diags = append(diags, &hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  "Invalid version constraint",
							Detail:   "This string does not use correct version constraint syntax.",
							Subject:  attr.Expr.Range().Ptr(),
						})
					} else {
						vc.Required = constraints
						rp.Requirement = vc
					}
				}
			}
			if expr.Type().HasAttribute("source") {
				source := expr.GetAttr("source")
				if !source.Type().Equals(cty.String) || source.IsNull() {
					diags = append(diags, &hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Invalid source",
						Detail:   "Source must be specified as a string.",
						Subject:  attr.Expr.Range().Ptr(),
					})
				} else {
					rp.Source = source.AsString()

					fqn, sourceDiags := addrs.ParseProviderSourceString(rp.Source)

					if sourceDiags.HasErrors() {
						hclDiags := sourceDiags.ToHCL()
						// The diagnostics from ParseProviderSourceString don't contain
						// source location information because it has no context to compute
						// them from, and so we'll add those in quickly here before we
						// return.
						for _, diag := range hclDiags {
							if diag.Subject == nil {
								diag.Subject = attr.Expr.Range().Ptr()
							}
						}
						diags = append(diags, hclDiags...)
					} else {
						rp.Type = fqn
					}
				}
			}

		default:
			// should not happen
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid required_providers syntax",
				Detail:   "required_providers entries must be strings or objects.",
				Subject:  attr.Expr.Range().Ptr(),
			})
		}

		if rp.Type.IsZero() && !diags.HasErrors() { // Don't try to generate an FQN if we've encountered errors
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

	return ret, diags
}
