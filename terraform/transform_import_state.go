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
			Addr:         addr,
			ID:           target.ID,
			ProviderName: target.Provider,
		})
	}

	// Build the graph vertices
	for _, n := range nodes {
		g.Add(n)
	}

	return nil
}

type graphNodeImportState struct {
	Addr             *ResourceAddress // Addr is the resource address to import to
	ID               string           // ID is the ID to import as
	ProviderName     string           // Provider string
	ResolvedProvider string           // provider node address

	states []*InstanceState
}

func (n *graphNodeImportState) Name() string {
	return fmt.Sprintf("%s (import id: %s)", n.Addr, n.ID)
}

func (n *graphNodeImportState) ProvidedBy() string {
	return resourceProvider(n.Addr.Type, n.ProviderName)
}

func (n *graphNodeImportState) SetProvider(p string) {
	n.ResolvedProvider = p
}

// GraphNodeSubPath
func (n *graphNodeImportState) Path() []string {
	return normalizeModulePath(n.Addr.Path)
}

// GraphNodeEvalable impl.
func (n *graphNodeImportState) EvalTree() EvalNode {
	var provider ResourceProvider
	info := &InstanceInfo{
		Id:         fmt.Sprintf("%s.%s", n.Addr.Type, n.Addr.Name),
		ModulePath: n.Path(),
		Type:       n.Addr.Type,
	}

	// Reset our states
	n.states = nil

	// Return our sequence
	return &EvalSequence{
		Nodes: []EvalNode{
			&EvalGetProvider{
				Name:   n.ResolvedProvider,
				Output: &provider,
			},
			&EvalImportState{
				Provider: &provider,
				Info:     info,
				Id:       n.ID,
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

	// nameCounter is used to de-dup names in the state.
	nameCounter := make(map[string]int)

	// Compile the list of addresses that we'll be inserting into the state.
	// We do this ahead of time so we can verify that we aren't importing
	// something that already exists.
	addrs := make([]*ResourceAddress, len(n.states))
	for i, state := range n.states {
		addr := *n.Addr
		if t := state.Ephemeral.Type; t != "" {
			addr.Type = t
		}

		// Determine if we need to suffix the name to de-dup
		key := addr.String()
		count, ok := nameCounter[key]
		if ok {
			count++
			addr.Name += fmt.Sprintf("-%d", count)
		}
		nameCounter[key] = count

		// Add it to our list
		addrs[i] = &addr
	}

	// Verify that all the addresses are clear
	state, lock := ctx.State()
	lock.RLock()
	defer lock.RUnlock()
	filter := &StateFilter{State: state}
	for _, addr := range addrs {
		result, err := filter.Filter(addr.String())
		if err != nil {
			return nil, fmt.Errorf("Error verifying address %s: %s", addr, err)
		}

		// Go through the filter results and it is an error if we find
		// a matching InstanceState, meaning that we would have a collision.
		for _, r := range result {
			if _, ok := r.Value.(*InstanceState); ok {
				return nil, fmt.Errorf(
					"Can't import %s, would collide with an existing resource.\n\n"+
						"Please remove or rename this resource before continuing.",
					addr)
			}
		}
	}

	// For each of the states, we add a node to handle the refresh/add to state.
	// "n.states" is populated by our own EvalTree with the result of
	// ImportState. Since DynamicExpand is always called after EvalTree, this
	// is safe.
	for i, state := range n.states {
		g.Add(&graphNodeImportStateSub{
			Target:           addrs[i],
			Path_:            n.Path(),
			State:            state,
			ProviderName:     n.ProviderName,
			ResolvedProvider: n.ResolvedProvider,
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
	Target           *ResourceAddress
	State            *InstanceState
	Path_            []string
	ProviderName     string
	ResolvedProvider string
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
		Id:         fmt.Sprintf("%s.%s", n.Target.Type, n.Target.Name),
		ModulePath: n.Path_,
		Type:       n.State.Ephemeral.Type,
	}

	// Key is the resource key
	key := &ResourceStateKey{
		Name:  n.Target.Name,
		Type:  info.Type,
		Index: n.Target.Index,
	}

	// The eval sequence
	var provider ResourceProvider
	return &EvalSequence{
		Nodes: []EvalNode{
			&EvalGetProvider{
				Name:   n.ResolvedProvider,
				Output: &provider,
			},
			&EvalRefresh{
				Provider: &provider,
				State:    &state,
				Info:     info,
				Output:   &state,
			},
			&EvalImportStateVerify{
				Info:  info,
				Id:    n.State.ID,
				State: &state,
			},
			&EvalWriteState{
				Name:         key.String(),
				ResourceType: info.Type,
				Provider:     n.ResolvedProvider,
				State:        &state,
			},
		},
	}
}
