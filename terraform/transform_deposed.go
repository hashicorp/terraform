package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/states"
)

// DeposedTransformer is a GraphTransformer that adds nodes to the graph for
// the deposed objects associated with a given resource instance.
type DeposedTransformer struct {
	// State is the global state, from which we'll retrieve the state for
	// the instance given in InstanceAddr.
	State *states.State

	// InstanceAddr is the address of the instance whose deposed objects will
	// have graph nodes created.
	InstanceAddr addrs.AbsResourceInstance

	// The provider used by the resourced which were deposed
	ResolvedProvider addrs.AbsProviderConfig
}

func (t *DeposedTransformer) Transform(g *Graph) error {
	rs := t.State.Resource(t.InstanceAddr.ContainingResource())
	if rs == nil {
		// If the resource has no state then there can't be deposed objects.
		return nil
	}
	is := rs.Instances[t.InstanceAddr.Resource.Key]
	if is == nil {
		// If the instance has no state then there can't be deposed objects.
		return nil
	}

	providerAddr := rs.ProviderConfig

	for k := range is.Deposed {
		g.Add(&graphNodeDeposedResource{
			Addr:             t.InstanceAddr,
			DeposedKey:       k,
			RecordedProvider: providerAddr,
			ResolvedProvider: t.ResolvedProvider,
		})
	}

	return nil
}

// graphNodeDeposedResource is the graph vertex representing a deposed resource.
type graphNodeDeposedResource struct {
	Addr             addrs.AbsResourceInstance
	DeposedKey       states.DeposedKey
	RecordedProvider addrs.AbsProviderConfig
	ResolvedProvider addrs.AbsProviderConfig
}

var (
	_ GraphNodeProviderConsumer = (*graphNodeDeposedResource)(nil)
	_ GraphNodeEvalable         = (*graphNodeDeposedResource)(nil)
)

func (n *graphNodeDeposedResource) Name() string {
	return fmt.Sprintf("%s (deposed %s)", n.Addr.String(), n.DeposedKey)
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

	var provider providers.Interface
	var providerSchema *ProviderSchema
	var state *states.ResourceInstanceObject

	seq := &EvalSequence{Nodes: make([]EvalNode, 0, 5)}

	// Refresh the resource
	seq.Nodes = append(seq.Nodes, &EvalOpFilter{
		Ops: []walkOperation{walkRefresh},
		Node: &EvalSequence{
			Nodes: []EvalNode{
				&EvalGetProvider{
					Addr:   n.ResolvedProvider,
					Output: &provider,
					Schema: &providerSchema,
				},
				&EvalReadStateDeposed{
					Addr:   addr.Resource,
					Key:    n.DeposedKey,
					Output: &state,
				},
				&EvalRefresh{
					Addr:     addr.Resource,
					Provider: &provider,
					State:    &state,
					Output:   &state,
				},
				&EvalWriteStateDeposed{
					Addr:           addr.Resource,
					Key:            n.DeposedKey,
					ProviderAddr:   n.ResolvedProvider,
					ProviderSchema: &providerSchema,
					State:          &state,
				},
			},
		},
	})

	// Apply
	var change *plans.ResourceInstanceChange
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
					Addr:           addr.Resource,
					Output:         &state,
					Key:            n.DeposedKey,
					Provider:       &provider,
					ProviderSchema: &providerSchema,
				},
				&EvalDiffDestroy{
					Addr:   addr.Resource,
					State:  &state,
					Output: &change,
				},
				// Call pre-apply hook
				&EvalApplyPre{
					Addr:   addr.Resource,
					State:  &state,
					Change: &change,
				},
				&EvalApply{
					Addr:     addr.Resource,
					State:    &state,
					Change:   &change,
					Provider: &provider,
					Output:   &state,
					Error:    &err,
				},
				// Always write the resource back to the state deposed... if it
				// was successfully destroyed it will be pruned. If it was not, it will
				// be caught on the next run.
				&EvalWriteStateDeposed{
					Addr:           addr.Resource,
					Key:            n.DeposedKey,
					ProviderAddr:   n.ResolvedProvider,
					ProviderSchema: &providerSchema,
					State:          &state,
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
