package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/config/module"
	"github.com/hashicorp/terraform/dag"
)

// OrphanTransformer is a GraphTransformer that adds orphans to the
// graph. This transformer adds both resource and module orphans.
type OrphanTransformer struct {
	// State is the global state. We require the global state to
	// properly find module orphans at our path.
	State *State

	// Module is the root module. We'll look up the proper configuration
	// using the graph path.
	Module *module.Tree
}

func (t *OrphanTransformer) Transform(g *Graph) error {
	var config *config.Config
	if module := t.Module.Child(g.Path[1:]); module != nil {
		config = module.Config()
	}

	var resourceVertexes []dag.Vertex
	if state := t.State.ModuleByPath(g.Path); state != nil {
		// If we have state, then we can have orphan resources

		// Go over each resource orphan and add it to the graph.
		resourceOrphans := state.Orphans(config)
		resourceVertexes = make([]dag.Vertex, len(resourceOrphans))
		for i, k := range resourceOrphans {
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
	seq := &EvalSequence{Nodes: make([]EvalNode, 0, 5)}

	// Build instance info
	info := &InstanceInfo{Id: n.ResourceName, Type: n.ResourceType}
	seq.Nodes = append(seq.Nodes, &EvalInstanceInfo{Info: info})

	// Refresh the resource
	seq.Nodes = append(seq.Nodes, &EvalOpFilter{
		Ops: []walkOperation{walkRefresh},
		Node: &EvalWriteState{
			Name:         n.ResourceName,
			ResourceType: n.ResourceType,
			Dependencies: n.DependentOn(),
			State: &EvalRefresh{
				Info:     info,
				Provider: &EvalGetProvider{Name: n.ProvidedBy()[0]},
				State: &EvalReadState{
					Name: n.ResourceName,
				},
			},
		},
	})

	// Diff the resource
	var diff InstanceDiff
	seq.Nodes = append(seq.Nodes, &EvalOpFilter{
		Ops: []walkOperation{walkPlan},
		Node: &EvalSequence{
			Nodes: []EvalNode{
				&EvalDiffDestroy{
					Info:   info,
					State:  &EvalReadState{Name: n.ResourceName},
					Output: &diff,
				},
				&EvalWriteDiff{
					Name: n.ResourceName,
					Diff: &diff,
				},
			},
		},
	})

	return seq
}

func (n *graphNodeOrphanResource) dependableName() string {
	return n.ResourceName
}
