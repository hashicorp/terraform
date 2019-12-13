package applying

import (
	"context"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/tfdiags"
)

type namedValueActions struct {
	RootInputVariables   map[string]*rootInputVariableActions
	ModuleInputVariables map[string]*moduleInputVariableActions
	LocalValues          map[string]*localValueActions
	OutputValues         map[string]*outputValueActions
}

type rootInputVariableActions struct {
	Eval         *rootInputVariableEvalAction
	Dependencies []addrs.Referenceable
}

type moduleInputVariableActions struct {
	Eval         *moduleInputVariableEvalAction
	Dependencies []addrs.Referenceable
}

type localValueActions struct {
	Eval         *localValueEvalAction
	Dependencies []addrs.Referenceable
}

type outputValueActions struct {
	Eval         *outputValueEvalAction
	Dependencies []addrs.Referenceable
}

type rootInputVariableEvalAction struct {
	Addr  addrs.AbsInputVariableInstance // Module is always addrs.RootModuleInstance
	Value cty.Value
}

func (a *rootInputVariableEvalAction) Name() string {
	return "Evaluate " + a.Addr.String()
}

func (a *rootInputVariableEvalAction) Execute(ctx context.Context, data *actionData) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	diags = diags.Append(tfdiags.Sourceless(
		tfdiags.Error,
		"Root input variable evaluation not yet implemented",
		"The prototype apply codepath does not yet support evaluating input variables in the root module.",
	))

	return diags
}

type moduleInputVariableEvalAction struct {
	Addr addrs.AbsInputVariableInstance
	Expr hcl.Expression
}

func (a *moduleInputVariableEvalAction) Name() string {
	return "Evaluate " + a.Addr.String()
}

func (a *moduleInputVariableEvalAction) Execute(ctx context.Context, data *actionData) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	diags = diags.Append(tfdiags.Sourceless(
		tfdiags.Error,
		"Module input variable evaluation not yet implemented",
		"The prototype apply codepath does not yet support evaluating input variables in a non-root module.",
	))

	return diags
}

type localValueEvalAction struct {
	Addr addrs.AbsLocalValue
	Expr hcl.Expression
}

func (a *localValueEvalAction) Name() string {
	return "Evaluate " + a.Addr.String()
}

func (a *localValueEvalAction) Execute(ctx context.Context, data *actionData) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	diags = diags.Append(tfdiags.Sourceless(
		tfdiags.Error,
		"Local value evaluation not yet implemented",
		"The prototype apply codepath does not yet support evaluating local values.",
	))

	return diags
}

type outputValueEvalAction struct {
	Addr addrs.AbsOutputValue
	Expr hcl.Expression
}

func (a *outputValueEvalAction) Name() string {
	return "Evaluate " + a.Addr.String()
}

func (a *outputValueEvalAction) Execute(ctx context.Context, data *actionData) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	diags = diags.Append(tfdiags.Sourceless(
		tfdiags.Error,
		"Local value evaluation not yet implemented",
		"The prototype apply codepath does not yet support evaluating local values.",
	))

	return diags
}
