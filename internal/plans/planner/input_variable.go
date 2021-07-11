package planner

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
)

type inputVariable struct {
	planner *planner
	addr    addrs.AbsInputVariableInstance
}

func (v inputVariable) Addr() addrs.AbsInputVariableInstance {
	return v.addr
}

func (v inputVariable) IsDeclared() bool {
	return v.Config() != nil
}

func (v inputVariable) Config() *configs.Variable {
	moduleConfig := v.planner.Config().DescendentForInstance(v.addr.Module)
	if moduleConfig == nil {
		return nil
	}

	return moduleConfig.Module.Variables[v.addr.Variable.Name]
}

func (v inputVariable) ModuleInstance() moduleInstance {
	return v.planner.ModuleInstance(v.addr.Module)
}

func (v inputVariable) Value() cty.Value {
	config := v.Config()
	if config == nil {
		// Reference to an undeclared variable should be caught during
		// the validation step, but we'll tolerate it here to allow other
		// evaluation to complete.
		return cty.DynamicVal
	}

	var val cty.Value
	var valRange tfdiags.SourceRange
	if v.addr.Module.IsRoot() {
		val = v.planner.PlanOptions().RootVariableVals[v.addr.Variable.Name]
		if val == cty.NilVal {
			val = cty.NullVal(cty.DynamicPseudoType)
		}

		// TODO: We should give the planner access to the source ranges of
		// the root module variables, because some of them will be in .tfvars
		// files where we really can return a useful source range. For now
		// though, we'll just blame the declaration.
		valRange = tfdiags.SourceRangeFromHCL(config.DeclRange)
	} else {
		// TODO: Evaluate the expression in the module call, and record its
		// source range in valRange.
		val = cty.DynamicVal
	}

	wantType := cty.DynamicPseudoType
	if config.Type != cty.NilType {
		wantType = config.Type
	}

	val, err := convert.Convert(val, wantType)
	if err != nil {
		v.planner.AddDiagnostics(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  `Incorrect variable type`,
			Detail:   fmt.Sprintf(`The resolved value of variable %q is not appropriate: %s.`, v.addr.Variable.Name, err),
			Subject:  valRange.ToHCL().Ptr(),
		})
		val = cty.UnknownVal(wantType)
	}

	return val
}
