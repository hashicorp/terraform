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
	"github.com/hashicorp/terraform/internal/lang/langrefs"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/moduletest/mocking"
	"github.com/hashicorp/terraform/internal/namedvals"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/plans/deferring"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// nodeExpandOutput is the placeholder for a non-root module output that has
// not yet had its module path expanded.
type nodeExpandOutput struct {
	Addr        addrs.OutputValue
	Module      addrs.Module
	Config      *configs.Output
	Destroying  bool
	RefreshOnly bool

	// Planning is set to true when this node is in a graph that was produced
	// by the plan graph builder, as opposed to the apply graph builder.
	// This quirk is just because we share the same node type between both
	// phases but in practice there are a few small differences in the actions
	// we need to take between plan and apply. See method DynamicExpand for
	// details.
	Planning bool

	// Overrides is the set of overrides applied by the testing framework. We
	// may need to override the value for this output and if we do the value
	// comes from here.
	Overrides *mocking.Overrides

	Dependencies []addrs.ConfigResource

	Dependants []*addrs.Reference
}

var (
	_ GraphNodeReferenceable      = (*nodeExpandOutput)(nil)
	_ GraphNodeReferencer         = (*nodeExpandOutput)(nil)
	_ GraphNodeReferenceOutside   = (*nodeExpandOutput)(nil)
	_ GraphNodeDynamicExpandable  = (*nodeExpandOutput)(nil)
	_ graphNodeTemporaryValue     = (*nodeExpandOutput)(nil)
	_ GraphNodeAttachDependencies = (*nodeExpandOutput)(nil)
	_ graphNodeExpandsInstances   = (*nodeExpandOutput)(nil)
)

func (n *nodeExpandOutput) expandsInstances() {}

func (n *nodeExpandOutput) temporaryValue() bool {
	// non root outputs are temporary
	return !n.Module.IsRoot()
}

// GraphNodeAttachDependencies
func (n *nodeExpandOutput) AttachDependencies(resources []addrs.ConfigResource) {
	n.Dependencies = resources
}

func (n *nodeExpandOutput) DynamicExpand(ctx EvalContext) (*Graph, tfdiags.Diagnostics) {
	expander := ctx.InstanceExpander()
	changes := ctx.Changes()

	// If this is an output value that participates in custom condition checks
	// (i.e. it has preconditions or postconditions) then the check state
	// wants to know the addresses of the checkable objects so that it can
	// treat them as unknown status if we encounter an error before actually
	// visiting the checks.
	//
	// We must do this only during planning, because the apply phase will start
	// with all of the same checkable objects that were registered during the
	// planning phase. Consumers of our JSON plan and state formats expect
	// that the set of checkable objects will be consistent between the plan
	// and any state snapshots created during apply, and that only the statuses
	// of those objects will have changed.
	var checkableAddrs addrs.Set[addrs.Checkable]
	if n.Planning {
		if checkState := ctx.Checks(); checkState.ConfigHasChecks(n.Addr.InModule(n.Module)) {
			checkableAddrs = addrs.MakeSet[addrs.Checkable]()
		}
	}

	var g Graph
	forEachModuleInstance(
		expander, n.Module, true,
		func(module addrs.ModuleInstance) {
			absAddr := n.Addr.Absolute(module)
			if checkableAddrs != nil {
				checkableAddrs.Add(absAddr)
			}

			// Find any recorded change for this output
			var change *plans.OutputChange
			var outputChanges []*plans.OutputChange
			if module.IsRoot() {
				outputChanges = changes.GetRootOutputChanges()
			} else {
				parent, call := module.Call()
				outputChanges = changes.GetOutputChanges(parent, call)
			}
			for _, c := range outputChanges {
				if c.Addr.String() == absAddr.String() {
					change = c
					break
				}
			}

			var node dag.Vertex
			switch {
			case module.IsRoot() && n.Destroying:
				node = &NodeDestroyableOutput{
					Addr:     absAddr,
					Planning: n.Planning,
				}

			default:
				node = &NodeApplyableOutput{
					Addr:         absAddr,
					Config:       n.Config,
					Change:       change,
					RefreshOnly:  n.RefreshOnly,
					DestroyApply: n.Destroying,
					Planning:     n.Planning,
					Override:     n.getOverrideValue(absAddr.Module),
					Dependencies: n.Dependencies,
					Dependants:   n.Dependants,
				}
			}

			log.Printf("[TRACE] Expanding output: adding %s as %T", absAddr.String(), node)
			g.Add(node)
		},
		func(pem addrs.PartialExpandedModule) {
			absAddr := addrs.ObjectInPartialExpandedModule(pem, n.Addr)
			node := &nodeOutputInPartialModule{
				Addr:        absAddr,
				Config:      n.Config,
				RefreshOnly: n.RefreshOnly,
			}
			// We don't need to handle the module.IsRoot() && n.Destroying case
			// seen in the fully-expanded case above, because the root module
			// instance is always "fully expanded" (it's always a singleton)
			// and so we can't get here for output values in the root module.
			log.Printf("[TRACE] Expanding output: adding placeholder for all %s as %T", absAddr.String(), node)
			g.Add(node)
		},
	)
	addRootNodeToGraph(&g)

	if checkableAddrs != nil {
		checkState := ctx.Checks()
		checkState.ReportCheckableObjects(n.Addr.InModule(n.Module), checkableAddrs)
	}

	return &g, nil
}

