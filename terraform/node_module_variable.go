package terraform

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/dag"
	"github.com/hashicorp/terraform/lang"
	"github.com/zclconf/go-cty/cty"
)

// NodePlannableModuleVariable is the placeholder for an variable that has not yet had
// its module path expanded.
type NodePlannableModuleVariable struct {
	Addr   addrs.InputVariable
	Module addrs.Module
	Config *configs.Variable
	Expr   hcl.Expression
}

var (
	_ GraphNodeDynamicExpandable = (*NodePlannableModuleVariable)(nil)
	_ GraphNodeReferenceOutside  = (*NodePlannableModuleVariable)(nil)
	_ GraphNodeReferenceable     = (*NodePlannableModuleVariable)(nil)
	_ GraphNodeReferencer        = (*NodePlannableModuleVariable)(nil)
	_ GraphNodeSubPath           = (*NodePlannableModuleVariable)(nil)
	_ RemovableIfNotTargeted     = (*NodePlannableModuleVariable)(nil)
)

func (n *NodePlannableModuleVariable) DynamicExpand(ctx EvalContext) (*Graph, error) {
	var g Graph
	expander := ctx.InstanceExpander()
	for _, module := range expander.ExpandModule(ctx.Path().Module()) {
		o := &NodeApplyableModuleVariable{
			Addr:   n.Addr.Absolute(module),
			Config: n.Config,
			Expr:   n.Expr,
		}
		g.Add(o)
	}
	return &g, nil
}

func (n *NodePlannableModuleVariable) Name() string {
	return fmt.Sprintf("%s.%s", n.Module, n.Addr.String())
}

// GraphNodeSubPath
func (n *NodePlannableModuleVariable) Path() addrs.ModuleInstance {
	// Return an UnkeyedInstanceShim as our placeholder,
	// given that modules will be unexpanded at this point in the walk
	return n.Module.UnkeyedInstanceShim()
}

// GraphNodeModulePath
func (n *NodePlannableModuleVariable) ModulePath() addrs.Module {
	return n.Module
}

// GraphNodeReferencer
func (n *NodePlannableModuleVariable) References() []*addrs.Reference {

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
	refs, _ := lang.ReferencesInExpr(n.Expr)
	return refs
}

// GraphNodeReferenceOutside implementation
func (n *NodePlannableModuleVariable) ReferenceOutside() (selfPath, referencePath addrs.Module) {
	return n.Module, n.Module.Parent()
}

// GraphNodeReferenceable
func (n *NodePlannableModuleVariable) ReferenceableAddrs() []addrs.Referenceable {
	// FIXME: References for module variables probably need to be thought out a bit more
	// Otherwise, we can reference the output via the address itself, or the
	// module call
	_, call := n.Module.Call()
	return []addrs.Referenceable{n.Addr, call}
}

// RemovableIfNotTargeted
func (n *NodePlannableModuleVariable) RemoveIfNotTargeted() bool {
	return true
}

// GraphNodeTargetDownstream
func (n *NodePlannableModuleVariable) TargetDownstream(targetedDeps, untargetedDeps dag.Set) bool {
	return true
}

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

// GraphNodeModulePath
func (n *NodeApplyableModuleVariable) ModulePath() addrs.Module {
	return n.Addr.Module.Parent().Module()
}

// RemovableIfNotTargeted
func (n *NodeApplyableModuleVariable) RemoveIfNotTargeted() bool {
	// We need to add this so that this node will be removed if
	// it isn't targeted or a dependency of a target.
	return true
}

// GraphNodeReferenceOutside implementation
func (n *NodeApplyableModuleVariable) ReferenceOutside() (selfPath, referencePath addrs.Module) {

	// Module input variables have their value expressions defined in the
	// context of their calling (parent) module, and so references from
	// a node of this type should be resolved in the parent module instance.
	referencePath = n.Addr.Module.Parent().Module()

	// Input variables are _referenced_ from their own module, though.
	selfPath = n.Addr.Module.Module()

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

			&evalVariableValidations{
				Addr:   n.Addr,
				Config: n.Config,
				Expr:   n.Expr,

				IgnoreDiagnostics: false,
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
