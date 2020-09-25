package terraform

import (
	"log"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/dag"
	"github.com/hashicorp/terraform/lang"
	"github.com/hashicorp/terraform/tfdiags"
)

// nodeExpandApplyableResource handles the first layer of resource
// expansion during apply. Even though the resource instances themselves are
// already expanded from the plan, we still need to expand the
// NodeApplyableResource nodes into their respective modules.
type nodeExpandApplyableResource struct {
	*NodeAbstractResource
}

var (
	_ GraphNodeDynamicExpandable    = (*nodeExpandApplyableResource)(nil)
	_ GraphNodeReferenceable        = (*nodeExpandApplyableResource)(nil)
	_ GraphNodeReferencer           = (*nodeExpandApplyableResource)(nil)
	_ GraphNodeConfigResource       = (*nodeExpandApplyableResource)(nil)
	_ GraphNodeAttachResourceConfig = (*nodeExpandApplyableResource)(nil)
	_ graphNodeExpandsInstances     = (*nodeExpandApplyableResource)(nil)
	_ GraphNodeTargetable           = (*nodeExpandApplyableResource)(nil)
)

func (n *nodeExpandApplyableResource) expandsInstances() {}

func (n *nodeExpandApplyableResource) References() []*addrs.Reference {
	return (&NodeApplyableResource{NodeAbstractResource: n.NodeAbstractResource}).References()
}

func (n *nodeExpandApplyableResource) Name() string {
	return n.NodeAbstractResource.Name() + " (expand)"
}

func (n *nodeExpandApplyableResource) DynamicExpand(ctx EvalContext) (*Graph, error) {
	var g Graph

	expander := ctx.InstanceExpander()
	moduleInstances := expander.ExpandModule(n.Addr.Module)
	var resources []addrs.AbsResource
	for _, module := range moduleInstances {
		resAddr := n.Addr.Resource.Absolute(module)
		resources = append(resources, resAddr)
		g.Add(&NodeApplyableResource{
			NodeAbstractResource: n.NodeAbstractResource,
			Addr:                 n.Addr.Resource.Absolute(module),
		})
	}

	return &g, nil
}

// NodeApplyableResource represents a resource that is "applyable":
// it may need to have its record in the state adjusted to match configuration.
//
// Unlike in the plan walk, this resource node does not DynamicExpand. Instead,
// it should be inserted into the same graph as any instances of the nodes
// with dependency edges ensuring that the resource is evaluated before any
// of its instances, which will turn ensure that the whole-resource record
// in the state is suitably prepared to receive any updates to instances.
type NodeApplyableResource struct {
	*NodeAbstractResource

	Addr addrs.AbsResource
}

var (
	_ GraphNodeModuleInstance       = (*NodeApplyableResource)(nil)
	_ GraphNodeConfigResource       = (*NodeApplyableResource)(nil)
	_ GraphNodeExecutable           = (*NodeApplyableResource)(nil)
	_ GraphNodeProviderConsumer     = (*NodeApplyableResource)(nil)
	_ GraphNodeAttachResourceConfig = (*NodeApplyableResource)(nil)
	_ GraphNodeReferencer           = (*NodeApplyableResource)(nil)
)

func (n *NodeApplyableResource) Path() addrs.ModuleInstance {
	return n.Addr.Module
}

func (n *NodeApplyableResource) References() []*addrs.Reference {
	if n.Config == nil {
		log.Printf("[WARN] NodeApplyableResource %q: no configuration, so can't determine References", dag.VertexName(n))
		return nil
	}

	var result []*addrs.Reference

	// Since this node type only updates resource-level metadata, we only
	// need to worry about the parts of the configuration that affect
	// our "each mode": the count and for_each meta-arguments.
	refs, _ := lang.ReferencesInExpr(n.Config.Count)
	result = append(result, refs...)
	refs, _ = lang.ReferencesInExpr(n.Config.ForEach)
	result = append(result, refs...)

	return result
}

// GraphNodeExecutable
func (n *NodeApplyableResource) Execute(ctx EvalContext, op walkOperation) error {
	if n.Config == nil {
		// Nothing to do, then.
		log.Printf("[TRACE] NodeApplyableResource: no configuration present for %s", n.Name())
		return nil
	}

	var diags tfdiags.Diagnostics
	state := ctx.State()

	// We'll record our expansion decision in the shared "expander" object
	// so that later operations (i.e. DynamicExpand and expression evaluation)
	// can refer to it. Since this node represents the abstract module, we need
	// to expand the module here to create all resources.
	expander := ctx.InstanceExpander()

	switch {
	case n.Config.Count != nil:
		count, countDiags := evaluateCountExpression(n.Config.Count, ctx)
		diags = diags.Append(countDiags)
		if countDiags.HasErrors() {
			return diags.Err()
		}

		state.SetResourceProvider(n.Addr, n.ResolvedProvider)
		expander.SetResourceCount(n.Addr.Module, n.Addr.Resource, count)

	case n.Config.ForEach != nil:
		forEach, forEachDiags := evaluateForEachExpression(n.Config.ForEach, ctx)
		diags = diags.Append(forEachDiags)
		if forEachDiags.HasErrors() {
			return diags.Err()
		}

		// This method takes care of all of the business logic of updating this
		// while ensuring that any existing instances are preserved, etc.
		state.SetResourceProvider(n.Addr, n.ResolvedProvider)
		expander.SetResourceForEach(n.Addr.Module, n.Addr.Resource, forEach)

	default:
		state.SetResourceProvider(n.Addr, n.ResolvedProvider)
		expander.SetResourceSingle(n.Addr.Module, n.Addr.Resource)
	}
	return nil
}