func (n *nodeExpandOutput) Name() string {
	path := n.Module.String()
	addr := n.Addr.String() + " (expand)"
	if path != "" {
		return path + "." + addr
	}
	return addr
}

// GraphNodeModulePath
func (n *nodeExpandOutput) ModulePath() addrs.Module {
	return n.Module
}

// GraphNodeReferenceable
func (n *nodeExpandOutput) ReferenceableAddrs() []addrs.Referenceable {
	// An output in the root module can't be referenced at all.
	if n.Module.IsRoot() {
		return nil
	}

	// the output is referenced through the module call, and via the
	// module itself.
	_, call := n.Module.Call()
	callOutput := addrs.ModuleCallOutput{
		Call: call,
		Name: n.Addr.Name,
	}

	// Otherwise, we can reference the output via the
	// module call itself
	return []addrs.Referenceable{call, callOutput}
}

// GraphNodeReferenceOutside implementation
func (n *nodeExpandOutput) ReferenceOutside() (selfPath, referencePath addrs.Module) {
	// Output values have their expressions resolved in the context of the
	// module where they are defined.
	referencePath = n.Module

	// ...but they are referenced in the context of their calling module.
	selfPath = referencePath.Parent()

	return // uses named return values
}

// GraphNodeReferencer
func (n *nodeExpandOutput) References() []*addrs.Reference {
	// DestroyNodes do not reference anything.
	if n.Module.IsRoot() && n.Destroying {
		return nil
	}

	return referencesForOutput(n.Config)
}

func (n *nodeExpandOutput) getOverrideValue(inst addrs.ModuleInstance) cty.Value {
	// First check if we have any overrides at all, this is a shorthand for
	// "are we running terraform test".
	if n.Overrides.Empty() {
		// cty.NilVal means no override
		return cty.NilVal
	}

	// We have overrides, let's see if we have one for this module instance.
	if override, ok := n.Overrides.GetModuleOverride(inst); ok {

		output := n.Addr.Name
		values := override.Values

		// The values.Type() should be an object type, but it might have
		// been set to nil by a test or something. We can handle it in the
		// same way as the attribute just not being specified. It's
		// functionally the same for us and not something we need to raise
		// alarms about.
		if values.Type().IsObjectType() && values.Type().HasAttribute(output) {
			return values.GetAttr(output)
		}

		// If we don't have a value provided for an output, then we'll
		// just set it to be null.
		//
		// TODO(liamcervante): Can we generate a value here? Probably
		//   not as we don't know the type.
		return cty.NullVal(cty.DynamicPseudoType)
	}

	// cty.NilVal indicates no override.
	return cty.NilVal
}

