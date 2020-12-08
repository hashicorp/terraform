package terraform

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/provisioners"
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

			// Validate Provisioner Config
			diags = diags.Append(n.validateProvisioner(ctx, p, hasCount, hasForEach))
			if diags.HasErrors() {
				return diags
			}
		}
	}
	return diags
}

// validateProvisioner validates the configuration of a provisioner belonging to
// a resource. The provisioner config is expected to contain the merged
// connection configurations.
func (n *NodeValidatableResource) validateProvisioner(ctx EvalContext, p *configs.Provisioner, hasCount, hasForEach bool) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	provisioner := ctx.Provisioner(p.Type)
	if provisioner == nil {
		return diags.Append(fmt.Errorf("provisioner %s not initialized", p.Type))
	}
	provisionerSchema := ctx.ProvisionerSchema(p.Type)
	if provisionerSchema == nil {
		return diags.Append(fmt.Errorf("provisioner %s not initialized", p.Type))
	}

	// Validate the provisioner's own config first
	configVal, _, configDiags := n.evaluateBlock(ctx, p.Config, provisionerSchema, hasCount, hasForEach)
	diags = diags.Append(configDiags)

	if configVal == cty.NilVal {
		// Should never happen for a well-behaved EvaluateBlock implementation
		return diags.Append(fmt.Errorf("EvaluateBlock returned nil value"))
	}

	req := provisioners.ValidateProvisionerConfigRequest{
		Config: configVal,
	}

	resp := provisioner.ValidateProvisionerConfig(req)
	diags = diags.Append(resp.Diagnostics)

	if p.Connection != nil {
		// We can't comprehensively validate the connection config since its
		// final structure is decided by the communicator and we can't instantiate
		// that until we have a complete instance state. However, we *can* catch
		// configuration keys that are not valid for *any* communicator, catching
		// typos early rather than waiting until we actually try to run one of
		// the resource's provisioners.
		_, _, connDiags := n.evaluateBlock(ctx, p.Connection.Config, connectionBlockSupersetSchema, hasCount, hasForEach)
		diags = diags.Append(connDiags)
	}
	return diags
}

func (n *NodeValidatableResource) evaluateBlock(ctx EvalContext, body hcl.Body, schema *configschema.Block, hasCount, hasForEach bool) (cty.Value, hcl.Body, tfdiags.Diagnostics) {
	keyData := EvalDataForNoInstanceKey
	selfAddr := n.ResourceAddr().Resource.Instance(addrs.NoKey)

	if hasCount {
		// For a resource that has count, we allow count.index but don't
		// know at this stage what it will return.
		keyData = InstanceKeyEvalData{
			CountIndex: cty.UnknownVal(cty.Number),
		}

		// "self" can't point to an unknown key, but we'll force it to be
		// key 0 here, which should return an unknown value of the
		// expected type since none of these elements are known at this
		// point anyway.
		selfAddr = n.ResourceAddr().Resource.Instance(addrs.IntKey(0))
	} else if hasForEach {
		// For a resource that has for_each, we allow each.value and each.key
		// but don't know at this stage what it will return.
		keyData = InstanceKeyEvalData{
			EachKey:   cty.UnknownVal(cty.String),
			EachValue: cty.DynamicVal,
		}

		// "self" can't point to an unknown key, but we'll force it to be
		// key "" here, which should return an unknown value of the
		// expected type since none of these elements are known at
		// this point anyway.
		selfAddr = n.ResourceAddr().Resource.Instance(addrs.StringKey(""))
	}

	return ctx.EvaluateBlock(body, schema, selfAddr, keyData)
}

// connectionBlockSupersetSchema is a schema representing the superset of all
// possible arguments for "connection" blocks across all supported connection
// types.
//
// This currently lives here because we've not yet updated our communicator
// subsystem to be aware of schema itself. Once that is done, we can remove
// this and use a type-specific schema from the communicator to validate
// exactly what is expected for a given connection type.
var connectionBlockSupersetSchema = &configschema.Block{
	Attributes: map[string]*configschema.Attribute{
		// NOTE: "type" is not included here because it's treated special
		// by the config loader and stored away in a separate field.

		// Common attributes for both connection types
		"host": {
			Type:     cty.String,
			Required: true,
		},
		"type": {
			Type:     cty.String,
			Optional: true,
		},
		"user": {
			Type:     cty.String,
			Optional: true,
		},
		"password": {
			Type:     cty.String,
			Optional: true,
		},
		"port": {
			Type:     cty.String,
			Optional: true,
		},
		"timeout": {
			Type:     cty.String,
			Optional: true,
		},
		"script_path": {
			Type:     cty.String,
			Optional: true,
		},
		// For type=ssh only (enforced in ssh communicator)
		"target_platform": {
			Type:     cty.String,
			Optional: true,
		},
		"private_key": {
			Type:     cty.String,
			Optional: true,
		},
		"certificate": {
			Type:     cty.String,
			Optional: true,
		},
		"host_key": {
			Type:     cty.String,
			Optional: true,
		},
		"agent": {
			Type:     cty.Bool,
			Optional: true,
		},
		"agent_identity": {
			Type:     cty.String,
			Optional: true,
		},
		"bastion_host": {
			Type:     cty.String,
			Optional: true,
		},
		"bastion_host_key": {
			Type:     cty.String,
			Optional: true,
		},
		"bastion_port": {
			Type:     cty.Number,
			Optional: true,
		},
		"bastion_user": {
			Type:     cty.String,
			Optional: true,
		},
		"bastion_password": {
			Type:     cty.String,
			Optional: true,
		},
		"bastion_private_key": {
			Type:     cty.String,
			Optional: true,
		},
		"bastion_certificate": {
			Type:     cty.String,
			Optional: true,
		},

		// For type=winrm only (enforced in winrm communicator)
		"https": {
			Type:     cty.Bool,
			Optional: true,
		},
		"insecure": {
			Type:     cty.Bool,
			Optional: true,
		},
		"cacert": {
			Type:     cty.String,
			Optional: true,
		},
		"use_ntlm": {
			Type:     cty.Bool,
			Optional: true,
		},
	},
}
