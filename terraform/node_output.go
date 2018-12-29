package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/plans"

	"github.com/hashicorp/terraform/states"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/dag"
	"github.com/hashicorp/terraform/lang"
)

// NodePlannableOutput represents an output that is "plannable":
// we want to check if the output value has changed and record that change
// in the plan if so.
//
// If the node has no attached configuration, the planned change will be to
// remove the output altogether.
type NodePlannableOutput struct {
	Addr   addrs.AbsOutputValue
	Config *configs.Output // Config is the output in the config
}

var (
	_ GraphNodeSubPath          = (*NodePlannableOutput)(nil)
	_ RemovableIfNotTargeted    = (*NodePlannableOutput)(nil)
	_ GraphNodeTargetDownstream = (*NodePlannableOutput)(nil)
	_ GraphNodeReferenceable    = (*NodePlannableOutput)(nil)
	_ GraphNodeReferencer       = (*NodePlannableOutput)(nil)
	_ GraphNodeReferenceOutside = (*NodePlannableOutput)(nil)
	_ GraphNodeEvalable         = (*NodePlannableOutput)(nil)
	_ dag.GraphNodeDotter       = (*NodePlannableOutput)(nil)
)

// NewOutputPlanNode constructs a new graph node that will make a plan for
// an output with the given address and configuration.
//
// The configuration may be nil, in which case the node will plan to remove
// the output from the state altogether.
func NewOutputPlanNode(addr addrs.AbsOutputValue, cfg *configs.Output) dag.Vertex {
	return &NodePlannableOutput{
		Addr:   addr,
		Config: cfg,
	}
}

func (n *NodePlannableOutput) Name() string {
	return n.Addr.String()
}

// GraphNodeSubPath
func (n *NodePlannableOutput) Path() addrs.ModuleInstance {
	return n.Addr.Module
}

// RemovableIfNotTargeted
func (n *NodePlannableOutput) RemoveIfNotTargeted() bool {
	// We need to add this so that this node will be removed if
	// it isn't targeted or a dependency of a target.
	return true
}

// GraphNodeTargetDownstream
func (n *NodePlannableOutput) TargetDownstream(targetedDeps, untargetedDeps *dag.Set) bool {
	// If any of the direct dependencies of an output are targeted then
	// the output must always be targeted as well, so its value will always
	// be up-to-date at the completion of an apply walk.
	return true
}

// GraphNodeReferenceOutside implementation
func (n *NodePlannableOutput) ReferenceOutside() (selfPath, referencePath addrs.ModuleInstance) {
	return referenceOutsideForOutput(n.Addr)
}

// GraphNodeReferenceable
func (n *NodePlannableOutput) ReferenceableAddrs() []addrs.Referenceable {
	return referenceableAddrsForOutput(n.Addr)
}

// GraphNodeReferencer
func (n *NodePlannableOutput) References() []*addrs.Reference {
	return appendResourceDestroyReferences(referencesForOutput(n.Config))
}

// GraphNodeEvalable
func (n *NodePlannableOutput) EvalTree() EvalNode {
	var state *states.OutputValue
	var change *plans.OutputChange

	return &EvalSequence{
		Nodes: []EvalNode{
			&EvalReadOutputState{
				Addr:   n.Addr.OutputValue,
				Output: &state,
			},
			&EvalIf{
				If: func(EvalContext) (bool, error) {
					if state == nil && n.Config == nil {
						// Nothing to do, then. (Shouldn't have created a graph node at all.)
						return false, EvalEarlyExitError{}
					}
					return n.Config != nil, nil
				},
				Then: &EvalPlanOutputChange{
					Addr:       n.Addr.OutputValue,
					Config:     n.Config,
					PriorState: &state,
					Output:     &change,
				},
				Else: &EvalPlanOutputDestroy{
					Addr:       n.Addr.OutputValue,
					PriorState: &state,
					Output:     &change,
				},
			},
			&EvalWriteOutputChange{
				Change: &change,
			},
		},
	}
}

// dag.GraphNodeDotter impl.
func (n *NodePlannableOutput) DotNode(name string, opts *dag.DotOpts) *dag.DotNode {
	return &dag.DotNode{
		Name: name,
		Attrs: map[string]string{
			"label": n.Name(),
			"shape": "note",
		},
	}
}

// NewOutputApplyNode constructs a new graph node that will update the state
// for an output with the given address and configuration.
//
// The configuration may be nil, in which case the node will remove the output
// from the state altogether.
func NewOutputApplyNode(addr addrs.AbsOutputValue, cfg *configs.Output) dag.Vertex {
	if cfg == nil {
		return &NodeDestroyableOutput{
			Addr: addr,
		}
	}
	return &NodeApplyableOutput{
		Addr:   addr,
		Config: cfg,
	}
}

// NodeApplyableOutput represents an output that is "applyable":
// it needs its state value updated to reflect its configuration.
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
func (n *NodeApplyableOutput) TargetDownstream(targetedDeps, untargetedDeps *dag.Set) bool {
	// If any of the direct dependencies of an output are targeted then
	// the output must always be targeted as well, so its value will always
	// be up-to-date at the completion of an apply walk.
	return true
}

func referenceOutsideForOutput(addr addrs.AbsOutputValue) (selfPath, referencePath addrs.ModuleInstance) {

	// Output values have their expressions resolved in the context of the
	// module where they are defined.
	referencePath = addr.Module

	// ...but they are referenced in the context of their calling module.
	selfPath = addr.Module.Parent()

	return // uses named return values

}

// GraphNodeReferenceOutside implementation
func (n *NodeApplyableOutput) ReferenceOutside() (selfPath, referencePath addrs.ModuleInstance) {
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
	var state *states.OutputValue

	return &EvalSequence{
		Nodes: []EvalNode{
			&EvalOutput{
				Addr:   n.Addr.OutputValue,
				Config: n.Config,
				Output: &state,
			},
			&EvalWriteOutputState{
				Addr:  n.Addr.OutputValue,
				State: &state,
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

// NodeDestroyableOutput represents an output that is "destroyable":
// evaluating it will remove the output from the state altogether.
type NodeDestroyableOutput struct {
	Addr addrs.AbsOutputValue
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
	return n.Addr.Module
}

// RemovableIfNotTargeted
func (n *NodeDestroyableOutput) RemoveIfNotTargeted() bool {
	// We need to add this so that this node will be removed if
	// it isn't targeted or a dependency of a target.
	return true
}

// This will keep the destroy node in the graph if its corresponding output
// node is also in the destroy graph.
func (n *NodeDestroyableOutput) TargetDownstream(targetedDeps, untargetedDeps *dag.Set) bool {
	return true
}

// GraphNodeReferencer
func (n *NodeDestroyableOutput) References() []*addrs.Reference {
	return nil
}

// GraphNodeEvalable
func (n *NodeDestroyableOutput) EvalTree() EvalNode {
	return &EvalDeleteOutput{
		Addr: n.Addr.OutputValue,
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
