package terraform

import (
	"fmt"

	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/config/configschema"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/tfdiags"
	"github.com/zclconf/go-cty/cty"
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
	Provider *ResourceProvider
	Config   *configs.Provider
}

func (n *EvalValidateProvider) Eval(ctx EvalContext) (interface{}, error) {
	var diags tfdiags.Diagnostics
	provider := *n.Provider

	var sourceBody hcl.Body
	if n.Config != nil && n.Config.Config != nil {
		sourceBody = n.Config.Config
	} else {
		// If the provider configuration is implicit (no block in configuration
		// but referred to by resources) then we'll assume an empty body
		// as a placeholder.
		sourceBody = hcl.EmptyBody()
	}

	schema, err := provider.GetSchema(&ProviderSchemaRequest{})
	if err != nil {
		diags = diags.Append(err)
		return nil, diags.NonFatalErr()
	}
	if schema == nil {
		return nil, fmt.Errorf("no schema is available for %s", n.Addr)
	}

	configSchema := schema.Provider
	configBody := buildProviderConfig(ctx, n.Addr, sourceBody)
	configVal, configBody, evalDiags := ctx.EvaluateBlock(configBody, configSchema, nil, addrs.NoKey)
	diags = diags.Append(evalDiags)
	if evalDiags.HasErrors() {
		return nil, diags.NonFatalErr()
	}

	// The provider API expects our legacy ResourceConfig type, so we'll need
	// to shim here.
	rc := NewResourceConfigShimmed(configVal, configSchema)

	warns, errs := provider.Validate(rc)
	if len(warns) == 0 && len(errs) == 0 {
		return nil, nil
	}

	// FIXME: Once provider.Validate itself returns diagnostics, just
	// return diags.NonFatalErr() immediately here.
	for _, warn := range warns {
		diags = diags.Append(tfdiags.SimpleWarning(warn))
	}
	for _, err := range errs {
		diags = diags.Append(err)
	}

	return nil, diags.NonFatalErr()
}

// EvalValidateProvisioner is an EvalNode implementation that validates
// the configuration of a provisioner belonging to a resource.
type EvalValidateProvisioner struct {
	ResourceAddr addrs.ResourceInstance
	Provisioner  *ResourceProvisioner
	Schema       **configschema.Block
	Config       *configs.Provisioner
	ConnConfig   *configs.Connection
}

func (n *EvalValidateProvisioner) Eval(ctx EvalContext) (interface{}, error) {
	provisioner := *n.Provisioner
	config := *n.Config
	schema := *n.Schema

	var warns []string
	var errs []error

	var diags tfdiags.Diagnostics

	{
		// Validate the provisioner's own config first

		configVal, _, configDiags := ctx.EvaluateBlock(config.Config, schema, n.ResourceAddr, n.ResourceAddr.Key)
		diags = diags.Append(configDiags)
		if configDiags.HasErrors() {
			return nil, diags.Err()
		}

		if configVal == cty.NilVal {
			// Should never happen for a well-behaved EvaluateBlock implementation
			return nil, fmt.Errorf("EvaluateBlock returned nil value")
		}

		// The provisioner API still uses our legacy ResourceConfig type, so
		// we need to shim it.
		legacyRC := NewResourceConfigShimmed(configVal, schema)

		w, e := provisioner.Validate(legacyRC)
		warns = append(warns, w...)
		errs = append(errs, e...)

		// FIXME: Once the provisioner API itself returns diagnostics, just
		// return diags.NonFatalErr() here.
		for _, warn := range warns {
			diags = diags.Append(tfdiags.SimpleWarning(warn))
		}
		for _, err := range errs {
			diags = diags.Append(err)
		}
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
	_, _, configDiags := ctx.EvaluateBlock(config.Config, connectionBlockSupersetSchema, self, n.ResourceAddr.Key)
	diags = diags.Append(configDiags)

	return diags
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
		"user": {
			Type:     cty.String,
			Required: false,
		},
		"password": {
			Type:     cty.String,
			Required: false,
		},
		"host": {
			Type:     cty.String,
			Required: false,
		},
		"port": {
			Type:     cty.Number,
			Required: false,
		},
		"timeout": {
			Type:     cty.String,
			Required: false,
		},
		"script_path": {
			Type:     cty.String,
			Required: false,
		},

		// For type=ssh only (enforced in ssh communicator)
		"private_key": {
			Type:     cty.String,
			Required: false,
		},
		"host_key": {
			Type:     cty.String,
			Required: false,
		},
		"agent": {
			Type:     cty.Bool,
			Required: false,
		},
		"agent_identity": {
			Type:     cty.String,
			Required: false,
		},
		"bastion_host": {
			Type:     cty.String,
			Required: false,
		},
		"bastion_host_key": {
			Type:     cty.String,
			Required: false,
		},
		"bastion_port": {
			Type:     cty.Number,
			Required: false,
		},
		"bastion_user": {
			Type:     cty.String,
			Required: false,
		},
		"bastion_password": {
			Type:     cty.String,
			Required: false,
		},
		"bastion_private_key": {
			Type:     cty.String,
			Required: false,
		},

		// For type=winrm only (enforced in winrm communicator)
		"https": {
			Type:     cty.Bool,
			Required: false,
		},
		"insecure": {
			Type:     cty.Bool,
			Required: false,
		},
		"cacert": {
			Type:     cty.String,
			Required: false,
		},
		"use_ntlm": {
			Type:     cty.Bool,
			Required: false,
		},
	},
}

