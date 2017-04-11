package schema

import (
	"context"

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

func (b *Backend) Input(
	input terraform.UIInput,
	c *terraform.ResourceConfig) (*terraform.ResourceConfig, error) {
	if b == nil {
		return c, nil
	}

	return schemaMap(b.Schema).Input(input, c)
}

func (b *Backend) Validate(c *terraform.ResourceConfig) ([]string, []error) {
	if b == nil {
		return nil, nil
	}

	return schemaMap(b.Schema).Validate(c)
}

func (b *Backend) Configure(c *terraform.ResourceConfig) error {
	if b == nil {
		return nil
	}

	sm := schemaMap(b.Schema)

	// Get a ResourceData for this configuration. To do this, we actually
	// generate an intermediary "diff" although that is never exposed.
	diff, err := sm.Diff(nil, c)
	if err != nil {
		return err
	}

	data, err := sm.Data(nil, diff)
	if err != nil {
		return err
	}
	b.config = data

	if b.ConfigureFunc != nil {
		err = b.ConfigureFunc(context.WithValue(
			context.Background(), backendConfigKey, data))
		if err != nil {
			return err
		}
	}

	return nil
}

// Config returns the configuration. This is available after Configure is
// called.
func (b *Backend) Config() *ResourceData {
	return b.config
}
