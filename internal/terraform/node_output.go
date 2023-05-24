// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/lang"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/plans"
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
}

var (
	_ GraphNodeReferenceable     = (*nodeExpandOutput)(nil)
	_ GraphNodeReferencer        = (*nodeExpandOutput)(nil)
	_ GraphNodeReferenceOutside  = (*nodeExpandOutput)(nil)
	_ GraphNodeDynamicExpandable = (*nodeExpandOutput)(nil)
	_ graphNodeTemporaryValue    = (*nodeExpandOutput)(nil)
	_ graphNodeExpandsInstances  = (*nodeExpandOutput)(nil)
)

func (n *nodeExpandOutput) expandsInstances() {}

func (n *nodeExpandOutput) temporaryValue() bool {
	// non root outputs are temporary
	return !n.Module.IsRoot()
}

func (n *nodeExpandOutput) DynamicExpand(ctx EvalContext) (*Graph, error) {
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
	for _, module := range expander.ExpandModule(n.Module) {
		absAddr := n.Addr.Absolute(module)
		if checkableAddrs != nil {
			checkableAddrs.Add(absAddr)
		}

		// Find any recorded change for this output
		var change *plans.OutputChangeSrc
		var outputChanges []*plans.OutputChangeSrc
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

		case n.Destroying:
			// nothing is done here for non-root outputs
			continue

		default:
			node = &NodeApplyableOutput{
				Addr:         absAddr,
				Config:       n.Config,
				Change:       change,
				RefreshOnly:  n.RefreshOnly,
				DestroyApply: n.Destroying,
				Planning:     n.Planning,
			}
		}

		log.Printf("[TRACE] Expanding output: adding %s as %T", absAddr.String(), node)
		g.Add(node)
	}
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

// NodeApplyableOutput represents an output that is "applyable":
// it is ready to be applied.
type NodeApplyableOutput struct {
	Addr   addrs.AbsOutputValue
	Config *configs.Output // Config is the output in the config
	// If this is being evaluated during apply, we may have a change recorded already
	Change *plans.OutputChangeSrc

	// Refresh-only mode means that any failing output preconditions are
	// reported as warnings rather than errors
	RefreshOnly bool

	// DestroyApply indicates that we are applying a destroy plan, and do not
	// need to account for conditional blocks.
	DestroyApply bool

	Planning bool
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

	impRefs, _ := lang.ReferencesInExpr(c.Expr)
	expRefs, _ := lang.References(c.DependsOn)

	refs = append(refs, impRefs...)
	refs = append(refs, expRefs...)

	for _, check := range c.Preconditions {
		condRefs, _ := lang.ReferencesInExpr(check.Condition)
		refs = append(refs, condRefs...)
		errRefs, _ := lang.ReferencesInExpr(check.ErrorMessage)
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
		change, err := n.Change.Decode()
		diags = diags.Append(err)
		if err == nil {
			val = change.After
		}
	}

	// Checks are not evaluated during a destroy. The checks may fail, may not
	// be valid, or may not have been registered at all.
	if !n.DestroyApply {
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

	// handling the interpolation error
	if diags.HasErrors() {
		if flagWarnOutputErrors {
			log.Printf("[ERROR] Output interpolation %q failed: %s", n.Addr, diags.Err())
			// if we're continuing, make sure the output is included, and
			// marked as unknown. If the evaluator was able to find a type
			// for the value in spite of the error then we'll use it.
			n.setValue(state, changes, cty.UnknownVal(val.Type()))

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
	n.setValue(state, changes, val)

	// If we were able to evaluate a new value, we can update that in the
	// refreshed state as well.
	if state = ctx.RefreshState(); state != nil && val.IsWhollyKnown() {
		// we only need to update the state, do not pass in the changes again
		n.setValue(state, nil, val)
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
		if o, ok := mod.OutputValues[n.Addr.OutputValue.Name]; ok {
			sensitiveBefore = o.Sensitive
			before = o.Value
		} else {
			// If the output was not in state, a delete change would
			// be meaningless, so exit early.
			return nil

		}
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

		cs, err := change.Encode()
		if err != nil {
			// Should never happen, since we just constructed this right above
			panic(fmt.Sprintf("planned change for %s could not be encoded: %s", n.Addr, err))
		}
		log.Printf("[TRACE] NodeDestroyableOutput: Saving %s change for %s in changeset", change.Action, n.Addr)

		changes.RemoveOutputChange(n.Addr) // remove any existing planned change, if present
		changes.AppendOutputChange(cs)     // add the new planned change
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

func (n *NodeApplyableOutput) setValue(state *states.SyncState, changes *plans.ChangesSync, val cty.Value) {
	if changes != nil && n.Planning {
		// if this is a root module, try to get a before value from the state for
		// the diff
		sensitiveBefore := false
		before := cty.NullVal(cty.DynamicPseudoType)

		// is this output new to our state?
		newOutput := true

		mod := state.Module(n.Addr.Module)
		if n.Addr.Module.IsRoot() && mod != nil {
			for name, o := range mod.OutputValues {
				if name == n.Addr.OutputValue.Name {
					before = o.Value
					sensitiveBefore = o.Sensitive
					newOutput = false
					break
				}
			}
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

		change := &plans.OutputChange{
			Addr:      n.Addr,
			Sensitive: sensitiveChange,
			Change: plans.Change{
				Action: action,
				Before: before,
				After:  val,
			},
		}

		cs, err := change.Encode()
		if err != nil {
			// Should never happen, since we just constructed this right above
			panic(fmt.Sprintf("planned change for %s could not be encoded: %s", n.Addr, err))
		}
		log.Printf("[TRACE] setValue: Saving %s change for %s in changeset", change.Action, n.Addr)
		changes.AppendOutputChange(cs) // add the new planned change
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

	log.Printf("[TRACE] setValue: Saving value for %s in state", n.Addr)

	// non-root outputs need to keep sensitive marks for evaluation, but are
	// not serialized.
	if n.Addr.Module.IsRoot() {
		val, _ = val.UnmarkDeep()
		val = cty.UnknownAsNull(val)
	}

	state.SetOutputValue(n.Addr, val, n.Config.Sensitive)
}