// EvalValidateResource is an EvalNode implementation that validates
// the configuration of a resource.
type EvalValidateResource struct {
	Addr           addrs.ResourceInstance
	Provider       *ResourceProvider
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

	var warns []string
	var errs []error

	// Provider entry point varies depending on resource mode, because
	// managed resources and data resources are two distinct concepts
	// in the provider abstraction.
	switch mode {
	case addrs.ManagedResourceMode:
		schema, exists := schema.ResourceTypes[cfg.Type]
		if !exists {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid resource type",
				Detail:   fmt.Sprintf("The provider %s does not support resource type %q.", cfg.ProviderConfigAddr(), cfg.Type),
				Subject:  &cfg.TypeRange,
			})
			return nil, diags.Err()
		}

		configVal, _, valDiags := ctx.EvaluateBlock(cfg.Config, schema, nil, n.Addr.Key)
		diags = diags.Append(valDiags)
		if valDiags.HasErrors() {
			return nil, diags.Err()
		}

		// The provider API still expects our legacy types, so we must do some
		// shimming here.
		legacyCfg := NewResourceConfigShimmed(configVal, schema)
		warns, errs = provider.ValidateResource(cfg.Type, legacyCfg)

		if n.ConfigVal != nil {
			*n.ConfigVal = configVal
		}

	case addrs.DataResourceMode:
		schema, exists := schema.DataSources[cfg.Type]
		if !exists {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid data source",
				Detail:   fmt.Sprintf("The provider %s does not support data source %q.", cfg.ProviderConfigAddr(), cfg.Type),
				Subject:  &cfg.TypeRange,
			})
			return nil, diags.Err()
		}

		configVal, _, valDiags := ctx.EvaluateBlock(cfg.Config, schema, nil, n.Addr.Key)
		diags = diags.Append(valDiags)
		if valDiags.HasErrors() {
			return nil, diags.Err()
		}

		// The provider API still expects our legacy types, so we must do some
		// shimming here.
		legacyCfg := NewResourceConfigShimmed(configVal, schema)
		warns, errs = provider.ValidateDataSource(cfg.Type, legacyCfg)

		if n.ConfigVal != nil {
			*n.ConfigVal = configVal
		}
	}

	// FIXME: Update the provider API to actually return diagnostics here,
	// and then we can remove all this shimming and use its diagnostics
	// directly.
	for _, warn := range warns {
		diags = diags.Append(tfdiags.SimpleWarning(warn))
	}
	for _, err := range errs {
		diags = diags.Append(err)
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
