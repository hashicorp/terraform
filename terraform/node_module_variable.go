package terraform

import (
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/dag"
	"github.com/hashicorp/terraform/lang"
	"github.com/zclconf/go-cty/cty"
)

// NodeApplyableModuleVariable represents a module variable input during
// the apply step.
type NodeApplyableModuleVariable struct {
	Addr   addrs.AbsInputVariableInstance
	Config *configs.Variable // Config is the var in the config
	Expr   hcl.Expression    // Expr is the value expression given in the call
}

// Ensure that we are implementing all of the interfaces we think we are
// implementing.
var (
	_ GraphNodeSubPath          = (*NodeApplyableModuleVariable)(nil)
	_ RemovableIfNotTargeted    = (*NodeApplyableModuleVariable)(nil)
	_ GraphNodeReferenceOutside = (*NodeApplyableModuleVariable)(nil)
	_ GraphNodeReferenceable    = (*NodeApplyableModuleVariable)(nil)
	_ GraphNodeReferencer       = (*NodeApplyableModuleVariable)(nil)
	_ GraphNodeEvalable         = (*NodeApplyableModuleVariable)(nil)
	_ dag.GraphNodeDotter       = (*NodeApplyableModuleVariable)(nil)
)

func (n *NodeApplyableModuleVariable) Name() string {
	return n.Addr.String()
}

// GraphNodeSubPath
func (n *NodeApplyableModuleVariable) Path() addrs.ModuleInstance {
	// We execute in the parent scope (above our own module) because
	// expressions in our value are resolved in that context.
	return n.Addr.Module.Parent()
}

// RemovableIfNotTargeted
func (n *NodeApplyableModuleVariable) RemoveIfNotTargeted() bool {
	// We need to add this so that this node will be removed if
	// it isn't targeted or a dependency of a target.
	return true
}

// GraphNodeReferenceOutside implementation
func (n *NodeApplyableModuleVariable) ReferenceOutside() (selfPath, referencePath addrs.ModuleInstance) {

	// Module input variables have their value expressions defined in the
	// context of their calling (parent) module, and so references from
	// a node of this type should be resolved in the parent module instance.
	referencePath = n.Addr.Module.Parent()

	// Input variables are _referenced_ from their own module, though.
	selfPath = n.Addr.Module

	return // uses named return values
}

// GraphNodeReferenceable
func (n *NodeApplyableModuleVariable) ReferenceableAddrs() []addrs.Referenceable {
	return []addrs.Referenceable{n.Addr.Variable}
}

// GraphNodeReferencer
func (n *NodeApplyableModuleVariable) References() []*addrs.Reference {

	// If we have no value expression, we cannot depend on anything.
	if n.Expr == nil {
		return nil
	}

	// Variables in the root don't depend on anything, because their values
	// are gathered prior to the graph walk and recorded in the context.
	if len(n.Addr.Module) == 0 {
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

// GraphNodeEvalable
func (n *NodeApplyableModuleVariable) EvalTree() EvalNode {
	// If we have no value, do nothing
	if n.Expr == nil {
		return &EvalNoop{}
	}

	// Otherwise, interpolate the value of this variable and set it
	// within the variables mapping.
	vals := make(map[string]cty.Value)

	_, call := n.Addr.Module.CallInstance()

	return &EvalSequence{
		Nodes: []EvalNode{
			&EvalOpFilter{
				Ops: []walkOperation{walkRefresh, walkPlan, walkApply,
					walkDestroy, walkValidate},
				Node: &EvalModuleCallArgument{
					Addr:   n.Addr.Variable,
					Config: n.Config,
					Expr:   n.Expr,
					Values: vals,

					IgnoreDiagnostics: false,
				},
			},

			&EvalSetModuleCallArguments{
				Module: call,
				Values: vals,
			},
		},
	}
}

// dag.GraphNodeDotter impl.
func (n *NodeApplyableModuleVariable) DotNode(name string, opts *dag.DotOpts) *dag.DotNode {
	return &dag.DotNode{
		Name: name,
		Attrs: map[string]string{
			"label": n.Name(),
			"shape": "note",
		},
	}
}
