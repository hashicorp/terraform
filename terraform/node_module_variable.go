package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/dag"
	"github.com/hashicorp/terraform/instances"
	"github.com/hashicorp/terraform/lang"
	"github.com/hashicorp/terraform/tfdiags"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
)

// nodeExpandModuleVariable is the placeholder for an variable that has not yet had
// its module path expanded.
type nodeExpandModuleVariable struct {
	Addr   addrs.InputVariable
	Module addrs.Module
	Config *configs.Variable
	Expr   hcl.Expression
}

var (
	_ GraphNodeDynamicExpandable = (*nodeExpandModuleVariable)(nil)
	_ GraphNodeReferenceOutside  = (*nodeExpandModuleVariable)(nil)
	_ GraphNodeReferenceable     = (*nodeExpandModuleVariable)(nil)
	_ GraphNodeReferencer        = (*nodeExpandModuleVariable)(nil)
	_ graphNodeTemporaryValue    = (*nodeExpandModuleVariable)(nil)
	_ graphNodeExpandsInstances  = (*nodeExpandModuleVariable)(nil)
)

func (n *nodeExpandModuleVariable) expandsInstances() {}

func (n *nodeExpandModuleVariable) temporaryValue() bool {
	return true
}

func (n *nodeExpandModuleVariable) DynamicExpand(ctx EvalContext) (*Graph, error) {
	var g Graph
	expander := ctx.InstanceExpander()
	for _, module := range expander.ExpandModule(n.Module) {
		o := &nodeModuleVariable{
			Addr:           n.Addr.Absolute(module),
			Config:         n.Config,
			Expr:           n.Expr,
			ModuleInstance: module,
		}
		g.Add(o)
	}
	return &g, nil
}

func (n *nodeExpandModuleVariable) Name() string {
	return fmt.Sprintf("%s.%s (expand)", n.Module, n.Addr.String())
}

// GraphNodeModulePath
func (n *nodeExpandModuleVariable) ModulePath() addrs.Module {
	return n.Module
}

// GraphNodeReferencer
func (n *nodeExpandModuleVariable) References() []*addrs.Reference {

	// If we have no value expression, we cannot depend on anything.
	if n.Expr == nil {
		return nil
	}

	// Variables in the root don't depend on anything, because their values
	// are gathered prior to the graph walk and recorded in the context.
	if len(n.Module) == 0 {
		return nil
	}

	// Otherwise, we depend on anything referenced by our value expression.
	// We ignore diagnostics here under the assumption that we'll re-eval
	// all these things later and catch them then; for our purposes here,
	// we only care about valid references.
	//
	// Due to our GraphNodeReferenceOutside implementation, the addresses
	// returned by this function are interpreted in the _parent_ module from
	// where our associated variable was declared, which is correct because
	// our value expression is assigned within a "module" block in the parent
	// module.
	refs, _ := lang.ReferencesInExpr(n.Expr)
	return refs
}

// GraphNodeReferenceOutside implementation
func (n *nodeExpandModuleVariable) ReferenceOutside() (selfPath, referencePath addrs.Module) {
	return n.Module, n.Module.Parent()
}

// GraphNodeReferenceable
func (n *nodeExpandModuleVariable) ReferenceableAddrs() []addrs.Referenceable {
	return []addrs.Referenceable{n.Addr}
}

// nodeModuleVariable represents a module variable input during
// the apply step.
type nodeModuleVariable struct {
	Addr   addrs.AbsInputVariableInstance
	Config *configs.Variable // Config is the var in the config
	Expr   hcl.Expression    // Expr is the value expression given in the call
	// ModuleInstance in order to create the appropriate context for evaluating
	// ModuleCallArguments, ex. so count.index and each.key can resolve
	ModuleInstance addrs.ModuleInstance
}

// Ensure that we are implementing all of the interfaces we think we are
// implementing.
var (
	_ GraphNodeModuleInstance = (*nodeModuleVariable)(nil)
	_ graphNodeTemporaryValue = (*nodeModuleVariable)(nil)
	_ dag.GraphNodeDotter     = (*nodeModuleVariable)(nil)
)

