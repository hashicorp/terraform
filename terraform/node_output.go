package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/dag"
	"github.com/hashicorp/terraform/lang"
)

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
	return &EvalSequence{
		Nodes: []EvalNode{
			&EvalOpFilter{
				// Don't let interpolation errors stop Input, since it happens
				// before Refresh.
				Ops: []walkOperation{walkInput},
				Node: &EvalWriteOutput{
					Addr:          n.Addr.OutputValue,
					Sensitive:     n.Config.Sensitive,
					Expr:          n.Config.Expr,
					ContinueOnErr: true,
				},
			},
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
	Addr   addrs.AbsOutputValue
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
	return referencesForOutput(n.Config)
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
