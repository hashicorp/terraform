package terraform

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// NodeRootVariable represents a root variable input.
type NodeRootVariable struct {
	Addr   addrs.InputVariable
	Config *configs.Variable
}

var (
	_ GraphNodeModuleInstance = (*NodeRootVariable)(nil)
	_ GraphNodeReferenceable  = (*NodeRootVariable)(nil)
)

func (n *NodeRootVariable) Name() string {
	return n.Addr.String()
}

// GraphNodeModuleInstance
func (n *NodeRootVariable) Path() addrs.ModuleInstance {
	return addrs.RootModuleInstance
}

func (n *NodeRootVariable) ModulePath() addrs.Module {
	return addrs.RootModule
}

// GraphNodeReferenceable
func (n *NodeRootVariable) ReferenceableAddrs() []addrs.Referenceable {
	return []addrs.Referenceable{n.Addr}
}

// GraphNodeExecutable
func (n *NodeRootVariable) Execute(ctx EvalContext, op walkOperation) tfdiags.Diagnostics {
	// We don't actually need to _evaluate_ a root module variable, because
	// its value is always constant and already stashed away in our EvalContext.
	// However, we might need to run some user-defined validation rules against
	// the value.

	if n.Config == nil || len(n.Config.Validations) == 0 {
		return nil // nothing to do
	}

	return evalVariableValidations(
		addrs.RootModuleInstance.InputVariable(n.Addr.Name),
		n.Config,
		nil, // not set for root module variables
		ctx,
	)
}

// dag.GraphNodeDotter impl.
func (n *NodeRootVariable) DotNode(name string, opts *dag.DotOpts) *dag.DotNode {
	return &dag.DotNode{
		Name: name,
		Attrs: map[string]string{
			"label": n.Name(),
			"shape": "note",
		},
	}
}
