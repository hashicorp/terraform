package terraform

import (
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/providers"
	"github.com/zclconf/go-cty/cty"
)

// NodeValidatableResource represents a resource that is used for validation
// only.
type NodeValidatableResource struct {
	*NodeAbstractResource
}

var (
	_ GraphNodeSubPath              = (*NodeValidatableResource)(nil)
	_ GraphNodeEvalable             = (*NodeValidatableResource)(nil)
	_ GraphNodeReferenceable        = (*NodeValidatableResource)(nil)
	_ GraphNodeReferencer           = (*NodeValidatableResource)(nil)
	_ GraphNodeResource             = (*NodeValidatableResource)(nil)
	_ GraphNodeAttachResourceConfig = (*NodeValidatableResource)(nil)
)

// GraphNodeEvalable
func (n *NodeValidatableResource) EvalTree() EvalNode {
	addr := n.ResourceAddr()
	config := n.Config

	// Declare the variables will be used are used to pass values along
	// the evaluation sequence below. These are written to via pointers
	// passed to the EvalNodes.
	var provider providers.Interface
	var providerSchema *ProviderSchema
	var configVal cty.Value

	seq := &EvalSequence{
		Nodes: []EvalNode{
			&EvalGetProvider{
				Addr:   n.ResolvedProvider,
				Output: &provider,
				Schema: &providerSchema,
			},
			&EvalValidateResource{
				Addr:           addr.Resource,
				Provider:       &provider,
				ProviderSchema: &providerSchema,
				Config:         config,
				ConfigVal:      &configVal,
			},
		},
	}

	if managed := n.Config.Managed; managed != nil {
		hasCount := n.Config.Count != nil

		// Validate all the provisioners
		for _, p := range managed.Provisioners {
			var provisioner ResourceProvisioner
			var provisionerSchema *configschema.Block
			seq.Nodes = append(
				seq.Nodes,
				&EvalGetProvisioner{
					Name:   p.Type,
					Output: &provisioner,
					Schema: &provisionerSchema,
				},
				&EvalValidateProvisioner{
					ResourceAddr:     addr.Resource,
					Provisioner:      &provisioner,
					Schema:           &provisionerSchema,
					Config:           p,
					ResourceHasCount: hasCount,
				},
			)
		}
	}

	return seq
}
