package schema

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform/tfdiags"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/config/hcl2shim"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/terraform"
	ctyconvert "github.com/zclconf/go-cty/cty/convert"
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

func (b *Backend) PrepareConfig(configVal cty.Value) (cty.Value, tfdiags.Diagnostics) {
	if b == nil {
		return configVal, nil
	}
	var diags tfdiags.Diagnostics
	var err error

	// In order to use Transform below, this needs to be filled out completely
	// according the schema.
	configVal, err = b.CoreConfigSchema().CoerceValue(configVal)
	if err != nil {
		return configVal, diags.Append(err)
	}

	// lookup any required, top-level attributes that are Null, and see if we
	// have a Default value available.
	configVal, err = cty.Transform(configVal, func(path cty.Path, val cty.Value) (cty.Value, error) {
		// we're only looking for top-level attributes
		if len(path) != 1 {
			return val, nil
		}

		// nothing to do if we already have a value
		if !val.IsNull() {
			return val, nil
		}

		// get the Schema definition for this attribute
		getAttr, ok := path[0].(cty.GetAttrStep)
		// these should all exist, but just ignore anything strange
		if !ok {
			return val, nil
		}

		attrSchema := b.Schema[getAttr.Name]
		// continue to ignore anything that doesn't match
		if attrSchema == nil {
			return val, nil
		}

		// this is deprecated, so don't set it
		if attrSchema.Deprecated != "" || attrSchema.Removed != "" {
			return val, nil
		}

		// find a default value if it exists
		def, err := attrSchema.DefaultValue()
		if err != nil {
			diags = diags.Append(fmt.Errorf("error getting default for %q: %s", getAttr.Name, err))
			return val, err
		}

		// no default
		if def == nil {
			return val, nil
		}

		// create a cty.Value and make sure it's the correct type
		tmpVal := hcl2shim.HCL2ValueFromConfigValue(def)

		// helper/schema used to allow setting "" to a bool
		if val.Type() == cty.Bool && tmpVal.RawEquals(cty.StringVal("")) {
			// return a warning about the conversion
			diags = diags.Append("provider set empty string as default value for bool " + getAttr.Name)
			tmpVal = cty.False
		}

		val, err = ctyconvert.Convert(tmpVal, val.Type())
		if err != nil {
			diags = diags.Append(fmt.Errorf("error setting default for %q: %s", getAttr.Name, err))
		}

		return val, err
	})
	if err != nil {
		// any error here was already added to the diagnostics
		return configVal, diags
	}

	shimRC := b.shimConfig(configVal)
	warns, errs := schemaMap(b.Schema).Validate(shimRC)
	for _, warn := range warns {
		diags = diags.Append(tfdiags.SimpleWarning(warn))
	}
	for _, err := range errs {
		diags = diags.Append(err)
	}
	return configVal, diags
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
	diff, err := sm.Diff(nil, shimRC, nil, nil, true)
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
