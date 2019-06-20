package terraform

import (
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/dag"
	"github.com/hashicorp/terraform/lang"
)

// NodeOutput is an interface implemented by all graph nodes that represent
// a single output value.
type NodeOutput interface {
	outputAddr() addrs.AbsOutputValue
}

// NodeAbstractOutput represents an output without implying any particular
// operation.
//
// Embed this in nodes representing actions on outputs to get default
// implementations of various interfaces used during graph construction.
type NodeAbstractOutput struct {
	Addr   addrs.AbsOutputValue
	Config *configs.Output // Config is the output in the config
}

var (
	_ NodeOutput                = (*NodeAbstractOutput)(nil)
	_ GraphNodeSubPath          = (*NodeAbstractOutput)(nil)
	_ RemovableIfNotTargeted    = (*NodeAbstractOutput)(nil)
	_ GraphNodeTargetDownstream = (*NodeAbstractOutput)(nil)
	_ GraphNodeReferenceable    = (*NodeAbstractOutput)(nil)
	_ GraphNodeReferencer       = (*NodeAbstractOutput)(nil)
	_ GraphNodeReferenceOutside = (*NodeAbstractOutput)(nil)
	_ dag.GraphNodeDotter       = (*NodeAbstractOutput)(nil)
)

func (n *NodeAbstractOutput) outputAddr() addrs.AbsOutputValue {
	return n.Addr
}

func (n *NodeAbstractOutput) Name() string {
	if n.Config == nil {
		return n.Addr.String() + " (removed)"
	}
	return n.Addr.String()
}

// GraphNodeSubPath
func (n *NodeAbstractOutput) Path() addrs.ModuleInstance {
	return n.Addr.Module
}

// RemovableIfNotTargeted
func (n *NodeAbstractOutput) RemoveIfNotTargeted() bool {
	// We need to add this so that this node will be removed if
	// it isn't targeted or a dependency of a target.
	return true
}

// GraphNodeTargetDownstream
func (n *NodeAbstractOutput) TargetDownstream(targetedDeps, untargetedDeps *dag.Set) bool {
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
func (n *NodeAbstractOutput) ReferenceOutside() (selfPath, referencePath addrs.ModuleInstance) {
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
func (n *NodeAbstractOutput) ReferenceableAddrs() []addrs.Referenceable {
	return referenceableAddrsForOutput(n.Addr)
}

func referencesForOutput(c *configs.Output) []*addrs.Reference {
	if c == nil {
		return nil
	}
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
func (n *NodeAbstractOutput) References() []*addrs.Reference {
	return appendResourceDestroyReferences(referencesForOutput(n.Config))
}

// dag.GraphNodeDotter impl.
func (n *NodeAbstractOutput) DotNode(name string, opts *dag.DotOpts) *dag.DotNode {
	return &dag.DotNode{
		Name: name,
		Attrs: map[string]string{
			"label": n.Name(),
			"shape": "note",
		},
	}
}

// NodeRefreshableOutput is a graph node representing an output value that
// is to be refreshed.
type NodeRefreshableOutput struct {
	*NodeAbstractOutput
}

var (
	_ GraphNodeSubPath          = (*NodeRefreshableOutput)(nil)
	_ RemovableIfNotTargeted    = (*NodeRefreshableOutput)(nil)
	_ GraphNodeTargetDownstream = (*NodeRefreshableOutput)(nil)
	_ GraphNodeReferencer       = (*NodeRefreshableOutput)(nil)
	_ GraphNodeEvalable         = (*NodeRefreshableOutput)(nil)
	_ dag.GraphNodeDotter       = (*NodeRefreshableOutput)(nil)
)

// GraphNodeEvalable
func (n *NodeRefreshableOutput) EvalTree() EvalNode {
	return &EvalRefreshOutput{
		Addr:      n.Addr.OutputValue,
		Sensitive: n.Config.Sensitive,
		Expr:      n.Config.Expr,
	}
}

// NodeValidatableOutput is a graph node representing an output value that
// is to be validated.
type NodeValidatableOutput struct {
	*NodeAbstractOutput
}

var (
	_ GraphNodeSubPath          = (*NodeValidatableOutput)(nil)
	_ RemovableIfNotTargeted    = (*NodeValidatableOutput)(nil)
	_ GraphNodeTargetDownstream = (*NodeValidatableOutput)(nil)
	_ GraphNodeReferencer       = (*NodeValidatableOutput)(nil)
	_ GraphNodeEvalable         = (*NodeValidatableOutput)(nil)
	_ dag.GraphNodeDotter       = (*NodeValidatableOutput)(nil)
)

// GraphNodeEvalable
func (n *NodeValidatableOutput) EvalTree() EvalNode {
	return &EvalValidateOutput{
		Addr:   n.Addr.OutputValue,
		Config: n.Config,
	}
}

// NodePlannableOutput is a graph node representing an output value that
// should have a planned change computed for it.
type NodePlannableOutput struct {
	*NodeAbstractOutput
	ForceDestroy bool
}

var (
	_ GraphNodeSubPath          = (*NodePlannableOutput)(nil)
	_ RemovableIfNotTargeted    = (*NodePlannableOutput)(nil)
	_ GraphNodeTargetDownstream = (*NodePlannableOutput)(nil)
	_ GraphNodeReferencer       = (*NodePlannableOutput)(nil)
	_ GraphNodeEvalable         = (*NodePlannableOutput)(nil)
	_ dag.GraphNodeDotter       = (*NodePlannableOutput)(nil)
)

// GraphNodeEvalable
func (n *NodePlannableOutput) EvalTree() EvalNode {
	return &EvalPlanOutput{
		Addr:         n.Addr.OutputValue,
		Config:       n.Config,
		ForceDestroy: n.ForceDestroy,
	}
}

// NodeApplyableOutput is a graph node representing an output value that
// has a non-Delete change ready to apply.
type NodeApplyableOutput struct {
	*NodeAbstractOutput
}

var (
	_ GraphNodeSubPath          = (*NodeApplyableOutput)(nil)
	_ RemovableIfNotTargeted    = (*NodeApplyableOutput)(nil)
	_ GraphNodeTargetDownstream = (*NodeApplyableOutput)(nil)
	_ GraphNodeReferencer       = (*NodeApplyableOutput)(nil)
	_ GraphNodeEvalable         = (*NodeApplyableOutput)(nil)
	_ dag.GraphNodeDotter       = (*NodeApplyableOutput)(nil)
)

// GraphNodeEvalable
func (n *NodeApplyableOutput) EvalTree() EvalNode {
	return &EvalApplyOutput{
		Addr: n.Addr.OutputValue,
		Expr: n.Config.Expr,
	}
}

// NodeDestroyableOutput is a graph node representing an output value that
// has a Delete change ready to apply.
type NodeDestroyableOutput struct {
	*NodeAbstractOutput
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
	return n.Addr.String() + " (remove)"
}

// GraphNodeReferencer
func (n *NodeDestroyableOutput) References() []*addrs.Reference {
	// Destroying an output doesn't require evaluating its expression,
	// so we have no references at all in this case.
	return nil
}

// GraphNodeEvalable
func (n *NodeDestroyableOutput) EvalTree() EvalNode {
	// Uses the same EvalNode as EvalApplyableOutput; this separate node type
	// exists only to alter how a destroyable output participates in the
	// dependency graph.
	return &EvalApplyOutput{
		Addr: n.Addr.OutputValue,
		Expr: nil, // no configuration needed during destroy
	}
}
