// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package configs

import (
	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/lang/langrefs"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

// ProviderRequirementExprEvaluator evaluates expressions used in
// required_providers entries.
type ProviderRequirementExprEvaluator interface {
	EvaluateStringExpr(hcl.Expression) (cty.Value, tfdiags.Diagnostics)
}

// HCLEvalExprEvaluator evaluates expressions using a standard HCL eval context.
//
// Nil Ctx is valid and means no variables/functions are available.
type HCLEvalExprEvaluator struct {
	Ctx *hcl.EvalContext
}

func (e HCLEvalExprEvaluator) EvaluateStringExpr(expr hcl.Expression) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	val, hclDiags := expr.Value(e.Ctx)
	diags = diags.Append(hclDiags)
	if hclDiags.HasErrors() {
		return cty.DynamicVal, diags
	}
	return val, diags
}

// ProviderRequirementResolveOpts controls how expression resolution behaves.
type ProviderRequirementResolveOpts struct {
	// DeferUnresolved causes unknown/unavailable expression values to be treated
	// as unresolved (and therefore deferred) rather than as errors.
	DeferUnresolved bool
}

// ResolveProviderRequirement resolves one required_providers expression entry.
//
// If opts.DeferUnresolved is true and an expression cannot be resolved yet,
// resolved will be false and diagnostics will be empty.
func ResolveProviderRequirement(
	name string,
	expr *ProviderRequirementExpr,
	eval ProviderRequirementExprEvaluator,
	opts ProviderRequirementResolveOpts,
) (rp *RequiredProvider, resolved bool, diags tfdiags.Diagnostics) {
	rp = &RequiredProvider{
		Name:      name,
		Aliases:   expr.ConfigAliases,
		DeclRange: expr.DeclRange,
	}

	sourceResolved := expr.SourceExpr == nil
	if expr.SourceExpr != nil {
		sourceStr, sourceType, ok, sourceDiags := resolveProviderSource(expr.SourceExpr, eval, opts)
		diags = diags.Append(sourceDiags)
		if sourceDiags.HasErrors() {
			return nil, false, diags
		}
		if ok {
			rp.Source = sourceStr
			rp.Type = sourceType
			sourceResolved = true
		}
	}

	versionResolved := expr.VersionExpr == nil
	if expr.VersionExpr != nil {
		vc, ok, vcDiags := resolveProviderVersion(expr.VersionExpr, eval, opts)
		diags = diags.Append(vcDiags)
		if vcDiags.HasErrors() {
			return nil, false, diags
		}
		if ok {
			rp.Requirement = vc
			versionResolved = true
		}
	}

	if !sourceResolved || !versionResolved {
		return nil, false, diags
	}

	if rp.Type.IsZero() {
		pType, err := addrs.ParseProviderPart(name)
		if err != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Invalid provider name",
				err.Error(),
			))
			return nil, false, diags
		}
		rp.Type = addrs.ImpliedProviderForUnqualifiedType(pType)
	}

	return rp, true, diags
}

func resolveProviderSource(
	sourceExpr hcl.Expression,
	eval ProviderRequirementExprEvaluator,
	opts ProviderRequirementResolveOpts,
) (string, addrs.Provider, bool, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	refs, refsDiags := langrefs.ReferencesInExpr(addrs.ParseRef, sourceExpr)
	diags = diags.Append(refsDiags)
	if diags.HasErrors() {
		return "", addrs.Provider{}, false, diags
	}

	for _, ref := range refs {
		switch ref.Subject.(type) {
		case addrs.InputVariable, addrs.LocalValue:
			// Allowed references
		default:
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Unknown provider source",
				Detail:   "Only literal values and const variables can be evaluated during init.",
				Subject:  ref.SourceRange.ToHCL().Ptr(),
			})
			return "", addrs.Provider{}, false, diags
		}
	}

	value, valueDiags := eval.EvaluateStringExpr(sourceExpr)
	if valueDiags.HasErrors() {
		if opts.DeferUnresolved {
			return "", addrs.Provider{}, false, nil
		}
		return "", addrs.Provider{}, false, valueDiags
	}

	if !value.IsWhollyKnown() {
		if opts.DeferUnresolved {
			return "", addrs.Provider{}, false, nil
		}
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Unknown provider source",
			Detail:   "Only literal values and const variables can be evaluated during init.",
			Subject:  sourceExpr.Range().Ptr(),
		})
		return "", addrs.Provider{}, false, diags
	}

	if value.Type() != cty.String {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid source",
			Detail:   "Provider source must be a string in the format namespace/type.",
			Subject:  sourceExpr.Range().Ptr(),
		})
		return "", addrs.Provider{}, false, diags
	}

	sourceStr := value.AsString()
	fqn, sourceDiags := addrs.ParseProviderSourceString(sourceStr)
	if sourceDiags.HasErrors() {
		hclDiags := sourceDiags.ToHCL()
		for _, d := range hclDiags {
			if d.Subject == nil {
				d.Subject = sourceExpr.Range().Ptr()
			}
		}
		diags = diags.Append(hclDiags)
		return "", addrs.Provider{}, false, diags
	}

	return sourceStr, fqn, true, diags
}

func resolveProviderVersion(
	versionExpr hcl.Expression,
	eval ProviderRequirementExprEvaluator,
	opts ProviderRequirementResolveOpts,
) (VersionConstraint, bool, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	ret := VersionConstraint{
		DeclRange: versionExpr.Range(),
	}

	refs, refsDiags := langrefs.ReferencesInExpr(addrs.ParseRef, versionExpr)
	diags = diags.Append(refsDiags)
	if diags.HasErrors() {
		return ret, false, diags
	}

	for _, ref := range refs {
		switch ref.Subject.(type) {
		case addrs.InputVariable, addrs.LocalValue:
			// Allowed references
		default:
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Unknown provider version",
				Detail:   "Only literal values and const variables can be evaluated during init.",
				Subject:  ref.SourceRange.ToHCL().Ptr(),
			})
			return ret, false, diags
		}
	}

	value, valueDiags := eval.EvaluateStringExpr(versionExpr)
	if valueDiags.HasErrors() {
		if opts.DeferUnresolved {
			return ret, false, nil
		}
		return ret, false, valueDiags
	}

	if !value.IsWhollyKnown() {
		if opts.DeferUnresolved {
			return ret, false, nil
		}
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Unknown provider version",
			Detail:   "Only literal values and const variables can be evaluated during init.",
			Subject:  versionExpr.Range().Ptr(),
		})
		return ret, false, diags
	}

	if value.Type() != cty.String {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid version constraint",
			Detail:   "Version must be specified as a string.",
			Subject:  versionExpr.Range().Ptr(),
		})
		return ret, false, diags
	}

	constraintStr := value.AsString()
	constraints, err := version.NewConstraint(constraintStr)
	if err != nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid version constraint",
			Detail:   "This string is not a valid version constraint.",
			Subject:  versionExpr.Range().Ptr(),
		})
		return ret, false, diags
	}

	ret.Required = constraints
	return ret, true, diags
}