// NodeApplyableOutput represents an output that is "applyable":
// it is ready to be applied.
type NodeApplyableOutput struct {
	Addr   addrs.AbsOutputValue
	Config *configs.Output // Config is the output in the config
	// If this is being evaluated during apply, we may have a change recorded already
	Change *plans.OutputChange

	// Refresh-only mode means that any failing output preconditions are
	// reported as warnings rather than errors
	RefreshOnly bool

	// DestroyApply indicates that we are applying a destroy plan, and do not
	// need to account for conditional blocks.
	DestroyApply bool

	Planning bool

	// Override provides the value to use for this output, if any. This can be
	// set by testing framework when a module is overridden.
	Override cty.Value

	// Dependencies is the full set of resources that are referenced by this
	// output.
	Dependencies []addrs.ConfigResource

	Dependants []*addrs.Reference
}

var (
	_ GraphNodeModuleInstance   = (*NodeApplyableOutput)(nil)
	_ GraphNodeReferenceable    = (*NodeApplyableOutput)(nil)
	_ GraphNodeReferencer       = (*NodeApplyableOutput)(nil)
	_ GraphNodeReferenceOutside = (*NodeApplyableOutput)(nil)
	_ GraphNodeExecutable       = (*NodeApplyableOutput)(nil)
	_ graphNodeTemporaryValue   = (*NodeApplyableOutput)(nil)
	_ dag.GraphNodeDotter       = (*NodeApplyableOutput)(nil)
)

func (n *NodeApplyableOutput) temporaryValue() bool {
	// this must always be evaluated if it is a root module output
	return !n.Addr.Module.IsRoot()
}

func (n *NodeApplyableOutput) Name() string {
	return n.Addr.String()
}

// GraphNodeModuleInstance
func (n *NodeApplyableOutput) Path() addrs.ModuleInstance {
	return n.Addr.Module
}

// GraphNodeModulePath
func (n *NodeApplyableOutput) ModulePath() addrs.Module {
	return n.Addr.Module.Module()
}

func referenceOutsideForOutput(addr addrs.AbsOutputValue) (selfPath, referencePath addrs.Module) {
	// Output values have their expressions resolved in the context of the
	// module where they are defined.
	referencePath = addr.Module.Module()

	// ...but they are referenced in the context of their calling module.
	selfPath = addr.Module.Parent().Module()

	return // uses named return values
}

// GraphNodeReferenceOutside implementation
func (n *NodeApplyableOutput) ReferenceOutside() (selfPath, referencePath addrs.Module) {
	return referenceOutsideForOutput(n.Addr)
}

func referenceableAddrsForOutput(addr addrs.AbsOutputValue) []addrs.Referenceable {
	// An output in the root module can't be referenced at all.
	if addr.Module.IsRoot() {
		return nil
	}

	// Otherwise, we can be referenced via a reference to our output name
	// on the parent module's call, or via a reference to the entire call.
	// e.g. module.foo.bar or just module.foo .
	// Note that our ReferenceOutside method causes these addresses to be
	// relative to the calling module, not the module where the output
	// was declared.
	_, outp := addr.ModuleCallOutput()
	_, call := addr.Module.CallInstance()

	return []addrs.Referenceable{outp, call}
}

// GraphNodeReferenceable
func (n *NodeApplyableOutput) ReferenceableAddrs() []addrs.Referenceable {
	return referenceableAddrsForOutput(n.Addr)
}

func referencesForOutput(c *configs.Output) []*addrs.Reference {
	var refs []*addrs.Reference

	impRefs, _ := langrefs.ReferencesInExpr(addrs.ParseRef, c.Expr)
	expRefs, _ := langrefs.References(addrs.ParseRef, c.DependsOn)

	refs = append(refs, impRefs...)
	refs = append(refs, expRefs...)

	for _, check := range c.Preconditions {
		condRefs, _ := langrefs.ReferencesInExpr(addrs.ParseRef, check.Condition)
		refs = append(refs, condRefs...)
		errRefs, _ := langrefs.ReferencesInExpr(addrs.ParseRef, check.ErrorMessage)
		refs = append(refs, errRefs...)
	}

	return refs
}

