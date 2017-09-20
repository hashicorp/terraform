package terraform

import (
	"github.com/hashicorp/terraform/dag"
)

// NodeValidatableResource represents a resource that is used for validation
// only.
type NodeValidatableResource struct {
	*NodeAbstractCountResource
}

// GraphNodeEvalable
func (n *NodeValidatableResource) EvalTree() EvalNode {
	// Ensure we're validating
	c := n.NodeAbstractCountResource
	c.Validate = true
	return c.EvalTree()
}

// GraphNodeDynamicExpandable
func (n *NodeValidatableResource) DynamicExpand(ctx EvalContext) (*Graph, error) {
	// Grab the state which we read
	state, lock := ctx.State()
	lock.RLock()
	defer lock.RUnlock()

	// Expand the resource count which must be available by now from EvalTree
	count := 1
	if n.Config.RawCount.Value() != unknownValue() {
		var err error
		count, err = n.Config.Count()
		if err != nil {
			return nil, err
		}
	}

	// The concrete resource factory we'll use
	concreteResource := func(a *NodeAbstractResource) dag.Vertex {
		// Add the config and state since we don't do that via transforms
		a.Config = n.Config

		return &NodeValidatableResourceInstance{
			NodeAbstractResource: a,
		}
	}

	// Start creating the steps
	steps := []GraphTransformer{
		// Expand the count.
		&ResourceCountTransformer{
			Concrete: concreteResource,
			Count:    count,
			Addr:     n.ResourceAddr(),
		},

		// Attach the state
		&AttachStateTransformer{State: state},

		// Targeting
		&TargetsTransformer{ParsedTargets: n.Targets},

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

	return b.Build(ctx.Path())
}

// This represents a _single_ resource instance to validate.
type NodeValidatableResourceInstance struct {
	*NodeAbstractResource
}

// GraphNodeEvalable
func (n *NodeValidatableResourceInstance) EvalTree() EvalNode {
	addr := n.NodeAbstractResource.Addr

	// Build the resource for eval
	resource := &Resource{
		Name:       addr.Name,
		Type:       addr.Type,
		CountIndex: addr.Index,
	}
	if resource.CountIndex < 0 {
		resource.CountIndex = 0
	}

	// Declare a bunch of variables that are used for state during
	// evaluation. Most of this are written to by-address below.
	var config *ResourceConfig
	var provider ResourceProvider

	seq := &EvalSequence{
		Nodes: []EvalNode{
			&EvalValidateResourceSelfRef{
				Addr:   &addr,
				Config: &n.Config.RawConfig,
			},
			&EvalGetProvider{
				Name:   n.ProvidedBy()[0],
				Output: &provider,
			},
			&EvalInterpolate{
				Config:   n.Config.RawConfig.Copy(),
				Resource: resource,
				Output:   &config,
			},
			&EvalValidateResource{
				Provider:     &provider,
				Config:       &config,
				ResourceName: n.Config.Name,
				ResourceType: n.Config.Type,
				ResourceMode: n.Config.Mode,
			},
		},
	}

	// Validate all the provisioners
	for _, p := range n.Config.Provisioners {
		var provisioner ResourceProvisioner
		var connConfig *ResourceConfig
		seq.Nodes = append(
			seq.Nodes,
			&EvalGetProvisioner{
				Name:   p.Type,
				Output: &provisioner,
			},
			&EvalInterpolate{
				Config:   p.RawConfig.Copy(),
				Resource: resource,
				Output:   &config,
			},
			&EvalInterpolate{
				Config:   p.ConnInfo.Copy(),
				Resource: resource,
				Output:   &connConfig,
			},
			&EvalValidateProvisioner{
				Provisioner: &provisioner,
				Config:      &config,
				ConnConfig:  &connConfig,
			},
		)
	}

	return seq
}
