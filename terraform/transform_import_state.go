package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/tfdiags"
)

// ImportStateTransformer is a GraphTransformer that adds nodes to the
// graph to represent the imports we want to do for resources.
type ImportStateTransformer struct {
	Targets []*ImportTarget
}

func (t *ImportStateTransformer) Transform(g *Graph) error {
	for _, target := range t.Targets {
		// The ProviderAddr may not be supplied for non-aliased providers.
		// This will be populated if the targets come from the cli, but tests
		// may not specify implied provider addresses.
		providerAddr := target.ProviderAddr
		if providerAddr.ProviderConfig.Type == "" {
			providerAddr = target.Addr.Resource.Resource.DefaultProviderConfig().Absolute(target.Addr.Module)
		}

		node := &graphNodeImportState{
			Addr:         target.Addr,
			ID:           target.ID,
			ProviderAddr: providerAddr,
		}
		g.Add(node)
	}
	return nil
}

type graphNodeImportState struct {
	Addr             addrs.AbsResourceInstance // Addr is the resource address to import into
	ID               string                    // ID is the ID to import as
	ProviderAddr     addrs.AbsProviderConfig   // Provider address given by the user, or implied by the resource type
	ResolvedProvider addrs.AbsProviderConfig   // provider node address after resolution

	states []providers.ImportedResource
}

var (
	_ GraphNodeSubPath           = (*graphNodeImportState)(nil)
	_ GraphNodeEvalable          = (*graphNodeImportState)(nil)
	_ GraphNodeProviderConsumer  = (*graphNodeImportState)(nil)
	_ GraphNodeDynamicExpandable = (*graphNodeImportState)(nil)
)

func (n *graphNodeImportState) Name() string {
	return fmt.Sprintf("%s (import id %q)", n.Addr, n.ID)
}

// GraphNodeProviderConsumer
func (n *graphNodeImportState) ProvidedBy() (addrs.AbsProviderConfig, bool) {
	// We assume that n.ProviderAddr has been properly populated here.
	// It's the responsibility of the code creating a graphNodeImportState
	// to populate this, possibly by calling DefaultProviderConfig() on the
	// resource address to infer an implied provider from the resource type
	// name.
	return n.ProviderAddr, false
}

// GraphNodeProviderConsumer
func (n *graphNodeImportState) SetProvider(addr addrs.AbsProviderConfig) {
	n.ResolvedProvider = addr
}

// GraphNodeSubPath
func (n *graphNodeImportState) Path() addrs.ModuleInstance {
	return n.Addr.Module
}

// GraphNodeEvalable impl.
func (n *graphNodeImportState) EvalTree() EvalNode {
	var provider providers.Interface

	// Reset our states
	n.states = nil

	// Return our sequence
	return &EvalSequence{
		Nodes: []EvalNode{
			&EvalGetProvider{
				Addr:   n.ResolvedProvider,
				Output: &provider,
			},
			&EvalImportState{
				Addr:     n.Addr.Resource,
				Provider: &provider,
				ID:       n.ID,
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
	var diags tfdiags.Diagnostics

	g := &Graph{Path: ctx.Path()}

	// nameCounter is used to de-dup names in the state.
	nameCounter := make(map[string]int)

	// Compile the list of addresses that we'll be inserting into the state.
	// We do this ahead of time so we can verify that we aren't importing
	// something that already exists.
	addrs := make([]addrs.AbsResourceInstance, len(n.states))
	for i, state := range n.states {
		addr := n.Addr
		if t := state.TypeName; t != "" {
			addr.Resource.Resource.Type = t
		}

		// Determine if we need to suffix the name to de-dup
		key := addr.String()
		count, ok := nameCounter[key]
		if ok {
			count++
			addr.Resource.Resource.Name += fmt.Sprintf("-%d", count)
		}
		nameCounter[key] = count

		// Add it to our list
		addrs[i] = addr
	}

	// Verify that all the addresses are clear
	state := ctx.State()
	for _, addr := range addrs {
		existing := state.ResourceInstance(addr)
		if existing != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Resource already managed by Terraform",
				fmt.Sprintf("Terraform is already managing a remote object for %s. To import to this address you must first remove the existing object from the state.", addr),
			))
			continue
		}
	}
	if diags.HasErrors() {
		// Bail out early, then.
		return nil, diags.Err()
	}

	// For each of the states, we add a node to handle the refresh/add to state.
	// "n.states" is populated by our own EvalTree with the result of
	// ImportState. Since DynamicExpand is always called after EvalTree, this
	// is safe.
	for i, state := range n.states {
		g.Add(&graphNodeImportStateSub{
			TargetAddr:       addrs[i],
			State:            state,
			ResolvedProvider: n.ResolvedProvider,
		})
	}

	// Root transform for a single root
	t := &RootTransformer{}
	if err := t.Transform(g); err != nil {
		return nil, err
	}

	// Done!
	return g, diags.Err()
}

// graphNodeImportStateSub is the sub-node of graphNodeImportState
// and is part of the subgraph. This node is responsible for refreshing
// and adding a resource to the state once it is imported.
type graphNodeImportStateSub struct {
	TargetAddr       addrs.AbsResourceInstance
	State            providers.ImportedResource
	ResolvedProvider addrs.AbsProviderConfig
}

var (
	_ GraphNodeSubPath  = (*graphNodeImportStateSub)(nil)
	_ GraphNodeEvalable = (*graphNodeImportStateSub)(nil)
)

func (n *graphNodeImportStateSub) Name() string {
	return fmt.Sprintf("import %s result", n.TargetAddr)
}

func (n *graphNodeImportStateSub) Path() addrs.ModuleInstance {
	return n.TargetAddr.Module
}

// GraphNodeEvalable impl.
func (n *graphNodeImportStateSub) EvalTree() EvalNode {
	// If the Ephemeral type isn't set, then it is an error
	if n.State.TypeName == "" {
		err := fmt.Errorf("import of %s didn't set type", n.TargetAddr.String())
		return &EvalReturnError{Error: &err}
	}

	state := n.State.AsInstanceObject()

	var provider providers.Interface
	var providerSchema *ProviderSchema
	return &EvalSequence{
		Nodes: []EvalNode{
			&EvalGetProvider{
				Addr:   n.ResolvedProvider,
				Output: &provider,
				Schema: &providerSchema,
			},
			&EvalRefresh{
				Addr:           n.TargetAddr.Resource,
				ProviderAddr:   n.ResolvedProvider,
				Provider:       &provider,
				ProviderSchema: &providerSchema,
				State:          &state,
				Output:         &state,
			},
			&EvalImportStateVerify{
				Addr:  n.TargetAddr.Resource,
				State: &state,
			},
			&EvalWriteState{
				Addr:           n.TargetAddr.Resource,
				ProviderAddr:   n.ResolvedProvider,
				ProviderSchema: &providerSchema,
				State:          &state,
			},
		},
	}
}
