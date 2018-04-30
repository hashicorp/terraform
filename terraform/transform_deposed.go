package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/addrs"
)

// DeposedTransformer is a GraphTransformer that adds deposed resources
// to the graph.
type DeposedTransformer struct {
	// State is the global state. We'll automatically find the correct
	// ModuleState based on the Graph.Path that is being transformed.
	State *State

	// View, if non-empty, is the ModuleState.View used around the state
	// to find deposed resources.
	View string

	// The provider used by the resourced which were deposed
	ResolvedProvider addrs.AbsProviderConfig
}

func (t *DeposedTransformer) Transform(g *Graph) error {
	state := t.State.ModuleByPath(g.Path)
	if state == nil {
		// If there is no state for our module there can't be any deposed
		// resources, since they live in the state.
		return nil
	}

	// If we have a view, apply it now
	if t.View != "" {
		state = state.View(t.View)
	}

	// Go through all the resources in our state to look for deposed resources
	for k, rs := range state.Resources {
		// If we have no deposed resources, then move on
		if len(rs.Deposed) == 0 {
			continue
		}

		legacyAddr, err := parseResourceAddressInternal(k)
		if err != nil {
			return fmt.Errorf("invalid instance key %q in state: %s", k, err)
		}
		addr := legacyAddr.AbsResourceInstanceAddr()

		providerAddr, err := rs.ProviderAddr()
		if err != nil {
			return fmt.Errorf("invalid instance provider address %q in state: %s", rs.Provider, err)
		}

		for i := range rs.Deposed {
			g.Add(&graphNodeDeposedResource{
				Addr:             addr,
				Index:            i,
				RecordedProvider: providerAddr,
			})
		}
	}

	return nil
}

// graphNodeDeposedResource is the graph vertex representing a deposed resource.
type graphNodeDeposedResource struct {
	Addr             addrs.AbsResourceInstance
	Index            int // Index into the "deposed" list in state
	RecordedProvider addrs.AbsProviderConfig
	ResolvedProvider addrs.AbsProviderConfig
}

var (
	_ GraphNodeProviderConsumer = (*graphNodeDeposedResource)(nil)
	_ GraphNodeEvalable         = (*graphNodeDeposedResource)(nil)
)

func (n *graphNodeDeposedResource) Name() string {
	return fmt.Sprintf("%s (deposed #%d)", n.Addr.String(), n.Index)
}

func (n *graphNodeDeposedResource) ProvidedBy() (addrs.AbsProviderConfig, bool) {
	return n.RecordedProvider, true
}

func (n *graphNodeDeposedResource) SetProvider(addr addrs.AbsProviderConfig) {
	// Because our ProvidedBy returns exact=true, this is actually rather
	// pointless and should always just be the address we asked for.
	n.RecordedProvider = addr
}

// GraphNodeEvalable impl.
func (n *graphNodeDeposedResource) EvalTree() EvalNode {
	addr := n.Addr

	var provider ResourceProvider
	var state *InstanceState

	seq := &EvalSequence{Nodes: make([]EvalNode, 0, 5)}

	stateKey := NewLegacyResourceInstanceAddress(addr).stateId()

	// Refresh the resource
	seq.Nodes = append(seq.Nodes, &EvalOpFilter{
		Ops: []walkOperation{walkRefresh},
		Node: &EvalSequence{
			Nodes: []EvalNode{
				&EvalGetProvider{
					Addr:   n.ResolvedProvider,
					Output: &provider,
				},
				&EvalReadStateDeposed{
					Name:   stateKey,
					Output: &state,
					Index:  n.Index,
				},
				&EvalRefresh{
					Addr:     addr.Resource,
					Provider: &provider,
					State:    &state,
					Output:   &state,
				},
				&EvalWriteStateDeposed{
					Name:         stateKey,
					ResourceType: n.Addr.Resource.Resource.Type,
					Provider:     n.ResolvedProvider.String(), // FIXME: Change underlying struct to use addrs.AbsProviderConfig itself
					State:        &state,
					Index:        n.Index,
				},
			},
		},
	})

	// Apply
	var diff *InstanceDiff
	var err error
	seq.Nodes = append(seq.Nodes, &EvalOpFilter{
		Ops: []walkOperation{walkApply, walkDestroy},
		Node: &EvalSequence{
			Nodes: []EvalNode{
				&EvalGetProvider{
					Addr:   n.ResolvedProvider,
					Output: &provider,
				},
				&EvalReadStateDeposed{
					Name:   stateKey,
					Output: &state,
					Index:  n.Index,
				},
				&EvalDiffDestroy{
					Addr:   addr.Resource,
					State:  &state,
					Output: &diff,
				},
				// Call pre-apply hook
				&EvalApplyPre{
					Addr:  addr.Resource,
					State: &state,
					Diff:  &diff,
				},
				&EvalApply{
					Addr:     addr.Resource,
					State:    &state,
					Diff:     &diff,
					Provider: &provider,
					Output:   &state,
					Error:    &err,
				},
				// Always write the resource back to the state deposed... if it
				// was successfully destroyed it will be pruned. If it was not, it will
				// be caught on the next run.
				&EvalWriteStateDeposed{
					Name:         stateKey,
					ResourceType: n.Addr.Resource.Resource.Type,
					Provider:     n.ResolvedProvider.String(), // FIXME: Change underlying struct to use addrs.AbsProviderConfig itself
					State:        &state,
					Index:        n.Index,
				},
				&EvalApplyPost{
					Addr:  addr.Resource,
					State: &state,
					Error: &err,
				},
				&EvalReturnError{
					Error: &err,
				},
				&EvalUpdateStateHook{},
			},
		},
	})

	return seq
}
