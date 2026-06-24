// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"log"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/lang/langrefs"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

type nodeResolveProviderRequirements struct {
	Addr   addrs.ModuleInstance
	Module *configs.Module
}

var (
	_ GraphNodeExecutable     = (*nodeResolveProviderRequirements)(nil)
	_ GraphNodeReferencer     = (*nodeResolveProviderRequirements)(nil)
	_ GraphNodeModuleInstance = (*nodeResolveProviderRequirements)(nil)
	_ dag.NamedVertex         = (*nodeResolveProviderRequirements)(nil)
)

func (n *nodeResolveProviderRequirements) Path() addrs.ModuleInstance {
	return n.Addr
}

func (n *nodeResolveProviderRequirements) Name() string {
	return n.Addr.String()
}

func (n *nodeResolveProviderRequirements) ModulePath() addrs.Module {
	return n.Addr.Module()
}

func (n *nodeResolveProviderRequirements) References() []*addrs.Reference {
	var refs []*addrs.Reference

	for _, req := range n.Module.ProviderRequirements.RequiredProviders {
		if req.SourceExpr != nil {
			sourceRefs, _ := langrefs.ReferencesInExpr(
				addrs.ParseRef, req.SourceExpr,
			)
			refs = append(refs, sourceRefs...)
		}

		if req.RequirementExpression != nil {
			versionRefs, _ := langrefs.ReferencesInExpr(
				addrs.ParseRef, req.RequirementExpression,
			)
			refs = append(refs, versionRefs...)
		}
	}

	return refs
}

func (n *nodeResolveProviderRequirements) Execute(
	ctx EvalContext,
	_ walkOperation,
) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	log.Printf("[TRACE] nodeResolveProviderRequirements: In %s with %#v", n.Path(), n.Module.ProviderRequirements.RequiredProviders)
	for _, req := range n.Module.ProviderRequirements.RequiredProviders {
		if req.SourceExpr != nil {
			sourceStr, sourceType, sourceDiags :=
				evalProviderSource(req.SourceExpr, ctx)
			diags = diags.Append(sourceDiags)
			if sourceDiags.HasErrors() {
				return diags
			}
			req.SourceR = sourceStr
			req.Type = sourceType
		}

		if req.RequirementExpression != nil {
			vc, vcDiags := evalProviderVersion(req.RequirementExpression, ctx)
			diags = diags.Append(vcDiags)
			if vcDiags.HasErrors() {
				return diags
			}
			req.RequirementR = vc
		}
	}

	return diags
}

func evalProviderSource(
	sourceExpr hcl.Expression,
	ctx EvalContext,
) (string, addrs.Provider, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	refs, refsDiags := langrefs.ReferencesInExpr(addrs.ParseRef, sourceExpr)
	diags = diags.Append(refsDiags)
	if diags.HasErrors() {
		return "", addrs.Provider{}, diags
	}

	for _, ref := range refs {
		// Limit references to vars, locals
		switch ref.Subject.(type) {
		case addrs.InputVariable, addrs.LocalValue:
		// Allowed
		default:
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Unknown provider source",
				Detail:   "Only literal values and const variables can be evaluated during init.",
				Subject:  ref.SourceRange.ToHCL().Ptr(),
			})
			return "", addrs.Provider{}, diags
		}
	}

	value, valueDiags := ctx.EvaluateExpr(sourceExpr, cty.String, nil)
	diags = diags.Append(valueDiags)
	if diags.HasErrors() {
		return "", addrs.Provider{}, diags
	}

	if !value.IsWhollyKnown() {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Unknown provider source",
			Detail:   "Only literal values and const variables can be evaluated during init.",
			Subject:  sourceExpr.Range().Ptr(),
		})
		return "", addrs.Provider{}, diags
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
		return "", addrs.Provider{}, diags
	}

	return sourceStr, fqn, diags
}

func evalProviderVersion(
	versionExpr hcl.Expression,
	ctx EvalContext,
) (configs.VersionConstraint, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	ret := configs.VersionConstraint{
		DeclRange: versionExpr.Range(),
	}

	refs, refsDiags := langrefs.ReferencesInExpr(addrs.ParseRef, versionExpr)
	diags = diags.Append(refsDiags)
	if diags.HasErrors() {
		return ret, diags
	}

	for _, ref := range refs {
		switch ref.Subject.(type) {
		case addrs.InputVariable, addrs.LocalValue:
		// Allowed
		default:
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Unknown provider version",
				Detail:   "Only literal values and const variables can be evaluated during init.",
				Subject:  ref.SourceRange.ToHCL().Ptr(),
			})
			return ret, diags
		}
	}

	value, valueDiags := ctx.EvaluateExpr(versionExpr, cty.String, nil)
	diags = diags.Append(valueDiags)
	if diags.HasErrors() {
		return ret, diags
	}

	if !value.IsWhollyKnown() {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Unknown provider version",
			Detail:   "Only literal values and const variables can be evaluated during init.",
			Subject:  versionExpr.Range().Ptr(),
		})
		return ret, diags
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
		return ret, diags
	}

	ret.Required = constraints
	return ret, diags
}