// GraphNodeReferencer
func (n *NodeApplyableOutput) References() []*addrs.Reference {
	return referencesForOutput(n.Config)
}

// GraphNodeExecutable
func (n *NodeApplyableOutput) Execute(ctx EvalContext, op walkOperation) (diags tfdiags.Diagnostics) {
	if op == walkValidate && n.Config.DeprecatedSet && len(n.Dependants) > 0 {
		for _, d := range n.Dependants {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagWarning,
				Summary:  "Usage of deprecated output",
				Detail:   n.Config.Deprecated,
				Subject:  d.SourceRange.ToHCL().Ptr(),
			})
		}
	}

	state := ctx.State()
	if state == nil {
		return
	}

	changes := ctx.Changes() // may be nil, if we're not working on a changeset

	val := cty.UnknownVal(cty.DynamicPseudoType)
	changeRecorded := n.Change != nil
	// we we have a change recorded, we don't need to re-evaluate if the value
	// was known
	if changeRecorded {
		val = n.Change.After
	}

	if n.Addr.Module.IsRoot() && n.Config.Ephemeral {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Ephemeral output not allowed",
			Detail:   "Ephemeral outputs are not allowed in context of a root module",
			Subject:  n.Config.DeclRange.Ptr(),
		})
		return
	}

	// Checks are not evaluated during a destroy. The checks may fail, may not
	// be valid, or may not have been registered at all.
	// We also don't evaluate checks for overridden outputs. This is because
	// any references within the checks will likely not have been created.
	if !n.DestroyApply && n.Override == cty.NilVal {
		checkRuleSeverity := tfdiags.Error
		if n.RefreshOnly {
			checkRuleSeverity = tfdiags.Warning
		}
		checkDiags := evalCheckRules(
			addrs.OutputPrecondition,
			n.Config.Preconditions,
			ctx, n.Addr, EvalDataForNoInstanceKey,
			checkRuleSeverity,
		)
		diags = diags.Append(checkDiags)
		if diags.HasErrors() {
			return diags // failed preconditions prevent further evaluation
		}
	}

	// If there was no change recorded, or the recorded change was not wholly
	// known, then we need to re-evaluate the output
	if !changeRecorded || !val.IsWhollyKnown() {

		// First, we check if we have an overridden value. If we do, then we
		// use that and we don't try and evaluate the underlying expression.
		val = n.Override
		if val == cty.NilVal {
			// This has to run before we have a state lock, since evaluation also
			// reads the state
			var evalDiags tfdiags.Diagnostics
			val, evalDiags = ctx.EvaluateExpr(n.Config.Expr, cty.DynamicPseudoType, nil)
			diags = diags.Append(evalDiags)

			// We'll handle errors below, after we have loaded the module.
			// Outputs don't have a separate mode for validation, so validate
			// depends_on expressions here too
			diags = diags.Append(validateDependsOn(ctx, n.Config.DependsOn))

			// For root module outputs in particular, an output value must be
			// statically declared as sensitive in order to dynamically return
			// a sensitive result, to help avoid accidental exposure in the state
			// of a sensitive value that the user doesn't want to include there.
			if n.Addr.Module.IsRoot() {
				if !n.Config.Sensitive && marks.Contains(val, marks.Sensitive) {
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Output refers to sensitive values",
						Detail: `To reduce the risk of accidentally exporting sensitive data that was intended to be only internal, Terraform requires that any root module output containing sensitive data be explicitly marked as sensitive, to confirm your intent.

If you do intend to export this data, annotate the output value as sensitive by adding the following argument:
    sensitive = true`,
						Subject: n.Config.DeclRange.Ptr(),
					})
				}
			}
		}
	}

	// handling the interpolation error
	if diags.HasErrors() {
		if flagWarnOutputErrors {
			log.Printf("[ERROR] Output interpolation %q failed: %s", n.Addr, diags.Err())
			// if we're continuing, make sure the output is included, and
			// marked as unknown. If the evaluator was able to find a type
			// for the value in spite of the error then we'll use it.
			n.setValue(ctx.NamedValues(), state, changes, ctx.Deferrals(), cty.UnknownVal(val.Type()))

			// Keep existing warnings, while converting errors to warnings.
			// This is not meant to be the normal path, so there no need to
			// make the errors pretty.
			var warnings tfdiags.Diagnostics
			for _, d := range diags {
				switch d.Severity() {
				case tfdiags.Warning:
					warnings = warnings.Append(d)
				case tfdiags.Error:
					desc := d.Description()
					warnings = warnings.Append(tfdiags.SimpleWarning(fmt.Sprintf("%s:%s", desc.Summary, desc.Detail)))
				}
			}

			return warnings
		}
		return diags
	}

	// The checks below this point are intentionally not opted out by
	// "flagWarnOutputErrors", because they relate to features that were added
	// more recently than the historical change to treat invalid output values
	// as errors rather than warnings.
	if n.Config.Ephemeral && !marks.Has(val, marks.Ephemeral) {
		// An ephemeral output value must always be ephemeral
		// This is to prevent accidental persistence upstream
		// from here.
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Value not allowed in ephemeral output",
			Detail:   "This output value is declared as returning an ephemeral value, so it can only be set to an ephemeral value.",
			Subject:  n.Config.Expr.Range().Ptr(),
		})
		return diags
	} else if !n.Config.Ephemeral && marks.Contains(val, marks.Ephemeral) {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Ephemeral value not allowed",
			Detail:   "This output value is not declared as returning an ephemeral value, so it cannot be set to a result derived from an ephemeral value.",
			Subject:  n.Config.Expr.Range().Ptr(),
		})
		return diags
	}

	n.setValue(ctx.NamedValues(), state, changes, ctx.Deferrals(), val)

	// If we were able to evaluate a new value, we can update that in the
	// refreshed state as well.
	if state = ctx.RefreshState(); state != nil && val.IsWhollyKnown() {
		// we only need to update the state, do not pass in the changes again
		n.setValue(nil, state, nil, ctx.Deferrals(), val)
	}

	return diags
}

