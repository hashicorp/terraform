package terraform

import (
	"log"

	"github.com/hashicorp/terraform/config/configschema"
	"github.com/hashicorp/terraform/dag"
	"github.com/hashicorp/terraform/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

// NodeValidatableResource represents a resource that is used for validation
// only.
type NodeValidatableResource struct {
	*NodeAbstractResource
}

var (
	_ GraphNodeSubPath              = (*NodeValidatableResource)(nil)
	_ GraphNodeDynamicExpandable    = (*NodeValidatableResource)(nil)
	_ GraphNodeReferenceable        = (*NodeValidatableResource)(nil)
	_ GraphNodeReferencer           = (*NodeValidatableResource)(nil)
	_ GraphNodeResource             = (*NodeValidatableResource)(nil)
	_ GraphNodeAttachResourceConfig = (*NodeValidatableResource)(nil)
)

// GraphNodeDynamicExpandable
func (n *NodeValidatableResource) DynamicExpand(ctx EvalContext) (*Graph, error) {
	var diags tfdiags.Diagnostics

	count, countDiags := evaluateResourceCountExpression(n.Config.Count, ctx)
	diags = diags.Append(countDiags)
	if countDiags.HasErrors() {
		log.Printf("[TRACE] %T %s: count expression has errors", n, n.Name())
		return nil, diags.Err()
	}
	if count >= 0 {
		log.Printf("[TRACE] %T %s: count expression evaluates to %d", n, n.Name(), count)
	} else {
		log.Printf("[TRACE] %T %s: no count argument present", n, n.Name())
	}

	// Next we need to potentially rename an instance address in the state
	// if we're transitioning whether "count" is set at all.
	fixResourceCountSetTransition(ctx, n.ResourceAddr().Resource, count != -1)

	// Grab the state which we read
	state, lock := ctx.State()
	lock.RLock()
	defer lock.RUnlock()

	// The concrete resource factory we'll use
	concreteResource := func(a *NodeAbstractResourceInstance) dag.Vertex {
		// Add the config and state since we don't do that via transforms
		a.Config = n.Config
		a.ResolvedProvider = n.ResolvedProvider

		return &NodeValidatableResourceInstance{
			NodeAbstractResourceInstance: a,
		}
	}

	// Start creating the steps
	steps := []GraphTransformer{
		// Expand the count.
		&ResourceCountTransformer{
			Concrete: concreteResource,
			Schema:   n.Schema,
			Count:    count,
			Addr:     n.ResourceAddr(),
		},

		// Attach the state
		&AttachStateTransformer{State: state},

		// Targeting
		&TargetsTransformer{Targets: n.Targets},

		// Connect references so ordering is correct
		&ReferenceTransformer{},

		// Make sure there is a single root
		&RootTransformer{},
	}

	// Build the graph
	b := &BasicGraphBuilder{
		Steps:    steps,
		Validate: true,
		Name:     "NodeValidatableResource",
	}

	graph, diags := b.Build(ctx.Path())
	return graph, diags.ErrWithWarnings()
}

// This represents a _single_ resource instance to validate.
type NodeValidatableResourceInstance struct {
	*NodeAbstractResourceInstance
}

var (
	_ GraphNodeSubPath              = (*NodeValidatableResourceInstance)(nil)
	_ GraphNodeReferenceable        = (*NodeValidatableResourceInstance)(nil)
	_ GraphNodeReferencer           = (*NodeValidatableResourceInstance)(nil)
	_ GraphNodeResource             = (*NodeValidatableResourceInstance)(nil)
	_ GraphNodeResourceInstance     = (*NodeValidatableResourceInstance)(nil)
	_ GraphNodeAttachResourceConfig = (*NodeValidatableResourceInstance)(nil)
	_ GraphNodeAttachResourceState  = (*NodeValidatableResourceInstance)(nil)
	_ GraphNodeEvalable             = (*NodeValidatableResourceInstance)(nil)
)

// GraphNodeEvalable
func (n *NodeValidatableResourceInstance) EvalTree() EvalNode {
	addr := n.ResourceInstanceAddr()
	config := n.Config

	// Declare a bunch of variables that are used for state during
	// evaluation. These are written to via pointers passed to the EvalNodes
	// below.
	var provider ResourceProvider
	var providerSchema *ProviderSchema
	var configVal cty.Value

	seq := &EvalSequence{
		Nodes: []EvalNode{
			&EvalValidateSelfRef{
				Addr:   addr.Resource,
				Config: config.Config,
			},
			&EvalGetProvider{
				Addr:   n.ResolvedProvider,
				Output: &provider,
				Schema: &providerSchema,
			},
			&EvalValidateResource{
				Addr:           addr.Resource,
				Provider:       &provider,
				ProviderSchema: &providerSchema,
				Config:         config,
				ConfigVal:      &configVal,
			},
		},
	}

	if managed := n.Config.Managed; managed != nil {
		// Validate all the provisioners
		for _, p := range managed.Provisioners {
			var provisioner ResourceProvisioner
			var provisionerSchema *configschema.Block
			seq.Nodes = append(
				seq.Nodes,
				&EvalGetProvisioner{
					Name:   p.Type,
					Output: &provisioner,
					Schema: &provisionerSchema,
				},
				&EvalValidateProvisioner{
					ResourceAddr: addr.Resource,
					Provisioner:  &provisioner,
					Schema:       &provisionerSchema,
					Config:       p,
				},
			)
		}
	}

	return seq
}
