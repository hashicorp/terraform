package terraform

import (
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/tfdiags"
)

// NodeDestroyableDataResourceInstance represents a resource that is "destroyable":
// it is ready to be destroyed.
type NodeDestroyableDataResourceInstance struct {
	*NodeAbstractResourceInstance
}

// GraphNodeEvalable
func (n *NodeDestroyableDataResourceInstance) EvalTree() EvalNode {
	addr := n.ResourceInstanceAddr()

	var providerSchema *ProviderSchema
	// We don't need the provider, but we're calling EvalGetProvider to load the
	// schema.
	var provider providers.Interface

	// Just destroy it.
	var state *states.ResourceInstanceObject
	return &EvalSequence{
		Nodes: []EvalNode{
			&EvalGetProvider{
				Addr:   n.ResolvedProvider,
				Output: &provider,
				Schema: &providerSchema,
			},
			&EvalWriteState{
				Addr:           addr.Resource,
				State:          &state,
				ProviderAddr:   n.ResolvedProvider,
				ProviderSchema: &providerSchema,
			},
		},
	}
}

func (n *NodeDestroyableDataResourceInstance) Execute(ctx EvalContext) tfdiags.Diagnostics {
	addr := n.ResourceInstanceAddr()
	var diags tfdiags.Diagnostics
	var providerSchema *ProviderSchema
	// We don't need the provider, but we're calling EvalGetProvider to load the
	// schema.
	var provider providers.Interface

	// Just destroy it.
	var state *states.ResourceInstanceObject

	step1 := &EvalGetProvider{
		Addr:   n.ResolvedProvider,
		Output: &provider,
		Schema: &providerSchema,
	}
	_, err := step1.Eval(ctx)
	if err != nil {
		diags = diags.Append(err)
		return diags
	}

	step2 := &EvalWriteState{
		Addr:           addr.Resource,
		State:          &state,
		ProviderAddr:   n.ResolvedProvider,
		ProviderSchema: &providerSchema,
	}
	_, err = step2.Eval(ctx)
	if err != nil {
		diags = diags.Append(err)
	}

	return diags
}
