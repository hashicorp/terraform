package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/dag"
	"github.com/hashicorp/terraform/lang"
)

// NodePlannableOutput is the placeholder for an output that has not yet had
// its module path expanded.
type NodePlannableOutput struct {
	Addr   addrs.OutputValue
	Module addrs.Module
	Config *configs.Output
}

var (
	_ GraphNodeSubPath       = (*NodePlannableOutput)(nil)
	_ RemovableIfNotTargeted = (*NodePlannableOutput)(nil)
	_ GraphNodeReferenceable = (*NodePlannableOutput)(nil)
	//_ GraphNodeEvalable          = (*NodePlannableOutput)(nil)
	_ GraphNodeReferencer        = (*NodePlannableOutput)(nil)
	_ GraphNodeDynamicExpandable = (*NodePlannableOutput)(nil)
)

func (n *NodePlannableOutput) DynamicExpand(ctx EvalContext) (*Graph, error) {
	var g Graph
	expander := ctx.InstanceExpander()
	for _, module := range expander.ExpandModule(ctx.Path().Module()) {
		o := &NodeApplyableOutput{
			Addr:   n.Addr.Absolute(module),
			Config: n.Config,
		}
		// log.Printf("[TRACE] Expanding output: adding %s as %T", o.Addr.String(), o)
		g.Add(o)
	}
	return &g, nil
}

func (n *NodePlannableOutput) Name() string {
	return n.Addr.Absolute(n.Module.UnkeyedInstanceShim()).String()
}

// GraphNodeSubPath
func (n *NodePlannableOutput) Path() addrs.ModuleInstance {
	// Return an UnkeyedInstanceShim as our placeholder,
	// given that modules will be unexpanded at this point in the walk
	return n.Module.UnkeyedInstanceShim()
}

// GraphNodeReferenceable
func (n *NodePlannableOutput) ReferenceableAddrs() []addrs.Referenceable {
	// An output in the root module can't be referenced at all.
	if n.Module.IsRoot() {
		return nil
	}

	// the output is referenced through the module call, and via the
	// module itself.
	_, call := n.Module.Call()

	// FIXME: make something like ModuleCallOutput for this type of reference
	// that doesn't need an instance shim
	callOutput := addrs.ModuleCallOutput{
		Call: call.Instance(addrs.NoKey),
		Name: n.Addr.Name,
	}

	// Otherwise, we can reference the output via the
	// module call itself
	return []addrs.Referenceable{call, callOutput}
}

// GraphNodeReferenceOutside implementation
func (n *NodePlannableOutput) ReferenceOutside() (selfPath, referencePath addrs.Module) {
	// Output values have their expressions resolved in the context of the
	// module where they are defined.
	referencePath = n.Module

	// ...but they are referenced in the context of their calling module.
	selfPath = referencePath.Parent()

	return // uses named return values
}

// GraphNodeReferencer
func (n *NodePlannableOutput) References() []*addrs.Reference {
	return appendResourceDestroyReferences(referencesForOutput(n.Config))
}

// RemovableIfNotTargeted
func (n *NodePlannableOutput) RemoveIfNotTargeted() bool {
	return true
}

// GraphNodeTargetDownstream
func (n *NodePlannableOutput) TargetDownstream(targetedDeps, untargetedDeps dag.Set) bool {
	return true
}

// NodeApplyableOutput represents an output that is "applyable":
// it is ready to be applied.
type NodeApplyableOutput struct {
	Addr   addrs.AbsOutputValue
	Config *configs.Output // Config is the output in the config
}

var (
	_ GraphNodeSubPath          = (*NodeApplyableOutput)(nil)
	_ RemovableIfNotTargeted    = (*NodeApplyableOutput)(nil)
	_ GraphNodeTargetDownstream = (*NodeApplyableOutput)(nil)
	_ GraphNodeReferenceable    = (*NodeApplyableOutput)(nil)
	_ GraphNodeReferencer       = (*NodeApplyableOutput)(nil)
	_ GraphNodeReferenceOutside = (*NodeApplyableOutput)(nil)
	_ GraphNodeEvalable         = (*NodeApplyableOutput)(nil)
	_ dag.GraphNodeDotter       = (*NodeApplyableOutput)(nil)
)

func (n *NodeApplyableOutput) Name() string {
	return n.Addr.String()
}

// GraphNodeSubPath
func (n *NodeApplyableOutput) Path() addrs.ModuleInstance {
	return n.Addr.Module
}

// RemovableIfNotTargeted
func (n *NodeApplyableOutput) RemoveIfNotTargeted() bool {
	// We need to add this so that this node will be removed if
	// it isn't targeted or a dependency of a target.
	return true
}

// GraphNodeTargetDownstream
func (n *NodeApplyableOutput) TargetDownstream(targetedDeps, untargetedDeps dag.Set) bool {
	// If any of the direct dependencies of an output are targeted then
	// the output must always be targeted as well, so its value will always
	// be up-to-date at the completion of an apply walk.
	return true
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
	return appendResourceDestroyReferences(referencesForOutput(n.Config))
}

// GraphNodeEvalable
func (n *NodeApplyableOutput) EvalTree() EvalNode {
	return &EvalSequence{
		Nodes: []EvalNode{
			&EvalOpFilter{
				Ops: []walkOperation{walkRefresh, walkPlan, walkApply, walkValidate, walkDestroy, walkPlanDestroy},
				Node: &EvalWriteOutput{
					Addr:      n.Addr.OutputValue,
					Sensitive: n.Config.Sensitive,
					Expr:      n.Config.Expr,
				},
			},
		},
	}
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
	Addr   addrs.OutputValue
	Module addrs.Module
	Config *configs.Output // Config is the output in the config
}

var (
	_ GraphNodeSubPath          = (*NodeDestroyableOutput)(nil)
	_ RemovableIfNotTargeted    = (*NodeDestroyableOutput)(nil)
	_ GraphNodeTargetDownstream = (*NodeDestroyableOutput)(nil)
	_ GraphNodeReferencer       = (*NodeDestroyableOutput)(nil)
	_ GraphNodeEvalable         = (*NodeDestroyableOutput)(nil)
	_ dag.GraphNodeDotter       = (*NodeDestroyableOutput)(nil)
)

func (n *NodeDestroyableOutput) Name() string {
	return fmt.Sprintf("%s (destroy)", n.Addr.String())
}

// GraphNodeSubPath
func (n *NodeDestroyableOutput) Path() addrs.ModuleInstance {
	return n.Module.UnkeyedInstanceShim()
}

// RemovableIfNotTargeted
func (n *NodeDestroyableOutput) RemoveIfNotTargeted() bool {
	// We need to add this so that this node will be removed if
	// it isn't targeted or a dependency of a target.
	return true
}

// This will keep the destroy node in the graph if its corresponding output
// node is also in the destroy graph.
func (n *NodeDestroyableOutput) TargetDownstream(targetedDeps, untargetedDeps dag.Set) bool {
	return true
}

// GraphNodeReferencer
func (n *NodeDestroyableOutput) References() []*addrs.Reference {
	return referencesForOutput(n.Config)
}

// GraphNodeEvalable
func (n *NodeDestroyableOutput) EvalTree() EvalNode {
	return &EvalDeleteOutput{
		Addr: n.Addr,
	}
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
