package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
)

// nodeExpandMoved is the placeholder for a moved block that hasn't yet
// had its module path expanded to take into account modules with count
// or for_each.
type nodeExpandMoved struct {
	Module addrs.Module
	Config *configs.Moved

	// AllConfig is the config object representing the root
	// module in the module tree. We need this so that we
	// can find the objects that the moved block refers to,
	// if present.
	AllConfig *configs.Config

	// Concrete is a callback function to optionally transform
	// the nodeMovedAbstract object we'd produce by default
	// into some other node type during expansion.
	Concrete func(abstract *nodeMovedAbstract) *dag.Vertex
}

var (
	_ GraphNodeDynamicExpandable = (*nodeExpandMoved)(nil)
	_ GraphNodeReferencer        = (*nodeExpandMoved)(nil)
	_ GraphNodeReferenceOutside  = (*nodeExpandMoved)(nil)
	_ graphNodeExpandsInstances  = (*nodeExpandMoved)(nil)
)

func (n *nodeExpandMoved) Name() string {
	return fmt.Sprintf(
		"%s: moved %s -> %s (expand)",
		n.Module,
		n.Config.From.Subject.String(),
		n.Config.To.Subject.String(),
	)
}

func (n *nodeExpandMoved) ModulePath() addrs.Module {
	return n.Module
}

func (n *nodeExpandMoved) References() []*addrs.Reference {
	// Validation of a move statement involves evaluating the
	// "count" or "for_each" expression of the resource
	// at the from address, if it's still present. This
	// is architecturally a bit awkward because this node
	// type is ostensibly a generic one representing a
	// moved block with no assumed behavior, but since we
	// have no use-case for these other than the validation
	// we'll just accept this assuming that we're going to
	// ultimately end up making nodeMovedValidate objects.
	modCfg := n.AllConfig.Descendent(n.Module)
	if modCfg == nil {
		// This doesn't make a whole lot of sense -- how
		// can a moved block exist if its containing module
		// doesn't?
		panic(fmt.Sprintf("moved block in unconfigured module %s", n.Module))
	}

}

func (n *nodeExpandMoved) ReferenceOutside() (selfPath, referencePath addrs.Module) {
	// The addresses returned by method References are in the
	// context of whatever module the "from" object is in,
	// because they are actually the for_each and count
	// references for that resource.
	return n.Module, n.Module.Parent()
}

func (n *nodeExpandMoved) expandsInstances() {}

func (n *nodeExpandMoved) DynamicExpand(ctx EvalContext) (*Graph, error) {
	var g Graph
	expander := ctx.InstanceExpander()
	for _, module := range expander.ExpandModule(n.Module) {
		an := &nodeMovedAbstract{
			Module: module,
			Config: n.Config,
		}
		var en dag.Vertex
		if n.Concrete != nil {
			en = n.Concrete(an)
		} else {
			en = an
		}
		g.Add(en)
	}
	return &g, nil
}

// nodeMovedAbstract is the generic node type for nodes representing
// "moved" blocks in the configuration, after expansion to take into
// account parent module instances.
type nodeMovedAbstract struct {
	Module addrs.ModuleInstance
	Config *configs.Moved
}

func (n *nodeMovedAbstract) Name() string {
	return fmt.Sprintf(
		"%s: moved %s -> %s",
		n.Module,
		n.Config.From.Subject.String(),
		n.Config.To.Subject.String(),
	)
}

// nodeMovedValidate is a specialization of nodeMovedAbstract which
// deals with the validation rules for a particular "moved" block in
// isolation.
//
// Although this node is called "validate", it can actually achieve
// full validation only during the plan walk, because it needs to
// consider the final expanded set of instance keys for any module
// or resource that is mentioned in the move statement.
//
// It can't consider the validation rules which relate to interactions
// _between_ moved blocks; those are dealt with as a preprocessing
// step while creating a terraform.Context, and so both must
// happen in order to achieve full validation.
type nodeMovedValidate struct {
	nodeMovedAbstract
}

var (
	_ GraphNodeExecutable = (*nodeMovedValidate)(nil)
)
