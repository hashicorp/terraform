package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/provisioners"
	"github.com/hashicorp/terraform/tfdiags"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
	"github.com/zclconf/go-cty/cty/gocty"
)

// EvalValidateCount is an EvalNode implementation that validates
// the count of a resource.
type EvalValidateCount struct {
	Resource *configs.Resource
}

// TODO: test
func (n *EvalValidateCount) Eval(ctx EvalContext) (interface{}, error) {
	var diags tfdiags.Diagnostics
	var count int
	var err error

	val, valDiags := ctx.EvaluateExpr(n.Resource.Count, cty.Number, nil)
	diags = diags.Append(valDiags)
	if valDiags.HasErrors() {
		goto RETURN
	}
	if val.IsNull() || !val.IsKnown() {
		goto RETURN
	}

	err = gocty.FromCtyValue(val, &count)
	if err != nil {
		// The EvaluateExpr call above already guaranteed us a number value,
		// so if we end up here then we have something that is out of range
		// for an int, and the error message will include a description of
		// the valid range.
		rawVal := val.AsBigFloat()
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid count value",
			Detail:   fmt.Sprintf("The number %s is not a valid count value: %s.", rawVal, err),
			Subject:  n.Resource.Count.Range().Ptr(),
		})
	} else if count < 0 {
		rawVal := val.AsBigFloat()
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid count value",
			Detail:   fmt.Sprintf("The number %s is not a valid count value: count must not be negative.", rawVal),
			Subject:  n.Resource.Count.Range().Ptr(),
		})
	}

RETURN:
	return nil, diags.NonFatalErr()
}

// EvalValidateProvider is an EvalNode implementation that validates
// a provider configuration.
type EvalValidateProvider struct {
	Addr     addrs.ProviderConfig
	Provider *providers.Interface
	Config   *configs.Provider
}

func (n *EvalValidateProvider) Eval(ctx EvalContext) (interface{}, error) {
	var diags tfdiags.Diagnostics
	provider := *n.Provider

	configBody := buildProviderConfig(ctx, n.Addr, n.Config)

	resp := provider.GetSchema()
	diags = diags.Append(resp.Diagnostics)
	if diags.HasErrors() {
		return nil, diags.NonFatalErr()
	}

	configSchema := resp.Provider.Block
	if configSchema == nil {
		// Should never happen in real code, but often comes up in tests where
		// mock schemas are being used that tend to be incomplete.
		log.Printf("[WARN] EvalValidateProvider: no config schema is available for %s, so using empty schema", n.Addr)
		configSchema = &configschema.Block{}
	}

	configVal, configBody, evalDiags := ctx.EvaluateBlock(configBody, configSchema, nil, EvalDataForNoInstanceKey)
	diags = diags.Append(evalDiags)
	if evalDiags.HasErrors() {
		return nil, diags.NonFatalErr()
	}

	req := providers.PrepareProviderConfigRequest{
		Config: configVal,
	}

	validateResp := provider.PrepareProviderConfig(req)
	diags = diags.Append(validateResp.Diagnostics)

	return nil, diags.NonFatalErr()
}

// EvalValidateProvisioner is an EvalNode implementation that validates
// the configuration of a provisioner belonging to a resource.
type EvalValidateProvisioner struct {
	ResourceAddr     addrs.Resource
	Provisioner      *provisioners.Interface
	Schema           **configschema.Block
	Config           *configs.Provisioner
	ConnConfig       *configs.Connection
	ResourceHasCount bool
}

func (n *EvalValidateProvisioner) Eval(ctx EvalContext) (interface{}, error) {
	provisioner := *n.Provisioner
	config := *n.Config
	schema := *n.Schema

	var diags tfdiags.Diagnostics

	{
		// Validate the provisioner's own config first

		configVal, _, configDiags := n.evaluateBlock(ctx, config.Config, schema)
		diags = diags.Append(configDiags)
		if configDiags.HasErrors() {
			return nil, diags.Err()
		}

		if configVal == cty.NilVal {
			// Should never happen for a well-behaved EvaluateBlock implementation
			return nil, fmt.Errorf("EvaluateBlock returned nil value")
		}

		req := provisioners.ValidateProvisionerConfigRequest{
			Config: configVal,
		}

		resp := provisioner.ValidateProvisionerConfig(req)
		diags = diags.Append(resp.Diagnostics)
	}

	{
		// Now validate the connection config, which might either be from
		// the provisioner block itself or inherited from the resource's
		// shared connection info.
		connDiags := n.validateConnConfig(ctx, n.ConnConfig, n.ResourceAddr)
		diags = diags.Append(connDiags)
	}

	return nil, diags.NonFatalErr()
}

