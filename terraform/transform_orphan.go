package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/config/module"
	"github.com/hashicorp/terraform/dag"
)

// GraphNodeStateRepresentative is an interface that can be implemented by
// a node to say that it is representing a resource in the state.
type GraphNodeStateRepresentative interface {
	StateId() []string
}

// OrphanTransformer is a GraphTransformer that adds orphans to the
// graph. This transformer adds both resource and module orphans.
type OrphanTransformer struct {
	// State is the global state. We require the global state to
	// properly find module orphans at our path.
	State *State

	// Module is the root module. We'll look up the proper configuration
	// using the graph path.
	Module *module.Tree

	// View, if non-nil will set a view on the module state.
	View string
}

func (t *OrphanTransformer) Transform(g *Graph) error {
	if t.State == nil {
		// If the entire state is nil, there can't be any orphans
		return nil
	}

	// Build up all our state representatives
	resourceRep := make(map[string]struct{})
	for _, v := range g.Vertices() {
		if sr, ok := v.(GraphNodeStateRepresentative); ok {
			for _, k := range sr.StateId() {
				resourceRep[k] = struct{}{}
			}
		}
	}

	var config *config.Config
	if t.Module != nil {
		if module := t.Module.Child(g.Path[1:]); module != nil {
			config = module.Config()
		}
	}

	var resourceVertexes []dag.Vertex
	if state := t.State.ModuleByPath(g.Path); state != nil {
		// If we have state, then we can have orphan resources

		// If we have a view, get the view
		if t.View != "" {
			state = state.View(t.View)
		}

		// Go over each resource orphan and add it to the graph.
		resourceOrphans := state.Orphans(config)
		resourceVertexes = make([]dag.Vertex, len(resourceOrphans))
		for i, k := range resourceOrphans {
			// If this orphan is represented by some other node somehow,
			// then ignore it.
			if _, ok := resourceRep[k]; ok {
				continue
			}

			rs := state.Resources[k]

			resourceVertexes[i] = g.Add(&graphNodeOrphanResource{
				ResourceName: k,
				ResourceType: rs.Type,
				dependentOn:  rs.Dependencies,
			})
		}
	}

	// Go over each module orphan and add it to the graph. We store the
	// vertexes and states outside so that we can connect dependencies later.
	moduleOrphans := t.State.ModuleOrphans(g.Path, config)
	moduleVertexes := make([]dag.Vertex, len(moduleOrphans))
	for i, path := range moduleOrphans {
		moduleVertexes[i] = g.Add(&graphNodeOrphanModule{
			Path:        path,
			dependentOn: t.State.ModuleByPath(path).Dependencies,
		})
	}

	// Now do the dependencies. We do this _after_ adding all the orphan
	// nodes above because there are cases in which the orphans themselves
	// depend on other orphans.

	// Resource dependencies
	for _, v := range resourceVertexes {
		g.ConnectDependent(v)
	}

	// Module dependencies
	for _, v := range moduleVertexes {
		g.ConnectDependent(v)
	}

	return nil
}

// graphNodeOrphanModule is the graph vertex representing an orphan resource..
type graphNodeOrphanModule struct {
	Path []string

	dependentOn []string
}

func (n *graphNodeOrphanModule) DependableName() []string {
	return []string{n.dependableName()}
}

func (n *graphNodeOrphanModule) DependentOn() []string {
	return n.dependentOn
}

func (n *graphNodeOrphanModule) Name() string {
	return fmt.Sprintf("%s (orphan)", n.dependableName())
}

func (n *graphNodeOrphanModule) dependableName() string {
	return fmt.Sprintf("module.%s", n.Path[len(n.Path)-1])
}

// GraphNodeExpandable
func (n *graphNodeOrphanModule) Expand(b GraphBuilder) (GraphNodeSubgraph, error) {
	g, err := b.Build(n.Path)
	if err != nil {
		return nil, err
	}

	return &GraphNodeBasicSubgraph{
		NameValue: n.Name(),
		Graph:     g,
	}, nil
}

// graphNodeOrphanResource is the graph vertex representing an orphan resource..
type graphNodeOrphanResource struct {
	ResourceName string
	ResourceType string

	dependentOn []string
}

func (n *graphNodeOrphanResource) DependableName() []string {
	return []string{n.dependableName()}
}

func (n *graphNodeOrphanResource) DependentOn() []string {
	return n.dependentOn
}

func (n *graphNodeOrphanResource) Name() string {
	return fmt.Sprintf("%s (orphan)", n.ResourceName)
}

func (n *graphNodeOrphanResource) ProvidedBy() []string {
	return []string{resourceProvider(n.ResourceName)}
}

// GraphNodeEvalable impl.
func (n *graphNodeOrphanResource) EvalTree() EvalNode {
	var provider ResourceProvider
	var state *InstanceState

	seq := &EvalSequence{Nodes: make([]EvalNode, 0, 5)}

	// Build instance info
	info := &InstanceInfo{Id: n.ResourceName, Type: n.ResourceType}
	seq.Nodes = append(seq.Nodes, &EvalInstanceInfo{Info: info})

	// Refresh the resource
	seq.Nodes = append(seq.Nodes, &EvalOpFilter{
		Ops: []walkOperation{walkRefresh},
		Node: &EvalSequence{
			Nodes: []EvalNode{
				&EvalGetProvider{
					Name:   n.ProvidedBy()[0],
					Output: &provider,
				},
				&EvalReadState{
					Name:   n.ResourceName,
					Output: &state,
				},
				&EvalRefresh{
					Info:     info,
					Provider: &provider,
					State:    &state,
					Output:   &state,
				},
				&EvalWriteState{
					Name:         n.ResourceName,
					ResourceType: n.ResourceType,
					Dependencies: n.DependentOn(),
					State:        &state,
				},
			},
		},
	})

	// Diff the resource
	var diff *InstanceDiff
	seq.Nodes = append(seq.Nodes, &EvalOpFilter{
		Ops: []walkOperation{walkPlan, walkPlanDestroy},
		Node: &EvalSequence{
			Nodes: []EvalNode{
				&EvalReadState{
					Name:   n.ResourceName,
					Output: &state,
				},
				&EvalDiffDestroy{
					Info:   info,
					State:  &state,
					Output: &diff,
				},
				&EvalWriteDiff{
					Name: n.ResourceName,
					Diff: &diff,
				},
			},
		},
	})

	// Apply
	seq.Nodes = append(seq.Nodes, &EvalOpFilter{
		Ops: []walkOperation{walkApply},
		Node: &EvalSequence{
			Nodes: []EvalNode{
				&EvalReadDiff{
					Name: n.ResourceName,
					Diff: &diff,
				},
				&EvalGetProvider{
					Name:   n.ProvidedBy()[0],
					Output: &provider,
				},
				&EvalReadState{
					Name:   n.ResourceName,
					Output: &state,
				},
				&EvalApply{
					Info:     info,
					State:    &state,
					Diff:     &diff,
					Provider: &provider,
					Output:   &state,
				},
				&EvalWriteState{
					Name:         n.ResourceName,
					ResourceType: n.ResourceType,
					Dependencies: n.DependentOn(),
					State:        &state,
				},
				&EvalUpdateStateHook{},
			},
		},
	})

	return seq
}

func (n *graphNodeOrphanResource) dependableName() string {
	return n.ResourceName
}
