package schema

import (
	"context"

	"github.com/hashicorp/terraform/tfdiags"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/config/configschema"
	"github.com/hashicorp/terraform/config/hcl2shim"
	"github.com/hashicorp/terraform/terraform"
)

// Backend represents a partial backend.Backend implementation and simplifies
// the creation of configuration loading and validation.
//
// Unlike other schema structs such as Provider, this struct is meant to be
// embedded within your actual implementation. It provides implementations
// only for Input and Configure and gives you a method for accessing the
// configuration in the form of a ResourceData that you're expected to call
// from the other implementation funcs.
type Backend struct {
	// Schema is the schema for the configuration of this backend. If this
	// Backend has no configuration this can be omitted.
	Schema map[string]*Schema

	// ConfigureFunc is called to configure the backend. Use the
	// FromContext* methods to extract information from the context.
	// This can be nil, in which case nothing will be called but the
	// config will still be stored.
	ConfigureFunc func(context.Context) error

	config *ResourceData
}

var (
	backendConfigKey = contextKey("backend config")
)

// FromContextBackendConfig extracts a ResourceData with the configuration
// from the context. This should only be called by Backend functions.
func FromContextBackendConfig(ctx context.Context) *ResourceData {
	return ctx.Value(backendConfigKey).(*ResourceData)
}

func (b *Backend) ConfigSchema() *configschema.Block {
	// This is an alias of CoreConfigSchema just to implement the
	// backend.Backend interface.
	return b.CoreConfigSchema()
}

func (b *Backend) ValidateConfig(obj cty.Value) tfdiags.Diagnostics {
	if b == nil {
		return nil
	}

	var diags tfdiags.Diagnostics
	shimRC := b.shimConfig(obj)
	warns, errs := schemaMap(b.Schema).Validate(shimRC)
	for _, warn := range warns {
		diags = diags.Append(tfdiags.SimpleWarning(warn))
	}
	for _, err := range errs {
		diags = diags.Append(err)
	}
	return diags
}

func (b *Backend) Configure(obj cty.Value) tfdiags.Diagnostics {
	if b == nil {
		return nil
	}

	var diags tfdiags.Diagnostics
	sm := schemaMap(b.Schema)
	shimRC := b.shimConfig(obj)

	// Get a ResourceData for this configuration. To do this, we actually
	// generate an intermediary "diff" although that is never exposed.
	diff, err := sm.Diff(nil, shimRC, nil, nil)
	if err != nil {
		diags = diags.Append(err)
		return diags
	}

	data, err := sm.Data(nil, diff)
	if err != nil {
		diags = diags.Append(err)
		return diags
	}
	b.config = data

	if b.ConfigureFunc != nil {
		err = b.ConfigureFunc(context.WithValue(
			context.Background(), backendConfigKey, data))
		if err != nil {
			diags = diags.Append(err)
			return diags
		}
	}

	return diags
}

// shimConfig turns a new-style cty.Value configuration (which must be of
// an object type) into a minimal old-style *terraform.ResourceConfig object
// that should be populated enough to appease the not-yet-updated functionality
// in this package. This should be removed once everything is updated.
func (b *Backend) shimConfig(obj cty.Value) *terraform.ResourceConfig {
	shimMap := hcl2shim.ConfigValueFromHCL2(obj).(map[string]interface{})
	return &terraform.ResourceConfig{
		Config: shimMap,
		Raw:    shimMap,
	}
}

// Config returns the configuration. This is available after Configure is
// called.
func (b *Backend) Config() *ResourceData {
	return b.config
}
