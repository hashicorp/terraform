package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/dag"
)

// ResourceCountTransformer is a GraphTransformer that expands the count
// out for a specific resource.
type ResourceCountTransformer struct {
	Resource *config.Resource
	Destroy  bool
}

func (t *ResourceCountTransformer) Transform(g *Graph) error {
	// Expand the resource count
	count, err := t.Resource.Count()
	if err != nil {
		return err
	}

	// Don't allow the count to be negative
	if count < 0 {
		return fmt.Errorf("negative count: %d", count)
	}

	// For each count, build and add the node
	nodes := make([]dag.Vertex, count)
	for i := 0; i < count; i++ {
		// Set the index. If our count is 1 we special case it so that
		// we handle the "resource.0" and "resource" boundary properly.
		index := i
		if count == 1 {
			index = -1
		}

		// Save the node for later so we can do connections. Make the
		// proper node depending on if we're just a destroy node or if
		// were a regular node.
		var node dag.Vertex = &graphNodeExpandedResource{
			Index:    index,
			Resource: t.Resource,
		}
		if t.Destroy {
			node = &graphNodeExpandedResourceDestroy{
				graphNodeExpandedResource: node.(*graphNodeExpandedResource),
			}
		}

		// Add the node now
		nodes[i] = node
		g.Add(nodes[i])
	}

	// Make the dependency connections
	for _, n := range nodes {
		// Connect the dependents. We ignore the return value for missing
		// dependents since that should've been caught at a higher level.
		g.ConnectDependent(n)
	}

	return nil
}

type graphNodeExpandedResource struct {
	Index    int
	Resource *config.Resource
}

func (n *graphNodeExpandedResource) Name() string {
	if n.Index == -1 {
		return n.Resource.Id()
	}

	return fmt.Sprintf("%s #%d", n.Resource.Id(), n.Index)
}

// GraphNodeDependable impl.
func (n *graphNodeExpandedResource) DependableName() []string {
	return []string{
		n.Resource.Id(),
		n.stateId(),
	}
}

// GraphNodeDependent impl.
func (n *graphNodeExpandedResource) DependentOn() []string {
	config := &GraphNodeConfigResource{Resource: n.Resource}
	return config.DependentOn()
}

// GraphNodeProviderConsumer
func (n *graphNodeExpandedResource) ProvidedBy() []string {
	return []string{resourceProvider(n.Resource.Type)}
}

// GraphNodeEvalable impl.
func (n *graphNodeExpandedResource) EvalTree() EvalNode {
	// Build the resource. If we aren't part of a multi-resource, then
	// we still consider ourselves as count index zero.
	index := n.Index
	if index < 0 {
		index = 0
	}
	resource := &Resource{CountIndex: index}

	// Shared node for interpolation of configuration
	interpolateNode := &EvalInterpolate{
		Config:   n.Resource.RawConfig,
		Resource: resource,
	}

	seq := &EvalSequence{Nodes: make([]EvalNode, 0, 5)}

	// Validate the resource
	vseq := &EvalSequence{Nodes: make([]EvalNode, 0, 5)}
	vseq.Nodes = append(vseq.Nodes, &EvalValidateResource{
		Provider:     &EvalGetProvider{Name: n.ProvidedBy()[0]},
		Config:       interpolateNode,
		ResourceName: n.Resource.Name,
		ResourceType: n.Resource.Type,
	})

	// Validate all the provisioners
	for _, p := range n.Resource.Provisioners {
		vseq.Nodes = append(vseq.Nodes, &EvalValidateProvisioner{
			Provisioner: &EvalGetProvisioner{Name: p.Type},
			Config: &EvalInterpolate{
				Config: p.RawConfig, Resource: resource},
		})
	}

	// Add the validation operations
	seq.Nodes = append(seq.Nodes, &EvalOpFilter{
		Ops:  []walkOperation{walkValidate},
		Node: vseq,
	})

	// Build instance info
	info := &InstanceInfo{Id: n.stateId(), Type: n.Resource.Type}
	seq.Nodes = append(seq.Nodes, &EvalInstanceInfo{Info: info})

	// Refresh the resource
	seq.Nodes = append(seq.Nodes, &EvalOpFilter{
		Ops: []walkOperation{walkRefresh},
		Node: &EvalWriteState{
			Name:         n.stateId(),
			ResourceType: n.Resource.Type,
			Dependencies: n.DependentOn(),
			State: &EvalRefresh{
				Info:     info,
				Provider: &EvalGetProvider{Name: n.ProvidedBy()[0]},
				State:    &EvalReadState{Name: n.stateId()},
			},
		},
	})

	// Diff the resource
	var diff InstanceDiff
	seq.Nodes = append(seq.Nodes, &EvalOpFilter{
		Ops: []walkOperation{walkPlan},
		Node: &EvalSequence{
			Nodes: []EvalNode{
				&EvalWriteState{
					Name:         n.stateId(),
					ResourceType: n.Resource.Type,
					Dependencies: n.DependentOn(),
					State: &EvalDiff{
						Info:     info,
						Config:   interpolateNode,
						Provider: &EvalGetProvider{Name: n.ProvidedBy()[0]},
						State:    &EvalReadState{Name: n.stateId()},
						Output:   &diff,
					},
				},
				&EvalWriteDiff{
					Name: n.stateId(),
					Diff: &diff,
				},
			},
		},
	})

	// Diff the resource for destruction
	seq.Nodes = append(seq.Nodes, &EvalOpFilter{
		Ops: []walkOperation{walkPlanDestroy},
		Node: &EvalSequence{
			Nodes: []EvalNode{
				&EvalDiffDestroy{
					Info:   info,
					State:  &EvalReadState{Name: n.stateId()},
					Output: &diff,
				},
				&EvalWriteDiff{
					Name: n.stateId(),
					Diff: &diff,
				},
			},
		},
	})

	return seq
}

// stateId is the name used for the state key
func (n *graphNodeExpandedResource) stateId() string {
	if n.Index == -1 {
		return n.Resource.Id()
	}

	return fmt.Sprintf("%s.%d", n.Resource.Id(), n.Index)
}

// GraphNodeStateRepresentative impl.
func (n *graphNodeExpandedResource) StateId() []string {
	return []string{n.stateId()}
}

// graphNodeExpandedResourceDestroy represents an expanded resource that
// is to be destroyed.
type graphNodeExpandedResourceDestroy struct {
	*graphNodeExpandedResource
}

func (n *graphNodeExpandedResourceDestroy) Name() string {
	return fmt.Sprintf("%s (destroy)", n.graphNodeExpandedResource.Name())
}

// GraphNodeEvalable impl.
func (n *graphNodeExpandedResourceDestroy) EvalTree() EvalNode {
	// TODO: We need an eval tree that destroys when there is a
	// RequiresNew.
	return EvalNoop{}
}