// dag.GraphNodeDotter impl.
func (n *NodeApplyableOutput) DotNode(name string, opts *dag.DotOpts) *dag.DotNode {
	return &dag.DotNode{
		Name: name,
		Attrs: map[string]string{
			"label": n.Name(),
			"shape": "note",
		},
	}
}

// nodeOutputInPartialModule represents an infinite set of possible output value
// instances beneath a partially-expanded module instance prefix.
//
// Its job is to find a suitable placeholder value that approximates the
// values of all of those possible instances. Ideally that's a concrete
// known value if all instances would have the same value, an unknown value
// of a specific type if the definition produces a known type, or a
// totally-unknown value of unknown type in the worst case.
type nodeOutputInPartialModule struct {
	Addr   addrs.InPartialExpandedModule[addrs.OutputValue]
	Config *configs.Output

	// Refresh-only mode means that any failing output preconditions are
	// reported as warnings rather than errors
	RefreshOnly bool
}

// Path implements [GraphNodePartialExpandedModule], meaning that the
// Execute method receives an [EvalContext] that's set up for partial-expanded
// evaluation instead of full evaluation.
func (n *nodeOutputInPartialModule) Path() addrs.PartialExpandedModule {
	return n.Addr.Module
}

func (n *nodeOutputInPartialModule) Execute(ctx EvalContext, op walkOperation) tfdiags.Diagnostics {
	// Our job here is to make sure that the output value definition is
	// valid for all instances of this output value across all of the possible
	// module instances under our partially-expanded prefix, and to record
	// a placeholder value that captures as precisely as possible what all
	// of those results have in common. In the worst case where they have
	// absolutely nothing in common cty.DynamicVal is the ultimate fallback,
	// but we should try to do better when possible to give operators earlier
	// feedback about any problems they would definitely encounter on a
	// subsequent plan where the output values get evaluated concretely.

	namedVals := ctx.NamedValues()

	// this "ctx" is preconfigured to evaluate in terms of other placeholder
	// values generated in the same unexpanded module prefix, rather than
	// from the active state/plan, so this result is likely to be derived
	// from unknown value placeholders itself.
	val, diags := ctx.EvaluateExpr(n.Config.Expr, cty.DynamicPseudoType, nil)
	if val == cty.NilVal {
		val = cty.DynamicVal
	}

	// We'll also check that the depends_on argument is valid, since that's
	// a static concern anyway and so cannot vary between instances of the
	// same module.
	diags = diags.Append(validateDependsOn(ctx, n.Config.DependsOn))

	namedVals.SetOutputValuePlaceholder(n.Addr, val)
	return diags
}

