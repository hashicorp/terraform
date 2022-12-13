package testconfigs

import (
	"fmt"
	"path/filepath"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type Step struct {
	Name string

	ModuleDir    string
	Providers    []*configs.PassedProviderConfig
	PlanMode     plans.Mode
	ApplyPlan    bool
	VariableDefs map[addrs.InputVariable]hcl.Expression

	ExpectFailure  addrs.Set[addrs.Checkable]
	Postconditions []*configs.CheckRule

	DeclRange hcl.Range
}

func decodeStepBlock(block *hcl.Block) (*Step, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	planOpts := *terraform.DefaultPlanOpts // shallow copy
	ret := &Step{
		Name:      block.Labels[0],
		DeclRange: block.DefRange,
	}
	planOpts.SetVariables = make(terraform.InputValues)
	ret.VariableDefs = make(map[addrs.InputVariable]hcl.Expression)

	if !hclsyntax.ValidIdentifier(ret.Name) {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid step name",
			Detail:   "A test scenario step name must be a valid identifier.",
			Subject:  block.LabelRanges[0].Ptr(),
		})
	}

	content, hclDiags := block.Body.Content(&stepBlockSchema)
	diags = diags.Append(hclDiags)

	if attr, ok := content.Attributes["module"]; ok {
		hclDiags = gohcl.DecodeExpression(attr.Expr, nil, &ret.ModuleDir)
		diags = diags.Append(hclDiags)
		if !hclDiags.HasErrors() {
			if ret.ModuleDir == "" {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid module directory",
					Detail:   "Must be a path to a local directory containing the Terraform module to test.",
					Subject:  attr.Expr.Range().Ptr(),
				})
			}
		}
		ret.ModuleDir = filepath.Clean(ret.ModuleDir)
	}

	ret.PlanMode = plans.NormalMode // Unless overridden by "plan_mode" argument
	if attr, ok := content.Attributes["plan_mode"]; ok {
		switch kw := hcl.ExprAsKeyword(attr.Expr); kw {
		case "normal":
			ret.PlanMode = plans.NormalMode
		case "destroy":
			ret.PlanMode = plans.DestroyMode
		case "refresh_only":
			ret.PlanMode = plans.RefreshOnlyMode
		default:
			if kw != "" {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid planning mode",
					Detail:   fmt.Sprintf("Cannot use %q as a planning mode.", kw),
					Subject:  attr.Expr.Range().Ptr(),
				})
			} else {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid planning mode",
					Detail:   "Must be one of the supported planning mode keywords.",
					Subject:  attr.Expr.Range().Ptr(),
				})
			}
		}
	}

	ret.ApplyPlan = true // Unless overridden by "apply" argument
	if attr, ok := content.Attributes["apply"]; ok {
		hclDiags = gohcl.DecodeExpression(attr.Expr, nil, &ret.ApplyPlan)
		diags = diags.Append(hclDiags)
	}

	if attr, ok := content.Attributes["variables"]; ok {
		pairs, hclDiags := hcl.ExprMap(attr.Expr)
		diags = diags.Append(hclDiags)
		for _, pair := range pairs {
			name := hcl.ExprAsKeyword(pair.Key)
			if name == "" {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid variable name",
					Detail:   "A variable name must be a valid identifier.",
					Subject:  pair.Key.Range().Ptr(),
				})
				continue
			}
			addr := addrs.InputVariable{Name: name}
			if existing, exists := ret.VariableDefs[addr]; exists {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Duplicate variable definition",
					Detail:   fmt.Sprintf("A value for variable %q was already defined at %s.", name, existing.Range()),
					Subject:  pair.Key.Range().Ptr(),
				})
				continue
			}
			ret.VariableDefs[addr] = pair.Value
		}
	}

	if attr, ok := content.Attributes["providers"]; ok {
		pairs, hclDiags := hcl.ExprMap(attr.Expr)
		diags = diags.Append(hclDiags)

		// Caller uses the nil-ness of ret.Providers to recognize when this
		// argument wasn't set to provide a default.
		ret.Providers = make([]*configs.PassedProviderConfig, 0, len(pairs))

		for _, pair := range pairs {
			inChild, hclDiags := configs.DecodeProviderConfigRef(pair.Key, "providers")
			diags = diags.Append(hclDiags)
			if hclDiags.HasErrors() {
				continue
			}
			inParent, hclDiags := configs.DecodeProviderConfigRef(pair.Value, "providers")
			diags = diags.Append(hclDiags)
			if hclDiags.HasErrors() {
				continue
			}
			ret.Providers = append(ret.Providers, &configs.PassedProviderConfig{
				InChild:  inChild,
				InParent: inParent,
			})
		}
	}

	if attr, ok := content.Attributes["expect_failure"]; ok {
		ret.ExpectFailure = addrs.MakeSet[addrs.Checkable]()
		exprs, hclDiags := hcl.ExprList(attr.Expr)
		diags = diags.Append(hclDiags)
		for _, expr := range exprs {
			traversal, hclDiags := hcl.AbsTraversalForExpr(expr)
			diags = diags.Append(hclDiags)
			if diags.HasErrors() {
				continue
			}
			// HACK: Because of historical ambiguity in our address syntaxes,
			// there isn't a single general ParseCheckable function that can
			// deal with addresses of any type. Instead, we need to specify
			// what kind of address we're trying to parse.
			//
			// As a temporary hack here we'll try all of the kinds in a fixed
			// preference order and just take the first one that succeeds.
			// The main thing about this ordering is that it tries resources
			// only after all of the other kinds and so it isn't possible to
			// refer to a resource which has type "output" or "smoke_test".
			var addr addrs.Checkable
			for _, kind := range expectFailureCheckableKinds {
				candidate, moreDiags := addrs.ParseCheckable(kind, traversal)
				if !moreDiags.HasErrors() {
					addr = candidate
					break
				}
			}
			if addr == nil {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid checkable object address",
					Detail:   "Must be an absolute address for a checkable object.",
					Subject:  expr.Range().Ptr(),
				})
				continue
			}
			if ret.ExpectFailure.Has(addr) {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Redundant expected failure",
					Detail:   fmt.Sprintf("The address %s appears twice in this set of expected failures.", addr),
					Subject:  expr.Range().Ptr(),
				})
			}
			ret.ExpectFailure.Add(addr)
		}
	}

	for _, block := range content.Blocks {
		switch block.Type {
		case "postcondition":
			rule, hclDiags := configs.DecodeCheckRuleBlock(block)
			diags = diags.Append(hclDiags)
			if !diags.HasErrors() {
				ret.Postconditions = append(ret.Postconditions, rule)
			}
		default:
			// Should never get here because the cases above should cover all
			// of the block types in stepBlockSchema.
			panic(fmt.Sprintf("unhandled block type %q", block.Type))
		}
	}

	return ret, diags
}

var stepBlockSchema = hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{Name: "module"},
		{Name: "plan_mode"},
		{Name: "apply"},
		{Name: "variables"},
		{Name: "providers"},
		{Name: "expect_failure"},
	},
	Blocks: []hcl.BlockHeaderSchema{
		{Type: "postcondition"},
	},
}

var expectFailureCheckableKinds = []addrs.CheckableKind{
	addrs.CheckableSmokeTest,
	addrs.CheckableOutputValue,
	addrs.CheckableResource,
}
