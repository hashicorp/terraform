package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/states"

	"github.com/hashicorp/terraform/addrs"
	"github.com/zclconf/go-cty/cty"
)

// NodePlannableResourceInstance represents a _single_ resource
// instance that is plannable. This means this represents a single
// count index, for example.
type NodePlannableResourceInstance struct {
	*NodeAbstractResourceInstance
	ForceCreateBeforeDestroy bool
}

var (
	_ GraphNodeModuleInstance       = (*NodePlannableResourceInstance)(nil)
	_ GraphNodeReferenceable        = (*NodePlannableResourceInstance)(nil)
	_ GraphNodeReferencer           = (*NodePlannableResourceInstance)(nil)
	_ GraphNodeConfigResource       = (*NodePlannableResourceInstance)(nil)
	_ GraphNodeResourceInstance     = (*NodePlannableResourceInstance)(nil)
	_ GraphNodeAttachResourceConfig = (*NodePlannableResourceInstance)(nil)
	_ GraphNodeAttachResourceState  = (*NodePlannableResourceInstance)(nil)
	_ GraphNodeEvalable             = (*NodePlannableResourceInstance)(nil)
)

// GraphNodeEvalable
func (n *NodePlannableResourceInstance) EvalTree() EvalNode {
	addr := n.ResourceInstanceAddr()

	// Eval info is different depending on what kind of resource this is
	switch addr.Resource.Resource.Mode {
	case addrs.ManagedResourceMode:
		return n.evalTreeManagedResource(addr)
	case addrs.DataResourceMode:
		return n.evalTreeDataResource(addr)
	default:
		panic(fmt.Errorf("unsupported resource mode %s", n.Config.Mode))
	}
}

func (n *NodePlannableResourceInstance) evalTreeDataResource(addr addrs.AbsResourceInstance) EvalNode {
	config := n.Config
	var provider providers.Interface
	var providerSchema *ProviderSchema
	var change *plans.ResourceInstanceChange
	var state *states.ResourceInstanceObject
	var configVal cty.Value

	forcePlanRead := new(bool)

	return &EvalSequence{
		Nodes: []EvalNode{
			&EvalGetProvider{
				Addr:   n.ResolvedProvider,
				Output: &provider,
				Schema: &providerSchema,
			},

			&EvalReadState{
				Addr:           addr.Resource,
				Provider:       &provider,
				ProviderSchema: &providerSchema,

				Output: &state,
			},

			// If we already have a non-planned state then we already dealt
			// with this during the refresh walk and so we have nothing to do
			// here.
			&EvalIf{
				If: func(ctx EvalContext) (bool, error) {
					depChanges := false

					// Check and see if any depends_on dependencies have
					// changes, since they won't show up as changes in the
					// configuration.
					changes := ctx.Changes()
					depChanges = func() bool {
						for _, d := range n.dependsOn {
							for _, change := range changes.GetConfigResourceChanges(d) {
								if change != nil && change.Action != plans.NoOp {
									return true
								}
							}
						}
						return false
					}()

					*forcePlanRead = depChanges
					return true, nil
				},
				Then: EvalNoop{},
			},

			&EvalValidateSelfRef{
				Addr:           addr.Resource,
				Config:         config.Config,
				ProviderSchema: &providerSchema,
			},

			&EvalReadData{
				Addr:           addr.Resource,
				Config:         n.Config,
				Provider:       &provider,
				ProviderAddr:   n.ResolvedProvider,
				ProviderMetas:  n.ProviderMetas,
				ProviderSchema: &providerSchema,
				ForcePlanRead:  forcePlanRead,
				OutputChange:   &change,
				OutputValue:    &configVal,
				OutputState:    &state,
			},

			&EvalWriteState{
				Addr:           addr.Resource,
				ProviderAddr:   n.ResolvedProvider,
				ProviderSchema: &providerSchema,
				State:          &state,
			},

			&EvalWriteDiff{
				Addr:           addr.Resource,
				ProviderSchema: &providerSchema,
				Change:         &change,
			},
		},
	}
}

func (n *NodePlannableResourceInstance) evalTreeManagedResource(addr addrs.AbsResourceInstance) EvalNode {
	config := n.Config
	var provider providers.Interface
	var providerSchema *ProviderSchema
	var change *plans.ResourceInstanceChange
	var state *states.ResourceInstanceObject

	return &EvalSequence{
		Nodes: []EvalNode{
			&EvalGetProvider{
				Addr:   n.ResolvedProvider,
				Output: &provider,
				Schema: &providerSchema,
			},

			&EvalReadState{
				Addr:           addr.Resource,
				Provider:       &provider,
				ProviderSchema: &providerSchema,

				Output: &state,
			},

			&EvalValidateSelfRef{
				Addr:           addr.Resource,
				Config:         config.Config,
				ProviderSchema: &providerSchema,
			},

			&EvalDiff{
				Addr:                addr.Resource,
				Config:              n.Config,
				CreateBeforeDestroy: n.ForceCreateBeforeDestroy,
				Provider:            &provider,
				ProviderAddr:        n.ResolvedProvider,
				ProviderMetas:       n.ProviderMetas,
				ProviderSchema:      &providerSchema,
				State:               &state,
				OutputChange:        &change,
				OutputState:         &state,
			},
			&EvalCheckPreventDestroy{
				Addr:   addr.Resource,
				Config: n.Config,
				Change: &change,
			},
			&EvalWriteState{
				Addr:           addr.Resource,
				ProviderAddr:   n.ResolvedProvider,
				State:          &state,
				ProviderSchema: &providerSchema,
			},
			&EvalWriteDiff{
				Addr:           addr.Resource,
				ProviderSchema: &providerSchema,
				Change:         &change,
			},
		},
	}
}
