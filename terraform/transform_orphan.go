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

		resourceOrphans := state.Orphans(config)

		resourceVertexes = make([]dag.Vertex, len(resourceOrphans))
		for i, k := range resourceOrphans {
			// If this orphan is represented by some other node somehow,
			// then ignore it.
			if _, ok := resourceRep[k]; ok {
				continue
			}

			rs := state.Resources[k]

			rsk, err := ParseResourceStateKey(k)
			if err != nil {
				return err
			}
			resourceVertexes[i] = g.Add(&graphNodeOrphanResource{
				Path:        g.Path,
				ResourceKey: rsk,
				Provider:    rs.Provider,
				dependentOn: rs.Dependencies,
			})
		}
	}

	// Go over each module orphan and add it to the graph. We store the
	// vertexes and states outside so that we can connect dependencies later.
	moduleOrphans := t.State.ModuleOrphans(g.Path, config)
	moduleVertexes := make([]dag.Vertex, len(moduleOrphans))
	for i, path := range moduleOrphans {
		var deps []string
		if s := t.State.ModuleByPath(path); s != nil {
			deps = s.Dependencies
		}

		moduleVertexes[i] = g.Add(&graphNodeOrphanModule{
			Path:        path,
			dependentOn: deps,
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
	Path        []string
	ResourceKey *ResourceStateKey
	Provider    string

	dependentOn []string
}

func (n *graphNodeOrphanResource) ConfigType() GraphNodeConfigType {
	return GraphNodeConfigTypeResource
}

func (n *graphNodeOrphanResource) ResourceAddress() *ResourceAddress {
	return &ResourceAddress{
		Index:        n.ResourceKey.Index,
		InstanceType: TypePrimary,
		Name:         n.ResourceKey.Name,
		Path:         n.Path[1:],
		Type:         n.ResourceKey.Type,
		Mode:         n.ResourceKey.Mode,
	}
}

func (n *graphNodeOrphanResource) DependableName() []string {
	return []string{n.dependableName()}
}

func (n *graphNodeOrphanResource) DependentOn() []string {
	return n.dependentOn
}

func (n *graphNodeOrphanResource) Flatten(p []string) (dag.Vertex, error) {
	return &graphNodeOrphanResourceFlat{
		graphNodeOrphanResource: n,
		PathValue:               p,
	}, nil
}

func (n *graphNodeOrphanResource) Name() string {
	return fmt.Sprintf("%s (orphan)", n.ResourceKey)
}

func (n *graphNodeOrphanResource) ProvidedBy() []string {
	return []string{resourceProvider(n.ResourceKey.Type, n.Provider)}
}

// GraphNodeEvalable impl.
func (n *graphNodeOrphanResource) EvalTree() EvalNode {

	seq := &EvalSequence{Nodes: make([]EvalNode, 0, 5)}

	// Build instance info
	info := &InstanceInfo{Id: n.ResourceKey.String(), Type: n.ResourceKey.Type}
	seq.Nodes = append(seq.Nodes, &EvalInstanceInfo{Info: info})

	// Each resource mode has its own lifecycle
	switch n.ResourceKey.Mode {
	case config.ManagedResourceMode:
		seq.Nodes = append(
			seq.Nodes,
			n.managedResourceEvalNodes(info)...,
		)
	case config.DataResourceMode:
		seq.Nodes = append(
			seq.Nodes,
			n.dataResourceEvalNodes(info)...,
		)
	default:
		panic(fmt.Errorf("unsupported resource mode %s", n.ResourceKey.Mode))
	}

	return seq
}

func (n *graphNodeOrphanResource) managedResourceEvalNodes(info *InstanceInfo) []EvalNode {
	var provider ResourceProvider
	var state *InstanceState

	nodes := make([]EvalNode, 0, 3)

	// Refresh the resource
	nodes = append(nodes, &EvalOpFilter{
		Ops: []walkOperation{walkRefresh},
		Node: &EvalSequence{
			Nodes: []EvalNode{
				&EvalGetProvider{
					Name:   n.ProvidedBy()[0],
					Output: &provider,
				},
				&EvalReadState{
					Name:   n.ResourceKey.String(),
					Output: &state,
				},
				&EvalRefresh{
					Info:     info,
					Provider: &provider,
					State:    &state,
					Output:   &state,
				},
				&EvalWriteState{
					Name:         n.ResourceKey.String(),
					ResourceType: n.ResourceKey.Type,
					Provider:     n.Provider,
					Dependencies: n.DependentOn(),
					State:        &state,
				},
			},
		},
	})

	// Diff the resource
	var diff *InstanceDiff
	nodes = append(nodes, &EvalOpFilter{
		Ops: []walkOperation{walkPlan, walkPlanDestroy},
		Node: &EvalSequence{
			Nodes: []EvalNode{
				&EvalReadState{
					Name:   n.ResourceKey.String(),
					Output: &state,
				},
				&EvalDiffDestroy{
					Info:   info,
					State:  &state,
					Output: &diff,
				},
				&EvalWriteDiff{
					Name: n.ResourceKey.String(),
					Diff: &diff,
				},
			},
		},
	})

	// Apply
	var err error
	nodes = append(nodes, &EvalOpFilter{
		Ops: []walkOperation{walkApply, walkDestroy},
		Node: &EvalSequence{
			Nodes: []EvalNode{
				&EvalReadDiff{
					Name: n.ResourceKey.String(),
					Diff: &diff,
				},
				&EvalGetProvider{
					Name:   n.ProvidedBy()[0],
					Output: &provider,
				},
				&EvalReadState{
					Name:   n.ResourceKey.String(),
					Output: &state,
				},
				&EvalApply{
					Info:     info,
					State:    &state,
					Diff:     &diff,
					Provider: &provider,
					Output:   &state,
					Error:    &err,
				},
				&EvalWriteState{
					Name:         n.ResourceKey.String(),
					ResourceType: n.ResourceKey.Type,
					Provider:     n.Provider,
					Dependencies: n.DependentOn(),
					State:        &state,
				},
				&EvalApplyPost{
					Info:  info,
					State: &state,
					Error: &err,
				},
				&EvalUpdateStateHook{},
			},
		},
	})

	return nodes
}

func (n *graphNodeOrphanResource) dataResourceEvalNodes(info *InstanceInfo) []EvalNode {
	nodes := make([]EvalNode, 0, 3)

	// This will remain nil, since we don't retain states for orphaned
	// data resources.
	var state *InstanceState

	// On both refresh and apply we just drop our state altogether,
	// since the config resource validation pass will have proven that the
	// resources remaining in the configuration don't need it.
	nodes = append(nodes, &EvalOpFilter{
		Ops: []walkOperation{walkRefresh, walkApply},
		Node: &EvalSequence{
			Nodes: []EvalNode{
				&EvalWriteState{
					Name:         n.ResourceKey.String(),
					ResourceType: n.ResourceKey.Type,
					Provider:     n.Provider,
					Dependencies: n.DependentOn(),
					State:        &state, // state is nil
				},
			},
		},
	})

	return nodes
}

func (n *graphNodeOrphanResource) dependableName() string {
	return n.ResourceKey.String()
}

// GraphNodeDestroyable impl.
func (n *graphNodeOrphanResource) DestroyNode() GraphNodeDestroy {
	return n
}

// GraphNodeDestroy impl.
func (n *graphNodeOrphanResource) CreateBeforeDestroy() bool {
	return false
}

func (n *graphNodeOrphanResource) CreateNode() dag.Vertex {
	return n
}

// Same as graphNodeOrphanResource, but for flattening
type graphNodeOrphanResourceFlat struct {
	*graphNodeOrphanResource

	PathValue []string
}

func (n *graphNodeOrphanResourceFlat) Name() string {
	return fmt.Sprintf(
		"%s.%s", modulePrefixStr(n.PathValue), n.graphNodeOrphanResource.Name())
}

func (n *graphNodeOrphanResourceFlat) Path() []string {
	return n.PathValue
}

// GraphNodeDestroyable impl.
func (n *graphNodeOrphanResourceFlat) DestroyNode() GraphNodeDestroy {
	return n
}

// GraphNodeDestroy impl.
func (n *graphNodeOrphanResourceFlat) CreateBeforeDestroy() bool {
	return false
}

func (n *graphNodeOrphanResourceFlat) CreateNode() dag.Vertex {
	return n
}

func (n *graphNodeOrphanResourceFlat) ProvidedBy() []string {
	return modulePrefixList(
		n.graphNodeOrphanResource.ProvidedBy(),
		modulePrefixStr(n.PathValue))
}