func (n *nodeModuleVariable) temporaryValue() bool {
	return true
}

func (n *nodeModuleVariable) Name() string {
	return n.Addr.String()
}

// GraphNodeModuleInstance
func (n *nodeModuleVariable) Path() addrs.ModuleInstance {
	// We execute in the parent scope (above our own module) because
	// expressions in our value are resolved in that context.
	return n.Addr.Module.Parent()
}

// GraphNodeModulePath
func (n *nodeModuleVariable) ModulePath() addrs.Module {
	return n.Addr.Module.Module()
}

// GraphNodeExecable
func (n *nodeModuleVariable) Execute(ctx EvalContext) tfdiags.Diagnostics {
	log.Printf("[TRACE] %T: Execute\n", n)
	// If we have no value, do nothing
	if n.Expr == nil {
		return nil
	}

	// Otherwise, interpolate the value of this variable and set it
	// within the variables mapping.
	vals := make(map[string]cty.Value)
	var diags tfdiags.Diagnostics
	_, call := n.Addr.Module.CallInstance()

	var req *EvalModuleCallArgument
	op := ctx.(*BuiltinEvalContext).Evaluator.Operation
	switch op {
	case walkRefresh, walkPlan, walkApply, walkDestroy:
		req = &EvalModuleCallArgument{
			Addr:           n.Addr.Variable,
			Config:         n.Config,
			Expr:           n.Expr,
			ModuleInstance: n.ModuleInstance,
			Values:         vals,
		}
		err := execModuleCallArgument(req, ctx)
		if err != nil {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Detail:   err.Error(),
			})
		}
	case walkValidate:
		req = &EvalModuleCallArgument{
			Addr:           n.Addr.Variable,
			Config:         n.Config,
			Expr:           n.Expr,
			ModuleInstance: n.ModuleInstance,
			Values:         vals,
			validateOnly:   true,
		}
		err := execModuleCallArgument(req, ctx)
		if err != nil {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Detail:   err.Error(),
			})
		}
	}

	ctx.SetModuleCallArguments(call, vals)

	err := execVariableValidations(&evalVariableValidations{
		Addr:   n.Addr,
		Config: n.Config,
		Expr:   n.Expr,
	}, ctx)
	if err != nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Detail:   err.Error(),
		})
	}

	return diags
}

func execModuleCallArgument(n *EvalModuleCallArgument, ctx EvalContext) error {
	// Clear out the existing mapping
	for k := range n.Values {
		delete(n.Values, k)
	}

	wantType := n.Config.Type
	name := n.Addr.Name
	expr := n.Expr

	if expr == nil {
		// Should never happen, but we'll bail out early here rather than
		// crash in case it does. We set no value at all in this case,
		// making a subsequent call to EvalContext.SetModuleCallArguments
		// a no-op.
		return fmt.Errorf("[ERROR] attempt to evaluate %s with nil expression", n.Addr.String())
	}

	var moduleInstanceRepetitionData instances.RepetitionData

	switch {
	case n.validateOnly:
		// the instance expander does not track unknown expansion values, so we
		// have to assume all RepetitionData is unknown.
		moduleInstanceRepetitionData = instances.RepetitionData{
			CountIndex: cty.UnknownVal(cty.Number),
			EachKey:    cty.UnknownVal(cty.String),
			EachValue:  cty.DynamicVal,
		}

	default:
		// Get the repetition data for this module instance,
		// so we can create the appropriate scope for evaluating our expression
		moduleInstanceRepetitionData = ctx.InstanceExpander().GetModuleInstanceRepetitionData(n.ModuleInstance)
	}

	scope := ctx.EvaluationScope(nil, moduleInstanceRepetitionData)
	val, diags := scope.EvalExpr(expr, cty.DynamicPseudoType)

	// We intentionally passed DynamicPseudoType to EvalExpr above because
	// now we can do our own local type conversion and produce an error message
	// with better context if it fails.
	var convErr error
	val, convErr = convert.Convert(val, wantType)
	if convErr != nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid value for module argument",
			Detail: fmt.Sprintf(
				"The given value is not suitable for child module variable %q defined at %s: %s.",
				name, n.Config.DeclRange.String(), convErr,
			),
			Subject: expr.Range().Ptr(),
		})
		// We'll return a placeholder unknown value to avoid producing
		// redundant downstream errors.
		val = cty.UnknownVal(wantType)
	}

	n.Values[name] = val
	return diags.ErrWithWarnings()
}

