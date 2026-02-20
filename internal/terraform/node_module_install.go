package terraform

import (
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/getmodules/moduleaddrs"
	"github.com/hashicorp/terraform/internal/lang/langrefs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type nodeInstallModule struct {
	// We're using a ModuleInstance here,
	// because the downstream graph builder requires it.
	// But it was constructed with addrs.NoKey
	Addr       addrs.ModuleInstance
	ModuleCall *configs.ModuleCall
	Parent     *configs.Config
	Installer  configs.ModuleWalker

	// Stores the configuration of the installed module
	Config *configs.Config
	// Stores the version of the installed module
	Version *version.Version
}

var (
	_ GraphNodeExecutable        = (*nodeInstallModule)(nil)
	_ GraphNodeReferencer        = (*nodeInstallModule)(nil)
	_ GraphNodeDynamicExpandable = (*nodeInstallModule)(nil)
	_ GraphNodeModuleInstance    = (*nodeInstallModule)(nil)
)

func (n *nodeInstallModule) Path() addrs.ModuleInstance {
	return n.Addr.Parent()
}

func (n *nodeInstallModule) Name() string {
	return n.Addr.String()
}

func (n *nodeInstallModule) ModulePath() addrs.Module {
	return n.Addr.Module().Parent()
}

func (n *nodeInstallModule) References() []*addrs.Reference {
	var refs []*addrs.Reference

	sourceRefs, _ := langrefs.ReferencesInExpr(addrs.ParseRef, n.ModuleCall.SourceExpr)
	refs = append(refs, sourceRefs...)
	versionRefs, _ := langrefs.ReferencesInExpr(addrs.ParseRef, n.ModuleCall.VersionExpr)
	refs = append(refs, versionRefs...)

	// We need to resolve all module inputs as well, because some might be used
	// in the module as a constant variable to build a nested module source
	attrs, _ := n.ModuleCall.Config.JustAttributes()
	for _, attr := range attrs {
		inputRefs, _ := langrefs.ReferencesInExpr(addrs.ParseRef, attr.Expr)
		refs = append(refs, inputRefs...)
	}

	return refs
}

func (n *nodeInstallModule) Execute(ctx EvalContext, walkOp walkOperation) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	var version configs.VersionConstraint
	if n.ModuleCall.VersionExpr != nil {
		var versionDiags tfdiags.Diagnostics
		version, versionDiags = decodeVersionConstraint(n.ModuleCall.VersionExpr, ctx)
		diags = diags.Append(versionDiags)
		if diags.HasErrors() {
			return diags
		}
	}

	hasVersion := n.ModuleCall.VersionExpr != nil
	source, sourceDiags := decodeSource(n.ModuleCall.SourceExpr, hasVersion, ctx)
	diags = diags.Append(sourceDiags)
	if diags.HasErrors() {
		return diags
	}

	req := &configs.ModuleRequest{
		Name:              n.ModuleCall.Name,
		Path:              n.Addr.Module(),
		SourceAddr:        source,
		SourceAddrRange:   n.ModuleCall.SourceExpr.Range(),
		VersionConstraint: version,
		Parent:            n.Parent,
		CallRange:         n.ModuleCall.DeclRange,
	}

	cfg, v, modDiags := n.Installer.LoadModule(req)
	diags = diags.Append(modDiags)
	if diags.HasErrors() {
		return diags
	}

	config := &configs.Config{
		Module:          cfg,
		Parent:          n.Parent,
		Path:            n.Addr.Module(),
		Root:            n.Parent.Root,
		Children:        map[string]*configs.Config{},
		CallRange:       n.ModuleCall.DeclRange,
		SourceAddr:      source,
		SourceAddrRange: n.ModuleCall.SourceExpr.Range(),
		Version:         v,
	}

	// Insert the installed module into the children of the current module
	currentModuleKey := n.Addr[len(n.Addr)-1].Name
	n.Parent.Children[currentModuleKey] = config

	n.Config = config
	n.Version = v

	return nil
}

func (n *nodeInstallModule) DynamicExpand(ctx EvalContext) (*Graph, tfdiags.Diagnostics) {
	var g Graph
	var diags tfdiags.Diagnostics

	expander := ctx.InstanceExpander()
	_, call := n.Addr.Call()
	expander.SetModuleSingle(n.Path(), call)

	graph, graphDiags := (&InitGraphBuilder{
		Config: n.Config,
		Walker: n.Installer,
	}).Build(n.Addr)
	diags = diags.Append(graphDiags)
	if graphDiags.HasErrors() {
		return nil, diags
	}
	g.Subsume(&graph.AcyclicGraph.Graph)

	addRootNodeToGraph(&g)

	return &g, nil
}