// NodeDestroyableOutput represents an output that is "destroyable":
// its application will remove the output from the state.
type NodeDestroyableOutput struct {
	Addr     addrs.AbsOutputValue
	Planning bool
}

var (
	_ GraphNodeExecutable = (*NodeDestroyableOutput)(nil)
	_ dag.GraphNodeDotter = (*NodeDestroyableOutput)(nil)
)

func (n *NodeDestroyableOutput) Name() string {
	return fmt.Sprintf("%s (destroy)", n.Addr.String())
}

// GraphNodeModulePath
func (n *NodeDestroyableOutput) ModulePath() addrs.Module {
	return n.Addr.Module.Module()
}

func (n *NodeDestroyableOutput) temporaryValue() bool {
	// this must always be evaluated if it is a root module output
	return !n.Addr.Module.IsRoot()
}

// GraphNodeExecutable
func (n *NodeDestroyableOutput) Execute(ctx EvalContext, op walkOperation) tfdiags.Diagnostics {
	state := ctx.State()
	if state == nil {
		return nil
	}

	// if this is a root module, try to get a before value from the state for
	// the diff
	sensitiveBefore := false
	before := cty.NullVal(cty.DynamicPseudoType)
	mod := state.Module(n.Addr.Module)
	if n.Addr.Module.IsRoot() && mod != nil {
		s := state.Lock()
		rootOutputs := s.RootOutputValues
		if o, ok := rootOutputs[n.Addr.OutputValue.Name]; ok {
			sensitiveBefore = o.Sensitive
			before = o.Value
		} else {
			// If the output was not in state, a delete change would
			// be meaningless, so exit early.
			state.Unlock()
			return nil
		}
		state.Unlock()
	}

	changes := ctx.Changes()
	if changes != nil && n.Planning {
		change := &plans.OutputChange{
			Addr:      n.Addr,
			Sensitive: sensitiveBefore,
			Change: plans.Change{
				Action: plans.Delete,
				Before: before,
				After:  cty.NullVal(cty.DynamicPseudoType),
			},
		}

		changes.RemoveOutputChange(n.Addr) // remove any existing planned change, if present
		changes.AppendOutputChange(change) // add the new planned change
	}
	state.RemoveOutputValue(n.Addr)

	return nil
}

// dag.GraphNodeDotter impl.
func (n *NodeDestroyableOutput) DotNode(name string, opts *dag.DotOpts) *dag.DotNode {
	return &dag.DotNode{
		Name: name,
		Attrs: map[string]string{
			"label": n.Name(),
			"shape": "note",
		},
	}
}

