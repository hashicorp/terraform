package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/dag"
	"github.com/hashicorp/terraform/lang"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/states"
	"github.com/zclconf/go-cty/cty"
)

// nodeExpandOutput is the placeholder for an output that has not yet had
// its module path expanded.
type nodeExpandOutput struct {
	Addr   addrs.OutputValue
	Module addrs.Module
	Config *configs.Output
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
	// this must always be evaluated if it is a root module output
	return !n.Module.IsRoot()
}

func (n *nodeExpandOutput) DynamicExpand(ctx EvalContext) (*Graph, error) {
	var g Graph
	expander := ctx.InstanceExpander()
	for _, module := range expander.ExpandModule(n.Module) {
		o := &NodeApplyableOutput{
			Addr:   n.Addr.Absolute(module),
			Config: n.Config,
		}
		log.Printf("[TRACE] Expanding output: adding %s as %T", o.Addr.String(), o)
		g.Add(o)
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
	return referencesForOutput(n.Config)
}

// NodeApplyableOutput represents an output that is "applyable":
// it is ready to be applied.
type NodeApplyableOutput struct {
	Addr   addrs.AbsOutputValue
	Config *configs.Output // Config is the output in the config
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
	impRefs, _ := lang.ReferencesInExpr(c.Expr)
	expRefs, _ := lang.References(c.DependsOn)
	l := len(impRefs) + len(expRefs)
	if l == 0 {
		return nil
	}
	refs := make([]*addrs.Reference, 0, l)
	refs = append(refs, impRefs...)
	refs = append(refs, expRefs...)
	return refs

}

// GraphNodeReferencer
func (n *NodeApplyableOutput) References() []*addrs.Reference {
	return referencesForOutput(n.Config)
}

// GraphNodeExecutable
func (n *NodeApplyableOutput) Execute(ctx EvalContext, op walkOperation) error {
	// This has to run before we have a state lock, since evaluation also
	// reads the state
	val, diags := ctx.EvaluateExpr(n.Config.Expr, cty.DynamicPseudoType, nil)
	// We'll handle errors below, after we have loaded the module.

	// Outputs don't have a separate mode for validation, so validate
	// depends_on expressions here too
	diags = diags.Append(validateDependsOn(ctx, n.Config.DependsOn))

	// Ensure that non-sensitive outputs don't include sensitive values
	_, marks := val.UnmarkDeep()
	_, hasSensitive := marks["sensitive"]
	if !n.Config.Sensitive && hasSensitive {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Output refers to sensitive values",
			Detail:   "Expressions used in outputs can only refer to sensitive values if the sensitive attribute is true.",
			Subject:  n.Config.DeclRange.Ptr(),
		})
	}

	state := ctx.State()
	if state == nil {
		return nil
	}

	changes := ctx.Changes() // may be nil, if we're not working on a changeset

	// handling the interpolation error
	if diags.HasErrors() {
		if flagWarnOutputErrors {
			log.Printf("[ERROR] Output interpolation %q failed: %s", n.Addr, diags.Err())
			// if we're continuing, make sure the output is included, and
			// marked as unknown. If the evaluator was able to find a type
			// for the value in spite of the error then we'll use it.
			n.setValue(state, changes, cty.UnknownVal(val.Type()))
			return EvalEarlyExitError{}
		}
		return diags.Err()
	}
	n.setValue(state, changes, val)

	// If we were able to evaluate a new value, we can update that in the
	// refreshed state as well.
	if state = ctx.RefreshState(); state != nil && val.IsWhollyKnown() {
		n.setValue(state, changes, val)
	}

	return nil
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

// NodeDestroyableOutput represents an output that is "destroybale":
// its application will remove the output from the state.
type NodeDestroyableOutput struct {
	Addr   addrs.AbsOutputValue
	Config *configs.Output // Config is the output in the config
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
func (n *NodeDestroyableOutput) Execute(ctx EvalContext, op walkOperation) error {
	state := ctx.State()
	if state == nil {
		return nil
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
	if val.IsKnown() && !val.IsNull() {
		// The state itself doesn't represent unknown values, so we null them
		// out here and then we'll save the real unknown value in the planned
		// changeset below, if we have one on this graph walk.
		log.Printf("[TRACE] EvalWriteOutput: Saving value for %s in state", n.Addr)
		unmarkedVal, _ := val.UnmarkDeep()
		stateVal := cty.UnknownAsNull(unmarkedVal)
		state.SetOutputValue(n.Addr, stateVal, n.Config.Sensitive)
	} else {
		log.Printf("[TRACE] EvalWriteOutput: Removing %s from state (it is now null)", n.Addr)
		state.RemoveOutputValue(n.Addr)
	}

	// If we also have an active changeset then we'll replicate the value in
	// there. This is used in preference to the state where present, since it
	// *is* able to represent unknowns, while the state cannot.
	if changes != nil {
		// For the moment we are not properly tracking changes to output
		// values, and just marking them always as "Create" or "Destroy"
		// actions. A future release will rework the output lifecycle so we
		// can track their changes properly, in a similar way to how we work
		// with resource instances.

		var change *plans.OutputChange
		if !val.IsNull() {
			change = &plans.OutputChange{
				Addr:      n.Addr,
				Sensitive: n.Config.Sensitive,
				Change: plans.Change{
					Action: plans.Create,
					Before: cty.NullVal(cty.DynamicPseudoType),
					After:  val,
				},
			}
		} else {
			change = &plans.OutputChange{
				Addr:      n.Addr,
				Sensitive: n.Config.Sensitive,
				Change: plans.Change{
					// This is just a weird placeholder delete action since
					// we don't have an actual prior value to indicate.
					// FIXME: Generate real planned changes for output values
					// that include the old values.
					Action: plans.Delete,
					Before: cty.NullVal(cty.DynamicPseudoType),
					After:  cty.NullVal(cty.DynamicPseudoType),
				},
			}
		}

		cs, err := change.Encode()
		if err != nil {
			// Should never happen, since we just constructed this right above
			panic(fmt.Sprintf("planned change for %s could not be encoded: %s", n.Addr, err))
		}
		log.Printf("[TRACE] ExecuteWriteOutput: Saving %s change for %s in changeset", change.Action, n.Addr)
		changes.RemoveOutputChange(n.Addr) // remove any existing planned change, if present
		changes.AppendOutputChange(cs)     // add the new planned change
	}
}