func decodeSource(sourceExpr hcl.Expression, hasVersion bool, ctx EvalContext) (addrs.ModuleSource, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	var addr addrs.ModuleSource
	var err error

	refs, refsDiags := langrefs.ReferencesInExpr(addrs.ParseRef, sourceExpr)
	diags = diags.Append(refsDiags)
	if diags.HasErrors() {
		return nil, diags
	}

	for _, ref := range refs {
		switch ref.Subject.(type) {
		case addrs.InputVariable, addrs.LocalValue:
			// These are allowed
		default:
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid module source",
				Detail:   "The module source can only reference input variables and local values.",
				Subject:  ref.SourceRange.ToHCL().Ptr(),
			})
			return nil, diags
		}
	}

	value, valueDiags := ctx.EvaluateExpr(sourceExpr, cty.String, nil)
	diags = diags.Append(valueDiags)
	if diags.HasErrors() {
		return nil, diags
	}

	if !value.IsWhollyKnown() {
		tExpr, ok := sourceExpr.(*hclsyntax.TemplateExpr)
		if !ok {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid module source",
				Detail:   "The module source contains a reference that is unknown during init.",
				Subject:  sourceExpr.Range().Ptr(),
			})
			return nil, diags
		}
		for _, part := range tExpr.Parts {
			partVal, partDiags := ctx.EvaluateExpr(part, cty.DynamicPseudoType, nil)
			diags = diags.Append(partDiags)
			if diags.HasErrors() {
				return nil, diags
			}

			scope := ctx.EvaluationScope(nil, nil, EvalDataForNoInstanceKey)
			hclCtx, evalDiags := scope.EvalContext(refs)
			diags = diags.Append(evalDiags)
			if diags.HasErrors() {
				return nil, diags
			}
			if !partVal.IsKnown() {
				diags = diags.Append(&hcl.Diagnostic{
					Severity:    hcl.DiagError,
					Summary:     "Invalid module source",
					Detail:      "The value of a reference in the module source is unknown.",
					Subject:     part.Range().Ptr(),
					Expression:  part,
					EvalContext: hclCtx,
					Extra:       diagnosticCausedByUnknown(true),
				})
				return nil, diags
			}
		}
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid module source",
			Detail:   "The module source contains a reference that is unknown.",
			Subject:  sourceExpr.Range().Ptr(),
		})
		return nil, diags
	}

	if hasVersion {
		addr, err = moduleaddrs.ParseModuleSourceRegistry(value.AsString())
	} else {
		addr, err = moduleaddrs.ParseModuleSource(value.AsString())
	}
	if err != nil {
		diags = diags.Append(diags, err)
	}

	return addr, nil
}

func decodeVersionConstraint(versionExpr hcl.Expression, ctx EvalContext) (configs.VersionConstraint, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	rng := versionExpr.Range()

	ret := configs.VersionConstraint{
		DeclRange: rng,
	}

	refs, refsDiags := langrefs.ReferencesInExpr(addrs.ParseRef, versionExpr)
	diags = diags.Append(refsDiags)
	if diags.HasErrors() {
		return ret, diags
	}

	for _, ref := range refs {
		switch ref.Subject.(type) {
		case addrs.InputVariable, addrs.LocalValue:
			// These are allowed
		default:
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid module version",
				Detail:   "The module version can only reference input variables and local values.",
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

	if value.IsNull() {
		// A null version constraint is strange, but we'll just treat it
		// like an empty constraint set.
		return ret, diags
	}

	if !value.IsWhollyKnown() {
		tExpr, ok := versionExpr.(*hclsyntax.TemplateExpr)
		if !ok {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid module version",
				Detail:   "The module version contains a reference that is unknown during init.",
				Subject:  versionExpr.Range().Ptr(),
			})
			return ret, diags
		}
		for _, part := range tExpr.Parts {
			partVal, partDiags := ctx.EvaluateExpr(part, cty.DynamicPseudoType, nil)
			diags = diags.Append(partDiags)
			if diags.HasErrors() {
				return ret, diags
			}

			scope := ctx.EvaluationScope(nil, nil, EvalDataForNoInstanceKey)
			hclCtx, evalDiags := scope.EvalContext(refs)
			diags = diags.Append(evalDiags)
			if diags.HasErrors() {
				return ret, diags
			}
			if !partVal.IsKnown() {
				diags = diags.Append(&hcl.Diagnostic{
					Severity:    hcl.DiagError,
					Summary:     "Invalid module version",
					Detail:      "The value of a reference in the module version is unknown.",
					Subject:     part.Range().Ptr(),
					Expression:  part,
					EvalContext: hclCtx,
					Extra:       diagnosticCausedByUnknown(true),
				})
				return ret, diags
			}
		}
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid module version",
			Detail:   "The module version contains a reference that is unknown.",
			Subject:  versionExpr.Range().Ptr(),
		})
		return ret, diags
	}

	constraintStr := value.AsString()
	constraints, err := version.NewConstraint(constraintStr)
	if err != nil {
		// NewConstraint doesn't return user-friendly errors, so we'll just
		// ignore the provided error and produce our own generic one.
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid version constraint",
			Detail:   "This string does not use correct version constraint syntax.", // Not very actionable :(
			Subject:  rng.Ptr(),
		})
		return ret, diags
	}

	ret.Required = constraints
	return ret, diags
}
