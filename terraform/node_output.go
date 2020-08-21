package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/dag"
	"github.com/hashicorp/terraform/lang"
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
	_ GraphNodeEvalable         = (*NodeApplyableOutput)(nil)
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

// GraphNodeEvalable
func (n *NodeApplyableOutput) EvalTree() EvalNode {
	return &EvalSequence{
		Nodes: []EvalNode{
			&EvalOpFilter{
				Ops: []walkOperation{walkEval, walkRefresh, walkPlan, walkApply, walkValidate, walkDestroy, walkPlanDestroy},
				Node: &EvalWriteOutput{
					Addr:   n.Addr.OutputValue,
					Config: n.Config,
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
	_ GraphNodeEvalable   = (*NodeDestroyableOutput)(nil)
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