func execVariableValidations(n *evalVariableValidations, ctx EvalContext) error {
	log.Printf("[TRACE] execVariableValidations: DO IT")
	if n.Config == nil || len(n.Config.Validations) == 0 {
		log.Printf("[TRACE] execVariableValidations: not active for %s, so skipping", n.Addr)
		return nil
	}

	var diags tfdiags.Diagnostics

	// Variable nodes evaluate in the parent module to where they were declared
	// because the value expression (n.Expr, if set) comes from the calling
	// "module" block in the parent module.
	//
	// Validation expressions are statically validated (during configuration
	// loading) to refer only to the variable being validated, so we can
	// bypass our usual evaluation machinery here and just produce a minimal
	// evaluation context containing just the required value, and thus avoid
	// the problem that ctx's evaluation functions refer to the wrong module.
	val := ctx.GetVariableValue(n.Addr)
	hclCtx := &hcl.EvalContext{
		Variables: map[string]cty.Value{
			"var": cty.ObjectVal(map[string]cty.Value{
				n.Config.Name: val,
			}),
		},
		Functions: ctx.EvaluationScope(nil, EvalDataForNoInstanceKey).Functions(),
	}

	for _, validation := range n.Config.Validations {
		const errInvalidCondition = "Invalid variable validation result"
		const errInvalidValue = "Invalid value for variable"

		result, moreDiags := validation.Condition.Value(hclCtx)
		diags = diags.Append(moreDiags)
		if moreDiags.HasErrors() {
			log.Printf("[TRACE] evalVariableValidations: %s rule %s condition expression failed: %s", n.Addr, validation.DeclRange, diags.Err().Error())
		}
		if !result.IsKnown() {
			log.Printf("[TRACE] evalVariableValidations: %s rule %s condition value is unknown, so skipping validation for now", n.Addr, validation.DeclRange)
			continue // We'll wait until we've learned more, then.
		}
		if result.IsNull() {
			diags = diags.Append(&hcl.Diagnostic{
				Severity:    hcl.DiagError,
				Summary:     errInvalidCondition,
				Detail:      "Validation condition expression must return either true or false, not null.",
				Subject:     validation.Condition.Range().Ptr(),
				Expression:  validation.Condition,
				EvalContext: hclCtx,
			})
			continue
		}
		var err error
		result, err = convert.Convert(result, cty.Bool)
		if err != nil {
			diags = diags.Append(&hcl.Diagnostic{
				Severity:    hcl.DiagError,
				Summary:     errInvalidCondition,
				Detail:      fmt.Sprintf("Invalid validation condition result value: %s.", tfdiags.FormatError(err)),
				Subject:     validation.Condition.Range().Ptr(),
				Expression:  validation.Condition,
				EvalContext: hclCtx,
			})
			continue
		}

		if result.False() {
			if n.Expr != nil {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  errInvalidValue,
					Detail:   fmt.Sprintf("%s\n\nThis was checked by the validation rule at %s.", validation.ErrorMessage, validation.DeclRange.String()),
					Subject:  n.Expr.Range().Ptr(),
				})
			} else {
				// Since we don't have a source expression for a root module
				// variable, we'll just report the error from the perspective
				// of the variable declaration itself.
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  errInvalidValue,
					Detail:   fmt.Sprintf("%s\n\nThis was checked by the validation rule at %s.", validation.ErrorMessage, validation.DeclRange.String()),
					Subject:  n.Config.DeclRange.Ptr(),
				})
			}
		}
	}

	return diags.ErrWithWarnings()
}

// dag.GraphNodeDotter impl.
func (n *nodeModuleVariable) DotNode(name string, opts *dag.DotOpts) *dag.DotNode {
	return &dag.DotNode{
		Name: name,
		Attrs: map[string]string{
			"label": n.Name(),
			"shape": "note",
		},
	}
}
