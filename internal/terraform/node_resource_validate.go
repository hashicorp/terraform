package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

// NodeValidatableResource represents a resource that is used for validation
// only.
type NodeValidatableResource struct {
	*NodeAbstractResource
}

var (
	_ GraphNodeModuleInstance            = (*NodeValidatableResource)(nil)
	_ GraphNodeExecutable                = (*NodeValidatableResource)(nil)
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
func (n *NodeValidatableResource) Execute(ctx EvalContext, op walkOperation) (diags tfdiags.Diagnostics) {
	addr := n.ResourceAddr()
	config := n.Config

	// Declare the variables will be used are used to pass values along
	// the evaluation sequence below. These are written to via pointers
	// passed to the EvalNodes.
	var configVal cty.Value
	provider, providerSchema, err := GetProvider(ctx, n.ResolvedProvider)
	diags = diags.Append(err)
	if diags.HasErrors() {
		return diags
	}

	evalValidateResource := &EvalValidateResource{
		Addr:           addr.Resource,
		Provider:       &provider,
		ProviderMetas:  n.ProviderMetas,
		ProviderSchema: &providerSchema,
		Config:         config,
		ConfigVal:      &configVal,
	}
	diags = diags.Append(evalValidateResource.Validate(ctx))
	if diags.HasErrors() {
		return diags
	}

	if managed := n.Config.Managed; managed != nil {
		hasCount := n.Config.Count != nil
		hasForEach := n.Config.ForEach != nil

		// Validate all the provisioners
		for _, p := range managed.Provisioners {
			if p.Connection == nil {
				p.Connection = config.Managed.Connection
			} else if config.Managed.Connection != nil {
				p.Connection.Config = configs.MergeBodies(config.Managed.Connection.Config, p.Connection.Config)
			}

			provisioner := ctx.Provisioner(p.Type)
			if provisioner == nil {
				diags = diags.Append(fmt.Errorf("provisioner %s not initialized", p.Type))
				return diags
			}
			provisionerSchema := ctx.ProvisionerSchema(p.Type)
			if provisionerSchema == nil {
				diags = diags.Append(fmt.Errorf("provisioner %s not initialized", p.Type))
				return diags
			}

			// Validate Provisioner Config
			validateProvisioner := &EvalValidateProvisioner{
				ResourceAddr:       addr.Resource,
				Provisioner:        &provisioner,
				Schema:             &provisionerSchema,
				Config:             p,
				ResourceHasCount:   hasCount,
				ResourceHasForEach: hasForEach,
			}
			diags = diags.Append(validateProvisioner.Validate(ctx))
			if diags.HasErrors() {
				return diags
			}
		}
	}
	return diags
}