func (n *NodeApplyableOutput) setValue(namedVals *namedvals.State, state *states.SyncState, changes *plans.ChangesSync, deferred *deferring.Deferred, val cty.Value) {
	if changes != nil && n.Planning {
		// if this is a root module, try to get a before value from the state for
		// the diff
		sensitiveBefore := false
		before := cty.NullVal(cty.DynamicPseudoType)

		// is this output new to our state?
		newOutput := true

		mod := state.Module(n.Addr.Module)
		if n.Addr.Module.IsRoot() && mod != nil {
			s := state.Lock()
			rootOutputs := s.RootOutputValues
			for name, o := range rootOutputs {
				if name == n.Addr.OutputValue.Name {
					before = o.Value
					sensitiveBefore = o.Sensitive
					newOutput = false
					break
				}
			}
			state.Unlock()
		}

		// We will not show the value if either the before or after are marked
		// as sensitive. We can show the value again once sensitivity is
		// removed from both the config and the state.
		sensitiveChange := sensitiveBefore || n.Config.Sensitive

		// strip any marks here just to be sure we don't panic on the True comparison
		unmarkedVal, _ := val.UnmarkDeep()

		action := plans.Update
		switch {
		case val.IsNull() && before.IsNull():
			// This is separate from the NoOp case below, since we can ignore
			// sensitivity here when there are only null values.
			action = plans.NoOp

		case newOutput:
			// This output was just added to the configuration
			action = plans.Create

		case val.IsWhollyKnown() &&
			unmarkedVal.Equals(before).True() &&
			n.Config.Sensitive == sensitiveBefore:
			// Sensitivity must also match to be a NoOp.
			// Theoretically marks may not match here, but sensitivity is the
			// only one we can act on, and the state will have been loaded
			// without any marks to consider.
			action = plans.NoOp
		}

		// Non-ephemeral output values get their changes recorded in the plan
		if !n.Config.Ephemeral {
			change := &plans.OutputChange{
				Addr:      n.Addr,
				Sensitive: sensitiveChange,
				Change: plans.Change{
					Action: action,
					Before: before,
					After:  val,
				},
			}

			log.Printf("[TRACE] setValue: Saving %s change for %s in changeset", change.Action, n.Addr)
			changes.AppendOutputChange(change) // add the new planned change
		}
	}

	if changes != nil && !n.Planning {
		// During apply there is no longer any change to track, so we must
		// ensure the state is updated and not overridden by a change.
		changes.RemoveOutputChange(n.Addr)
	}

	// Null outputs must be saved for modules so that they can still be
	// evaluated. Null root outputs are removed entirely, which is always fine
	// because they can't be referenced by anything else in the configuration.
	if n.Addr.Module.IsRoot() && val.IsNull() {
		log.Printf("[TRACE] setValue: Removing %s from state (it is now null)", n.Addr)
		state.RemoveOutputValue(n.Addr)
		return
	}

	// caller leaves namedVals nil if they've already called this function
	// with a different state, since we only have one namedVals regardless
	// of how many states are involved in an operation.
	if namedVals != nil {
		saveVal := val
		if n.Config.Ephemeral {
			// Downstream uses of this output value must propagate the
			// ephemerality.
			saveVal = saveVal.Mark(marks.Ephemeral)
		}
		namedVals.SetOutputValue(n.Addr, saveVal)
	}

	// Non-ephemeral output values get saved in the state too
	if !n.Config.Ephemeral {
		// The state itself doesn't represent unknown values, so we null them
		// out here and then we'll save the real unknown value in the planned
		// changeset, if we have one on this graph walk.
		log.Printf("[TRACE] setValue: Saving value for %s in state", n.Addr)
		// non-root outputs need to keep sensitive marks for evaluation, but are
		// not serialized.
		if n.Addr.Module.IsRoot() {
			val, _ = val.UnmarkDeep()
			if deferred.DependenciesDeferred(n.Dependencies) {
				// If the output is from deferred resources then we return a
				// simple null value representing that the value is really
				// unknown as the dependencies were not properly computed.
				val = cty.NullVal(val.Type())
			} else {
				val = cty.UnknownAsNull(val)
			}
		}
	}
	state.SetOutputValue(n.Addr, val, n.Config.Sensitive)
}
