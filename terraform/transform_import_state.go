package terraform

import (
	"fmt"
)

// ImportStateTransformer is a GraphTransformer that adds nodes to the
// graph to represent the imports we want to do for resources.
type ImportStateTransformer struct {
	Targets []*ImportTarget
}

func (t *ImportStateTransformer) Transform(g *Graph) error {
	nodes := make([]*graphNodeImportState, 0, len(t.Targets))
	for _, target := range t.Targets {
		addr, err := ParseResourceAddress(target.Addr)
		if err != nil {
			return fmt.Errorf(
				"failed to parse resource address '%s': %s",
				target.Addr, err)
		}

		nodes = append(nodes, &graphNodeImportState{
			Addr: addr,
			ID:   target.ID,
		})
	}

	// Build the graph vertices
	for _, n := range nodes {
		g.Add(n)
	}

	return nil
}

type graphNodeImportState struct {
	Addr *ResourceAddress // Addr is the resource address to import to
	ID   string           // ID is the ID to import as

	states []*InstanceState
}

func (n *graphNodeImportState) Name() string {
	return fmt.Sprintf("%s (import id: %s)", n.Addr, n.ID)
}

func (n *graphNodeImportState) ProvidedBy() []string {
	return []string{resourceProvider(n.Addr.Type, "")}
}

// GraphNodeSubPath
func (n *graphNodeImportState) Path() []string {
	return normalizeModulePath(n.Addr.Path)
}

// GraphNodeEvalable impl.
func (n *graphNodeImportState) EvalTree() EvalNode {
	var provider ResourceProvider
	info := &InstanceInfo{
		Id:         n.ID,
		ModulePath: n.Path(),
		Type:       n.Addr.Type,
	}

	// Reset our states
	n.states = nil

	// Return our sequence
	return &EvalSequence{
		Nodes: []EvalNode{
			&EvalGetProvider{
				Name:   n.ProvidedBy()[0],
				Output: &provider,
			},
			&EvalImportState{
				Provider: &provider,
				Info:     info,
				Output:   &n.states,
			},
		},
	}
}

// GraphNodeDynamicExpandable impl.
//
// We use DynamicExpand as a way to generate the subgraph of refreshes
// and state inserts we need to do for our import state. Since they're new
// resources they don't depend on anything else and refreshes are isolated
// so this is nearly a perfect use case for dynamic expand.
func (n *graphNodeImportState) DynamicExpand(ctx EvalContext) (*Graph, error) {
	g := &Graph{Path: ctx.Path()}

	// For each of the states, we add a node to handle the refresh/add to state.
	// "n.states" is populated by our own EvalTree with the result of
	// ImportState. Since DynamicExpand is always called after EvalTree, this
	// is safe.
	for _, state := range n.states {
		g.Add(&graphNodeImportStateSub{
			Target: n.Addr,
			Path_:  n.Path(),
			State:  state,
		})
	}

	// Root transform for a single root
	t := &RootTransformer{}
	if err := t.Transform(g); err != nil {
		return nil, err
	}

	// Done!
	return g, nil
}

// graphNodeImportStateSub is the sub-node of graphNodeImportState
// and is part of the subgraph. This node is responsible for refreshing
// and adding a resource to the state once it is imported.
type graphNodeImportStateSub struct {
	Target *ResourceAddress
	State  *InstanceState
	Path_  []string
}

func (n *graphNodeImportStateSub) Name() string {
	return fmt.Sprintf("import %s result: %s", n.Target, n.State.ID)
}

func (n *graphNodeImportStateSub) Path() []string {
	return n.Path_
}

// GraphNodeEvalable impl.
func (n *graphNodeImportStateSub) EvalTree() EvalNode {
	// If the Ephemeral type isn't set, then it is an error
	if n.State.Ephemeral.Type == "" {
		err := fmt.Errorf(
			"import of %s didn't set type for %s",
			n.Target.String(), n.State.ID)
		return &EvalReturnError{Error: &err}
	}

	// DeepCopy so we're only modifying our local copy
	state := n.State.DeepCopy()

	// Build the resource info
	info := &InstanceInfo{
		Id:         n.State.ID,
		ModulePath: n.Path_,
		Type:       n.State.Ephemeral.Type,
	}

	// Key is the resource key
	key := &ResourceStateKey{
		Name:  n.Target.Name,
		Type:  info.Type,
		Index: -1,
	}

	// The eval sequence
	var provider ResourceProvider
	return &EvalSequence{
		Nodes: []EvalNode{
			&EvalGetProvider{
				Name:   resourceProvider(info.Type, ""),
				Output: &provider,
			},
			&EvalRefresh{
				Provider: &provider,
				State:    &state,
				Info:     info,
				Output:   &state,
			},
			&EvalWriteState{
				Name:         key.String(),
				ResourceType: info.Type,
				Provider:     resourceProvider(info.Type, ""),
				State:        &state,
			},
		},
	}
}
