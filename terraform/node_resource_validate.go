package terraform

import (
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/provisioners"
	"github.com/zclconf/go-cty/cty"
)

// NodeValidatableResource represents a resource that is used for validation
// only.
type NodeValidatableResource struct {
	*NodeAbstractResource
}

var (
	_ GraphNodeModuleInstance            = (*NodeValidatableResource)(nil)
	_ GraphNodeEvalable                  = (*NodeValidatableResource)(nil)
	_ GraphNodeReferenceable             = (*NodeValidatableResource)(nil)
	_ GraphNodeReferencer                = (*NodeValidatableResource)(nil)
	_ GraphNodeConfigResource            = (*NodeValidatableResource)(nil)
	_ GraphNodeAttachResourceConfig      = (*NodeValidatableResource)(nil)
	_ GraphNodeAttachProviderMetaConfigs = (*NodeValidatableResource)(nil)
)

func (n *NodeValidatableResource) Path() addrs.ModuleInstance {
	// There is no expansion during validation, so we evaluate everything as
	// single module instances.
	return n.Addr.Module.UnkeyedInstanceShim()
}

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
				ProviderMetas:  n.ProviderMetas,
				ProviderSchema: &providerSchema,
				Config:         config,
				ConfigVal:      &configVal,
			},
		},
	}

	if managed := n.Config.Managed; managed != nil {
		hasCount := n.Config.Count != nil
		hasForEach := n.Config.ForEach != nil

		// Validate all the provisioners
		for _, p := range managed.Provisioners {
			var provisioner provisioners.Interface
			var provisionerSchema *configschema.Block

			if p.Connection == nil {
				p.Connection = config.Managed.Connection
			} else if config.Managed.Connection != nil {
				p.Connection.Config = configs.MergeBodies(config.Managed.Connection.Config, p.Connection.Config)
			}

			seq.Nodes = append(
				seq.Nodes,
				&EvalGetProvisioner{
					Name:   p.Type,
					Output: &provisioner,
					Schema: &provisionerSchema,
				},
				&EvalValidateProvisioner{
					ResourceAddr:       addr.Resource,
					Provisioner:        &provisioner,
					Schema:             &provisionerSchema,
					Config:             p,
					ResourceHasCount:   hasCount,
					ResourceHasForEach: hasForEach,
				},
			)
		}
	}

	return seq
}