func (n *EvalValidateProvisioner) validateConnConfig(ctx EvalContext, config *configs.Connection, self addrs.Referenceable) tfdiags.Diagnostics {
	// We can't comprehensively validate the connection config since its
	// final structure is decided by the communicator and we can't instantiate
	// that until we have a complete instance state. However, we *can* catch
	// configuration keys that are not valid for *any* communicator, catching
	// typos early rather than waiting until we actually try to run one of
	// the resource's provisioners.

	var diags tfdiags.Diagnostics

	if config == nil || config.Config == nil {
		// No block to validate
		return diags
	}

	// We evaluate here just by evaluating the block and returning any
	// diagnostics we get, since evaluation alone is enough to check for
	// extraneous arguments and incorrectly-typed arguments.
	_, _, configDiags := n.evaluateBlock(ctx, config.Config, connectionBlockSupersetSchema)
	diags = diags.Append(configDiags)

	return diags
}

func (n *EvalValidateProvisioner) evaluateBlock(ctx EvalContext, body hcl.Body, schema *configschema.Block) (cty.Value, hcl.Body, tfdiags.Diagnostics) {
	keyData := EvalDataForNoInstanceKey
	selfAddr := n.ResourceAddr.Instance(addrs.NoKey)

	if n.ResourceHasCount {
		// For a resource that has count, we allow count.index but don't
		// know at this stage what it will return.
		keyData = InstanceKeyEvalData{
			CountIndex: cty.UnknownVal(cty.Number),
		}

		// "self" can't point to an unknown key, but we'll force it to be
		// key 0 here, which should return an unknown value of the
		// expected type since none of these elements are known at this
		// point anyway.
		selfAddr = n.ResourceAddr.Instance(addrs.IntKey(0))
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

// connectionBlockSupersetSchema is a schema representing the superset of all
// possible arguments for "connection" blocks across all supported connection
// types.
//
// This currently lives here because we've not yet updated our communicator
// subsystem to be aware of schema itself. It's exported only for use in the
// configs/configupgrade package and should not be used from anywhere else.
// The caller may not modify any part of the returned schema data structure.
func ConnectionBlockSupersetSchema() *configschema.Block {
	return connectionBlockSupersetSchema
}

// EvalValidateResource is an EvalNode implementation that validates
// the configuration of a resource.
type EvalValidateResource struct {
	Addr           addrs.Resource
	Provider       *providers.Interface
	ProviderSchema **ProviderSchema
	Config         *configs.Resource

	// IgnoreWarnings means that warnings will not be passed through. This allows
	// "just-in-time" passes of validation to continue execution through warnings.
	IgnoreWarnings bool

	// ConfigVal, if non-nil, will be updated with the value resulting from
	// evaluating the given configuration body. Since validation is performed
	// very early, this value is likely to contain lots of unknown values,
	// but its type will conform to the schema of the resource type associated
	// with the resource instance being validated.
	ConfigVal *cty.Value
}

func (n *EvalValidateResource) Eval(ctx EvalContext) (interface{}, error) {
	if n.ProviderSchema == nil || *n.ProviderSchema == nil {
		return nil, fmt.Errorf("EvalValidateResource has nil schema for %s", n.Addr)
	}

	var diags tfdiags.Diagnostics
	provider := *n.Provider
	cfg := *n.Config
	schema := *n.ProviderSchema
	mode := cfg.Mode

	keyData := EvalDataForNoInstanceKey
	if n.Config.Count != nil {
		// If the config block has count, we'll evaluate with an unknown
		// number as count.index so we can still type check even though
		// we won't expand count until the plan phase.
		keyData = InstanceKeyEvalData{
			CountIndex: cty.UnknownVal(cty.Number),
		}

		// Basic type-checking of the count argument. More complete validation
		// of this will happen when we DynamicExpand during the plan walk.
		countDiags := n.validateCount(ctx, n.Config.Count)
		diags = diags.Append(countDiags)
	}

	for _, traversal := range n.Config.DependsOn {
		ref, refDiags := addrs.ParseRef(traversal)
		diags = diags.Append(refDiags)
		if len(ref.Remaining) != 0 {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid depends_on reference",
				Detail:   "References in depends_on must be to a whole object (resource, etc), not to an attribute of an object.",
				Subject:  ref.Remaining.SourceRange().Ptr(),
			})
		}

		// The ref must also refer to something that exists. To test that,
		// we'll just eval it and count on the fact that our evaluator will
		// detect references to non-existent objects.
		if !diags.HasErrors() {
			scope := ctx.EvaluationScope(nil, EvalDataForNoInstanceKey)
			if scope != nil { // sometimes nil in tests, due to incomplete mocks
				_, refDiags = scope.EvalReference(ref, cty.DynamicPseudoType)
				diags = diags.Append(refDiags)
			}
		}
	}

	// Provider entry point varies depending on resource mode, because
	// managed resources and data resources are two distinct concepts
	// in the provider abstraction.
	switch mode {
	case addrs.ManagedResourceMode:
		schema, _ := schema.SchemaForResourceType(mode, cfg.Type)
		if schema == nil {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid resource type",
				Detail:   fmt.Sprintf("The provider %s does not support resource type %q.", cfg.ProviderConfigAddr(), cfg.Type),
				Subject:  &cfg.TypeRange,
			})
			return nil, diags.Err()
		}

		configVal, _, valDiags := ctx.EvaluateBlock(cfg.Config, schema, nil, keyData)
		diags = diags.Append(valDiags)
		if valDiags.HasErrors() {
			return nil, diags.Err()
		}

		if cfg.Managed != nil { // can be nil only in tests with poorly-configured mocks
			for _, traversal := range cfg.Managed.IgnoreChanges {
				moreDiags := schema.StaticValidateTraversal(traversal)
				diags = diags.Append(moreDiags)
			}
		}

		req := providers.ValidateResourceTypeConfigRequest{
			TypeName: cfg.Type,
			Config:   configVal,
		}

		resp := provider.ValidateResourceTypeConfig(req)
		diags = diags.Append(resp.Diagnostics.InConfigBody(cfg.Config))

		if n.ConfigVal != nil {
			*n.ConfigVal = configVal
		}

	case addrs.DataResourceMode:
		schema, _ := schema.SchemaForResourceType(mode, cfg.Type)
		if schema == nil {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid data source",
				Detail:   fmt.Sprintf("The provider %s does not support data source %q.", cfg.ProviderConfigAddr(), cfg.Type),
				Subject:  &cfg.TypeRange,
			})
			return nil, diags.Err()
		}

		configVal, _, valDiags := ctx.EvaluateBlock(cfg.Config, schema, nil, keyData)
		diags = diags.Append(valDiags)
		if valDiags.HasErrors() {
			return nil, diags.Err()
		}

		req := providers.ValidateDataSourceConfigRequest{
			TypeName: cfg.Type,
			Config:   configVal,
		}

		resp := provider.ValidateDataSourceConfig(req)
		diags = diags.Append(resp.Diagnostics.InConfigBody(cfg.Config))
	}

	if n.IgnoreWarnings {
		// If we _only_ have warnings then we'll return nil.
		if diags.HasErrors() {
			return nil, diags.NonFatalErr()
		}
		return nil, nil
	} else {
		// We'll return an error if there are any diagnostics at all, even if
		// some of them are warnings.
		return nil, diags.NonFatalErr()
	}
}

