// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/lang/langrefs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// nodeExpandModuleVariable is the placeholder for an variable that has not yet had
// its module path expanded.
type nodeExpandModuleVariable struct {
	Addr   addrs.InputVariable
	Module addrs.Module
	Config *configs.Variable
	Expr   hcl.Expression

	// Planning must be set to true when building a planning graph, and must be
	// false when building an apply graph.
	Planning bool

	// DestroyApply must be set to true when planning or applying a destroy
	// operation, and false otherwise.
	DestroyApply bool
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

func (n *nodeExpandModuleVariable) DynamicExpand(ctx EvalContext) (*Graph, tfdiags.Diagnostics) {
	var g Graph

	// If this variable has preconditions, we need to report these checks now.
	//
	// We should only do this during planning as the apply phase starts with
	// all the same checkable objects that were registered during the plan.
	var checkableAddrs addrs.Set[addrs.Checkable]
	if n.Planning {
		if checkState := ctx.Checks(); checkState.ConfigHasChecks(n.Addr.InModule(n.Module)) {
			checkableAddrs = addrs.MakeSet[addrs.Checkable]()
		}
	}

	expander := ctx.InstanceExpander()
	forEachModuleInstance(expander, n.Module, false, func(module addrs.ModuleInstance) {
		addr := n.Addr.Absolute(module)
		if checkableAddrs != nil {
			log.Printf("[TRACE] nodeExpandModuleVariable: found checkable object %s", addr)
			checkableAddrs.Add(addr)
		}

		o := &nodeModuleVariable{
			Addr:           addr,
			Config:         n.Config,
			Expr:           n.Expr,
			ModuleInstance: module,
			DestroyApply:   n.DestroyApply,
		}
		g.Add(o)
	}, func(pem addrs.PartialExpandedModule) {
		addr := addrs.ObjectInPartialExpandedModule(pem, n.Addr)
		o := &nodeModuleVariableInPartialModule{
			Addr:           addr,
			Config:         n.Config,
			Expr:           n.Expr,
			ModuleInstance: pem,
			DestroyApply:   n.DestroyApply,
		}
		g.Add(o)
	})
	addRootNodeToGraph(&g)

	if checkableAddrs != nil {
		ctx.Checks().ReportCheckableObjects(n.Addr.InModule(n.Module), checkableAddrs)
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
	refs, _ := langrefs.ReferencesInExpr(addrs.ParseRef, n.Expr)
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

// variableValidationRules implements [graphNodeValidatableVariable].
func (n *nodeExpandModuleVariable) variableValidationRules() (addrs.ConfigInputVariable, []*configs.CheckRule, hcl.Range) {
	var defnRange hcl.Range
	if n.Expr != nil { // should always be set in real calls, but not always in tests
		defnRange = n.Expr.Range()
	}
	if n.DestroyApply {
		// We don't perform any variable validation during the apply phase
		// of a destroy, because validation rules typically aren't prepared
		// for dealing with things already having been destroyed.
		return n.Addr.InModule(n.Module), nil, defnRange
	}
	var rules []*configs.CheckRule
	if n.Config != nil { // always in normal code, but sometimes not in unit tests
		rules = n.Config.Validations
	}
	return n.Addr.InModule(n.Module), rules, defnRange
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

	// ModuleCallConfig is the module call that the expression in field Expr
	// came from, which helps decide what [instances.RepetitionData] we should
	// use when evaluating Expr.
	ModuleCallConfig *configs.ModuleCall

	// DestroyApply must be set to true when applying a destroy operation and
	// false otherwise.
	DestroyApply bool
}

// Ensure that we are implementing all of the interfaces we think we are
// implementing.
var (
	_ GraphNodeModuleInstance = (*nodeModuleVariable)(nil)
	_ GraphNodeExecutable     = (*nodeModuleVariable)(nil)
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

// GraphNodeExecutable
func (n *nodeModuleVariable) Execute(ctx EvalContext, op walkOperation) (diags tfdiags.Diagnostics) {
	log.Printf("[TRACE] nodeModuleVariable: evaluating %s", n.Addr)

	var val cty.Value
	var err error

	switch op {
	case walkValidate:
		val, err = n.evalModuleVariable(ctx, true)
		diags = diags.Append(err)
	default:
		val, err = n.evalModuleVariable(ctx, false)
		diags = diags.Append(err)
	}
	if diags.HasErrors() {
		return diags
	}

	// Set values for arguments of a child module call, for later retrieval
	// during expression evaluation.
	ctx.NamedValues().SetInputVariableValue(n.Addr, val)

	// Custom validation rules are handled by a separate graph node of type
	// nodeVariableValidation, added by variableValidationTransformer.

	return diags
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

// evalModuleVariable produces the value for a particular variable as will
// be used by a child module instance.
//
// validateOnly indicates that this evaluation is only for config
// validation, and we will not have any expansion module instance
// repetition data.
func (n *nodeModuleVariable) evalModuleVariable(ctx EvalContext, validateOnly bool) (cty.Value, error) {
	var diags tfdiags.Diagnostics
	var givenVal cty.Value
	var errSourceRange tfdiags.SourceRange
	if expr := n.Expr; expr != nil {
		var moduleInstanceRepetitionData instances.RepetitionData

		switch {
		case validateOnly:
			// the instance expander does not track unknown expansion values, so we
			// have to assume all RepetitionData is unknown.
			// TODO: Ideally we should vary the placeholder we use here based
			// on how the module call repetition was configured, but we don't
			// have enough information here to decide that.
			moduleInstanceRepetitionData = instances.TotallyUnknownRepetitionData

		default:
			// Get the repetition data for this module instance,
			// so we can create the appropriate scope for evaluating our expression
			moduleInstanceRepetitionData = ctx.InstanceExpander().GetModuleInstanceRepetitionData(n.ModuleInstance)
		}

		scope := ctx.EvaluationScope(nil, nil, moduleInstanceRepetitionData)
		val, moreDiags := scope.EvalExpr(expr, cty.DynamicPseudoType)
		diags = diags.Append(moreDiags)
		if moreDiags.HasErrors() {
			return cty.DynamicVal, diags.ErrWithWarnings()
		}
		givenVal = val
		errSourceRange = tfdiags.SourceRangeFromHCL(expr.Range())
	} else {
		// We'll use cty.NilVal to represent the variable not being set at all.
		givenVal = cty.NilVal
		errSourceRange = tfdiags.SourceRangeFromHCL(n.Config.DeclRange) // we use the declaration range as a fallback for an undefined variable
	}

	// We construct a synthetic InputValue here to pretend as if this were
	// a root module variable set from outside, just as a convenience so we
	// can reuse the InputValue type for this.
	rawVal := &InputValue{
		Value:       givenVal,
		SourceType:  ValueFromConfig,
		SourceRange: errSourceRange,
	}

	finalVal, moreDiags := prepareFinalInputVariableValue(n.Addr, rawVal, n.Config)
	diags = diags.Append(moreDiags)

	return finalVal, diags.ErrWithWarnings()
}

// nodeModuleVariableInPartialModule represents an infinite set of possible
// input variable instances beneath a partially-expanded module instance prefix.
//
// Its job is to find a suitable placeholder value that approximates the
// values of all of those possible instances. Ideally that's a concrete
// known value if all instances would have the same value, an unknown value
// of a specific type if the definition produces a known type, or a
// totally-unknown value of unknown type in the worst case.
type nodeModuleVariableInPartialModule struct {
	Addr   addrs.InPartialExpandedModule[addrs.InputVariable]
	Config *configs.Variable // Config is the var in the config
	Expr   hcl.Expression    // Expr is the value expression given in the call
	// ModuleInstance in order to create the appropriate context for evaluating
	// ModuleCallArguments, ex. so count.index and each.key can resolve
	ModuleInstance addrs.PartialExpandedModule

	// DestroyApply must be set to true when applying a destroy operation and
	// false otherwise.
	DestroyApply bool
}

func (n *nodeModuleVariableInPartialModule) Path() addrs.PartialExpandedModule {
	return n.Addr.Module
}

func (n *nodeModuleVariableInPartialModule) Execute(ctx EvalContext, op walkOperation) tfdiags.Diagnostics {
	// Our job here is to make sure that the input variable definition is
	// valid for all instances of this input variable across all of the possible
	// module instances under our partially-expanded prefix, and to record
	// a placeholder value that captures as precisely as possible what all
	// of those results have in common. In the worst case where they have
	// absolutely nothing in common cty.DynamicVal is the ultimate fallback,
	// but we should try to do better when possible to give operators earlier
	// feedback about any problems they would definitely encounter on a
	// subsequent plan where the input variables get evaluated concretely.

	namedVals := ctx.NamedValues()

	// TODO: Ideally we should vary the placeholder we use here based
	// on how the module call repetition was configured, but we don't
	// have enough information here to decide that.
	moduleInstanceRepetitionData := instances.TotallyUnknownRepetitionData

	// NOTE WELL: Input variables are a little strange in that they announce
	// themselves as belonging to the caller of the module they are declared
	// in, because that's where their definition expressions get evaluated.
	// Therefore this [EvalContext] is in the scope of the parent module,
	// while n.Addr describes an object in the child module (where the
	// variable declaration appeared).
	scope := ctx.EvaluationScope(nil, nil, moduleInstanceRepetitionData)
	val, diags := scope.EvalExpr(n.Expr, cty.DynamicPseudoType)

	namedVals.SetInputVariablePlaceholder(n.Addr, val)
	return diags
}
