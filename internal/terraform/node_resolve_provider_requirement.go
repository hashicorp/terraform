// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/lang/langrefs"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

type nodeResolveProviderRequirements struct {
	Addr   addrs.ModuleInstance
	Module *configs.Module
	Exprs  map[string]*configs.ProviderRequirementExpr
}

var (
	_ GraphNodeExecutable     = (*nodeResolveProviderRequirements)(nil)
	_ GraphNodeReferencer     = (*nodeResolveProviderRequirements)(nil)
	_ GraphNodeModuleInstance = (*nodeResolveProviderRequirements)(nil)
)

func (n *nodeResolveProviderRequirements) Execute(
	ctx EvalContext,
	_ walkOperation,
) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	for name, expr := range n.Exprs {
		rp, rpDiags := n.resolveProvider(name, expr, ctx)
		diags = append(diags, rpDiags...)
		if rpDiags.HasErrors() {
			continue
		}

		n.Module.ProviderRequirements.RequiredProviders[name] = rp
	}

	n.Module.GatherProviderLocalNames()

	return diags
}

func (n *nodeResolveProviderRequirements) resolveProvider(
	name string,
	expr *configs.ProviderRequirementExpr,
	ctx EvalContext,
) (*configs.RequiredProvider, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	rp := &configs.RequiredProvider{
		Name:      name,
		Aliases:   expr.ConfigAliases,
		DeclRange: expr.DeclRange,
	}

	if expr.SourceExpr != nil {
		sourceStr, sourceType, sourceDiags :=
			evalProviderSource(expr.SourceExpr, ctx)
		diags = diags.Append(sourceDiags)
		if sourceDiags.HasErrors() {
			return nil, diags
		}
		rp.Source = sourceStr
		rp.Type = sourceType
	} else { // Regular string parsing (no vars)
		pType, err := addrs.ParseProviderPart(name)
		if err != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Invalid provider name",
				err.Error(),
			))
			return nil, diags
		}
		rp.Type = addrs.ImpliedProviderForUnqualifiedType(pType)
	}

	if expr.VersionExpr != nil {
		vc, vcDiags := evalProviderVersion(expr.VersionExpr, ctx)
		diags = diags.Append(vcDiags)
		if vcDiags.HasErrors() {
			return nil, diags
		}
		rp.Requirement = vc
	}

	return rp, diags
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
				Summary:  "Invalid provider source",
				Detail: "The provider source can only reference constant " +
					"input variables and local values." + constVariableDetail,
				Subject: ref.SourceRange.ToHCL().Ptr(),
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
			Summary:  "Invalid provider source",
			Detail: "The provider source contains a reference that is " +
				"unknown during init.",
			Subject: sourceExpr.Range().Ptr(),
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
				Summary:  "Invalid provider version",
				Detail: "The provider version can only reference constant " +
					"input variables and local values." + constVariableDetail,
				Subject: ref.SourceRange.ToHCL().Ptr(),
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
			Summary:  "Invalid provider version",
			Detail: "The provider version contains a reference that is " +
				"unknown during init.",
			Subject: versionExpr.Range().Ptr(),
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

func (n *nodeResolveProviderRequirements) References() []*addrs.Reference {
	var refs []*addrs.Reference
	for _, expr := range n.Exprs {
		if expr.SourceExpr != nil {
			sourceRefs, _ := langrefs.ReferencesInExpr(
				addrs.ParseRef, expr.SourceExpr,
			)
			refs = append(refs, sourceRefs...)
		}

		if expr.VersionExpr != nil {
			versionRefs, _ := langrefs.ReferencesInExpr(
				addrs.ParseRef, expr.VersionExpr,
			)
			refs = append(refs, versionRefs...)
		}
	}

	return refs
}

func (n *nodeResolveProviderRequirements) Path() addrs.ModuleInstance {
	return n.Addr
}

func (n *nodeResolveProviderRequirements) ModulePath() addrs.Module {
	return n.Addr.Module()
}