func (n *EvalValidateResource) validateCount(ctx EvalContext, expr hcl.Expression) tfdiags.Diagnostics {
	if expr == nil {
		return nil
	}

	var diags tfdiags.Diagnostics

	countVal, countDiags := ctx.EvaluateExpr(expr, cty.Number, nil)
	diags = diags.Append(countDiags)
	if diags.HasErrors() {
		return diags
	}

	if countVal.IsNull() {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid count argument",
			Detail:   `The given "count" argument value is null. An integer is required.`,
			Subject:  expr.Range().Ptr(),
		})
		return diags
	}

	var err error
	countVal, err = convert.Convert(countVal, cty.Number)
	if err != nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid count argument",
			Detail:   fmt.Sprintf(`The given "count" argument value is unsuitable: %s.`, err),
			Subject:  expr.Range().Ptr(),
		})
		return diags
	}

	// If the value isn't known then that's the best we can do for now, but
	// we'll check more thoroughly during the plan walk.
	if !countVal.IsKnown() {
		return diags
	}

	// If we _do_ know the value, then we can do a few more checks here.
	var count int
	err = gocty.FromCtyValue(countVal, &count)
	if err != nil {
		// Isn't a whole number, etc.
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid count argument",
			Detail:   fmt.Sprintf(`The given "count" argument value is unsuitable: %s.`, err),
			Subject:  expr.Range().Ptr(),
		})
		return diags
	}

	if count < 0 {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid count argument",
			Detail:   `The given "count" argument value is unsuitable: count cannot be negative.`,
			Subject:  expr.Range().Ptr(),
		})
		return diags
	}

	return diags
}
