package terraform

import (
	"fmt"
	"strings"

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

	// Targets are user-specified resources to target. We need to be aware of
	// these so we don't improperly identify orphans when they've just been
	// filtered out of the graph via targeting.
	Targets []ResourceAddress

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
		if len(t.Targets) > 0 {
			var targetedOrphans []string
			for _, o := range resourceOrphans {
				targeted := false
				for _, t := range t.Targets {
					prefix := fmt.Sprintf("%s.%s.%d", t.Type, t.Name, t.Index)
					if strings.HasPrefix(o, prefix) {
						targeted = true
					}
				}
				if targeted {
					targetedOrphans = append(targetedOrphans, o)
				}
			}
			resourceOrphans = targetedOrphans
		}

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
				Provider:     rs.Provider,
				dependentOn:  rs.Dependencies,
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
	ResourceName string
	ResourceType string
	Provider     string

	dependentOn []string
}

func (n *graphNodeOrphanResource) ResourceAddress() *ResourceAddress {
	return n.ResourceAddress()
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
	return fmt.Sprintf("%s (orphan)", n.ResourceName)
}

func (n *graphNodeOrphanResource) ProvidedBy() []string {
	return []string{resourceProvider(n.ResourceName, n.Provider)}
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
					Provider:     n.Provider,
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
	var err error
	seq.Nodes = append(seq.Nodes, &EvalOpFilter{
		Ops: []walkOperation{walkApply, walkDestroy},
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
					Error:    &err,
				},
				&EvalWriteState{
					Name:         n.ResourceName,
					ResourceType: n.ResourceType,
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

	return seq
}

func (n *graphNodeOrphanResource) dependableName() string {
	return n.ResourceName
}

// GraphNodeDestroyable impl.
func (n *graphNodeOrphanResource) DestroyNode(mode GraphNodeDestroyMode) GraphNodeDestroy {
	if mode != DestroyPrimary {
		return nil
	}

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
func (n *graphNodeOrphanResourceFlat) DestroyNode(mode GraphNodeDestroyMode) GraphNodeDestroy {
	if mode != DestroyPrimary {
		return nil
	}

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
